package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"truck-inventory/agent"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/models/odataerrors"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/users/item/mailfolders"
	msgraphcore "github.com/microsoftgraph/msgraph-sdk-go-core"
)

type EmailMatcher struct {
	client              *msgraphsdk.GraphServiceClient
	db                  *DB
	userID              string
	readyDealsFolderID  string
}

type SearchCriteria struct {
	YearMin   *int
	YearMax   *int
	Make      *string
	Model     *string
	MaxPrice  *float64
	MinPrice  *float64
	MaxMiles  *int
	Location  *string
	RadiusMiles *int
}

func NewEmailMatcher(ctx context.Context, db *DB) (*EmailMatcher, error) {
	client, err := createGraphClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create Graph client: %w", err)
	}

	userID := getEnv("MICROSOFT_USER_ID", "")
	if userID == "" {
		return nil, fmt.Errorf("MICROSOFT_USER_ID environment variable is required for application permissions")
	}

	return &EmailMatcher{
		client: client,
		db:     db,
		userID: userID,
	}, nil
}

func createGraphClient(ctx context.Context) (*msgraphsdk.GraphServiceClient, error) {
	appID := getEnv("MICROSOFT_APP_ID", "")
	appSecret := getEnv("MICROSOFT_APP_SECRET", "")
	tenantID := getEnv("MICROSOFT_TENANT_ID", "")

	if appID == "" || appSecret == "" || tenantID == "" {
		return nil, fmt.Errorf("Microsoft Graph credentials not configured (MICROSOFT_APP_ID, MICROSOFT_APP_SECRET, MICROSOFT_TENANT_ID)")
	}

	// Create client secret credential for application permissions
	cred, err := azidentity.NewClientSecretCredential(tenantID, appID, appSecret, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create credential: %w", err)
	}

	// Create Graph client with credentials
	client, err := msgraphsdk.NewGraphServiceClientWithCredentials(cred, []string{"https://graph.microsoft.com/.default"})
	if err != nil {
		return nil, fmt.Errorf("failed to create Graph client: %w", err)
	}

	return client, nil
}

func (em *EmailMatcher) FetchNewEmails(ctx context.Context) error {
	days, _ := strconv.Atoi(getEnv("EMAIL_LOOKBACK_DAYS", "90"))
	if days <= 0 {
		days = 90
	}
	sinceTime := time.Now().Add(-time.Duration(days) * 24 * time.Hour).Format(time.RFC3339)
	requestFilter := fmt.Sprintf("receivedDateTime ge %s", sinceTime)

	top := int32(500)
	requestConfig := &mailfolders.ItemMailFoldersItemMessagesRequestBuilderGetRequestConfiguration{
		QueryParameters: &mailfolders.ItemMailFoldersItemMessagesRequestBuilderGetQueryParameters{
			Filter: &requestFilter,
			Select: []string{"from", "subject", "bodyPreview", "body", "receivedDateTime"},
			Top:    &top,
		},
	}

	result, err := em.client.Users().ByUserId(em.userID).MailFolders().ByMailFolderId("inbox").Messages().Get(ctx, requestConfig)
	if err != nil {
		if oDataError, ok := err.(*odataerrors.ODataError); ok {
			return fmt.Errorf("Graph API error: %s", oDataError.Error())
		}
		return fmt.Errorf("failed to fetch emails: %w", err)
	}

	pageIterator, err := msgraphcore.NewPageIterator[models.Messageable](result, em.client.GetAdapter(), models.CreateMessageCollectionResponseFromDiscriminatorValue)
	if err != nil {
		return fmt.Errorf("failed to create page iterator: %w", err)
	}

	var total int
	err = pageIterator.Iterate(ctx, func(message models.Messageable) bool {
		total++
		if err := em.processEmail(ctx, message); err != nil {
			log.Printf("  Error processing email: %v", err)
		}
		return true
	})
	if err != nil {
		if oDataError, ok := err.(*odataerrors.ODataError); ok {
			return fmt.Errorf("Graph API error: %s", oDataError.Error())
		}
		return fmt.Errorf("failed to iterate emails: %w", err)
	}

	log.Printf("Processing %d emails (lookback %d days)", total, days)
	return nil
}

func (em *EmailMatcher) processEmail(ctx context.Context, message models.Messageable) error {
	from := message.GetFrom()
	if from == nil {
		return fmt.Errorf("email has no from address")
	}

	emailAddress := from.GetEmailAddress()
	if emailAddress == nil {
		return fmt.Errorf("email has no email address")
	}

	fromEmail := emailAddress.GetAddress()
	if fromEmail == nil {
		return fmt.Errorf("email address is nil")
	}

	subject := ""
	if message.GetSubject() != nil {
		subject = *message.GetSubject()
	}

	body := ""
	if message.GetBodyPreview() != nil {
		body = *message.GetBodyPreview()
	} else if message.GetBody() != nil && message.GetBody().GetContent() != nil {
		body = *message.GetBody().GetContent()
	}

	receivedAt := message.GetReceivedDateTime()
	if receivedAt == nil {
		return fmt.Errorf("email has no received date")
	}

	// Check if already processed
	var existingID int
	err := em.db.pool.QueryRow(ctx,
		"SELECT id FROM email_search_requests WHERE from_email = $1 AND subject = $2 AND received_at = $3 LIMIT 1",
		*fromEmail, subject, *receivedAt,
	).Scan(&existingID)

	if err == nil {
		return nil // Already processed
	}

	// Extract search criteria (SLM with regex fallback)
	criteria := em.extractCriteriaWithFallback(ctx, subject, body)

	// Save email request
	req := &EmailSearchRequest{
		FromEmail:       *fromEmail,
		FromName:        emailAddress.GetName(),
		Subject:         &subject,
		Body:            &body,
		SearchYearMin:   criteria.YearMin,
		SearchYearMax:   criteria.YearMax,
		SearchMake:      criteria.Make,
		SearchModel:     criteria.Model,
		SearchMaxPrice:  criteria.MaxPrice,
		SearchMinPrice:  criteria.MinPrice,
		SearchMaxMiles:  criteria.MaxMiles,
		SearchLocation:  criteria.Location,
		ReceivedAt:      receivedAt,
	}

	requestID, err := em.db.SaveEmailRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to save email request: %w", err)
	}

	// Find matching trucks
	matches, err := em.db.FindMatchingListings(ctx, criteria)
	if err != nil {
		return fmt.Errorf("failed to find matches: %w", err)
	}

	log.Printf("  → Found %d matching trucks for %s", len(matches), *fromEmail)

	// Save matches
	for _, match := range matches {
		if err := em.db.SaveMatch(ctx, requestID, match.ID, 0.95); err != nil {
			log.Printf("    ⚠️  Error saving match: %v", err)
		}
	}

	// Create Ready Deals draft if matches found (no automated sending)
	if len(matches) > 0 {
		if em.readyDealsFolderID == "" {
			folderID, err := EnsureReadyDealsFolder(ctx, em.client, em.userID)
			if err != nil {
				log.Printf("  EnsureReadyDealsFolder failed: %v", err)
			} else {
				em.readyDealsFolderID = folderID
			}
		}
		if em.readyDealsFolderID != "" {
			if err := CreateReadyDealsDraft(ctx, em.client, em.userID, em.readyDealsFolderID, req, requestID, criteria, matches); err != nil {
				log.Printf("  CreateReadyDealsDraft failed: %v", err)
			}
		}
	}

	return nil
}

func (em *EmailMatcher) extractCriteriaWithFallback(ctx context.Context, subject, body string) *SearchCriteria {
	ac, err := agent.ExtractCriteria(ctx, subject, body)
	if err != nil || ac == nil {
		return em.extractSearchCriteria(subject, body)
	}
	return agentCriteriaToSearchCriteria(ac)
}

func agentCriteriaToSearchCriteria(c *agent.Criteria) *SearchCriteria {
	if c == nil {
		return &SearchCriteria{}
	}
	return &SearchCriteria{
		YearMin:   c.YearMin,
		YearMax:   c.YearMax,
		Make:      c.Make,
		Model:     c.Model,
		MaxPrice:  c.MaxPrice,
		MinPrice:  c.MinPrice,
		MaxMiles:  c.MaxMiles,
		Location:  c.Location,
	}
}

func (em *EmailMatcher) extractSearchCriteria(subject, body string) *SearchCriteria {
	criteria := &SearchCriteria{}
	text := strings.ToLower(subject + " " + body)

	// Year patterns: "2023", "2020-2024", "from 2020 to 2024"
	yearPattern := regexp.MustCompile(`(\d{4})\s*(?:to|-)\s*(\d{4})|year.*?(\d{4})`)
	yearMatch := yearPattern.FindStringSubmatch(text)
	if yearMatch != nil {
		if yearMatch[1] != "" && yearMatch[2] != "" {
			if min, err := strconv.Atoi(yearMatch[1]); err == nil {
				criteria.YearMin = &min
			}
			if max, err := strconv.Atoi(yearMatch[2]); err == nil {
				criteria.YearMax = &max
			}
		} else if yearMatch[3] != "" {
			if year, err := strconv.Atoi(yearMatch[3]); err == nil {
				criteria.YearMin = &year
				criteria.YearMax = &year
			}
		}
	}

	// Make/Model: "Peterbilt", "Freightliner 114SD", "Volvo VNL"
	makePattern := regexp.MustCompile(`(peterbilt|freightliner|volvo|kenworth|mack|international|daimler|cummins)`)
	makeMatch := makePattern.FindStringSubmatch(text)
	if makeMatch != nil {
		make := strings.ToUpper(makeMatch[1])
		criteria.Make = &make

		// Try to extract model
		modelPattern := regexp.MustCompile(fmt.Sprintf(`(?:%s)[\s-]+([\w\d]+)`, makeMatch[1]))
		modelMatch := modelPattern.FindStringSubmatch(text)
		if modelMatch != nil {
			model := strings.ToUpper(modelMatch[1])
			criteria.Model = &model
		}
	}

	// Price: "$50000", "$50k", "under $100000"
	pricePattern := regexp.MustCompile(`\$?(\d+)(?:k|000)?`)
	priceMatch := pricePattern.FindStringSubmatch(text)
	if priceMatch != nil {
		price, err := strconv.Atoi(priceMatch[1])
		if err == nil {
			if strings.Contains(priceMatch[0], "k") {
				price *= 1000
			}
			priceFloat := float64(price)
			criteria.MaxPrice = &priceFloat
			minPrice := priceFloat * 0.8
			criteria.MinPrice = &minPrice
		}
	}

	// Miles: "100000 miles", "under 500k miles"
	milesPattern := regexp.MustCompile(`(\d+)\s*(?:k|000)?\s*miles`)
	milesMatch := milesPattern.FindStringSubmatch(text)
	if milesMatch != nil {
		miles, err := strconv.Atoi(milesMatch[1])
		if err == nil {
			if strings.Contains(milesMatch[0], "k") {
				miles *= 1000
			}
			criteria.MaxMiles = &miles
		}
	}

	// Location: city names, state abbreviations
	locationPattern := regexp.MustCompile(`(?:in|near|around)\s+([a-z\s]+?)(?:,|\.|$)`)
	locationMatch := locationPattern.FindStringSubmatch(text)
	if locationMatch != nil {
		location := strings.TrimSpace(locationMatch[1])
		criteria.Location = &location
	}

	return criteria
}


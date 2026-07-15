package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/microsoftgraph/msgraph-sdk-go/models"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
)

const readyDealsFolderName = "Ready Deals"

// EnsureReadyDealsFolder finds or creates a "Ready Deals" child of Inbox and returns its ID.
func EnsureReadyDealsFolder(ctx context.Context, client *msgraphsdk.GraphServiceClient, userID string) (string, error) {
	childFolders, err := client.Users().ByUserId(userID).MailFolders().ByMailFolderId("inbox").ChildFolders().Get(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("list inbox child folders: %w", err)
	}

	if childFolders != nil && childFolders.GetValue() != nil {
		for _, f := range childFolders.GetValue() {
			if n := f.GetDisplayName(); n != nil && *n == readyDealsFolderName {
				if id := f.GetId(); id != nil {
					return *id, nil
				}
			}
		}
	}

	folder := models.NewMailFolder()
	folder.SetDisplayName(&readyDealsFolderName)
	created, err := client.Users().ByUserId(userID).MailFolders().ByMailFolderId("inbox").ChildFolders().Post(ctx, folder, nil)
	if err != nil {
		return "", fmt.Errorf("create Ready Deals folder: %w", err)
	}
	if id := created.GetId(); id != nil {
		return *id, nil
	}
	return "", fmt.Errorf("create Ready Deals folder: no id in response")
}

// CreateReadyDealsDraft creates a draft in the Ready Deals folder with matches.csv and inquiry.txt attachments.
// It does not send the message.
func CreateReadyDealsDraft(ctx context.Context, client *msgraphsdk.GraphServiceClient, userID, folderID string, req *EmailSearchRequest, requestID int, criteria *SearchCriteria, matches []*TruckListing) error {
	subject := "Ready: "
	if req.Subject != nil && *req.Subject != "" {
		subject += *req.Subject
	} else {
		subject = fmt.Sprintf("Deal %d - %s", requestID, req.FromEmail)
	}

	bodySnippet := ""
	if req.Body != nil && len(*req.Body) > 200 {
		bodySnippet = (*req.Body)[:200] + "..."
	} else if req.Body != nil {
		bodySnippet = *req.Body
	}

	var tableRows strings.Builder
	top := matches
	if len(top) > 5 {
		top = top[:5]
	}
	for _, m := range top {
		year := ""
		if m.Year != nil {
			year = strconv.Itoa(*m.Year)
		}
		make := ""
		if m.Make != nil {
			make = *m.Make
		}
		model := ""
		if m.Model != nil {
			model = *m.Model
		}
		price := ""
		if m.Price != nil {
			price = fmt.Sprintf("%.0f", *m.Price)
		}
		miles := ""
		if m.Miles != nil {
			miles = strconv.Itoa(*m.Miles)
		}
		loc := ""
		if m.Location != nil {
			loc = *m.Location
		}
		url := m.URL
		tableRows.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%s</td><td>$%s</td><td>%s</td><td>%s</td><td><a href=\"%s\">Link</a></td></tr>", year, make, model, price, miles, loc, url))
	}

	html := fmt.Sprintf("<p>Inquiry from %s</p><p><b>Subject:</b> %s</p><p><b>Body snippet:</b> %s</p><p>Top matches:</p><table border=\"1\"><tr><th>Year</th><th>Make</th><th>Model</th><th>Price</th><th>Miles</th><th>Location</th><th>URL</th></tr>%s</table>",
		req.FromEmail,
		ptrStr(req.Subject),
		bodySnippet,
		tableRows.String())

	message := models.NewMessage()
	message.SetSubject(&subject)
	message.SetIsDraft(true)

	body := models.NewItemBody()
	ct := "Html"
	body.SetContentType(&ct)
	body.SetContent(&html)
	message.SetBody(body)

	toRecipient := models.NewRecipient()
	addr := models.NewEmailAddress()
	addr.SetAddress(&req.FromEmail)
	if req.FromName != nil {
		addr.SetName(req.FromName)
	}
	toRecipient.SetEmailAddress(addr)
	message.SetToRecipients([]models.Recipientable{toRecipient})

	draft, err := client.Users().ByUserId(userID).MailFolders().ByMailFolderId(folderID).Messages().Post(ctx, message, nil)
	if err != nil {
		return fmt.Errorf("create draft: %w", err)
	}
	draftID := draft.GetId()
	if draftID == nil || *draftID == "" {
		return fmt.Errorf("create draft: no id in response")
	}

	// matches.csv
	csvBuf := new(bytes.Buffer)
	w := csv.NewWriter(csvBuf)
	_ = w.Write([]string{"id", "marketplace", "title", "year", "make", "model", "price", "miles", "location", "url", "description", "vin", "stock_number"})
	for _, m := range matches {
		_ = w.Write([]string{
			strconv.Itoa(m.ID),
			m.Marketplace,
			ptrStr(m.Title),
			ptrInt(m.Year),
			ptrStr(m.Make),
			ptrStr(m.Model),
			ptrFloat(m.Price),
			ptrInt(m.Miles),
			ptrStr(m.Location),
			m.URL,
			ptrStr(m.Description),
			ptrStr(m.VIN),
			ptrStr(m.StockNumber),
		})
	}
	w.Flush()
	csvBytes := csvBuf.Bytes()

	csvAttachment := models.NewFileAttachment()
	csvName := "matches.csv"
	csvAttachment.SetName(&csvName)
	csvCT := "text/csv"
	csvAttachment.SetContentType(&csvCT)
	csvAttachment.SetContentBytes(csvBytes)
	_, err = client.Users().ByUserId(userID).Messages().ByMessageId(*draftID).Attachments().Post(ctx, csvAttachment, nil)
	if err != nil {
		log.Printf("  Failed to attach matches.csv: %v", err)
		// continue; inquiry.txt is optional
	}

	// inquiry.txt
	critStr := criteriaSummary(criteria)
	recvAt := ""
	if req.ReceivedAt != nil {
		recvAt = req.ReceivedAt.Format(time.RFC3339)
	}
	inquiry := fmt.Sprintf("request_id: %d\nfrom_email: %s\nfrom_name: %s\nsubject: %s\nbody: %s\nreceived_at: %s\nextracted_criteria: %s\n",
		requestID, req.FromEmail, ptrStr(req.FromName), ptrStr(req.Subject), ptrStr(req.Body), recvAt, critStr)

	inqAttachment := models.NewFileAttachment()
	inqName := "inquiry.txt"
	inqAttachment.SetName(&inqName)
	inqCT := "text/plain"
	inqAttachment.SetContentType(&inqCT)
	inqAttachment.SetContentBytes([]byte(inquiry))
	_, err = client.Users().ByUserId(userID).Messages().ByMessageId(*draftID).Attachments().Post(ctx, inqAttachment, nil)
	if err != nil {
		log.Printf("  Failed to attach inquiry.txt: %v", err)
	}

	log.Printf("  Created Ready Deals draft for %s (%d matches)", req.FromEmail, len(matches))
	return nil
}

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ptrInt(i *int) string {
	if i == nil {
		return ""
	}
	return strconv.Itoa(*i)
}

func ptrFloat(f *float64) string {
	if f == nil {
		return ""
	}
	return strconv.FormatFloat(*f, 'f', -1, 64)
}

func criteriaSummary(c *SearchCriteria) string {
	if c == nil {
		return ""
	}
	var parts []string
	if c.YearMin != nil {
		parts = append(parts, fmt.Sprintf("year_min=%d", *c.YearMin))
	}
	if c.YearMax != nil {
		parts = append(parts, fmt.Sprintf("year_max=%d", *c.YearMax))
	}
	if c.Make != nil {
		parts = append(parts, fmt.Sprintf("make=%s", *c.Make))
	}
	if c.Model != nil {
		parts = append(parts, fmt.Sprintf("model=%s", *c.Model))
	}
	if c.MaxPrice != nil {
		parts = append(parts, fmt.Sprintf("max_price=%.0f", *c.MaxPrice))
	}
	if c.MinPrice != nil {
		parts = append(parts, fmt.Sprintf("min_price=%.0f", *c.MinPrice))
	}
	if c.MaxMiles != nil {
		parts = append(parts, fmt.Sprintf("max_miles=%d", *c.MaxMiles))
	}
	if c.Location != nil {
		parts = append(parts, fmt.Sprintf("location=%s", *c.Location))
	}
	return strings.Join(parts, ", ")
}

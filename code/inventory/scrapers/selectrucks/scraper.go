package selectrucks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// SelecTrucksScraper handles HTTP-based scraping for SelecTrucks.com
type SelecTrucksScraper struct {
	client      *http.Client
	baseURL     string
	apiEndpoint string
	pageSize    int
}

// NewSelecTrucksScraper creates a new HTTP scraper for SelecTrucks
func NewSelecTrucksScraper() *SelecTrucksScraper {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &SelecTrucksScraper{
		client:      client,
		baseURL:     "https://www.selectrucks.com",
		apiEndpoint: "/api/v1/inventory/search/",
		pageSize:    50, // Use 50 per page for balance between speed and API limits
	}
}

// ScrapePage scrapes a single page of listings
func (s *SelecTrucksScraper) ScrapePage(pageNum int) ([]*agent.ListingExtract, *SearchResponse, error) {
	// Build JSON payload
	payload := SearchRequest{
		Page:     pageNum,
		PageSize: s.pageSize,
		Filters:  make(map[string]interface{}), // Empty filters to get all trucks
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create POST request
	reqURL := s.baseURL + s.apiEndpoint
	req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Origin", s.baseURL)
	req.Header.Set("Referer", s.baseURL+"/")

	// Retry logic
	maxRetries := 3
	var resp *http.Response

	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err = s.client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}

		if resp != nil {
			resp.Body.Close()
		}

		if attempt < maxRetries {
			backoff := time.Duration(attempt) * time.Second
			if err != nil {
				log.Printf("  Page %d request attempt %d failed: %v, retrying in %v...", pageNum, attempt, err, backoff)
			} else {
				log.Printf("  Page %d returned status %d, retrying in %v...", pageNum, resp.StatusCode, backoff)
			}
			time.Sleep(backoff)
		}
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch page %d after %d attempts: %w", pageNum, maxRetries, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("unexpected status code %d for page %d", resp.StatusCode, pageNum)
	}

	// Parse JSON response
	var searchResp SearchResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&searchResp); err != nil {
		return nil, nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	extracts := make([]*agent.ListingExtract, 0, len(searchResp.Items))
	for i := range searchResp.Items {
		ext := s.ConvertTruckToExtract(&searchResp.Items[i])
		if ext != nil {
			extracts = append(extracts, ext)
		}
	}

	return extracts, &searchResp, nil
}

// ConvertTruckToExtract converts API Truck to agent.ListingExtract.
// Returns nil when TruckId <= 0 (cannot build URL).
func (s *SelecTrucksScraper) ConvertTruckToExtract(truck *Truck) *agent.ListingExtract {
	if truck.TruckId <= 0 {
		return nil
	}
	ext := &agent.ListingExtract{
		URL: fmt.Sprintf("%s/trucks/%d", s.baseURL, truck.TruckId),
	}

	if truck.Manufacturer != "" {
		v := strings.TrimSpace(truck.Manufacturer)
		ext.Make = &v
	}
	if truck.Model != "" {
		v := strings.TrimSpace(truck.Model)
		ext.Model = &v
	}
	if truck.Year > 0 {
		ext.Year = &truck.Year
	}
	if truck.Price != "" {
		if v := parsePrice(truck.Price); v != nil {
			ext.Price = v
		}
	}
	if truck.Mileage != "" {
		if v := parseMiles(truck.Mileage); v != nil {
			ext.Miles = v
		}
	}
	if truck.StockNumber != "" {
		v := strings.TrimSpace(truck.StockNumber)
		ext.StockNumber = &v
	}
	if truck.ImageFileName != "" {
		u := s.constructImageURL(truck.ImageFileName)
		ext.ImageURL = &u
	}
	if truck.DealerName != "" {
		loc := truck.DealerName
		if truck.DealerCountryAbbreviation != "" {
			loc = fmt.Sprintf("%s, %s", loc, truck.DealerCountryAbbreviation)
		}
		ext.Location = &loc
	}

	if ext.Year != nil && ext.Make != nil && ext.Model != nil {
		v := fmt.Sprintf("%d %s %s", *ext.Year, *ext.Make, *ext.Model)
		ext.Title = &v
	} else if ext.Make != nil && ext.Model != nil {
		v := fmt.Sprintf("%s %s", *ext.Make, *ext.Model)
		ext.Title = &v
	}

	return ext
}

// constructImageURL builds the full image URL from ImageFileName
func (s *SelecTrucksScraper) constructImageURL(imageFileName string) string {
	// ImageFileName might be relative or absolute
	if strings.HasPrefix(imageFileName, "http") {
		return imageFileName
	}

	// Construct full URL (typical pattern: /images/trucks/{filename} or similar)
	if strings.HasPrefix(imageFileName, "/") {
		return s.baseURL + imageFileName
	}

	// If it's just a filename, construct path
	return fmt.Sprintf("%s/images/trucks/%s", s.baseURL, imageFileName)
}

// ScrapeAllPages scrapes all pages until no more listings are found
func (s *SelecTrucksScraper) ScrapeAllPages(ctx context.Context) ([]*agent.ListingExtract, error) {
	allExtracts := []*agent.ListingExtract{}
	page := 1
	total := 0
	maxPages := 1000

	for page <= maxPages {
		select {
		case <-ctx.Done():
			return allExtracts, ctx.Err()
		default:
		}

		log.Printf("  Scraping page %d...", page)

		extracts, resp, err := s.ScrapePage(page)
		if err != nil {
			return allExtracts, fmt.Errorf("error scraping page %d: %w", page, err)
		}

		if total == 0 && resp != nil {
			total = resp.Total
			log.Printf("  Total trucks available: %d", total)
		}

		if len(extracts) == 0 {
			log.Printf("  No more listings found at page %d", page)
			break
		}

		allExtracts = append(allExtracts, extracts...)
		log.Printf("  Found %d listings on page %d (total: %d)", len(extracts), page, len(allExtracts))

		if total > 0 && page*s.pageSize >= total {
			log.Printf("  Reached end (page %d * %d >= total %d)", page, s.pageSize, total)
			break
		}

		page++
		time.Sleep(1*time.Second + time.Duration(page%2)*time.Second)
	}

	return allExtracts, nil
}

// Helper functions for parsing (reused from other scrapers)

func parsePrice(text string) *float64 {
	re := regexp.MustCompile(`\$?([\d,]+(?:\.\d{2})?)`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		priceStr := strings.ReplaceAll(matches[1], ",", "")
		if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
			return &price
		}
	}
	return nil
}

func parseMiles(text string) *int {
	re := regexp.MustCompile(`(\d+(?:,\d{3})*)`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		milesStr := strings.ReplaceAll(matches[1], ",", "")
		if miles, err := strconv.Atoi(milesStr); err == nil {
			return &miles
		}
	}
	return nil
}

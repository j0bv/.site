package heavytruckdealers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"truck-inventory/agent"
)

// HeavyTruckDealersScraper handles HTTP-based scraping for HeavyTruckDealers.com
type HeavyTruckDealersScraper struct {
	client      *http.Client
	baseURL     string
	apiEndpoint string
}

// NewHeavyTruckDealersScraper creates a new HTTP scraper for HeavyTruckDealers
func NewHeavyTruckDealersScraper() *HeavyTruckDealersScraper {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &HeavyTruckDealersScraper{
		client:      client,
		baseURL:     "https://heavytruckdealers.com",
		apiEndpoint: "/wp-json/facetwp/v1/refresh",
	}
}

// ScrapePage scrapes a single page of listings
func (s *HeavyTruckDealersScraper) ScrapePage(ctx context.Context, pageNum int) ([]*agent.ListingExtract, error) {
	// Build JSON payload
	payload := FacetWPRequest{
		Action: "facetwp_refresh",
		Data: RequestData{
			Paged:        pageNum,
			Facets:       make(map[string]interface{}),
			FrozenFacets: make(map[string]interface{}),
			HTTPParams: HTTPParams{
				Get: []string{},
				URI: "listings/",
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create POST request
	reqURL := s.baseURL + s.apiEndpoint
	req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Origin", s.baseURL)
	req.Header.Set("Referer", s.baseURL+"/listings/")

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
		return nil, fmt.Errorf("failed to fetch page %d after %d attempts: %w", pageNum, maxRetries, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d for page %d", resp.StatusCode, pageNum)
	}

	// Parse JSON response
	var facetWPResp FacetWPResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&facetWPResp); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	// Extract listings HTML from template field
	listingsHTML, err := s.ExtractListingsHTML(&facetWPResp)
	if err != nil {
		return nil, fmt.Errorf("failed to extract listings HTML: %w", err)
	}

	if listingsHTML == "" {
		return []*agent.ListingExtract{}, nil
	}

	// LLM-based extraction (or shared fallback when SLM disabled)
	return agent.ExtractListingsFromHTML(ctx, listingsHTML, s.baseURL)
}

// ExtractListingsHTML extracts the HTML from the "template" field
func (s *HeavyTruckDealersScraper) ExtractListingsHTML(resp *FacetWPResponse) (string, error) {
	if resp.Template == "" {
		return "", fmt.Errorf("template field is empty")
	}
	return resp.Template, nil
}

// ScrapeAllPages scrapes all pages until no more listings are found
func (s *HeavyTruckDealersScraper) ScrapeAllPages(ctx context.Context) ([]*agent.ListingExtract, error) {
	allListings := []*agent.ListingExtract{}
	page := 1
	maxPages := 100 // Safety limit

	for page <= maxPages {
		select {
		case <-ctx.Done():
			return allListings, ctx.Err()
		default:
		}

		log.Printf("  Scraping page %d...", page)

		listings, err := s.ScrapePage(ctx, page)
		if err != nil {
			return allListings, fmt.Errorf("error scraping page %d: %w", page, err)
		}

		if len(listings) == 0 {
			log.Printf("  No more listings found at page %d", page)
			break
		}

		allListings = append(allListings, listings...)
		log.Printf("  Found %d listings on page %d (total: %d)", len(listings), page, len(allListings))

		page++

		// Human-like delay between pages
		time.Sleep(1*time.Second + time.Duration(page%2)*time.Second)
	}

	return allListings, nil
}

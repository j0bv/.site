package arrowtruck

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"truck-inventory/agent"
)

// ArrowTruckScraper handles HTTP-based scraping for ArrowTruck.com
type ArrowTruckScraper struct {
	client      *http.Client
	baseURL     string
	apiEndpoint string
	pageSize    int
}

// NewArrowTruckScraper creates a new HTTP scraper for ArrowTruck
func NewArrowTruckScraper() *ArrowTruckScraper {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &ArrowTruckScraper{
		client:      client,
		baseURL:     "https://www.arrowtruck.com",
		apiEndpoint: "https://api.arrowl5secure.com/services/searchproducts",
		pageSize:    30, // Use 30 per page as shown in API response
	}
}

// ScrapePage scrapes a single page of listings
func (s *ArrowTruckScraper) ScrapePage(pageNum int) ([]*agent.ListingExtract, *SearchResponse, error) {
	// Build JSON payload
	payload := SearchRequest{
		PageIndex: pageNum,
		PageSize:  s.pageSize,
		// Empty filters to get all trucks
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", s.apiEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

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

	extracts := make([]*agent.ListingExtract, 0, len(searchResp.Products))
	for i := range searchResp.Products {
		ext := s.ConvertProductToExtract(&searchResp.Products[i])
		if ext != nil {
			extracts = append(extracts, ext)
		}
	}

	return extracts, &searchResp, nil
}

// ConvertProductToExtract converts API Product to agent.ListingExtract.
// Returns nil when StockNumber is empty (cannot build URL).
func (s *ArrowTruckScraper) ConvertProductToExtract(product *Product) *agent.ListingExtract {
	if product.StockNumber == "" {
		return nil
	}
	ext := &agent.ListingExtract{
		URL: fmt.Sprintf("%s/trucks/%s", s.baseURL, product.StockNumber),
	}

	if product.Make != "" {
		v := strings.TrimSpace(product.Make)
		ext.Make = &v
	}
	if product.Model != "" {
		v := strings.TrimSpace(product.Model)
		ext.Model = &v
	}
	if product.Year > 0 {
		ext.Year = &product.Year
	}
	if product.Price.Value > 0 {
		v := product.Price.Value
		ext.Price = &v
	}
	if product.Mileage.Value > 0 {
		v := product.Mileage.Value
		ext.Miles = &v
	}
	if product.StockNumber != "" {
		v := strings.TrimSpace(product.StockNumber)
		ext.StockNumber = &v
	}

	for _, mediaProvider := range product.Media {
		if mediaProvider.ProviderName == "Arrow" && len(mediaProvider.Media.Photos) > 0 {
			if u := mediaProvider.Media.Photos[0].MediaURL; u != "" {
				ext.ImageURL = &u
				break
			}
		}
	}
	if ext.ImageURL == nil {
		for _, mediaProvider := range product.Media {
			if len(mediaProvider.Media.Photos) > 0 {
				if u := mediaProvider.Media.Photos[0].MediaURL; u != "" {
					ext.ImageURL = &u
					break
				}
			}
		}
	}

	for _, mediaProvider := range product.Media {
		if mediaProvider.ProviderName == "Glo3D" && mediaProvider.Media.VIN != "" {
			v := strings.TrimSpace(mediaProvider.Media.VIN)
			ext.VIN = &v
			break
		}
	}

	if product.LocationCity != "" || product.LocationState != "" {
		var parts []string
		if product.LocationCity != "" {
			parts = append(parts, strings.TrimSpace(product.LocationCity))
		}
		if product.LocationState != "" {
			parts = append(parts, strings.TrimSpace(product.LocationState))
		}
		if len(parts) > 0 {
			v := strings.Join(parts, ", ")
			ext.Location = &v
		}
	}

	if product.Description != "" {
		v := strings.TrimSpace(product.Description)
		ext.Description = &v
	} else if product.Headline != "" {
		v := strings.TrimSpace(product.Headline)
		ext.Description = &v
	}

	if product.Headline != "" {
		v := strings.TrimSpace(product.Headline)
		ext.Title = &v
	} else if ext.Year != nil && ext.Make != nil && ext.Model != nil {
		v := fmt.Sprintf("%d %s %s", *ext.Year, *ext.Make, *ext.Model)
		ext.Title = &v
	} else if ext.Make != nil && ext.Model != nil {
		v := fmt.Sprintf("%s %s", *ext.Make, *ext.Model)
		ext.Title = &v
	}

	return ext
}

// ScrapeAllPages scrapes all pages until no more listings are found
func (s *ArrowTruckScraper) ScrapeAllPages(ctx context.Context) ([]*agent.ListingExtract, error) {
	allExtracts := []*agent.ListingExtract{}
	page := 1
	totalPages := 0
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

		if totalPages == 0 && resp != nil {
			totalPages = resp.TotalPages
			log.Printf("  Total pages: %d, Total trucks: %d", totalPages, resp.TotalItemCount)
		}

		if len(extracts) == 0 {
			log.Printf("  No more listings found at page %d", page)
			break
		}

		allExtracts = append(allExtracts, extracts...)
		log.Printf("  Found %d listings on page %d (total: %d)", len(extracts), page, len(allExtracts))

		if totalPages > 0 && page >= totalPages {
			log.Printf("  Reached end (page %d >= totalPages %d)", page, totalPages)
			break
		}

		page++
		time.Sleep(1*time.Second + time.Duration(page%2)*time.Second)
	}

	return allExtracts, nil
}

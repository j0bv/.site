package trucktractortrailer

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

// TruckTractorTrailerScraper handles HTTP-based scraping for TruckTractorTrailer.com
type TruckTractorTrailerScraper struct {
	client      *http.Client
	baseURL     string
	apiEndpoint string
}

// NewTruckTractorTrailerScraper creates a new HTTP scraper for TruckTractorTrailer
func NewTruckTractorTrailerScraper() *TruckTractorTrailerScraper {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &TruckTractorTrailerScraper{
		client:      client,
		baseURL:     "https://trucktractortrailer.com",
		apiEndpoint: "/filter",
	}
}

// ScrapePage scrapes a single page of listings
func (s *TruckTractorTrailerScraper) ScrapePage(ctx context.Context, pageNum int) ([]*agent.ListingExtract, error) {
	payload := map[string]interface{}{
		"page":      pageNum,
		"make":      "",
		"model":     "",
		"yearMin":   "",
		"yearMax":   "",
		"priceMin":  "",
		"priceMax":  "",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

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
	req.Header.Set("Referer", s.baseURL+"/")

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
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page %d after %d attempts: %w", pageNum, maxRetries, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d for page %d", resp.StatusCode, pageNum)
	}

	var searchResp SearchResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	if searchResp.Listings == "" {
		return []*agent.ListingExtract{}, nil
	}

	return agent.ExtractListingsFromHTML(ctx, searchResp.Listings, s.baseURL)
}

// ScrapeAllPages scrapes all pages until no more listings are found
func (s *TruckTractorTrailerScraper) ScrapeAllPages(ctx context.Context) ([]*agent.ListingExtract, error) {
	allListings := []*agent.ListingExtract{}
	page := 1
	maxPages := 100

	for page <= maxPages {
		select {
		case <-ctx.Done():
			return allListings, ctx.Err()
		default:
		}

		log.Printf("  Scraping page %d...", page)

		listings, err := s.ScrapePage(ctx, page)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				break
			}
			return allListings, fmt.Errorf("error scraping page %d: %w", page, err)
		}

		if len(listings) == 0 {
			break
		}

		allListings = append(allListings, listings...)
		log.Printf("  Found %d listings on page %d (total: %d)", len(listings), page, len(allListings))

		page++
		time.Sleep(2*time.Second + time.Duration(page%3)*time.Second)
	}

	return allListings, nil
}

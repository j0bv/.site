package truckplanet

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"truck-inventory/agent"
)

// TruckPlanetScraper handles HTTP-based scraping for TruckPlanet.com
type TruckPlanetScraper struct {
	client      *http.Client
	baseURL     string
	apiEndpoint string
	rowsPerPage int
}

// NewTruckPlanetScraper creates a new HTTP scraper for TruckPlanet
func NewTruckPlanetScraper() *TruckPlanetScraper {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &TruckPlanetScraper{
		client:      client,
		baseURL:     "https://www.truckplanet.com",
		apiEndpoint: "/jsp/s/search.ips",
		rowsPerPage: 100,
	}
}

// ScrapePage scrapes a single page of listings using offset
func (s *TruckPlanetScraper) ScrapePage(ctx context.Context, offset int) ([]*agent.ListingExtract, error) {
	pageURL := fmt.Sprintf("%s%s?prows=%d&sm=%d&json=true", s.baseURL, s.apiEndpoint, s.rowsPerPage, offset)

	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
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
		return nil, fmt.Errorf("failed to fetch offset %d after %d attempts: %w", offset, maxRetries, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d for offset %d", resp.StatusCode, offset)
	}

	var searchResp SearchResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	listingsHTML, err := s.ExtractListingsHTML(&searchResp)
	if err != nil {
		return nil, fmt.Errorf("failed to extract listings HTML: %w", err)
	}

	if listingsHTML == "" {
		return []*agent.ListingExtract{}, nil
	}

	return agent.ExtractListingsFromHTML(ctx, listingsHTML, s.baseURL)
}

// ExtractListingsHTML finds and extracts the HTML from the "sr_results" div
func (s *TruckPlanetScraper) ExtractListingsHTML(resp *SearchResponse) (string, error) {
	for _, div := range resp.Divs {
		if div.ID == "sr_results" {
			return div.Value, nil
		}
	}
	return "", fmt.Errorf("sr_results div not found in response")
}

// ScrapeAllPages scrapes all pages until no more listings are found
func (s *TruckPlanetScraper) ScrapeAllPages(ctx context.Context) ([]*agent.ListingExtract, error) {
	allListings := []*agent.ListingExtract{}
	offset := 0
	maxOffset := 10000

	for offset <= maxOffset {
		select {
		case <-ctx.Done():
			return allListings, ctx.Err()
		default:
		}

		log.Printf("  Scraping offset %d (page %d)...", offset, (offset/s.rowsPerPage)+1)

		listings, err := s.ScrapePage(ctx, offset)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				break
			}
			return allListings, fmt.Errorf("error scraping offset %d: %w", offset, err)
		}

		if len(listings) == 0 {
			break
		}

		allListings = append(allListings, listings...)
		log.Printf("  Found %d listings at offset %d (total: %d)", len(listings), offset, len(allListings))

		if len(listings) < s.rowsPerPage {
			break
		}

		offset += s.rowsPerPage
		time.Sleep(2*time.Second + time.Duration((offset/s.rowsPerPage)%3)*time.Second)
	}

	return allListings, nil
}

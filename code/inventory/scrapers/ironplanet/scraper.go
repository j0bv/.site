package ironplanet

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"truck-inventory/agent"
)

// IronPlanetScraper handles HTTP-based scraping for IronPlanet.com
type IronPlanetScraper struct {
	client       *http.Client
	baseURL      string
	itemsPerPage int
}

// NewIronPlanetScraper creates a new HTTP scraper for IronPlanet
func NewIronPlanetScraper() *IronPlanetScraper {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &IronPlanetScraper{
		client:       client,
		baseURL:      "https://www.ironplanet.com",
		itemsPerPage: 60,
	}
}

// ScrapePage scrapes a single page of listings using offset
func (s *IronPlanetScraper) ScrapePage(ctx context.Context, offset int) ([]*agent.ListingExtract, error) {
	pageURL := fmt.Sprintf("%s/Truck+Tractors?sm=%d", s.baseURL, offset)

	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

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

	if resp.StatusCode == http.StatusNotFound {
		return []*agent.ListingExtract{}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d for offset %d", resp.StatusCode, offset)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	return agent.ExtractListingsFromHTML(ctx, string(bodyBytes), s.baseURL)
}

// ScrapeAllPages scrapes all pages until no more listings are found
func (s *IronPlanetScraper) ScrapeAllPages(ctx context.Context) ([]*agent.ListingExtract, error) {
	allListings := []*agent.ListingExtract{}
	offset := 0
	maxOffset := 10000

	for offset <= maxOffset {
		select {
		case <-ctx.Done():
			return allListings, ctx.Err()
		default:
		}

		log.Printf("  Scraping offset %d (page %d)...", offset, (offset/s.itemsPerPage)+1)

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

		if len(listings) < s.itemsPerPage {
			break
		}

		offset += s.itemsPerPage
		time.Sleep(2*time.Second + time.Duration((offset/s.itemsPerPage)%3)*time.Second)
	}

	return allListings, nil
}

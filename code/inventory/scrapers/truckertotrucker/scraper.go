package truckertotrucker

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

// TruckertoTruckerScraper handles HTTP-based scraping for TruckertoTrucker.com
type TruckertoTruckerScraper struct {
	client  *http.Client
	baseURL string
}

// NewTruckertoTruckerScraper creates a new HTTP scraper for TruckertoTrucker
func NewTruckertoTruckerScraper() *TruckertoTruckerScraper {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &TruckertoTruckerScraper{
		client:  client,
		baseURL: "https://truckertotrucker.com",
	}
}

// ScrapePage scrapes a single page of listings
func (s *TruckertoTruckerScraper) ScrapePage(ctx context.Context, pageNum int) ([]*agent.ListingExtract, error) {
	pageURL := fmt.Sprintf("%s/ads?page=%d", s.baseURL, pageNum)

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
		return nil, fmt.Errorf("failed to fetch page %d after %d attempts: %w", pageNum, maxRetries, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return []*agent.ListingExtract{}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d for page %d", resp.StatusCode, pageNum)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	return agent.ExtractListingsFromHTML(ctx, string(bodyBytes), s.baseURL)
}

// ScrapeAllPages scrapes all pages until no more listings are found
func (s *TruckertoTruckerScraper) ScrapeAllPages(ctx context.Context) ([]*agent.ListingExtract, error) {
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

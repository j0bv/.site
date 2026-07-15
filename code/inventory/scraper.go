package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"truck-inventory/agent"
	"truck-inventory/scrapers/arrowtruck"
	"truck-inventory/scrapers/heavytruckdealers"
	"truck-inventory/scrapers/ironconnect"
	"truck-inventory/scrapers/trucktractortrailer"
	"truck-inventory/scrapers/truckertotrucker"
	"truck-inventory/scrapers/truckplanet"
	"truck-inventory/scrapers/selectrucks"
	"truck-inventory/scrapers/ironplanet"
)

// ScraperCoordinator coordinates all individual scrapers and saves results to database
type ScraperCoordinator struct{}

func NewScraperCoordinator() *ScraperCoordinator {
	return &ScraperCoordinator{}
}

// Start begins the scraping coordination loop
func (sc *ScraperCoordinator) Start(ctx context.Context, db *DB) {
	// Run initial scrape
	sc.runFullScrape(ctx, db)

	// Schedule periodic scrapes
	ticker := time.NewTicker(45 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Add random delay (45-75 minutes)
			delay := time.Duration(45+rand.Intn(30)) * time.Minute
			time.Sleep(delay)
			sc.runFullScrape(ctx, db)
		}
	}
}

func (sc *ScraperCoordinator) runFullScrape(ctx context.Context, db *DB) {
	scrapers := []struct {
		name    string
		scrape  func(context.Context) ([]*TruckListing, error)
	}{
		{"IronConnect", sc.scrapeIronConnect},
		{"TruckTractorTrailer", sc.scrapeTruckTractorTrailer},
		{"TruckertoTrucker", sc.scrapeTruckertoTrucker},
		{"TruckPlanet", sc.scrapeTruckPlanet},
		{"SelecTrucks", sc.scrapeSelecTrucks},
		{"ArrowTruck", sc.scrapeArrowTruck},
		{"HeavyTruckDealers", sc.scrapeHeavyTruckDealers},
		{"IronPlanet", sc.scrapeIronPlanet},
	}

	for _, s := range scrapers {
		select {
		case <-ctx.Done():
			return
		default:
		}

		log.Printf("\nScraping %s...", s.name)

		listings, err := s.scrape(ctx)
		if err != nil {
			log.Printf("  Error scraping %s: %v", s.name, err)
			continue
		}

		log.Printf("  Found %d listings", len(listings))

		for _, listing := range listings {
			if err := db.UpsertListing(ctx, listing); err != nil {
				log.Printf("  Error saving listing: %v", err)
			}
		}

		// Delay between marketplaces
		delay := 3*time.Second + time.Duration(rand.Intn(5000))*time.Millisecond
		time.Sleep(delay)
	}
}

func (sc *ScraperCoordinator) scrapeIronConnect(ctx context.Context) ([]*TruckListing, error) {
	httpScraper := ironconnect.NewIronConnectHTTPScraper()

	ironEmail := getEnv("IRON_EMAIL", "")
	ironPassword := getEnv("IRON_PASSWORD", "")
	if ironEmail != "" && ironPassword != "" {
		if err := httpScraper.Login(ironEmail, ironPassword); err != nil {
			return nil, fmt.Errorf("failed to login: %w", err)
		}
	}

	extracts, err := httpScraper.ScrapeAllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to scrape pages: %w", err)
	}
	return listingExtractsToTruckListings(extracts, "IronConnect"), nil
}

func (sc *ScraperCoordinator) scrapeTruckTractorTrailer(ctx context.Context) ([]*TruckListing, error) {
	httpScraper := trucktractortrailer.NewTruckTractorTrailerScraper()
	extracts, err := httpScraper.ScrapeAllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to scrape pages: %w", err)
	}
	return listingExtractsToTruckListings(extracts, "TruckTractorTrailer"), nil
}

func (sc *ScraperCoordinator) scrapeTruckertoTrucker(ctx context.Context) ([]*TruckListing, error) {
	httpScraper := truckertotrucker.NewTruckertoTruckerScraper()
	extracts, err := httpScraper.ScrapeAllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to scrape pages: %w", err)
	}
	return listingExtractsToTruckListings(extracts, "TruckertoTrucker"), nil
}

func (sc *ScraperCoordinator) scrapeTruckPlanet(ctx context.Context) ([]*TruckListing, error) {
	httpScraper := truckplanet.NewTruckPlanetScraper()
	extracts, err := httpScraper.ScrapeAllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to scrape pages: %w", err)
	}
	return listingExtractsToTruckListings(extracts, "TruckPlanet"), nil
}

func (sc *ScraperCoordinator) scrapeSelecTrucks(ctx context.Context) ([]*TruckListing, error) {
	httpScraper := selectrucks.NewSelecTrucksScraper()
	extracts, err := httpScraper.ScrapeAllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to scrape pages: %w", err)
	}
	return listingExtractsToTruckListings(extracts, "SelecTrucks"), nil
}

func (sc *ScraperCoordinator) scrapeArrowTruck(ctx context.Context) ([]*TruckListing, error) {
	httpScraper := arrowtruck.NewArrowTruckScraper()
	extracts, err := httpScraper.ScrapeAllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to scrape pages: %w", err)
	}
	return listingExtractsToTruckListings(extracts, "ArrowTruck"), nil
}

func (sc *ScraperCoordinator) scrapeHeavyTruckDealers(ctx context.Context) ([]*TruckListing, error) {
	httpScraper := heavytruckdealers.NewHeavyTruckDealersScraper()
	extracts, err := httpScraper.ScrapeAllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to scrape pages: %w", err)
	}
	return listingExtractsToTruckListings(extracts, "HeavyTruckDealers"), nil
}

func (sc *ScraperCoordinator) scrapeIronPlanet(ctx context.Context) ([]*TruckListing, error) {
	httpScraper := ironplanet.NewIronPlanetScraper()
	extracts, err := httpScraper.ScrapeAllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to scrape pages: %w", err)
	}
	return listingExtractsToTruckListings(extracts, "IronPlanet"), nil
}

// listingExtractToTruckListing converts agent.ListingExtract to TruckListing for DB upsert.
// Returns nil if ext.URL is empty.
func listingExtractToTruckListing(ext *agent.ListingExtract, marketplace string) *TruckListing {
	if ext == nil || ext.URL == "" {
		return nil
	}
	return &TruckListing{
		Marketplace: marketplace,
		Title:       ext.Title,
		Year:        ext.Year,
		Make:        ext.Make,
		Model:       ext.Model,
		Price:       ext.Price,
		Miles:       ext.Miles,
		Location:    ext.Location,
		ImageURL:    ext.ImageURL,
		URL:         ext.URL,
		Description: ext.Description,
		VIN:         ext.VIN,
		StockNumber: ext.StockNumber,
	}
}

func listingExtractsToTruckListings(extracts []*agent.ListingExtract, marketplace string) []*TruckListing {
	result := make([]*TruckListing, 0, len(extracts))
	for _, e := range extracts {
		if tl := listingExtractToTruckListing(e, marketplace); tl != nil {
			result = append(result, tl)
		}
	}
	return result
}

// Helper to get environment variable
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

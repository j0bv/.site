// test-extraction runs sample HTML through agent.ExtractListingFromHTML and
// agent.ExtractListingsFromHTML to validate the SLM or the shared fallback.
//
// Usage: go run ./cmd/test-extraction
// Loads .env from current directory if present. Uses SLM when SLM_ENABLED != "false";
// otherwise the fallback in ExtractListingsFromHTML is used.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"truck-inventory/agent"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Note: .env not loaded: %v", err)
	}

	ctx := context.Background()
	baseURL := "https://example.com"

	// Sample 1: minimal card-like HTML
	html1 := `
<div class="listing-card">
  <h2>2020 Peterbilt 389 - Great condition</h2>
  <span class="price">$85,000</span>
  <span class="mileage">450,000 miles</span>
  <span class="location">Chicago, IL</span>
  <a href="/listings/123">View details</a>
  <img src="/images/truck1.jpg" alt="truck"/>
</div>`

	// Sample 2: used by ExtractListingsFromHTML (full page fragment with container)
	html2 := `
<div class="fwpl-item">
  <h3>2019 Freightliner Cascadia</h3>
  <div class="price">$72,500</div>
  <div class="mileage">380,000 miles</div>
  <a href="/truck/456">Details</a>
</div>`

	log.Println("Test 1: ExtractListingFromHTML (single fragment)")
	ext1, err := agent.ExtractListingFromHTML(ctx, html1, baseURL)
	if err != nil {
		log.Printf("  ExtractListingFromHTML error: %v", err)
	} else if ext1 == nil {
		log.Println("  SLM disabled: got nil (fallback is used inside ExtractListingsFromHTML for multi-card HTML)")
	} else {
		log.Printf("  Title: %v", strPtr(ext1.Title))
		log.Printf("  Year: %v", intPtr(ext1.Year))
		log.Printf("  Make: %v", strPtr(ext1.Make))
		log.Printf("  Price: %v", floatPtr(ext1.Price))
		log.Printf("  Miles: %v", intPtr(ext1.Miles))
		log.Printf("  Location: %v", strPtr(ext1.Location))
		log.Printf("  URL: %s", ext1.URL)
	}

	log.Println("Test 2: ExtractListingsFromHTML (container + cards)")
	list2, err := agent.ExtractListingsFromHTML(ctx, html2, baseURL)
	if err != nil {
		log.Printf("  ExtractListingsFromHTML error: %v", err)
	} else {
		log.Printf("  Found %d listing(s)", len(list2))
		for i, e := range list2 {
			log.Printf("  [%d] URL=%s Title=%v Price=%v", i+1, e.URL, strPtr(e.Title), floatPtr(e.Price))
		}
	}

	if os.Getenv("SLM_ENABLED") == "false" {
		log.Println("SLM_ENABLED=false: extraction used shared fallback (selector-based).")
	} else {
		log.Println("SLM enabled: extraction used LLM. Set SLM_ENABLED=false to test fallback.")
	}
	fmt.Println("test-extraction done")
}

func strPtr(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

func intPtr(i *int) string {
	if i == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%d", *i)
}

func floatPtr(f *float64) string {
	if f == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%g", *f)
}

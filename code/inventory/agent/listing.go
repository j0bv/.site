package agent

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ListingExtract holds extracted truck listing fields for DB upsert.
// Marketplace is set by the caller when converting to TruckListing.
type ListingExtract struct {
	Title       *string
	Year        *int
	Make        *string
	Model       *string
	Price       *float64
	Miles       *int
	Location    *string
	ImageURL    *string
	URL         string
	Description *string
	VIN         *string
	StockNumber *string
}

type listingExtractJSON struct {
	Title       *string  `json:"title"`
	Year        *int     `json:"year"`
	Make        *string  `json:"make"`
	Model       *string  `json:"model"`
	Price       *float64 `json:"price"`
	Miles       *int     `json:"miles"`
	Location    *string  `json:"location"`
	ImageURL    *string  `json:"image_url"`
	URL         string   `json:"url"`
	Description *string  `json:"description"`
	VIN         *string  `json:"vin"`
	StockNumber *string  `json:"stock_number"`
}

// containerSelectors is the shared list for listing discovery only (no per-field selectors).
const containerSelectors = `.fwpl-item, .listing-item, .sr_item, .listing-card, div[class*="listing"], div[class*="item"], div[class*="card"], article`

const htmlTruncate = 4096

// ExtractListingFromHTML uses the SLM when SLM_ENABLED != "false" to extract one listing from an HTML fragment.
// Returns (nil, nil) when SLM_ENABLED=="false" so the caller can use the shared fallback.
func ExtractListingFromHTML(ctx context.Context, htmlFragment, baseURL string) (*ListingExtract, error) {
	if os.Getenv("SLM_ENABLED") == "false" {
		return nil, nil
	}

	prompt := `Extract a truck listing from this HTML. Return a single JSON object only. Keys: title, year (int), make, model, price (float), miles (int), location, image_url, url (absolute; if relative resolve with base ` + baseURL + `), description, vin, stock_number. Omit missing. Example: {"make":"PETERBILT","price":85000}

HTML:
`
	if len(htmlFragment) > htmlTruncate {
		htmlFragment = htmlFragment[:htmlTruncate]
	}
	prompt += htmlFragment

	content, err := Chat(ctx, prompt)
	if err != nil {
		return nil, err
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, nil
	}

	jsonStr := extractJSON(content)
	if jsonStr == "" {
		return nil, nil
	}

	var j listingExtractJSON
	if err := json.Unmarshal([]byte(jsonStr), &j); err != nil {
		return nil, err
	}

	ext := &ListingExtract{
		Title:       j.Title,
		Year:        j.Year,
		Make:        j.Make,
		Model:       j.Model,
		Price:       j.Price,
		Miles:       j.Miles,
		Location:    j.Location,
		ImageURL:    j.ImageURL,
		URL:         strings.TrimSpace(j.URL),
		Description: j.Description,
		VIN:         j.VIN,
		StockNumber: j.StockNumber,
	}
	resolveListingURLs(ext, baseURL)
	return ext, nil
}

func resolveListingURLs(ext *ListingExtract, baseURL string) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return
	}
	if ext.URL != "" && !strings.HasPrefix(ext.URL, "http") {
		if ref, err := url.Parse(ext.URL); err == nil && ref != nil {
			ext.URL = base.ResolveReference(ref).String()
		}
	}
	if ext.ImageURL != nil && *ext.ImageURL != "" && !strings.HasPrefix(*ext.ImageURL, "http") {
		if ref, err := url.Parse(*ext.ImageURL); err == nil && ref != nil {
			s := base.ResolveReference(ref).String()
			ext.ImageURL = &s
		}
	}
}

// ExtractListingsFromHTML finds listing containers via a shared selector list, then for each
// runs ExtractListingFromHTML (SLM) or extractListingFallback when SLM is disabled.
func ExtractListingsFromHTML(ctx context.Context, htmlContent, baseURL string) ([]*ListingExtract, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	var out []*ListingExtract
	doc.Find(containerSelectors).Each(func(i int, sel *goquery.Selection) {
		blockHTML, err := sel.Html()
		if err != nil {
			return
		}
		ext, err := ExtractListingFromHTML(ctx, blockHTML, baseURL)
		if err != nil {
			return
		}
		if ext != nil && ext.URL != "" {
			out = append(out, ext)
			return
		}
		// SLM disabled or empty: use fallback
		if ext == nil && err == nil {
			fb := extractListingFallback(sel, baseURL)
			if fb != nil && fb.URL != "" {
				out = append(out, fb)
			}
		}
	})
	return out, nil
}

// extractListingFallback uses 1-2 generic selectors per field when SLM is disabled.
func extractListingFallback(sel *goquery.Selection, baseURL string) *ListingExtract {
	ext := &ListingExtract{}

	// title: h2, h3, .title
	for _, s := range []string{"h2", "h3", ".title"} {
		if t := sel.Find(s).First().Text(); t != "" {
			t = strings.TrimSpace(t)
			ext.Title = &t
			break
		}
	}

	// year, make, model: from title or .year, .make, .model
	if ext.Title != nil {
		if y := parseYear(*ext.Title); y != nil {
			ext.Year = y
		}
	}
	for _, s := range []string{"[data-year]", ".year", ".model-year"} {
		if t := sel.Find(s).First().Text(); t != "" {
			if y := parseYear(t); y != nil {
				ext.Year = y
				break
			}
		}
	}
	for _, s := range []string{"[data-make]", ".make", ".brand"} {
		if t := sel.Find(s).First().Text(); t != "" {
			t = strings.TrimSpace(t)
			ext.Make = &t
			break
		}
	}
	for _, s := range []string{"[data-model]", ".model"} {
		if t := sel.Find(s).First().Text(); t != "" {
			t = strings.TrimSpace(t)
			ext.Model = &t
			break
		}
	}

	// price: .price, [data-price]
	for _, s := range []string{".price", "[data-price]", ".price-display", ".listing-price"} {
		if t := sel.Find(s).First().Text(); t != "" {
			if p := parsePrice(t); p != nil {
				ext.Price = p
				break
			}
		}
	}

	// miles: .mileage, .miles, [data-mileage]
	for _, s := range []string{"[data-mileage]", ".mileage", "[data-miles]", ".miles"} {
		if t := sel.Find(s).First().Text(); t != "" {
			if m := parseMiles(t); m != nil {
				ext.Miles = m
				break
			}
		}
	}

	// location: .location, [data-location]
	for _, s := range []string{".location", "[data-location]", ".dealer-location", ".listing-location"} {
		if t := sel.Find(s).First().Text(); t != "" {
			t = strings.TrimSpace(t)
			ext.Location = &t
			break
		}
	}

	// image_url: img[src]
	if src, ok := sel.Find("img[src]").First().Attr("src"); ok && src != "" {
		src = resolveURL(src, baseURL)
		ext.ImageURL = &src
	}

	// url: a[href] (required)
	for _, s := range []string{"a[href*='/listings/']", "a[href*='/listing/']", "a[href*='/truck/']", "a[href]"} {
		if href, ok := sel.Find(s).First().Attr("href"); ok && href != "" {
			ext.URL = resolveURL(href, baseURL)
			break
		}
	}
	if ext.URL == "" {
		return nil
	}

	// description: .description
	for _, s := range []string{".listing-description", ".description", ".details", ".truck-details"} {
		if t := sel.Find(s).First().Text(); t != "" {
			t = strings.TrimSpace(t)
			ext.Description = &t
			break
		}
	}

	// vin: [data-vin], .vin
	for _, s := range []string{"[data-vin]", ".vin"} {
		if t := sel.Find(s).First().Text(); t != "" {
			t = strings.TrimSpace(t)
			ext.VIN = &t
			break
		}
	}

	// stock_number: .stock, .stock-number
	for _, s := range []string{"[data-stock-number]", ".stock-number", ".stock", ".inventory-number"} {
		if t := sel.Find(s).First().Text(); t != "" {
			t = strings.TrimSpace(t)
			ext.StockNumber = &t
			break
		}
	}

	return ext
}

func resolveURL(rel, baseURL string) string {
	if rel == "" || strings.HasPrefix(rel, "http") {
		return rel
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return rel
	}
	ref, err := url.Parse(rel)
	if err != nil || ref == nil {
		return rel
	}
	return base.ResolveReference(ref).String()
}

func parseYear(text string) *int {
	re := regexp.MustCompile(`\b(19|20)\d{2}\b`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 0 {
		if y, err := strconv.Atoi(matches[0]); err == nil {
			return &y
		}
	}
	return nil
}

func parsePrice(text string) *float64 {
	re := regexp.MustCompile(`\$?([\d,]+(?:\.\d{2})?)`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		s := strings.ReplaceAll(matches[1], ",", "")
		if p, err := strconv.ParseFloat(s, 64); err == nil {
			return &p
		}
	}
	return nil
}

func parseMiles(text string) *int {
	re := regexp.MustCompile(`(\d+(?:,\d{3})*)`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		s := strings.ReplaceAll(matches[1], ",", "")
		if m, err := strconv.Atoi(s); err == nil {
			return &m
		}
	}
	return nil
}

// CleanupListing is a stub for a future phase: a second LLM pass to complete missing
// or correct improper fields (e.g. price 0, empty make when title suggests one).
// Call when ext has low completeness or after persisting to run a batch over stored
// listings with nulls. For now it returns ext unchanged.
func CleanupListing(ctx context.Context, ext *ListingExtract, rawHTML string) (*ListingExtract, error) {
	if ext == nil {
		return nil, nil
	}
	return ext, nil
}

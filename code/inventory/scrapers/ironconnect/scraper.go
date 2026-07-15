package ironconnect

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"truck-inventory/agent"

	"github.com/PuerkitoBio/goquery"
)

// IronConnectHTTPScraper handles HTTP-based scraping for IronConnect
type IronConnectHTTPScraper struct {
	client  *http.Client
	baseURL string
	loggedIn bool
}

// NewIronConnectHTTPScraper creates a new HTTP scraper for IronConnect
func NewIronConnectHTTPScraper() *IronConnectHTTPScraper {
	jar, _ := cookiejar.New(nil)
	
	client := &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Follow redirects but preserve cookies
			return nil
		},
	}

	return &IronConnectHTTPScraper{
		client:  client,
		baseURL: "https://ironconnect.com",
		loggedIn: false,
	}
}

// Login authenticates with IronConnect and stores session cookies
func (s *IronConnectHTTPScraper) Login(email, password string) error {
	loginURL := fmt.Sprintf("%s/Home/SignIn", s.baseURL)
	log.Printf("🔐 Attempting programmatic login to IronConnect...")

	// Retry logic for login
	maxRetries := 3
	var err error
	var resp *http.Response

	// Step 1: Get the login page to extract form fields and CSRF tokens
	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, _ := http.NewRequest("GET", loginURL, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		
		resp, err = s.client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		
		if resp != nil {
			resp.Body.Close()
		}
		
		if attempt < maxRetries {
			backoff := time.Duration(attempt) * time.Second
			log.Printf("  ⚠️  Failed to get login page (attempt %d/%d), retrying in %v...", attempt, maxRetries, backoff)
			time.Sleep(backoff)
		}
	}
	
	if err != nil {
		return fmt.Errorf("failed to get login page after %d attempts: %w", maxRetries, err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return fmt.Errorf("login page returned status %d", resp.StatusCode)
	}

	// Parse the login form to extract all form fields
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("failed to parse login page: %w", err)
	}

	// Find the form action URL (might be relative or absolute)
	formAction := loginURL
	doc.Find("form").Each(func(i int, sel *goquery.Selection) {
		if action, exists := sel.Attr("action"); exists {
			if strings.HasPrefix(action, "http") {
				formAction = action
			} else if action != "" {
				base, _ := url.Parse(loginURL)
				rel, _ := url.Parse(action)
				formAction = base.ResolveReference(rel).String()
			}
		}
	})

	log.Printf("  📋 Form action: %s", formAction)

	// Prepare form data with credentials
	formData := url.Values{}
	
	// Try to find email/password field names (they might be different)
	emailFieldName := "email"
	passwordFieldName := "password"
	
	doc.Find("input[type='email'], input[name*='email'], input[name*='Email']").Each(func(i int, sel *goquery.Selection) {
		if name, exists := sel.Attr("name"); exists && name != "" {
			emailFieldName = name
		}
	})
	
	doc.Find("input[type='password'], input[name*='password'], input[name*='Password']").Each(func(i int, sel *goquery.Selection) {
		if name, exists := sel.Attr("name"); exists && name != "" {
			passwordFieldName = name
		}
	})
	
	formData.Set(emailFieldName, email)
	formData.Set(passwordFieldName, password)
	
	log.Printf("  📝 Using field names: %s, %s", emailFieldName, passwordFieldName)
	
	// Extract ALL hidden fields (CSRF tokens, viewstate, etc.)
	hiddenFields := 0
	doc.Find("input[type='hidden']").Each(func(i int, sel *goquery.Selection) {
		name, _ := sel.Attr("name")
		value, _ := sel.Attr("value")
		if name != "" {
			formData.Set(name, value)
			hiddenFields++
			// Log important fields (CSRF, viewstate, etc.)
			if strings.Contains(strings.ToLower(name), "token") || 
			   strings.Contains(strings.ToLower(name), "csrf") ||
			   strings.Contains(strings.ToLower(name), "viewstate") ||
			   strings.Contains(strings.ToLower(name), "validation") {
				log.Printf("  🔑 Found security field: %s", name)
			}
		}
	})
	
	log.Printf("  📦 Extracted %d hidden form fields", hiddenFields)

	// Submit login form
	req, err := http.NewRequest("POST", formAction, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", loginURL)
	req.Header.Set("Origin", s.baseURL)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	// Retry login POST
	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err = s.client.Do(req)
		if err == nil {
			break
		}
		
		if resp != nil {
			resp.Body.Close()
		}
		
		if attempt < maxRetries {
			backoff := time.Duration(attempt) * time.Second
			log.Printf("  ⚠️  Login POST attempt %d failed: %v, retrying in %v...", attempt, err, backoff)
			time.Sleep(backoff)
		}
	}
	
	if err != nil {
		return fmt.Errorf("failed to submit login after %d attempts: %w", maxRetries, err)
	}
	defer resp.Body.Close()

	log.Printf("  📊 Login response status: %d", resp.StatusCode)
	log.Printf("  🔗 Final URL: %s", resp.Request.URL.String())

	// Check response body for error messages
	bodyBytes := make([]byte, 4096)
	n, _ := resp.Body.Read(bodyBytes)
	bodyStr := string(bodyBytes[:n])
	
	// Check for common error indicators
	errorIndicators := []string{
		"invalid email or password",
		"incorrect password",
		"login failed",
		"authentication failed",
		"error",
	}
	
	for _, indicator := range errorIndicators {
		if strings.Contains(strings.ToLower(bodyStr), indicator) {
			log.Printf("  ⚠️  Found error indicator in response: %s", indicator)
		}
	}

	// Check cookies - successful login usually sets session cookies
	cookies := s.client.Jar.Cookies(resp.Request.URL)
	sessionCookies := 0
	for _, cookie := range cookies {
		if strings.Contains(strings.ToLower(cookie.Name), "session") ||
		   strings.Contains(strings.ToLower(cookie.Name), "auth") ||
		   strings.Contains(strings.ToLower(cookie.Name), "token") {
			sessionCookies++
			log.Printf("  🍪 Session cookie found: %s", cookie.Name)
		}
	}
	
	log.Printf("  🍪 Total cookies: %d (session cookies: %d)", len(cookies), sessionCookies)

	// Check if login was successful
	success := false
	finalURL := resp.Request.URL.String()
	
	// Success indicators:
	// 1. Redirected away from login page
	if !strings.Contains(finalURL, "/Home/SignIn") && !strings.Contains(finalURL, "/SignIn") {
		success = true
		log.Printf("  ✅ Redirected away from login page")
	}
	
	// 2. Status code indicates redirect
	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusSeeOther || resp.StatusCode == http.StatusMovedPermanently {
		success = true
		log.Printf("  ✅ Redirect status code: %d", resp.StatusCode)
	}
	
	// 3. Session cookies present
	if sessionCookies > 0 {
		success = true
		log.Printf("  ✅ Session cookies present")
	}
	
	// 4. Response body contains dashboard/listings indicators
	dashboardIndicators := []string{"dashboard", "my hub", "listings", "logout", "sign out"}
	for _, indicator := range dashboardIndicators {
		if strings.Contains(strings.ToLower(bodyStr), indicator) {
			success = true
			log.Printf("  ✅ Found dashboard indicator: %s", indicator)
			break
		}
	}

	if success {
		s.loggedIn = true
		log.Println("✅ IronConnect authenticated successfully via HTTP")
		return nil
	}

	// If we're still on login page, authentication likely failed
	return fmt.Errorf("login failed: still on login page (status: %d, URL: %s, session cookies: %d)", 
		resp.StatusCode, finalURL, sessionCookies)
}

// ScrapePage scrapes a single page of listings with retry logic
func (s *IronConnectHTTPScraper) ScrapePage(ctx context.Context, pageNum int) ([]*agent.ListingExtract, error) {
	pageURL := fmt.Sprintf("%s/Listings?Industry=Truck&PartEquipment=Equipment&LastCall=true", s.baseURL)
	if pageNum > 1 {
		pageURL = fmt.Sprintf("%s&page=%d", pageURL, pageNum)
	}

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
	htmlContent := string(bodyBytes)
	return agent.ExtractListingsFromHTML(ctx, htmlContent, s.baseURL)
}

// ScrapeAllPages scrapes all pages until no more listings are found
func (s *IronConnectHTTPScraper) ScrapeAllPages(ctx context.Context) ([]*agent.ListingExtract, error) {
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

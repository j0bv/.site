package ironconnect

import (
	"os"
	"testing"
)

// TestLogin is a helper function to test login programmatically
// Usage: go test -run TestLogin -v (with IRON_EMAIL and IRON_PASSWORD env vars set)
func TestLogin(t *testing.T) {
	email := os.Getenv("IRON_EMAIL")
	password := os.Getenv("IRON_PASSWORD")

	if email == "" || password == "" {
		t.Skip("Skipping login test: IRON_EMAIL and IRON_PASSWORD not set")
	}

	scraper := NewIronConnectHTTPScraper()
	err := scraper.Login(email, password)
	
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	if !scraper.loggedIn {
		t.Fatal("Login succeeded but loggedIn flag is false")
	}

	t.Log("✅ Login test passed!")
}

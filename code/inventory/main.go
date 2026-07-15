package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database
	db, err := NewDB(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize scraper coordinator
	scraperCoordinator := NewScraperCoordinator()

	// Initialize email matcher
	emailMatcher, err := NewEmailMatcher(ctx, db)
	if err != nil {
		log.Fatalf("Failed to initialize email matcher: %v", err)
	}

	var wg sync.WaitGroup

	// Start scraper coordinator in background
	wg.Add(1)
	go func() {
		defer wg.Done()
		scraperCoordinator.Start(ctx, db)
	}()

	// Start email checker in background
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		// Run immediately on startup
		if err := emailMatcher.FetchNewEmails(ctx); err != nil {
			log.Printf("Error fetching emails: %v", err)
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := emailMatcher.FetchNewEmails(ctx); err != nil {
					log.Printf("Error fetching emails: %v", err)
				}
			}
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	cancel()
	wg.Wait()
	log.Println("Shutdown complete")
}

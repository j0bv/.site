package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool *pgxpool.Pool
}

func NewDB(ctx context.Context) (*DB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_USER", "postgres"),
		getEnv("DB_PASSWORD", ""),
		getEnv("DB_NAME", "truck_inventory"),
	)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{pool: pool}

	// Initialize schema
	if err := db.InitSchema(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

func (db *DB) Close() {
	db.pool.Close()
}

func (db *DB) InitSchema(ctx context.Context) error {
	schema := `
	-- Main truck listings table
	CREATE TABLE IF NOT EXISTS truck_listings (
		id SERIAL PRIMARY KEY,
		marketplace VARCHAR(50) NOT NULL,
		
		-- Truck specifications
		title TEXT,
		year INTEGER,
		make VARCHAR(100),
		model VARCHAR(100),
		condition VARCHAR(50),
		transmission VARCHAR(50),
		engine_type VARCHAR(50),
		miles INTEGER,
		horsepower VARCHAR(50),
		fuel_type VARCHAR(50),
		
		-- Financial
		price DECIMAL(12, 2),
		down_payment DECIMAL(12, 2),
		monthly_payment DECIMAL(12, 2),
		
		-- Location
		location VARCHAR(100),
		city VARCHAR(50),
		state VARCHAR(50),
		zip_code VARCHAR(10),
		latitude DECIMAL(10, 8),
		longitude DECIMAL(11, 8),
		
		-- Details
		description TEXT,
		stock_number VARCHAR(50),
		vin VARCHAR(17),
		
		-- Media
		image_url TEXT,
		image_urls TEXT[],
		
		-- URLs
		url TEXT NOT NULL,
		detail_page_url TEXT,
		
		-- Tracking
		first_seen TIMESTAMP DEFAULT NOW(),
		last_seen TIMESTAMP DEFAULT NOW(),
		last_updated TIMESTAMP DEFAULT NOW(),
		
		-- Email matching
		matched_to_request_id INTEGER,
		matched_at TIMESTAMP,
		
		UNIQUE(marketplace, url)
	);

	-- Track client email requests
	CREATE TABLE IF NOT EXISTS email_search_requests (
		id SERIAL PRIMARY KEY,
		from_email VARCHAR(255) NOT NULL,
		from_name VARCHAR(255),
		subject TEXT,
		body TEXT,
		
		-- Extracted search criteria
		search_year_min INTEGER,
		search_year_max INTEGER,
		search_make VARCHAR(100),
		search_model VARCHAR(100),
		search_max_price DECIMAL(12, 2),
		search_min_price DECIMAL(12, 2),
		search_max_miles INTEGER,
		search_location VARCHAR(100),
		search_radius_miles INTEGER,
		
		received_at TIMESTAMP,
		processed_at TIMESTAMP,
		
		UNIQUE(from_email, subject, received_at)
	);

	-- Track matches
	CREATE TABLE IF NOT EXISTS email_matches (
		id SERIAL PRIMARY KEY,
		request_id INTEGER NOT NULL REFERENCES email_search_requests(id),
		listing_id INTEGER NOT NULL REFERENCES truck_listings(id),
		match_score DECIMAL(3, 2),
		matched_at TIMESTAMP DEFAULT NOW(),
		
		UNIQUE(request_id, listing_id)
	);

	-- Create indexes
	CREATE INDEX IF NOT EXISTS idx_marketplace ON truck_listings(marketplace);
	CREATE INDEX IF NOT EXISTS idx_make_model ON truck_listings(make, model);
	CREATE INDEX IF NOT EXISTS idx_price ON truck_listings(price);
	CREATE INDEX IF NOT EXISTS idx_year ON truck_listings(year);
	CREATE INDEX IF NOT EXISTS idx_location ON truck_listings(city, state);
	CREATE INDEX IF NOT EXISTS idx_last_seen ON truck_listings(last_seen);
	CREATE INDEX IF NOT EXISTS idx_matched_to_request ON truck_listings(matched_to_request_id);
	`

	_, err := db.pool.Exec(ctx, schema)
	return err
}

type TruckListing struct {
	ID                int
	Marketplace       string
	Title             *string
	Year              *int
	Make              *string
	Model             *string
	Condition         *string
	Transmission      *string
	EngineType        *string
	Miles             *int
	Horsepower        *string
	FuelType          *string
	Price             *float64
	DownPayment       *float64
	MonthlyPayment    *float64
	Location          *string
	City              *string
	State             *string
	ZipCode           *string
	Latitude          *float64
	Longitude         *float64
	Description      *string
	StockNumber      *string
	VIN               *string
	ImageURL          *string
	ImageURLs         []string
	URL               string
	DetailPageURL     *string
	FirstSeen         time.Time
	LastSeen          time.Time
	LastUpdated       time.Time
	MatchedToRequestID *int
	MatchedAt         *time.Time
}

func (db *DB) UpsertListing(ctx context.Context, listing *TruckListing) error {
	query := `
		INSERT INTO truck_listings (
			marketplace, title, year, make, model, price, miles,
			location, image_url, url, description, vin, stock_number, last_seen
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW()
		)
		ON CONFLICT (marketplace, url) DO UPDATE SET
			price = EXCLUDED.price,
			miles = EXCLUDED.miles,
			last_seen = NOW()
	`

	_, err := db.pool.Exec(ctx, query,
		listing.Marketplace,
		listing.Title,
		listing.Year,
		listing.Make,
		listing.Model,
		listing.Price,
		listing.Miles,
		listing.Location,
		listing.ImageURL,
		listing.URL,
		listing.Description,
		listing.VIN,
		listing.StockNumber,
	)

	return err
}

type EmailSearchRequest struct {
	ID              int
	FromEmail       string
	FromName        *string
	Subject         *string
	Body            *string
	SearchYearMin   *int
	SearchYearMax   *int
	SearchMake      *string
	SearchModel     *string
	SearchMaxPrice  *float64
	SearchMinPrice  *float64
	SearchMaxMiles  *int
	SearchLocation  *string
	SearchRadiusMiles *int
	ReceivedAt      *time.Time
	ProcessedAt     *time.Time
}

func (db *DB) SaveEmailRequest(ctx context.Context, req *EmailSearchRequest) (int, error) {
	query := `
		INSERT INTO email_search_requests (
			from_email, from_name, subject, body,
			search_year_min, search_year_max,
			search_make, search_model,
			search_max_price, search_min_price,
			search_max_miles, search_location,
			received_at, processed_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW()
		)
		ON CONFLICT (from_email, subject, received_at) DO NOTHING
		RETURNING id
	`

	var id int
	err := db.pool.QueryRow(ctx, query,
		req.FromEmail,
		req.FromName,
		req.Subject,
		req.Body,
		req.SearchYearMin,
		req.SearchYearMax,
		req.SearchMake,
		req.SearchModel,
		req.SearchMaxPrice,
		req.SearchMinPrice,
		req.SearchMaxMiles,
		req.SearchLocation,
		req.ReceivedAt,
	).Scan(&id)

	if err != nil {
		return 0, err
	}

	return id, nil
}

func (db *DB) FindMatchingListings(ctx context.Context, criteria *SearchCriteria) ([]*TruckListing, error) {
	query := `
		SELECT id, marketplace, title, year, make, model, price, miles, 
		       location, url, description, vin, stock_number
		FROM truck_listings
		WHERE 1=1
	`

	args := []interface{}{}
	argPos := 1

	if criteria.YearMin != nil {
		query += fmt.Sprintf(" AND year >= $%d", argPos)
		args = append(args, *criteria.YearMin)
		argPos++
	}

	if criteria.YearMax != nil {
		query += fmt.Sprintf(" AND year <= $%d", argPos)
		args = append(args, *criteria.YearMax)
		argPos++
	}

	if criteria.Make != nil {
		query += fmt.Sprintf(" AND UPPER(make) LIKE UPPER($%d)", argPos)
		args = append(args, "%"+*criteria.Make+"%")
		argPos++
	}

	if criteria.Model != nil {
		query += fmt.Sprintf(" AND UPPER(model) LIKE UPPER($%d)", argPos)
		args = append(args, "%"+*criteria.Model+"%")
		argPos++
	}

	if criteria.MaxPrice != nil {
		query += fmt.Sprintf(" AND price <= $%d", argPos)
		args = append(args, *criteria.MaxPrice)
		argPos++
	}

	if criteria.MinPrice != nil {
		query += fmt.Sprintf(" AND price >= $%d", argPos)
		args = append(args, *criteria.MinPrice)
		argPos++
	}

	if criteria.MaxMiles != nil {
		query += fmt.Sprintf(" AND miles <= $%d", argPos)
		args = append(args, *criteria.MaxMiles)
		argPos++
	}

	query += " LIMIT 50"

	rows, err := db.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var listings []*TruckListing
	for rows.Next() {
		var listing TruckListing
		err := rows.Scan(
			&listing.ID,
			&listing.Marketplace,
			&listing.Title,
			&listing.Year,
			&listing.Make,
			&listing.Model,
			&listing.Price,
			&listing.Miles,
			&listing.Location,
			&listing.URL,
			&listing.Description,
			&listing.VIN,
			&listing.StockNumber,
		)
		if err != nil {
			return nil, err
		}
		listings = append(listings, &listing)
	}

	return listings, rows.Err()
}

func (db *DB) SaveMatch(ctx context.Context, requestID, listingID int, score float64) error {
	query := `
		INSERT INTO email_matches (request_id, listing_id, match_score)
		VALUES ($1, $2, $3)
		ON CONFLICT (request_id, listing_id) DO NOTHING
	`

	_, err := db.pool.Exec(ctx, query, requestID, listingID, score)
	return err
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

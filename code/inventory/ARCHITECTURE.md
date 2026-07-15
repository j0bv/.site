# Architecture Overview

## System Components

### 1. Scraper (`scraper.go`)
- Uses **Rod** (Go-native browser automation) for web scraping
- Supports 8 truck marketplaces with configurable CSS selectors
- Implements human-like delays and scrolling behavior
- Handles IronConnect authentication
- Extracts truck data: title, year, make, model, price, miles, location, images, VIN, etc.

### 2. Database Layer (`database.go`)
- Uses **pgx/v5** for PostgreSQL connectivity
- Auto-creates schema on first run
- Implements upsert logic to handle duplicate listings
- Provides query methods for matching trucks to email criteria
- Tracks email requests and matches

### 3. Email Integration (`email.go`)
- Uses **Microsoft Graph SDK for Go** for Outlook integration
- Implements OAuth2 client credentials flow
- Fetches emails from inbox (last 24 hours)
- Extracts search criteria using regex patterns
- Matches trucks to email criteria
- Sends automated response emails

### 4. Main Orchestration (`main.go`)
- Coordinates scraper and email matcher
- Runs scraper every 45-75 minutes (randomized)
- Checks emails every 10 minutes
- Graceful shutdown on interrupt signals

## Data Flow

```
┌─────────────┐
│ Marketplace │
│   Websites  │
└──────┬──────┘
       │
       │ Scrape (Rod)
       ▼
┌─────────────┐
│   Scraper   │
└──────┬──────┘
       │
       │ Save Listings
       ▼
┌─────────────┐
│ PostgreSQL  │
│  Database   │
└──────┬──────┘
       │
       │ Query Matches
       ▲
┌──────┴──────┐
│Email Matcher│
└──────┬──────┘
       │
       │ Fetch Emails
       ▼
┌─────────────┐
│  Microsoft  │
│ Graph API    │
│  (Outlook)   │
└─────────────┘
```

## Key Design Decisions

### Why Rod Instead of Playwright?
- **Go-native**: No Node.js dependency
- **Lighter**: Smaller binary size
- **Simpler**: No async/await complexity
- **Production-proven**: 11.5k+ GitHub stars

### Why pgx Instead of GORM?
- **Performance**: Direct SQL, no ORM overhead
- **Type safety**: Better compile-time checking
- **Control**: Full SQL power when needed
- **Simplicity**: Less abstraction for this use case

### Why Microsoft Graph SDK?
- **Official**: Microsoft-maintained
- **Full-featured**: Complete Outlook API support
- **Well-documented**: Extensive examples available
- **Production-ready**: Used by enterprise applications

## Database Schema

### truck_listings
Stores all scraped truck listings with:
- Truck specifications (year, make, model, condition, etc.)
- Financial data (price, down payment, monthly payment)
- Location information
- Media URLs (images)
- Tracking timestamps
- Email matching references

### email_search_requests
Tracks client email requests with:
- Email metadata (from, subject, body)
- Extracted search criteria
- Processing timestamps

### email_matches
Links email requests to matching trucks:
- Request ID → Listing ID mapping
- Match confidence score
- Match timestamp

## Marketplace Support

Each marketplace has:
- **Base URL**: Starting point for scraping
- **Listing Selector**: CSS selector to find listing containers
- **Field Selectors**: Multiple fallback selectors for each field
- **Login Requirements**: Whether authentication is needed

### Selector Strategy
- Multiple fallback selectors per field (websites change frequently)
- Graceful degradation (missing fields don't break scraping)
- Data validation before saving

## Data Sources and Extraction

All scrapers produce `[]*agent.ListingExtract`, which the coordinator converts to `TruckListing` via `listingExtractsToTruckListings`. Extraction varies by source:

### Pure JSON (ArrowTruck, SelecTrucks)
- Call JSON API, decode into typed structs (`SearchResponse.Products`, `SearchResponse.Items`).
- Convert each item via `ConvertProductToExtract` / `ConvertTruckToExtract` to `*agent.ListingExtract`.
- No HTML parsing, no LLM. Coordinator uses `listingExtractsToTruckListings(extracts, marketplace)`.

### JSON-with-HTML (TruckPlanet, TruckTractorTrailer, HeavyTruckDealers)
- Decode JSON; read HTML from the known field (`Divs[sr_results]`, `Listings`, `Template`).
- Pass that HTML to `agent.ExtractListingsFromHTML`; outcome is `[]*agent.ListingExtract`.
- Coordinator uses `listingExtractsToTruckListings(extracts, marketplace)`.

### HTML-only (IronConnect, IronPlanet, TruckertoTrucker)
- Fetch HTML page (or fragment), pass to `agent.ExtractListingsFromHTML`.
- Coordinator uses `listingExtractsToTruckListings(extracts, marketplace)`.

## Email Matching Logic

### Criteria Extraction
Uses regex patterns to extract:
- **Year**: `(\d{4})`, `(\d{4})\s*(?:to|-)\s*(\d{4})`
- **Make**: `(peterbilt|freightliner|volvo|kenworth|mack|international|daimler|cummins)`
- **Model**: Extracted after make detection
- **Price**: `\$?(\d+)(?:k|000)?`
- **Miles**: `(\d+)\s*(?:k|000)?\s*miles`
- **Location**: `(?:in|near|around)\s+([a-z\s]+?)(?:,|\.|$)`

### Matching Query
Builds dynamic SQL query based on extracted criteria:
- Year range filtering
- Make/model pattern matching (case-insensitive LIKE)
- Price range filtering
- Mileage filtering
- Location filtering (future enhancement)

## Error Handling

- **Scraper errors**: Logged, continue with next marketplace
- **Database errors**: Logged, skip problematic listings
- **Email errors**: Logged, continue with next email
- **Authentication errors**: Fatal (system can't function without auth)

## Performance Considerations

- **Connection pooling**: pgx connection pool for database
- **Concurrent scraping**: Can be extended to scrape marketplaces in parallel
- **Batch operations**: Database upserts are efficient
- **Token caching**: Microsoft Graph tokens are cached and refreshed automatically

## Security

- **Credentials**: Stored in environment variables, never in code
- **Database**: Uses parameterized queries (SQL injection protection)
- **OAuth2**: Secure token-based authentication for Microsoft Graph
- **Service user**: Runs as non-root user in production

## Monitoring

- **Logging**: Structured logging to stdout/journal
- **Database**: Can query `truck_listings` table for scrape success
- **Email**: Check `email_search_requests` for processed emails
- **Systemd**: Built-in service monitoring via `systemctl status`

## Future Enhancements

1. **Parallel scraping**: Scrape multiple marketplaces concurrently
2. **Location-based matching**: Geocoding and radius-based matching
3. **Image processing**: Download and analyze truck images
4. **Price alerts**: Notify when price drops below threshold
5. **Webhook support**: Real-time notifications instead of email polling
6. **Admin dashboard**: Web UI for monitoring and configuration
7. **Selector auto-update**: Machine learning to adapt to website changes

## Known Limitations

1. **HTML extraction**: Uses LLM (Ollama) when `SLM_ENABLED` is not "false"; otherwise a shared selector fallback. LLM availability and prompt design affect quality.
2. **Email Parsing**: Regex-based when SLM is disabled; SLM when enabled.
3. **Single-threaded scraping**: One marketplace at a time (can be parallelized).
4. **No retry logic**: Failed scrapes are logged but not retried automatically.
5. **Microsoft Graph Auth**: May need adjustment based on actual SDK version.

## Testing Strategy

1. **Extraction testing**: Use `make test-extraction` to run sample HTML through `agent.ExtractListingFromHTML` and `agent.ExtractListingsFromHTML`.
2. **JSON converter tests**: `go test ./scrapers/arrowtruck/... ./scrapers/selectrucks/...` for `ConvertProductToExtract` and `ConvertTruckToExtract`.
3. **Database testing**: Verify schema creation and queries.
4. **Email testing**: Send test emails and verify matching.
5. **Integration testing**: Run full system with test data.
6. **Production monitoring**: Watch logs and database for issues.
7. **Cleanup agent** (planned): Optional second LLM pass to complete missing or improper fields.

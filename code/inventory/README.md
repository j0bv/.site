# Truck Inventory Scraper + Email Matcher

Production-ready Go application that scrapes truck listings from 8 major marketplaces and automatically matches them to client email requests via Outlook integration.

## Features

- **Multi-Marketplace Scraping**: Automatically scrapes listings from:
  - IronConnect
  - CommercialTruckTrader
  - TruckPlanet
  - TrukerToTrucker
  - SelecTrucks
  - PenskeUsedTrucks
  - IronPlanet
  - TLGPeterbilt

- **PostgreSQL Database**: Stores all listings with full truck specifications
- **Outlook Email Integration**: Monitors inbox for client requests via Microsoft Graph API
- **Intelligent Matching**: Parses email content to extract search criteria (year, make, model, price, miles, location)
- **Automated Responses**: Sends matching truck listings back to clients via email

- **Data sources**: ArrowTruck and SelecTrucks use pure JSON APIs; TruckPlanet, TruckTractorTrailer, and HeavyTruckDealers use JSON responses with embedded HTML; IronConnect, IronPlanet, and TruckertoTrucker use HTML pages. All feed into a unified `agent.ListingExtract` pipeline. See ARCHITECTURE.md.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│         TRUCK INVENTORY + OUTLOOK EMAIL SYSTEM          │
└─────────────────────────────────────────────────────────┘
                        │
        ┌───────────────┼───────────────┐
        │               │               │
   Marketplace      PostgreSQL    Microsoft Graph API
   Scraper         Database         (Outlook)
```

## Prerequisites

- Go 1.21 or later
- PostgreSQL 12 or later
- Microsoft Azure AD App Registration (for Outlook integration)
- IronConnect account (for authenticated scraping)

## Setup

### 1. Database Setup

```bash
# Create database
createdb truck_inventory

# The application will automatically create tables on first run
```

### 2. Microsoft Graph API Setup

1. Register an application in Azure AD
2. Grant the following permissions:
   - `Mail.Read` (Application permission)
   - `Mail.Send` (Application permission)
3. Create a client secret
4. Add credentials to `.env` file

### 3. Configuration

Copy `.env.example` to `.env` and fill in your credentials:

```bash
cp .env.example .env
# Edit .env with your credentials
```

### 4. Install Dependencies

```bash
go mod download
```

### 5. Build

```bash
go build -o truck-inventory main.go
```

## Usage

### Run Locally

```bash
./truck-inventory
```

The application will:
- Initialize database schema
- Start scraping marketplaces every 45-75 minutes
- Check for new emails every 10 minutes
- Match trucks to email requests and send responses

### Run as Systemd Service

See `deployment/truck-inventory.service` for systemd configuration.

```bash
sudo cp deployment/truck-inventory.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable truck-inventory
sudo systemctl start truck-inventory
```

## Testing extraction

To validate SLM-based listing extraction (or the shared fallback when `SLM_ENABLED=false`):

```bash
make test-extraction
```

Runs sample HTML fragments through `agent.ExtractListingFromHTML` and `agent.ExtractListingsFromHTML`.

## Database Schema

The application creates three main tables:

- `truck_listings`: Stores all scraped truck listings
- `email_search_requests`: Tracks client email requests
- `email_matches`: Links email requests to matching trucks

See `database.go` for the complete schema.

## Email Matching

The system extracts search criteria from email subject and body using regex patterns:

- **Year**: "2023", "2020-2024", "from 2020 to 2024"
- **Make/Model**: "Peterbilt", "Freightliner 114SD", "Volvo VNL"
- **Price**: "$50000", "$50k", "under $100000"
- **Miles**: "100000 miles", "under 500k miles"
- **Location**: "in Chicago", "near Texas"

## Deployment

The application compiles to a single binary (~18MB) with no runtime dependencies.

### Production Checklist

- [ ] PostgreSQL database configured and accessible
- [ ] Microsoft Graph API credentials configured
- [ ] IronConnect credentials configured
- [ ] CSS selectors tested on all marketplaces
- [ ] Systemd service configured (if using)
- [ ] Logging configured
- [ ] Monitoring set up

## Troubleshooting

### Scraper Issues

- Verify CSS selectors are still valid (websites change frequently)
- Check browser automation is working (Rod/Chromium)
- Verify IronConnect login credentials

### Email Issues

- Verify Microsoft Graph API credentials
- Check Azure AD app permissions
- Verify OAuth2 token acquisition

### Database Issues

- Check PostgreSQL connection string
- Verify database user has CREATE TABLE permissions
- Check for connection pool exhaustion

## License

MIT

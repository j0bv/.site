Here is the structured Page artifact synthesized from the group chat:

Page: Commercial Truck Scraping API Exploration

Context

The group explored multiple commercial truck listing websites to identify viable scraping strategies. The goal was to locate inventory endpoints, understand pagination, and evaluate anti‑scraping protections across different platforms. The conversation moved sequentially through several sites, documenting findings and strategies.

Sites Investigated & Findings

1. CommercialTruckTrader

Observation: Encountered DataDome bot‑protection endpoints (api-js.datadome.co/js/).

Issue: Inventory API hidden behind bot‑protection layer.

Finding: JSON responses contained only cookies, fingerprints, and tracking metadata.

Status: Blocked temporarily for using DevTools; scraping here is high‑difficulty.

2. TruckertoTrucker.com

Endpoint: https://truckertotrucker.com/ads?page=N

Structure: Simple pagination over HTML pages.

Finding: Listings embedded directly in HTML (<div class="ad"> blocks).

Difficulty: Very low; no bot protection, no hidden API.

Strategy: Increment page numbers until no listings remain.

3. TruckPlanet.com

Endpoint: https://www.truckplanet.com/jsp/s/search.ips?prows=100&sm=0&json=true

Structure: JSON envelope containing HTML fragments.

Key Divs:

sr_results → listing cards

sr_facets → filters

sr_pagination → pagination controls

Pagination: Controlled by sm offset (0, 100, 200…).

Difficulty: Medium; requires parsing HTML embedded in JSON.

4. SelecTrucks.com

Endpoint: POST https://www.selectrucks.com/api/v1/inventory/search/

Structure: Pure JSON API.

Fields: Dealer info, manufacturer, model, year, mileage, price, stock number, truck ID, image filename.

Pagination: Via POST body (page, pageSize).

Difficulty: Very low; clean JSON, no protection.

Status: One of the most scraper‑friendly APIs discovered.

5. IronPlanet.com

Observation: Captured tracking beacon endpoints (/jsp/track/hit.jsp, a.ironpla.net/dw/hit).

Finding: These are analytics, not inventory.

Real Data: Listings embedded in HTML pages (<div class="sr_item">).

Pagination: Controlled by sm parameter (increments of 60).

Difficulty: Medium; HTML parsing required.

6. ArrowTruck.com

Endpoint: POST https://api.arrowl5secure.com/services/searchproducts

Structure: Pure JSON API with rich truck specs.

Fields: Year, make, model, mileage, price, location, engine, axle, transmission, sleeper, wheelbase, tire size, media galleries.

Pagination: pageIndex and pageSize in POST body.

Difficulty: Very low; clean JSON, no protection.

Status: Alongside SelecTrucks, one of the best APIs found.

Comparative Difficulty Table

Site

Difficulty

Notes

ArrowTruck

⭐ Easiest

Full JSON API, rich fields, no protection

SelecTrucks

⭐ Easiest

Full JSON API, simple pagination

TruckertoTrucker

Low

HTML parsing, simple pagination

TruckPlanet

Medium

JSON wrapper with embedded HTML

IronPlanet

Medium

HTML only, noisy tracking layer

CommercialTruckTrader

🔥 Hardest

DataDome bot protection, hidden API

Next Steps

SelecTrucks & ArrowTruck: Build full Go scrapers using JSON APIs; export to CSV.

TruckertoTrucker, TruckPlanet, IronPlanet: Implement HTML parsing with Goquery; handle pagination loops.

CommercialTruckTrader: Consider advanced bypass strategies (cookies, headers, IP rotation) or deprioritize due to difficulty.

Normalization: Standardize fields (price, mileage, dealer info) across sites.

Export: Create unified CSV/DB schema for aggregated inventory.

Future Considerations

Add concurrency for faster scraping.

Handle media galleries and images.

Normalize inconsistent field formats (e.g., mileage as string vs integer).

Monitor for API changes or new anti‑scraping measures.

Consider ethical and legal implications of scraping commercial sites.

End of Document

This Page artifact consolidates the exploratory debugging and scraping strategy discussion into a structured, actionable reference for building scrapers across multiple truck listing platforms.
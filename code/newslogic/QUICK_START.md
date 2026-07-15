# BitNet Worker - Quick Start

## Complete Pipeline: XML → Lightpanda → BitNet → Neo4j

## Setup

1. **Install Neo4j** (if not already installed)
2. **Start Lightpanda** browser:
   ```bash
   docker run -d --name lightpanda -p 9222:9222 lightpanda/browser:nightly
   ```

3. **Set Environment Variables**:
   ```bash
   export NEO4J_PASSWORD=your_password
   export LIGHTPANDA_HOST=127.0.0.1
   export LIGHTPANDA_PORT=9222
   ```

4. **Place GeoJSON files** in `geojson/` directory (for 757 area locations)

## Run the Worker

```bash
cargo run -- bitnet-worker
```

## What It Does

1. **Reads RSS XML files** from `output/` directory
2. **For each article**:
   - Extracts full content using Lightpanda (if URL available)
   - Falls back to RSS content if extraction fails
   - Analyzes with BitNet agent:
     - Classifies event types (accident, crime, weather, etc.)
     - Extracts locations and geocodes using GeoJSON
     - Extracts entities (organizations, locations, people)
   - Stores in Neo4j:
     - Article nodes
     - Event nodes with coordinates
     - Entity nodes
     - Relationships between them

## Output

All data stored in Neo4j:
- **Articles** with metadata
- **Events** with lat/lon coordinates
- **Entities** (Person, Organization, Location)
- **Relationships**: Article → Event → Location, Article → Entity

## Performance

- Processes articles sequentially with 100ms delay
- Logs progress every 10 articles
- Handles failures gracefully (continues on error)

## Next: Frontend

Once data is in Neo4j, the frontend (757.was.news) can query it via the API server:
```bash
cargo run -- api-server  # (requires axum dependencies)
```

Or query Neo4j directly for static/local frontend.

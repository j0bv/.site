# BitNet AI Agent Architecture

## Overview
The BitNet agent extracts events from news articles, geocodes them using GeoJSON data for the 757 area, and stores them in Neo4j with spatial coordinates for visualization.

## Mapping Library Recommendation

### **Leaflet.js** (Recommended)
- **Why**: Most popular, simple API, excellent GeoJSON support
- **Pros**: 
  - Lightweight (~40KB)
  - Great documentation
  - Works seamlessly with GeoJSON
  - Large plugin ecosystem
  - No API keys required for basic use
- **Cons**: Less performant than Mapbox GL for very large datasets

### Alternative: Mapbox GL JS
- **Pros**: Better performance, 3D support, advanced styling
- **Cons**: Requires API key (free tier available), more complex

### Implementation
Create a simple HTML/JS frontend that:
1. Fetches events from Neo4j via REST API
2. Displays them on Leaflet map with GeoJSON overlays
3. Shows event details on click

## Architecture Components

### 1. GeoJSON Data Layer (`src/geo_data.rs`)
- Loads GeoJSON files from directory
- Indexes features by name
- Provides location lookup and geocoding
- Calculates polygon centers for area features

### 2. BitNet Agent (`src/bitnet_agent.rs`)
- Extracts events from article content
- Classifies event types (accident, crime, weather, etc.)
- Geocodes locations using GeoJSON index
- Extracts entities (people, organizations, locations)

### 3. Neo4j Storage (`src/memory_graph.rs`)
- Stores Events with coordinates
- Creates relationships: Article → Event → Location
- Supports spatial queries

## Data Flow

```
Article → BitNet Agent → Events + Locations → Neo4j → Web API → Leaflet Map
```

## Next Steps

1. **Fix GeoJSON crate compilation** - Ensure geojson crate is properly added
2. **Add main.rs command** - Create `bitnet-process` command to run agent
3. **Create web server** - Simple HTTP server to serve events as JSON
4. **Build Leaflet frontend** - HTML page with map visualization
5. **Enhance event extraction** - Add LLM integration for better extraction

## File Structure

```
newslogic/
├── src/
│   ├── geo_data.rs          # GeoJSON loading and lookup
│   ├── bitnet_agent.rs      # Event extraction agent
│   ├── memory_graph.rs      # Neo4j storage (extended)
│   └── main.rs              # CLI commands
├── geojson/                 # Your 757 area GeoJSON files
│   ├── norfolk.geojson
│   ├── virginia-beach.geojson
│   └── ...
└── web/                     # Frontend (to be created)
    ├── index.html
    ├── map.js
    └── style.css
```

## Environment Variables

```bash
NEO4J_URI=127.0.0.1:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=your_password
NEO4J_DATABASE=neo4j
GEOJSON_DIR=./geojson
```

## Usage

```bash
# Process articles and extract events
cargo run -- bitnet-process

# Start web server for visualization
cargo run -- bitnet-serve --port 8080
```

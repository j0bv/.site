# BitNet Worker Pipeline

## Overview
Complete pipeline: XML → Lightpanda → Document Extraction → BitNet → Neo4j

## Pipeline Flow

```
RSS XML Files → Parse Articles → Lightpanda Extract → BitNet Analysis → Neo4j Storage
```

## Components

### 1. XML Parser (`parser.rs`)
- Reads RSS XML files from `output/` directory
- Extracts article metadata (title, description, link, date, etc.)
- Deduplicates by content hash

### 2. Lightpanda Client (`lightpanda_client.rs`)
- Extracts full article content from URLs
- Removes ads, navigation, sidebars
- Returns clean text and HTML

### 3. BitNet Agent (`bitnet_agent.rs`)
- **Basic Prompt-Based Extraction**:
  - Event classification (accident, crime, weather, government, etc.)
  - Location extraction using GeoJSON index
  - Entity extraction (organizations, locations, people)
- Uses pattern matching and keyword detection
- Geocodes locations using 757 area GeoJSON data

### 4. Neo4j Storage (`memory_graph.rs`)
- Stores Articles as nodes
- Stores Events with coordinates
- Stores Entities (Person, Organization, Location, etc.)
- Creates relationships:
  - Article → DESCRIBES → Event
  - Event → OCCURRED_AT → Location
  - Article → MENTIONS_ENTITY → Entity

## Usage

```bash
# Set required environment variables
export NEO4J_PASSWORD=your_password
export LIGHTPANDA_HOST=127.0.0.1
export LIGHTPANDA_PORT=9222

# Run the worker
cargo run -- bitnet-worker
```

## What It Does

1. **Reads all RSS XML files** from `output/` directory
2. **For each article**:
   - Extracts full content using Lightpanda (if URL available)
   - Analyzes text with BitNet agent
   - Extracts events and entities
   - Geocodes locations using GeoJSON
   - Stores everything in Neo4j

## Output

- Articles stored in Neo4j with all metadata
- Events with coordinates and relationships
- Entities linked to articles
- Location nodes with lat/lon from GeoJSON

## Performance

- Processes articles in batches
- Uses connection pooling for Neo4j
- Rate limiting (100ms delay between articles)
- Progress logging every 10 articles

## Next Steps

1. Add LLM integration for better entity extraction
2. Add more sophisticated event classification
3. Add relationship extraction between entities
4. Add temporal event linking

# Backend API Server

## Overview
Fast, reliable Rust HTTP server using Axum that provides the API endpoints expected by the BitNet frontend.

## Performance Features

1. **Async/Await** - Full async support with Tokio
2. **Connection Pooling** - Neo4j connection pool (10 connections)
3. **CORS Support** - Configured for frontend access
4. **Request Tracing** - Built-in request logging
5. **Efficient Queries** - Optimized Neo4j queries with limits

## API Endpoints

### GET /api/map/events
Returns map events as JSON array.

**Query Parameters:**
- `start_date` (optional): ISO 8601 date string
- `end_date` (optional): ISO 8601 date string

**Response:**
```json
{
  "events": [
    {
      "id": "event_123",
      "title": "Event Title",
      "location_name": "Norfolk",
      "latitude": 36.8468,
      "longitude": -76.2852,
      "occurred_at": "2025-01-15T10:00:00Z",
      "entry_id": "article_abc",
      "entry_title": "Article Title"
    }
  ]
}
```

### GET /api/map/geojson
Returns map events as GeoJSON FeatureCollection (preferred by frontend).

**Query Parameters:**
- `start_date` (optional): ISO 8601 date string
- `end_date` (optional): ISO 8601 date string

**Response:**
```json
{
  "type": "FeatureCollection",
  "features": [
    {
      "id": "event_123",
      "type": "Feature",
      "geometry": {
        "type": "Point",
        "coordinates": [-76.2852, 36.8468]
      },
      "properties": {
        "id": "event_123",
        "title": "Event Title",
        "location_name": "Norfolk",
        "occurred_at": "2025-01-15T10:00:00Z"
      }
    }
  ]
}
```

### GET /api/memory/graph
Returns knowledge graph data for visualization.

**Query Parameters:**
- `entry_ids` (optional): Comma-separated entry IDs
- `limit` (optional): Maximum nodes to return (default: 1000)

**Response:**
```json
{
  "nodes": [
    {
      "id": "123",
      "label": "Article Title",
      "type": "Article",
      "properties": {
        "title": "Article Title",
        "id": "article_abc"
      }
    }
  ],
  "edges": [
    {
      "id": "456",
      "source": "123",
      "target": "789",
      "type": "OCCURRED_AT",
      "properties": {}
    }
  ]
}
```

## Running the Server

```bash
# Set required environment variables
export NEO4J_PASSWORD=your_password
export API_PORT=3001  # Optional, defaults to 3001

# Start the server
cargo run -- api-server
```

## Performance Optimizations

1. **Neo4j Query Limits** - All queries use LIMIT clauses
2. **Connection Reuse** - Single Neo4j client shared across requests
3. **Efficient Filtering** - Date filtering done in Rust after fetch
4. **JSON Serialization** - Fast serde_json serialization

## Reliability Features

1. **Error Handling** - Proper error responses with status codes
2. **Logging** - Request tracing and error logging
3. **Graceful Shutdown** - Proper cleanup on shutdown
4. **Connection Pooling** - Handles connection failures gracefully

## Frontend Integration

The frontend expects these endpoints at:
- Base URL: `http://localhost:3001` (or configured API_URL)
- CORS: Enabled for all origins
- Content-Type: `application/json`

## Next Steps

1. Add authentication/authorization if needed
2. Add rate limiting for production
3. Add response caching for frequently accessed data
4. Add health check endpoint
5. Add metrics endpoint

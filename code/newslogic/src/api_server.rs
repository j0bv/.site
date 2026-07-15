use anyhow::Context;
use axum::{
    extract::{Path, Query, Request, State},
    http::{Method, StatusCode},
    response::Json,
    routing::{get, patch, post},
    Router,
};
use serde_json::Value as JsonValue;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::Arc;
use tower_http::cors::{Any, CorsLayer};
use tower_http::trace::TraceLayer;

use crate::memory_graph::Neo4jClient;
use crate::error::AppResult;

#[derive(Debug, Clone)]
pub struct ApiState {
    pub neo4j: Arc<Neo4jClient>,
}

#[derive(Debug, Deserialize)]
pub struct MapQueryParams {
    pub start_date: Option<String>,
    pub end_date: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct MemoryGraphQueryParams {
    pub entry_ids: Option<String>,
    pub limit: Option<u64>,
}

#[derive(Debug, Serialize)]
pub struct MapEvent {
    pub id: String,
    pub title: String,
    pub location_name: String,
    pub latitude: f64,
    pub longitude: f64,
    pub occurred_at: Option<String>,
    pub entry_id: String,
    pub entry_title: String,
}

#[derive(Debug, Serialize)]
pub struct EventsResponse {
    pub events: Vec<MapEvent>,
}

#[derive(Debug, Serialize)]
pub struct GeoJSONFeature {
    pub id: Option<String>,
    #[serde(rename = "type")]
    pub feature_type: String,
    pub geometry: GeoJSONGeometry,
    pub properties: HashMap<String, serde_json::Value>,
}

#[derive(Debug, Serialize)]
pub struct GeoJSONGeometry {
    #[serde(rename = "type")]
    pub geometry_type: String,
    pub coordinates: serde_json::Value,
}

#[derive(Debug, Serialize)]
pub struct GeoJSONFeatureCollection {
    #[serde(rename = "type")]
    pub collection_type: String,
    pub features: Vec<GeoJSONFeature>,
}

#[derive(Debug, Serialize)]
pub struct GraphNode {
    pub id: String,
    pub label: String,
    #[serde(rename = "type")]
    pub node_type: String,
    pub properties: HashMap<String, serde_json::Value>,
}

#[derive(Debug, Serialize)]
pub struct GraphEdge {
    pub id: String,
    pub source: String,
    pub target: String,
    #[serde(rename = "type")]
    pub edge_type: String,
    pub properties: HashMap<String, serde_json::Value>,
}

#[derive(Debug, Serialize)]
pub struct MemoryGraphResponse {
    pub nodes: Vec<GraphNode>,
    pub edges: Vec<GraphEdge>,
}

pub async fn get_map_events(
    State(state): State<ApiState>,
    Query(params): Query<MapQueryParams>,
) -> Result<Json<EventsResponse>, StatusCode> {
    let events = state.neo4j.get_events_with_locations().await
        .map_err(|e| {
            tracing::error!("Failed to fetch events: {}", e);
            StatusCode::INTERNAL_SERVER_ERROR
        })?;

    let mut map_events = Vec::new();

    for event_json in events {
        let id = event_json.get("event_id")
            .and_then(|v| v.as_str())
            .map(|s| s.to_string())
            .unwrap_or_default();

        let title = event_json.get("title")
            .and_then(|v| v.as_str())
            .map(|s| s.to_string())
            .unwrap_or_default();

        let location_name = event_json.get("location_name")
            .and_then(|v| v.as_str())
            .map(|s| s.to_string())
            .unwrap_or_default();

        let latitude = event_json.get("latitude")
            .and_then(|v| v.as_f64())
            .unwrap_or(0.0);

        let longitude = event_json.get("longitude")
            .and_then(|v| v.as_f64())
            .unwrap_or(0.0);

        let timestamp = event_json.get("timestamp")
            .and_then(|v| v.as_str())
            .map(|s| s.to_string());

        let entry_id = event_json.get("article_id")
            .and_then(|v| v.as_str())
            .map(|s| s.to_string())
            .unwrap_or_default();
        let entry_title = event_json.get("article_title")
            .and_then(|v| v.as_str())
            .map(|s| s.to_string())
            .unwrap_or_default();

        let map_event = MapEvent {
            id,
            title,
            location_name,
            latitude,
            longitude,
            occurred_at: timestamp,
            entry_id,
            entry_title,
        };

        if let Some(start_date) = &params.start_date {
            if let Some(ref occurred_at) = map_event.occurred_at {
                if occurred_at < start_date {
                    continue;
                }
            }
        }

        if let Some(end_date) = &params.end_date {
            if let Some(ref occurred_at) = map_event.occurred_at {
                if occurred_at > end_date {
                    continue;
                }
            }
        }

        map_events.push(map_event);
    }

    Ok(Json(EventsResponse { events: map_events }))
}

pub async fn get_map_geojson(
    State(state): State<ApiState>,
    Query(params): Query<MapQueryParams>,
) -> Result<Json<GeoJSONFeatureCollection>, StatusCode> {
    let events = state.neo4j.get_events_with_locations().await
        .map_err(|e| {
            tracing::error!("Failed to fetch events: {}", e);
            StatusCode::INTERNAL_SERVER_ERROR
        })?;

    let mut features = Vec::new();

    for event_json in events {
        let id = event_json.get("event_id")
            .and_then(|v| v.as_str())
            .map(|s| s.to_string());

        let title = event_json.get("title")
            .and_then(|v| v.as_str())
            .map(|s| s.to_string())
            .unwrap_or_default();

        let location_name = event_json.get("location_name")
            .and_then(|v| v.as_str())
            .map(|s| s.to_string())
            .unwrap_or_default();

        let latitude = event_json.get("latitude")
            .and_then(|v| v.as_f64())
            .unwrap_or(0.0);

        let longitude = event_json.get("longitude")
            .and_then(|v| v.as_f64())
            .unwrap_or(0.0);

        let timestamp = event_json.get("timestamp")
            .and_then(|v| v.as_str())
            .map(|s| s.to_string());

        if latitude == 0.0 && longitude == 0.0 {
            continue;
        }

        if let Some(start_date) = &params.start_date {
            if let Some(ref occurred_at) = timestamp {
                if occurred_at < start_date {
                    continue;
                }
            }
        }

        if let Some(end_date) = &params.end_date {
            if let Some(ref occurred_at) = timestamp {
                if occurred_at > end_date {
                    continue;
                }
            }
        }

        let entry_id = event_json.get("article_id")
            .and_then(|v| v.as_str())
            .map(|s| s.to_string())
            .unwrap_or_default();
        let entry_title = event_json.get("article_title")
            .and_then(|v| v.as_str())
            .map(|s| s.to_string())
            .unwrap_or_default();

        let mut properties = HashMap::new();
        properties.insert("id".to_string(), serde_json::Value::String(id.clone().unwrap_or_default()));
        properties.insert("title".to_string(), serde_json::Value::String(title.clone()));
        properties.insert("location_name".to_string(), serde_json::Value::String(location_name));
        properties.insert("entry_id".to_string(), serde_json::Value::String(entry_id));
        properties.insert("entry_title".to_string(), serde_json::Value::String(entry_title));
        if let Some(ts) = timestamp {
            properties.insert("occurred_at".to_string(), serde_json::Value::String(ts));
        }

        let geometry = GeoJSONGeometry {
            geometry_type: "Point".to_string(),
            coordinates: serde_json::json!([longitude, latitude]),
        };

        let feature = GeoJSONFeature {
            id,
            feature_type: "Feature".to_string(),
            geometry,
            properties,
        };

        features.push(feature);
    }

    Ok(Json(GeoJSONFeatureCollection {
        collection_type: "FeatureCollection".to_string(),
        features,
    }))
}

pub async fn get_memory_graph(
    State(state): State<ApiState>,
    Query(params): Query<MemoryGraphQueryParams>,
) -> Result<Json<MemoryGraphResponse>, StatusCode> {
    let limit = params.limit.unwrap_or(1000);
    let _entry_ids_filter = params.entry_ids.as_ref()
        .map(|s| s.split(',').map(|id| id.trim().to_string()).collect::<Vec<_>>());

    let mut result = state.neo4j.graph.execute(neo4rs::query(
        r#"
        MATCH (n)
        OPTIONAL MATCH (n)-[r]->(m)
        RETURN n, r, m
        LIMIT $limit
        "#)
        .param("limit", limit as i64))
        .await
        .map_err(|e| {
            tracing::error!("Failed to execute graph query: {}", e);
            StatusCode::INTERNAL_SERVER_ERROR
        })?;

    let mut nodes = Vec::new();
    let mut edges = Vec::new();
    let mut seen_nodes = std::collections::HashSet::new();

    while let Ok(Some(row)) = result.next().await {
        if let Ok(node) = row.get::<neo4rs::Node>("n") {
            let node_id = format!("{}", node.id());
            if !seen_nodes.contains(&node_id) {
                seen_nodes.insert(node_id.clone());

                let labels = node.labels();
                let node_type = labels.first().cloned().unwrap_or_else(|| "Node".to_string());

                let mut properties = HashMap::new();
                for key in node.keys() {
                    if let Ok(value) = node.get::<String>(&key) {
                        properties.insert(key.clone(), serde_json::Value::String(value));
                    } else if let Ok(value) = node.get::<i64>(&key) {
                        properties.insert(key.clone(), serde_json::Value::Number(serde_json::Number::from(value)));
                    } else if let Ok(value) = node.get::<f64>(&key) {
                        properties.insert(key.clone(), serde_json::Value::Number(serde_json::Number::from_f64(value).unwrap()));
                    }
                }

                let label = properties.get("title")
                    .or_else(|| properties.get("name"))
                    .and_then(|v| v.as_str())
                    .map(|s| s.to_string())
                    .unwrap_or_else(|| node_type.clone());

                nodes.push(GraphNode {
                    id: node_id,
                    label,
                    node_type,
                    properties,
                });
            }
        }

        if let Ok(rel) = row.get::<neo4rs::Relation>("r") {
            let edge_id = format!("{}", rel.id());
            let start_id = format!("{}", rel.start_node_id());
            let end_id = format!("{}", rel.end_node_id());
            let rel_type = rel.typ();

            let mut properties = HashMap::new();
            for key in rel.keys() {
                if let Ok(value) = rel.get::<String>(&key) {
                    properties.insert(key.clone(), serde_json::Value::String(value));
                }
            }

            edges.push(GraphEdge {
                id: edge_id,
                source: start_id,
                target: end_id,
                edge_type: rel_type,
                properties,
            });
        }
    }

    Ok(Json(MemoryGraphResponse { nodes, edges }))
}

// --- Stub handlers for GleanRSS @was/api-client compatibility (757 MVP) ---

async fn stub_auth_me(State(_): State<ApiState>) -> Json<JsonValue> {
    Json(serde_json::json!({
        "id": "stub",
        "email": "local@newslogic",
        "name": "Local User"
    }))
}

async fn stub_auth_login(State(_): State<ApiState>) -> Json<JsonValue> {
    Json(serde_json::json!({
        "user": { "id": "stub", "email": "local@newslogic", "name": "Local User" },
        "tokens": { "access_token": "stub-token", "refresh_token": "stub-refresh" }
    }))
}

async fn stub_auth_register(State(_): State<ApiState>) -> Json<JsonValue> {
    Json(serde_json::json!({
        "user": { "id": "stub", "email": "local@newslogic", "name": "Local User" },
        "tokens": { "access_token": "stub-token", "refresh_token": "stub-refresh" }
    }))
}

async fn stub_auth_logout(State(_): State<ApiState>) -> (StatusCode, Json<JsonValue>) {
    (StatusCode::OK, Json(serde_json::json!({})))
}

async fn stub_entries_list(State(_): State<ApiState>) -> Json<JsonValue> {
    Json(serde_json::json!({
        "items": [],
        "page": 1,
        "per_page": 20,
        "total": 0,
        "total_pages": 0
    }))
}

async fn stub_entries_get_404(State(_): State<ApiState>, Path(_id): Path<String>) -> (StatusCode, Json<JsonValue>) {
    (StatusCode::NOT_FOUND, Json(serde_json::json!({ "detail": "Not found" })))
}

async fn stub_entries_patch_state(State(_): State<ApiState>, Path(_id): Path<String>) -> Json<JsonValue> {
    Json(serde_json::json!({}))
}

async fn stub_entries_mark_all_read(State(_): State<ApiState>) -> Json<JsonValue> {
    Json(serde_json::json!({}))
}

async fn stub_empty_array(State(_): State<ApiState>) -> Json<JsonValue> {
    Json(serde_json::json!([]))
}

async fn stub_empty_items(State(_): State<ApiState>) -> Json<JsonValue> {
    Json(serde_json::json!({ "items": [] }))
}

async fn stub_folders_list(State(_): State<ApiState>) -> Json<JsonValue> {
    Json(serde_json::json!({ "folders": [] }))
}

async fn stub_folders_create(State(_): State<ApiState>) -> Json<JsonValue> {
    Json(serde_json::json!({
        "id": "stub-folder-1",
        "name": "New Folder",
        "parent_id": null
    }))
}

async fn stub_preferences(State(_): State<ApiState>) -> Json<JsonValue> {
    Json(serde_json::json!({ "enabled": false }))
}

async fn stub_vectorization_status(State(_): State<ApiState>) -> Json<JsonValue> {
    Json(serde_json::json!({ "enabled": false }))
}

async fn stub_catchall(State(_): State<ApiState>, req: Request) -> (StatusCode, Json<JsonValue>) {
    let body = if req.method() == Method::GET {
        serde_json::json!([])
    } else {
        serde_json::json!({})
    };
    (StatusCode::OK, Json(body))
}

pub fn create_router(neo4j_client: Arc<Neo4jClient>) -> Router {
    let cors = CorsLayer::new()
        .allow_origin(Any)
        .allow_methods(Any)
        .allow_headers(Any);

    let state = ApiState { neo4j: neo4j_client };

    Router::new()
        .route("/api/map/events", get(get_map_events))
        .route("/api/map/geojson", get(get_map_geojson))
        .route("/api/memory/graph", get(get_memory_graph))
        // Auth (stubs)
        .route("/api/auth/me", get(stub_auth_me))
        .route("/api/me", get(stub_auth_me))
        .route("/api/auth/login", post(stub_auth_login))
        .route("/api/auth/register", post(stub_auth_register))
        .route("/api/auth/logout", post(stub_auth_logout))
        // Entries (stubs)
        .route("/api/entries", get(stub_entries_list))
        .route("/api/entries/mark-all-read", post(stub_entries_mark_all_read))
        .route("/api/entries/:id", get(stub_entries_get_404))
        .route("/api/entries/:id/state", patch(stub_entries_patch_state))
        // Feeds, subscriptions, folders (stubs)
        .route("/api/feeds", get(stub_empty_items))
        .route("/api/subscriptions", get(stub_empty_array))
        .route("/api/folders", get(stub_folders_list).post(stub_folders_create))
        // Tags, bookmarks, stories (stubs)
        .route("/api/tags", get(stub_empty_array))
        .route("/api/bookmarks", get(stub_empty_array))
        .route("/api/stories", get(stub_empty_array))
        // Preference, system/vectorization (stubs)
        .route("/api/preferences", get(stub_preferences))
        .route("/api/preference", get(stub_preferences))
        .route("/api/system/vectorization-status", get(stub_vectorization_status))
        .fallback(stub_catchall)
        .layer(cors)
        .layer(TraceLayer::new_for_http())
        .with_state(state)
}

pub async fn start_server(neo4j_client: Arc<Neo4jClient>, port: u16) -> AppResult<()> {
    let app = create_router(neo4j_client);

    let listener = tokio::net::TcpListener::bind(format!("0.0.0.0:{}", port)).await
        .context("Failed to bind server")?;

    tracing::info!("API server listening on http://0.0.0.0:{}", port);

    axum::serve(listener, app).await
        .context("Server error")?;

    Ok(())
}

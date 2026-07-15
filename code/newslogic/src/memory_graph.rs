use serde::{Deserialize, Serialize};
use serde_json::{Map, Value as JsonValue};
use crate::geo_data::{feed_id_to_location, resolve_location, GeoJsonIndex, LocationCoordinates};
use crate::parser::Article;
use crate::models::{ExtractedClaim, GeoLocation, VerificationResult};
use crate::nlp_processor::DocumentQuality;
use crate::error::{AppResult, Neo4jError};
use anyhow::Context;
use chrono::{DateTime, Utc};
#[allow(unused_imports)]
use futures::stream::StreamExt; // required for RowStream::next() in stream iteration
use neo4rs::{Graph, ConfigBuilder, query};
use std::collections::HashSet;
use std::sync::Arc;

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct DocumentWithMemories {
    pub id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub custom_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content_hash: Option<String>,
    pub org_id: String,
    pub user_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub connection_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub title: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub summary: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub source: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub r#type: Option<String>,
    pub status: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub metadata: Option<Map<String, JsonValue>>,
    pub is_dragging: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location: Option<LocationData>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct LocationData {
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
    pub name: Option<String>,
    pub geo_tags: Vec<String>,
}

impl DocumentWithMemories {
    pub fn from_article(
        article: &Article,
        org_id: &str,
        user_id: &str,
    ) -> Self {
        let mut metadata = Map::new();
        
        metadata.insert("feed_id".to_string(), JsonValue::String(article.feed_id.clone()));
        metadata.insert("feed_name".to_string(), JsonValue::String(article.feed_name.clone()));
        
        if !article.categories.is_empty() {
            let categories_json: Vec<JsonValue> = article.categories.iter()
                .map(|cat| JsonValue::String(cat.clone()))
                .collect();
            metadata.insert("categories".to_string(), JsonValue::Array(categories_json));
        }
        
        if let Some(ref author) = article.author {
            metadata.insert("author".to_string(), JsonValue::String(author.clone()));
        }
        
        if let Some(ref pub_date) = article.pub_date {
            metadata.insert("pub_date".to_string(), JsonValue::String(pub_date.to_rfc3339()));
        }
        
        let location = if !article.geo_tags.is_empty() {
            Some(LocationData {
                latitude: None,
                longitude: None,
                name: None,
                geo_tags: article.geo_tags.clone(),
            })
        } else {
            None
        };
        
        let full_text = article.full_text();
        let summary = article.description.clone()
            .or_else(|| {
                if full_text.len() > 200 {
                    Some(full_text.chars().take(200).collect::<String>() + "...")
                } else {
                    None
                }
            });
        
        let doc_id = format!("article_{}", article.content_hash);
        
        Self {
            id: doc_id.clone(),
            custom_id: article.guid.clone(),
            content_hash: Some(article.content_hash.clone()),
            org_id: org_id.to_string(),
            user_id: user_id.to_string(),
            connection_id: None,
            title: Some(article.title.clone()),
            content: Some(full_text),
            summary,
            url: article.link.clone(),
            source: Some(article.feed_name.clone()),
            r#type: Some("article".to_string()),
            status: "done".to_string(),
            metadata: Some(metadata),
            is_dragging: false,
            location,
        }
    }
    
    pub fn to_json(&self) -> AppResult<JsonValue> {
        serde_json::to_value(self)
            .context("Failed to serialize document to JSON")
            .map_err(Into::into)
    }
}

pub struct Neo4jClient {
    graph: Arc<Graph>,
    pub org_id: String,
    pub user_id: String,
    geojson_index: Option<GeoJsonIndex>,
}

impl Neo4jClient {
    async fn new_impl(org_id: String, user_id: String, geojson_index: Option<GeoJsonIndex>) -> AppResult<Self> {
        let uri = std::env::var("NEO4J_URI")
            .unwrap_or_else(|_| "127.0.0.1:7687".to_string());
        let user = std::env::var("NEO4J_USER")
            .unwrap_or_else(|_| "neo4j".to_string());
        let password = std::env::var("NEO4J_PASSWORD")
            .context("NEO4J_PASSWORD environment variable is required")?;
        let db = std::env::var("NEO4J_DATABASE")
            .unwrap_or_else(|_| "neo4j".to_string());
        let mut config_builder = ConfigBuilder::default()
            .uri(&uri)
            .user(&user)
            .password(&password)
            .max_connections(10);
        if !db.is_empty() && db != "neo4j" {
            config_builder = config_builder.db(db.as_str());
        }
        let config = config_builder.build().context("Failed to build Neo4j config")?;
        let graph = Graph::connect(config).await
            .map_err(|e| Neo4jError::ConnectionFailed(e.to_string()))?;
        Ok(Self {
            graph: Arc::new(graph),
            org_id,
            user_id,
            geojson_index,
        })
    }

    pub async fn new(org_id: String, user_id: String) -> AppResult<Self> {
        Self::new_impl(org_id, user_id, None).await
    }

    pub async fn new_with_geojson(org_id: String, user_id: String, geojson_index: Option<GeoJsonIndex>) -> AppResult<Self> {
        Self::new_impl(org_id, user_id, geojson_index).await
    }

    /// Venues from GeoJSON index within `max_km` of (lat, lon). Uses `find_nearest`.
    pub fn get_nearby_venues(&self, lat: f64, lon: f64, max_km: f64) -> Vec<LocationCoordinates> {
        self.geojson_index
            .as_ref()
            .map(|g| g.find_nearest(lat, lon, max_km))
            .unwrap_or_default()
    }
    
    pub async fn add_article(
        &self,
        article: &Article,
        extra_location_names: Option<&[String]>,
    ) -> AppResult<()> {
        let doc = DocumentWithMemories::from_article(article, &self.org_id, &self.user_id);
        
        let mut txn = self.graph.start_txn().await
            .context("Failed to start Neo4j transaction")?;
        
        let article_id = doc.id.clone();
        let content_hash = doc.content_hash.as_ref()
            .map(|s| s.as_str())
            .unwrap_or("");
        
        let title = doc.title.as_deref().unwrap_or("Untitled");
        let content = doc.content.as_deref().unwrap_or("");
        let summary = doc.summary.as_deref().unwrap_or("");
        let url = doc.url.as_deref().unwrap_or("");
        let custom_id = doc.custom_id.as_deref().unwrap_or("");
        
        let pub_date = article.pub_date
            .map(|dt| dt.to_rfc3339())
            .unwrap_or_else(String::new);
        
        let mut queries = Vec::new();
        
        let create_article_query = query(
            r#"
            MERGE (a:Article {id: $id, content_hash: $content_hash})
            SET a.title = $title,
                a.content = $content,
                a.summary = $summary,
                a.url = $url,
                a.guid = $guid,
                a.org_id = $org_id,
                a.user_id = $user_id,
                a.status = $status,
                a.feed_id = $feed_id,
                a.feed_name = $feed_name,
                a.published_date = $pub_date,
                a.created_at = datetime()
            "#)
            .param("id", article_id.clone())
            .param("content_hash", content_hash)
            .param("title", title)
            .param("content", content)
            .param("summary", summary)
            .param("url", url)
            .param("guid", custom_id)
            .param("org_id", self.org_id.as_str())
            .param("user_id", self.user_id.as_str())
            .param("status", doc.status.as_str())
            .param("feed_id", article.feed_id.as_str())
            .param("feed_name", article.feed_name.as_str())
            .param("pub_date", pub_date);
        
        queries.push(create_article_query);
        
        if let Some(author) = &article.author {
            let author_query = query(
                r#"
                MERGE (author:Author {name: $author_name})
                MERGE (a:Article {id: $article_id})
                MERGE (a)-[:AUTHORED_BY]->(author)
                "#)
                .param("author_name", author.as_str())
                .param("article_id", article_id.clone());
            queries.push(author_query);
        }
        
        for category in &article.categories {
            let category_query = query(
                r#"
                MERGE (cat:Category {name: $category_name})
                MERGE (a:Article {id: $article_id})
                MERGE (a)-[:IN_CATEGORY]->(cat)
                "#)
                .param("category_name", category.as_str())
                .param("article_id", article_id.clone());
            queries.push(category_query);
        }
        
        let feed_query = query(
            r#"
            MERGE (feed:Feed {id: $feed_id})
            SET feed.name = $feed_name
            MERGE (a:Article {id: $article_id})
            MERGE (a)-[:FROM_FEED]->(feed)
            "#)
            .param("feed_id", article.feed_id.as_str())
            .param("feed_name", article.feed_name.as_str())
            .param("article_id", article_id.clone());
        queries.push(feed_query);
        let mut loc_names: HashSet<String> = article.geo_tags.iter().cloned().collect();
        if let Some(f) = feed_id_to_location(&article.feed_id) {
            loc_names.insert(f);
        }
        if let Some(extra) = extra_location_names {
            for n in extra {
                loc_names.insert(n.clone());
            }
        }
        for name in loc_names {
            let coords = resolve_location(&name, self.geojson_index.as_ref());
            let (lat, lon) = coords.unwrap_or((0.0, 0.0));
            let loc_q = if coords.is_some() {
                query(
                    r#"
                    MERGE (loc:Location {name: $location_name})
                    SET loc.latitude = $latitude, loc.longitude = $longitude
                    MERGE (a:Article {id: $article_id})
                    MERGE (a)-[:MENTIONS_LOCATION]->(loc)
                    "#,
                )
                .param("location_name", name.as_str())
                .param("latitude", lat)
                .param("longitude", lon)
                .param("article_id", article_id.clone())
            } else {
                query(
                    r#"
                    MERGE (loc:Location {name: $location_name})
                    MERGE (a:Article {id: $article_id})
                    MERGE (a)-[:MENTIONS_LOCATION]->(loc)
                    "#,
                )
                .param("location_name", name.as_str())
                .param("article_id", article_id.clone())
            };
            queries.push(loc_q);
        }
        txn.run_queries(queries).await
            .context("Failed to execute queries in transaction")?;
        txn.commit().await
            .context("Failed to commit Neo4j transaction")?;
        Ok(())
    }

    pub async fn add_articles_batch(&self, articles: &[Article]) -> AppResult<()> {
        let mut txn = self.graph.start_txn().await
            .context("Failed to start Neo4j transaction")?;
        
        let mut queries = Vec::new();
        
        for article in articles {
            let doc = DocumentWithMemories::from_article(article, &self.org_id, &self.user_id);
            let article_id = doc.id.clone();
            let content_hash = doc.content_hash.as_ref()
                .map(|s| s.as_str())
                .unwrap_or("");
            
            let title = doc.title.as_deref().unwrap_or("Untitled");
            let content = doc.content.as_deref().unwrap_or("");
            let summary = doc.summary.as_deref().unwrap_or("");
            let url = doc.url.as_deref().unwrap_or("");
            let source = doc.source.as_deref().unwrap_or("");
            let custom_id = doc.custom_id.as_deref().unwrap_or("");
            
            let pub_date = article.pub_date
                .map(|dt| dt.to_rfc3339())
                .unwrap_or_else(String::new);
            
            let create_query = query(
                r#"
                MERGE (a:Article {id: $id, content_hash: $content_hash})
                SET a.title = $title,
                    a.content = $content,
                    a.summary = $summary,
                    a.url = $url,
                    a.source = $source,
                    a.guid = $guid,
                    a.org_id = $org_id,
                    a.user_id = $user_id,
                    a.status = $status,
                    a.feed_id = $feed_id,
                    a.feed_name = $feed_name,
                    a.published_date = $pub_date,
                    a.created_at = datetime()
                "#)
                .param("id", article_id.clone())
                .param("content_hash", content_hash)
                .param("title", title)
                .param("content", content)
                .param("summary", summary)
                .param("url", url)
                .param("source", source)
                .param("guid", custom_id)
                .param("org_id", self.org_id.as_str())
                .param("user_id", self.user_id.as_str())
                .param("status", doc.status.as_str())
                .param("feed_id", article.feed_id.as_str())
                .param("feed_name", article.feed_name.as_str())
                .param("pub_date", pub_date);
            
            queries.push(create_query);
            
            if let Some(author) = &article.author {
                let author_query = query(
                    r#"
                    MERGE (author:Author {name: $author_name})
                    MERGE (a:Article {id: $article_id})
                    MERGE (a)-[:AUTHORED_BY]->(author)
                    "#)
                    .param("author_name", author.as_str())
                    .param("article_id", article_id.clone());
                queries.push(author_query);
            }
            
            for category in &article.categories {
                let category_query = query(
                    r#"
                    MERGE (cat:Category {name: $category_name})
                    MERGE (a:Article {id: $article_id})
                    MERGE (a)-[:IN_CATEGORY]->(cat)
                    "#)
                    .param("category_name", category.as_str())
                    .param("article_id", article_id.clone());
                queries.push(category_query);
            }
            
            let feed_query = query(
                r#"
                MERGE (feed:Feed {id: $feed_id})
                SET feed.name = $feed_name
                MERGE (a:Article {id: $article_id})
                MERGE (a)-[:FROM_FEED]->(feed)
                "#)
                .param("feed_id", article.feed_id.as_str())
                .param("feed_name", article.feed_name.as_str())
                .param("article_id", article_id.clone());
            queries.push(feed_query);
            let mut loc_names: HashSet<String> = article.geo_tags.iter().cloned().collect();
            if let Some(f) = feed_id_to_location(&article.feed_id) {
                loc_names.insert(f);
            }
            for name in loc_names {
                let coords = resolve_location(&name, self.geojson_index.as_ref());
                let (lat, lon) = coords.unwrap_or((0.0, 0.0));
                let loc_q = if coords.is_some() {
                    query(
                        r#"
                        MERGE (loc:Location {name: $location_name})
                        SET loc.latitude = $latitude, loc.longitude = $longitude
                        MERGE (a:Article {id: $article_id})
                        MERGE (a)-[:MENTIONS_LOCATION]->(loc)
                        "#,
                    )
                    .param("location_name", name.as_str())
                    .param("latitude", lat)
                    .param("longitude", lon)
                    .param("article_id", article_id.clone())
                } else {
                    query(
                        r#"
                        MERGE (loc:Location {name: $location_name})
                        MERGE (a:Article {id: $article_id})
                        MERGE (a)-[:MENTIONS_LOCATION]->(loc)
                        "#,
                    )
                    .param("location_name", name.as_str())
                    .param("article_id", article_id.clone())
                };
                queries.push(loc_q);
            }
        }
        txn.run_queries(queries).await
            .context("Failed to execute batch queries")?;
        
        txn.commit().await
            .context("Failed to commit Neo4j transaction")?;
        
        Ok(())
    }
    
    pub fn export_to_json_file(&self, documents: &[DocumentWithMemories], path: &std::path::Path) -> AppResult<()> {
        use std::fs;
        use crate::error::wrap_io_error;
        
        let json_array: Vec<JsonValue> = documents.iter()
            .filter_map(|doc| doc.to_json().ok())
            .collect();
        
        let json_string = serde_json::to_string_pretty(&json_array)
            .context("Failed to serialize documents to JSON")?;
        
        if let Some(parent) = path.parent() {
            fs::create_dir_all(parent)
                .map_err(wrap_io_error)
                .context(format!("Failed to create output directory: {:?}", parent))?;
        }
        
        fs::write(path, json_string)
            .map_err(wrap_io_error)
            .context(format!("Failed to write JSON file: {:?}", path))?;
        
        Ok(())
    }

    /// Insert an extracted claim and its locations/entities into the graph.
    pub async fn insert_claim(&self, claim: &ExtractedClaim) -> AppResult<()> {
        let mut txn = self.graph.start_txn().await
            .context("Failed to start Neo4j transaction")?;
        let mut queries = Vec::new();

        let ts = claim.temporal_bounds.0.to_rfc3339();
        let te = claim.temporal_bounds.1.to_rfc3339();
        let ex = claim.extraction_timestamp.to_rfc3339();

        let q = query(
            r#"
            CREATE (c:Claim {
                id: $id,
                text: $text,
                article_id: $article_id,
                confidence: $confidence,
                temporal_start: datetime($ts),
                temporal_end: datetime($te),
                extracted_at: datetime($ex)
            })
            "#
        )
        .param("id", claim.id.as_str())
        .param("text", claim.claim_text.as_str())
        .param("article_id", claim.article_id.as_str())
        .param("confidence", claim.confidence)
        .param("ts", ts.as_str())
        .param("te", te.as_str())
        .param("ex", ex.as_str());
        queries.push(q);

        for loc in &claim.locations {
        let q = query(
            r#"
                MATCH (c:Claim {id: $claim_id})
                MERGE (place:Place {name: $name, lat: $lat, lng: $lng})
                MERGE (c)-[:LOCATED_AT]->(place)
                "#
        )
            .param("claim_id", claim.id.as_str())
            .param("name", loc.name.as_str())
            .param("lat", loc.latitude)
            .param("lng", loc.longitude);
            queries.push(q);
        }

        for ent in &claim.entities {
            let t = format!("{:?}", ent.entity_type);
        let q = query(
            r#"
                MATCH (c:Claim {id: $claim_id})
                MERGE (ent:Entity {name: $name, entity_type: $entity_type})
                MERGE (c)-[:MENTIONS]->(ent)
                "#
        )
            .param("claim_id", claim.id.as_str())
            .param("name", ent.name.as_str())
            .param("entity_type", t.as_str());
            queries.push(q);
        }

        txn.run_queries(queries).await
            .context("Failed to run insert_claim")?;
        txn.commit().await.context("Failed to commit insert_claim")?;
        Ok(())
    }

    /// Find claims overlapping in space and time. Returns minimal ExtractedClaim for reasoning.
    pub async fn find_overlapping_claims(
        &self,
        locations: &[GeoLocation],
        temporal: &(DateTime<Utc>, DateTime<Utc>),
    ) -> AppResult<Vec<ExtractedClaim>> {
        let time_start = temporal.0.to_rfc3339();
        let time_end = temporal.1.to_rfc3339();

        let mut q = query(
            r#"
            MATCH (c:Claim)
            WHERE c.temporal_start <= datetime($time_end)
              AND c.temporal_end >= datetime($time_start)
              AND c.confidence > 0.6
            RETURN c.id AS id, c.text AS claim_text, c.article_id AS article_id,
                   c.confidence AS confidence,
                   c.temporal_start AS ts, c.temporal_end AS te,
                   c.extracted_at AS ex
            LIMIT 20
            "#,
        )
        .param("time_start", time_start.as_str())
        .param("time_end", time_end.as_str());

        if !locations.is_empty() {
            let lat_min: f64 = locations.iter().map(|l| l.latitude).fold(f64::MAX, f64::min);
            let lat_max: f64 = locations.iter().map(|l| l.latitude).fold(f64::MIN, f64::max);
            let lng_min: f64 = locations.iter().map(|l| l.longitude).fold(f64::MAX, f64::min);
            let lng_max: f64 = locations.iter().map(|l| l.longitude).fold(f64::MIN, f64::max);
            q = query(
                r#"
                MATCH (c:Claim)-[:LOCATED_AT]->(place:Place)
                WHERE place.lat >= $lat_min AND place.lat <= $lat_max
                  AND place.lng >= $lng_min AND place.lng <= $lng_max
                  AND c.temporal_start <= datetime($time_end)
                  AND c.temporal_end >= datetime($time_start)
                  AND c.confidence > 0.6
                RETURN c.id AS id, c.text AS claim_text, c.article_id AS article_id,
                       c.confidence AS confidence,
                       c.temporal_start AS ts, c.temporal_end AS te,
                       c.extracted_at AS ex
                LIMIT 20
                "#,
            )
            .param("lat_min", lat_min)
            .param("lat_max", lat_max)
            .param("lng_min", lng_min)
            .param("lng_max", lng_max)
            .param("time_start", time_start.as_str())
            .param("time_end", time_end.as_str());
        }

        let mut result = self.graph.execute(q).await.context("find_overlapping_claims execute")?;
        let mut out = Vec::new();
        while let Ok(Some(row)) = result.next().await {
            let id: String = row.get::<String>("id").context("id")?;
            let claim_text: String = row.get::<String>("claim_text").context("claim_text")?;
            let article_id: String = row.get::<String>("article_id").context("article_id")?;
            let confidence: f32 = row.get::<f64>("confidence").map(|f| f as f32).unwrap_or(0.0);
            let ts: String = row.get::<String>("ts").unwrap_or_else(|_| Utc::now().to_rfc3339());
            let te: String = row.get::<String>("te").unwrap_or_else(|_| Utc::now().to_rfc3339());
            let ex: String = row.get::<String>("ex").unwrap_or_else(|_| Utc::now().to_rfc3339());
            let t_start = DateTime::parse_from_rfc3339(&ts)
                .map(|d| d.with_timezone(&Utc))
                .unwrap_or_else(|_| temporal.0);
            let t_end = DateTime::parse_from_rfc3339(&te)
                .map(|d| d.with_timezone(&Utc))
                .unwrap_or_else(|_| temporal.1);
            let ext_at = DateTime::parse_from_rfc3339(&ex)
                .map(|d| d.with_timezone(&Utc))
                .unwrap_or_else(|_| Utc::now());
            out.push(ExtractedClaim {
                id,
                claim_text,
                article_id,
                entities: vec![],
                locations: vec![],
                temporal_bounds: (t_start, t_end),
                confidence,
                extraction_timestamp: ext_at,
            });
        }
        Ok(out)
    }

    /// Update a claim node with verification verdict.
    pub async fn update_claim_verdict(&self, result: &VerificationResult) -> AppResult<()> {
        let verdict = format!("{:?}", result.verdict);
        let verified = result.verified_at.to_rfc3339();
        let q = query(
            r#"
            MATCH (c:Claim {id: $claim_id})
            SET c.verdict = $verdict,
                c.final_confidence = $confidence,
                c.verified_at = datetime($verified),
                c.conflict_count = $conflict_count
            "#,
        )
        .param("claim_id", result.claim_id.as_str())
        .param("verdict", verdict.as_str())
        .param("confidence", result.confidence)
        .param("verified", verified.as_str())
        .param("conflict_count", result.conflicting_claims.len() as i64);
        let mut stream = self.graph.execute(q).await.context("update_claim_verdict")?;
        while let Ok(Some(_)) = stream.next().await {}
        Ok(())
    }

    /// Store document quality on Article. Only scalar fields; word_tokens are not persisted.
    pub async fn add_document_quality(&self, article_id: &str, quality: &DocumentQuality) -> AppResult<()> {
        let mut txn = self.graph.start_txn().await
            .context("Failed to start Neo4j transaction for add_document_quality")?;
        let q = query(
            r#"
            MERGE (a:Article {id: $article_id})
            MERGE (q:DocumentQuality {article_id: $article_id})
            SET q.word_count = $word_count, q.sentence_count = $sentence_count,
                q.reading_time_min = $reading_time_min, q.language = $language,
                q.readability_score = $readability_score, q.created_at = datetime()
            MERGE (a)-[:HAS_QUALITY]->(q)
            "#,
        )
        .param("article_id", article_id)
        .param("word_count", quality.word_count as i64)
        .param("sentence_count", quality.sentence_count as i64)
        .param("reading_time_min", quality.reading_time_min)
        .param("language", quality.language.as_str())
        .param("readability_score", quality.readability_score);
        txn.run_queries([q]).await
            .context("Failed to run add_document_quality")?;
        txn.commit().await
            .context("Failed to commit add_document_quality")?;
        Ok(())
    }

    /// Create SIMILAR_TO edges. Caller ensures id1 < id2 per (id1, id2, score). Empty edges = no-op.
    pub async fn add_similarity_batch(&self, edges: &[(String, String, f64)]) -> AppResult<()> {
        if edges.is_empty() {
            return Ok(());
        }
        let mut txn = self.graph.start_txn().await
            .context("Failed to start Neo4j transaction for add_similarity_batch")?;
        let mut queries = Vec::new();
        for (id1, id2, score) in edges {
            let q = query(
                r#"
                MERGE (a1:Article {id: $id1})
                MERGE (a2:Article {id: $id2})
                MERGE (a1)-[r:SIMILAR_TO]->(a2)
                SET r.score = $score
                "#,
            )
            .param("id1", id1.as_str())
            .param("id2", id2.as_str())
            .param("score", *score);
            queries.push(q);
        }
        txn.run_queries(queries).await
            .context("Failed to run add_similarity_batch")?;
        txn.commit().await
            .context("Failed to commit add_similarity_batch")?;
        Ok(())
    }
}

impl Neo4jClient {
    pub async fn add_event(&self, event: &crate::bitnet_agent::Event) -> AppResult<()> {
        let mut txn = self.graph.start_txn().await
            .context("Failed to start Neo4j transaction")?;
        
        let mut queries = Vec::new();
        
        let create_event_query = query(
            r#"
            MERGE (e:Event {id: $event_id})
            SET e.title = $title,
                e.description = $description,
                e.event_type = $event_type,
                e.confidence = $confidence,
                e.org_id = $org_id,
                e.user_id = $user_id,
                e.created_at = datetime()
            "#)
            .param("event_id", event.id.clone())
            .param("title", event.title.as_str())
            .param("description", event.description.as_str())
            .param("event_type", event.event_type.as_str())
            .param("confidence", event.confidence)
            .param("org_id", self.org_id.as_str())
            .param("user_id", self.user_id.as_str());
        
        queries.push(create_event_query);
        
        if let Some(timestamp) = event.timestamp {
            let timestamp_query = query(
                r#"
                MERGE (e:Event {id: $event_id})
                SET e.timestamp = $timestamp
                "#)
                .param("event_id", event.id.clone())
                .param("timestamp", timestamp.to_rfc3339());
            queries.push(timestamp_query);
        }
        
        if let Some(ref location) = event.location {
            let (lat, lon) = resolve_location(&location.name, self.geojson_index.as_ref())
                .unwrap_or((location.latitude, location.longitude));
            let location_query = query(
                r#"
                MERGE (loc:Location {name: $location_name})
                SET loc.latitude = $latitude,
                    loc.longitude = $longitude
                MERGE (e:Event {id: $event_id})
                MERGE (e)-[:OCCURRED_AT]->(loc)
                "#)
                .param("location_name", location.name.as_str())
                .param("latitude", lat)
                .param("longitude", lon)
                .param("event_id", event.id.clone());
            queries.push(location_query);
        }
        
        let article_query = query(
            r#"
            MERGE (a:Article {id: $article_id})
            MERGE (e:Event {id: $event_id})
            MERGE (a)-[:DESCRIBES]->(e)
            "#)
            .param("article_id", event.source_article_id.clone())
            .param("event_id", event.id.clone());
        queries.push(article_query);
        
        for entity in &event.entities {
            let entity_query = query(
                r#"
                MERGE (ent:Entity {name: $entity_name, type: $entity_type})
                MERGE (e:Event {id: $event_id})
                MERGE (e)-[:INVOLVES]->(ent)
                "#)
                .param("entity_name", entity.name.as_str())
                .param("entity_type", format!("{:?}", entity.entity_type))
                .param("event_id", event.id.clone());
            queries.push(entity_query);
        }
        
        txn.run_queries(queries).await
            .map_err(|e| Neo4jError::QueryFailed(e.to_string()))?;
        
        txn.commit().await
            .map_err(|e| Neo4jError::TransactionError(e.to_string()))?;
        
        Ok(())
    }
    
    pub async fn add_entity(&self, article_id: &str, entity: &crate::bitnet_agent::Entity) -> AppResult<()> {
        let mut txn = self.graph.start_txn().await
            .context("Failed to start Neo4j transaction")?;
        let entity_type_str = match &entity.entity_type {
            crate::bitnet_agent::EntityType::Person => "Person",
            crate::bitnet_agent::EntityType::Organization => "Organization",
            crate::bitnet_agent::EntityType::Location => "Location",
            crate::bitnet_agent::EntityType::Event => "Event",
            crate::bitnet_agent::EntityType::Date => "Date",
            crate::bitnet_agent::EntityType::Other(s) => s.as_str(),
        };
        let entity_query = query(
            r#"
            MERGE (ent:Entity {name: $entity_name, type: $entity_type})
            SET ent.org_id = $org_id,
                ent.user_id = $user_id,
                ent.created_at = datetime()
            MERGE (a:Article {id: $article_id})
            MERGE (a)-[:MENTIONS_ENTITY]->(ent)
            "#)
            .param("entity_name", entity.name.as_str())
            .param("entity_type", entity_type_str)
            .param("article_id", article_id)
            .param("org_id", self.org_id.as_str())
            .param("user_id", self.user_id.as_str());
        let mut qs: Vec<_> = vec![entity_query];
        if matches!(entity.entity_type, crate::bitnet_agent::EntityType::Location) {
            if let Some((lat, lon)) = resolve_location(&entity.name, self.geojson_index.as_ref()) {
                let loc_q = query(
                    r#"
                    MERGE (loc:Location {name: $location_name})
                    SET loc.latitude = $latitude, loc.longitude = $longitude
                    MERGE (ent:Entity {name: $entity_name, type: $entity_type})
                    MERGE (ent)-[:LOCATED_IN]->(loc)
                    "#,
                )
                .param("location_name", entity.name.as_str())
                .param("latitude", lat)
                .param("longitude", lon)
                .param("entity_name", entity.name.as_str())
                .param("entity_type", entity_type_str);
                qs.push(loc_q);
            }
        }
        txn.run_queries(qs).await
            .context("Failed to execute entity query")?;
        txn.commit().await
            .context("Failed to commit entity transaction")?;
        Ok(())
    }
    
    /// Events with location data for API / map endpoints. Joins (Article)-[:DESCRIBES]->(Event)
    /// to include article_id and article_title for MapTimeline entry_id/entry_title.
    pub async fn get_events_with_locations(&self) -> AppResult<Vec<serde_json::Value>> {
        let mut result = self.graph.execute(query(
            r#"
            MATCH (e:Event)-[:OCCURRED_AT]->(loc:Location)
            OPTIONAL MATCH (a:Article)-[:DESCRIBES]->(e)
            RETURN e.id as event_id,
                   e.title as title,
                   e.event_type as event_type,
                   e.timestamp as timestamp,
                   loc.name as location_name,
                   loc.latitude as latitude,
                   loc.longitude as longitude,
                   a.id as article_id,
                   a.title as article_title
            ORDER BY e.timestamp DESC
            LIMIT 1000
            "#)).await
            .map_err(|e| Neo4jError::QueryFailed(e.to_string()))?;
        
        let mut events = Vec::new();
        while let Ok(Some(row)) = result.next().await {
            let mut event_json = serde_json::Map::new();
            
            if let Ok(event_id) = row.get::<String>("event_id") {
                event_json.insert("event_id".to_string(), serde_json::Value::String(event_id));
            }
            if let Ok(title) = row.get::<String>("title") {
                event_json.insert("title".to_string(), serde_json::Value::String(title));
            }
            if let Ok(event_type) = row.get::<String>("event_type") {
                event_json.insert("event_type".to_string(), serde_json::Value::String(event_type));
            }
            if let Ok(location_name) = row.get::<String>("location_name") {
                event_json.insert("location_name".to_string(), serde_json::Value::String(location_name));
            }
            if let Ok(lat) = row.get::<f64>("latitude") {
                event_json.insert("latitude".to_string(), serde_json::Value::Number(serde_json::Number::from_f64(lat).unwrap()));
            }
            if let Ok(lon) = row.get::<f64>("longitude") {
                event_json.insert("longitude".to_string(), serde_json::Value::Number(serde_json::Number::from_f64(lon).unwrap()));
            }
            if let Ok(timestamp) = row.get::<String>("timestamp") {
                event_json.insert("timestamp".to_string(), serde_json::Value::String(timestamp));
            }
            if let Ok(article_id) = row.get::<String>("article_id") {
                event_json.insert("article_id".to_string(), serde_json::Value::String(article_id));
            }
            if let Ok(article_title) = row.get::<String>("article_title") {
                event_json.insert("article_title".to_string(), serde_json::Value::String(article_title));
            }
            
            events.push(serde_json::Value::Object(event_json));
        }
        
        Ok(events)
    }
}

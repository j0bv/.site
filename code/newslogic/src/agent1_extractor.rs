//! Agent 1: Truth Extractor. Consumes EnrichedArticle from Redis, extracts claims via BitNet,
//! writes to Neo4j, and pushes ExtractedClaim to claims:unverified.

use crate::bitnet::BitNetInference;
use crate::error::RedisError;
use crate::memory_graph::Neo4jClient;
use crate::models::{Entity, EnrichedArticle, ExtractedClaim, GeoLocation};
use anyhow::Result;
use chrono::{Duration, Utc};
use redis::aio::ConnectionManager;
use std::sync::Arc;
use tokio::time::{sleep, Duration as TokioDuration};
use uuid::Uuid;

pub struct TruthExtractor {
    redis: ConnectionManager,
    memory_graph: Arc<Neo4jClient>,
    bitnet: Arc<BitNetInference>,
}

impl TruthExtractor {
    pub fn new(
        redis: ConnectionManager,
        memory_graph: Arc<Neo4jClient>,
        bitnet: Arc<BitNetInference>,
    ) -> Self {
        Self {
            redis,
            memory_graph,
            bitnet,
        }
    }

    pub async fn run(&self) -> Result<()> {
        tracing::info!("Agent 1: Truth Extractor started");
        let mut conn = self.redis.clone();
        loop {
            let v: Result<(String, String), _> = redis::cmd("BLPOP")
                .arg("articles:extracted")
                .arg(0)
                .query_async(&mut conn)
                .await;

            if let Ok((_, json_str)) = v {
                match serde_json::from_str::<EnrichedArticle>(&json_str) {
                    Ok(article) => {
                        if let Err(e) = self.extract_from_article(&article, &mut conn).await {
                            tracing::error!("Extraction failed for {}: {}", article.title, e);
                        }
                    }
                    Err(e) => tracing::error!("Failed to parse EnrichedArticle JSON: {}", e),
                }
            }

            sleep(TokioDuration::from_millis(100)).await;
        }
    }

    async fn extract_from_article(
        &self,
        article: &EnrichedArticle,
        conn: &mut ConnectionManager,
    ) -> Result<()> {
        tracing::debug!("Extracting from: {}", article.title);

        let entities = self.bitnet.extract_entities(&article.text).await?;
        let claims = self.bitnet.extract_claims(&article.text).await?;
        let locations = self.geocode_from_entities(&entities);
        let temporal_bounds = self.extract_temporal_bounds(article);

        for claim_text in claims {
            let extracted = ExtractedClaim {
                id: Uuid::new_v4().to_string(),
                claim_text,
                article_id: article.id.clone(),
                entities: entities.clone(),
                locations: locations.clone(),
                temporal_bounds,
                confidence: 0.82,
                extraction_timestamp: Utc::now(),
            };

            self.memory_graph.insert_claim(&extracted).await?;
            let claim_json = serde_json::to_string(&extracted)?;
            redis::cmd("RPUSH")
                .arg("claims:unverified")
                .arg(claim_json)
                .query_async::<()>(conn)
                .await
                .map_err(|e| RedisError::CommandFailed(e.to_string()))?;

            tracing::info!("Extracted claim: {}", extracted.id);
        }
        Ok(())
    }

    fn geocode_from_entities(&self, _entities: &[Entity]) -> Vec<GeoLocation> {
        // Placeholder: real impl would use geoutils or Nominatim
        Vec::new()
    }

    fn extract_temporal_bounds(&self, article: &EnrichedArticle) -> (chrono::DateTime<Utc>, chrono::DateTime<Utc>) {
        let start = article.published;
        let end = start + Duration::days(30);
        (start, end)
    }
}

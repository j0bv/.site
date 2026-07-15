//! Agent 2: Spatio-Temporal Verifier. Consumes ExtractedClaim from Redis, finds overlapping
//! claims in Neo4j, reasons via BitNet, and updates verdicts in the graph.

use crate::bitnet::{BitNetInference, VerdictReasoning};
use crate::memory_graph::Neo4jClient;
use crate::models::{ExtractedClaim, VerificationResult};
use anyhow::Result;
use chrono::Utc;
use redis::aio::ConnectionManager;
use std::sync::Arc;
use tokio::time::{sleep, Duration};

pub struct Verifier {
    redis: ConnectionManager,
    memory_graph: Arc<Neo4jClient>,
    bitnet: Arc<BitNetInference>,
}

impl Verifier {
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
        tracing::info!("Agent 2: Spatio-Temporal Verifier started");
        let mut conn = self.redis.clone();
        loop {
            let v: Result<(String, String), _> = redis::cmd("BLPOP")
                .arg("claims:unverified")
                .arg(0)
                .query_async(&mut conn)
                .await;

            if let Ok((_, json_str)) = v {
                match serde_json::from_str::<ExtractedClaim>(&json_str) {
                    Ok(claim) => {
                        if let Err(e) = self.verify_claim(&claim, &mut conn).await {
                            tracing::error!("Verification failed for {}: {}", claim.id, e);
                        }
                    }
                    Err(e) => tracing::error!("Failed to parse ExtractedClaim JSON: {}", e),
                }
            }

            sleep(Duration::from_millis(100)).await;
        }
    }

    async fn verify_claim(
        &self,
        claim: &ExtractedClaim,
        _conn: &mut ConnectionManager,
    ) -> Result<()> {
        tracing::debug!("Verifying claim: {}", claim.id);

        let overlapping = self
            .memory_graph
            .find_overlapping_claims(&claim.locations, &claim.temporal_bounds)
            .await?;

        let reasoning: VerdictReasoning = self
            .bitnet
            .reason_about_conflicts(&claim.claim_text, &overlapping)
            .await?;

        let result = VerificationResult {
            claim_id: claim.id.clone(),
            verdict: reasoning.verdict,
            confidence: reasoning.confidence,
            conflicting_claims: reasoning.conflicting_ids,
            supporting_claims: reasoning.supporting_ids,
            evidence_count: overlapping.len() as i32,
            verified_at: Utc::now(),
        };

        self.memory_graph.update_claim_verdict(&result).await?;
        tracing::info!(
            "Verified {}: {:?} (confidence: {})",
            result.claim_id,
            result.verdict,
            result.confidence
        );
        Ok(())
    }
}

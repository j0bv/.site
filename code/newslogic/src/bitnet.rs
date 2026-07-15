//! BitNet inference for claim extraction and conflict reasoning.
//! Integrates with BitNet/llama-server via HTTP POST /completion when BITNET_URL is set.

use crate::error::BitNetError;
use crate::models::{Entity, EntityType, ExtractedClaim, Verdict};
use anyhow::{Context, Result};
use tracing::{trace, warn};

/// Result of reasoning about claim conflicts.
#[derive(Debug)]
pub struct VerdictReasoning {
    pub verdict: Verdict,
    pub confidence: f32,
    pub conflicting_ids: Vec<String>,
    pub supporting_ids: Vec<String>,
}

/// BitNet inference facade. When BITNET_URL is unset, uses placeholder logic.
pub struct BitNetInference {
    client: reqwest::Client,
    base_url: Option<String>,
}

impl BitNetInference {
    pub fn new() -> Self {
        let base_url = std::env::var("BITNET_URL").ok();
        let client = reqwest::Client::builder()
            .timeout(std::time::Duration::from_secs(120))
            .build()
            .unwrap_or_else(|_| reqwest::Client::new());
        Self { client, base_url }
    }

    /// POST /completion with {prompt, n_predict}; returns the "content" string. Err if base_url is None or request fails.
    async fn generate(&self, prompt: &str, n_predict: u32) -> Result<String> {
        let base = self
            .base_url
            .as_ref()
            .ok_or_else(|| anyhow::anyhow!("BITNET_URL not set"))?;
        let url = format!("{}/completion", base.trim_end_matches('/'));
        let body = serde_json::json!({ "prompt": prompt, "n_predict": n_predict });
        let res = self
            .client
            .post(&url)
            .json(&body)
            .send()
            .await
            .map_err(|e| BitNetError::InferenceFailed(e.to_string()))?;
        let status = res.status();
        let text = res.text().await.context("BitNet /completion response body")?;
        if !status.is_success() {
            anyhow::bail!("BitNet /completion {}: {}", status, text);
        }
        let json: serde_json::Value =
            serde_json::from_str(&text).context("BitNet /completion: invalid JSON")?;
        let content = json
            .get("content")
            .and_then(|c| c.as_str())
            .unwrap_or("")
            .to_string();
        Ok(content)
    }

    /// Try to parse JSON from model output that may be wrapped in markdown or extra text.
    fn extract_json_array(content: &str) -> Option<serde_json::Value> {
        if let Ok(v) = serde_json::from_str::<serde_json::Value>(content) {
            if v.is_array() {
                return Some(v);
            }
        }
        // Heuristic: find first [ ... ] or { ... }
        let trimmed = content.trim();
        if let Some(start) = trimmed.find('[') {
            let rest = &trimmed[start..];
            let mut depth = 0u32;
            for (i, c) in rest.chars().enumerate() {
                match c {
                    '[' => depth += 1,
                    ']' => {
                        depth -= 1;
                        if depth == 0 {
                            let slice = &rest[..=i];
                            if let Ok(v) = serde_json::from_str(slice) {
                                return Some(v);
                            }
                            break;
                        }
                    }
                    _ => {}
                }
            }
        }
        None
    }

    fn extract_json_object(content: &str) -> Option<serde_json::Value> {
        if let Ok(v) = serde_json::from_str::<serde_json::Value>(content) {
            if v.is_object() {
                return Some(v);
            }
        }
        let trimmed = content.trim();
        if let Some(start) = trimmed.find('{') {
            let rest = &trimmed[start..];
            let mut depth = 0u32;
            for (i, c) in rest.chars().enumerate() {
                match c {
                    '{' => depth += 1,
                    '}' => {
                        depth -= 1;
                        if depth == 0 {
                            let slice = &rest[..=i];
                            if let Ok(v) = serde_json::from_str(slice) {
                                return Some(v);
                            }
                            break;
                        }
                    }
                    _ => {}
                }
            }
        }
        None
    }

    fn map_entity_type(s: &str) -> EntityType {
        match s.to_lowercase().as_str() {
            "person" => EntityType::Person,
            "organization" => EntityType::Organization,
            "geopolitical" => EntityType::Geopolitical,
            "location" => EntityType::Location,
            "event" => EntityType::Event,
            _ => EntityType::Other(s.to_string()),
        }
    }

    fn map_verdict(s: &str) -> Verdict {
        match s.to_lowercase().as_str() {
            "true" => Verdict::True,
            "false" => Verdict::False,
            "ambiguous" => Verdict::Ambiguous,
            "unverified" => Verdict::Unverified,
            _ => Verdict::Unverified,
        }
    }

    /// Named entity recognition. If BITNET_URL unset or request fails, returns empty vec.
    pub async fn extract_entities(&self, text: &str) -> Result<Vec<Entity>> {
        if self.base_url.is_none() {
            trace!("BitNet: BITNET_URL unset, extract_entities placeholder");
            return Ok(Vec::new());
        }
        let prompt = format!(
            r#"Extract named entities from the following text. Return ONLY a JSON array of objects, each with "name", "type" (one of: Person, Organization, Geopolitical, Location, Event, Other), and "mention_count" (integer). No other text.

Text:
{}
"#,
            text.chars().take(4000).collect::<String>()
        );
        let content = match self.generate(&prompt, 256).await {
            Ok(c) => c,
            Err(e) => {
                warn!("BitNet extract_entities: {}; falling back to empty", e);
                return Ok(Vec::new());
            }
        };
        let arr = match Self::extract_json_array(&content) {
            Some(serde_json::Value::Array(a)) => a,
            _ => {
                warn!("BitNet extract_entities: could not parse JSON array; falling back to empty");
                return Ok(Vec::new());
            }
        };
        let mut out = Vec::new();
        for v in arr {
            if let Some(obj) = v.as_object() {
                let name = obj
                    .get("name")
                    .and_then(|n| n.as_str())
                    .unwrap_or("")
                    .to_string();
                if name.is_empty() {
                    continue;
                }
                let typ = obj
                    .get("type")
                    .and_then(|t| t.as_str())
                    .unwrap_or("Other");
                let mentions = obj
                    .get("mention_count")
                    .and_then(|m| m.as_i64())
                    .unwrap_or(1)
                    .max(0) as i32;
                out.push(Entity {
                    name,
                    entity_type: Self::map_entity_type(typ),
                    mentions: if mentions > 0 { mentions } else { 1 },
                });
            }
        }
        Ok(out)
    }

    /// Claim extraction from text. If BITNET_URL unset or parse fails, uses heuristic (first sentence / 200 chars).
    pub async fn extract_claims(&self, text: &str) -> Result<Vec<String>> {
        let heuristic = || {
            let t = text.trim();
            if t.is_empty() {
                return Vec::new();
            }
            let end = t
                .char_indices()
                .find(|(_, c)| *c == '.' || *c == '!' || *c == '?')
                .map(|(i, _)| i + 1)
                .unwrap_or(t.len().min(200));
            let claim = t.chars().take(end).collect::<String>().trim().to_string();
            if claim.is_empty() {
                Vec::new()
            } else {
                vec![claim]
            }
        };

        if self.base_url.is_none() {
            return Ok(heuristic());
        }
        let prompt = format!(
            r#"Extract factual claims from the following text. Return ONLY a JSON array of strings, each string one claim. No other text.

Text:
{}
"#,
            text.chars().take(6000).collect::<String>()
        );
        let content = match self.generate(&prompt, 384).await {
            Ok(c) => c,
            Err(e) => {
                warn!("BitNet extract_claims: {}; using heuristic", e);
                return Ok(heuristic());
            }
        };
        let arr = match Self::extract_json_array(&content) {
            Some(serde_json::Value::Array(a)) => a,
            _ => {
                warn!("BitNet extract_claims: could not parse JSON array; using heuristic");
                return Ok(heuristic());
            }
        };
        let mut claims = Vec::new();
        for v in arr {
            if let Some(s) = v.as_str() {
                let c = s.trim().to_string();
                if !c.is_empty() {
                    claims.push(c);
                }
            }
        }
        if claims.is_empty() {
            Ok(heuristic())
        } else {
            Ok(claims)
        }
    }

    /// Reason about conflicts between a claim and overlapping claims. If BITNET_URL unset or parse fails, returns Verdict::True, 0.85.
    pub async fn reason_about_conflicts(
        &self,
        claim: &str,
        overlapping: &[ExtractedClaim],
    ) -> Result<VerdictReasoning> {
        let fallback = VerdictReasoning {
            verdict: Verdict::True,
            confidence: 0.85,
            conflicting_ids: Vec::new(),
            supporting_ids: overlapping.iter().map(|c| c.id.clone()).collect(),
        };

        if self.base_url.is_none() {
            return Ok(fallback);
        }

        let overlapping_list: String = overlapping
            .iter()
            .take(15)
            .map(|c| format!("- id: {} | text: {}", c.id, c.claim_text.chars().take(200).collect::<String>()))
            .collect::<Vec<_>>()
            .join("\n");

        let prompt = format!(
            r#"Given the main claim and a list of overlapping claims, reason about conflicts. Return ONLY a JSON object with: "verdict" (one of: True, False, Ambiguous, Unverified), "confidence" (0-1), "conflicting_ids" (array of claim ids that contradict), "supporting_ids" (array of claim ids that support). No other text.

Main claim: {}

Overlapping claims:
{}

JSON:
"#,
            claim.chars().take(500).collect::<String>(),
            if overlapping_list.is_empty() {
                "(none)".to_string()
            } else {
                overlapping_list
            }
        );

        let content = match self.generate(&prompt, 256).await {
            Ok(c) => c,
            Err(e) => {
                warn!("BitNet reason_about_conflicts: {}; using fallback", e);
                return Ok(fallback);
            }
        };

        let obj = match Self::extract_json_object(&content) {
            Some(serde_json::Value::Object(o)) => o,
            _ => {
                warn!("BitNet reason_about_conflicts: could not parse JSON object; using fallback");
                return Ok(fallback);
            }
        };

        let verdict = obj
            .get("verdict")
            .and_then(|v| v.as_str())
            .map(Self::map_verdict)
            .unwrap_or(Verdict::Unverified);
        let confidence = obj
            .get("confidence")
            .and_then(|c| c.as_f64())
            .map(|f| f as f32)
            .unwrap_or(0.5)
            .clamp(0.0, 1.0);
        let conflicting_ids: Vec<String> = obj
            .get("conflicting_ids")
            .and_then(|a| a.as_array())
            .map(|a| {
                a.iter()
                    .filter_map(|v| v.as_str().map(String::from))
                    .collect()
            })
            .unwrap_or_default();
        let supporting_ids: Vec<String> = obj
            .get("supporting_ids")
            .and_then(|a| a.as_array())
            .map(|a| {
                a.iter()
                    .filter_map(|v| v.as_str().map(String::from))
                    .collect()
            })
            .unwrap_or_else(|| overlapping.iter().map(|c| c.id.clone()).collect());

        Ok(VerdictReasoning {
            verdict,
            confidence,
            conflicting_ids,
            supporting_ids,
        })
    }
}

impl Default for BitNetInference {
    fn default() -> Self {
        Self::new()
    }
}

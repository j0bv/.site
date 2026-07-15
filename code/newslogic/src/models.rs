use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ArticleContent {
    pub title: String,
    pub main_text: String,
    pub html_content: String,
    pub images: Vec<ImageInfo>,
    pub published_date: Option<DateTime<Utc>>,
    pub url: String,
    pub extraction_timestamp: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ImageInfo {
    pub src: String,
    pub alt: String,
    pub caption: Option<String>,
}

/// Enriched article from Lightpanda (RSS + browser extraction) for the truth-engine pipeline.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EnrichedArticle {
    pub id: String,
    pub title: String,
    pub url: String,
    pub source: String,
    pub published: DateTime<Utc>,
    pub text: String,
    pub html: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExtractedClaim {
    pub id: String,
    pub claim_text: String,
    pub article_id: String,
    pub entities: Vec<Entity>,
    pub locations: Vec<GeoLocation>,
    pub temporal_bounds: (DateTime<Utc>, DateTime<Utc>),
    pub confidence: f32,
    pub extraction_timestamp: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Entity {
    pub name: String,
    pub entity_type: EntityType,
    pub mentions: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum EntityType {
    Person,
    Organization,
    Geopolitical,
    Location,
    Event,
    Other(String),
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GeoLocation {
    pub name: String,
    pub latitude: f64,
    pub longitude: f64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub geojson: Option<serde_json::Value>,
    pub geocoder_confidence: f32,
}#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerificationResult {
    pub claim_id: String,
    pub verdict: Verdict,
    pub confidence: f32,
    pub conflicting_claims: Vec<String>,
    pub supporting_claims: Vec<String>,
    pub evidence_count: i32,
    pub verified_at: DateTime<Utc>,
}#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum Verdict {
    True,
    False,
    Ambiguous,
    Unverified,
}

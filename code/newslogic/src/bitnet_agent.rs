use crate::parser::Article;
use crate::geo_data::GeoJsonIndex;
use crate::error::{AppResult, BitNetError};
use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Event {
    pub id: String,
    pub title: String,
    pub description: String,
    pub event_type: String,
    pub timestamp: Option<DateTime<Utc>>,
    pub location: Option<EventLocation>,
    pub source_article_id: String,
    pub confidence: f64,
    pub entities: Vec<Entity>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EventLocation {
    pub name: String,
    pub latitude: f64,
    pub longitude: f64,
    pub address: Option<String>,
    pub geojson_properties: Option<serde_json::Value>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Entity {
    pub name: String,
    pub entity_type: EntityType,
    pub mentions: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum EntityType {
    Person,
    Organization,
    Location,
    Event,
    Date,
    Other(String),
}

pub struct BitNetAgent {
    geojson_index: GeoJsonIndex,
}

impl BitNetAgent {
    pub fn new(geojson_dir: &std::path::Path) -> AppResult<Self> {
        let geojson_index = crate::geo_data::load_geojson_directory(geojson_dir)
            .map_err(|e| BitNetError::ModelLoadFailed(e.to_string()))?;
        
        Ok(Self { geojson_index })
    }
    
    /// Legacy; worker uses extract_events(article, text_override) instead.
    #[allow(dead_code)]
    pub async fn extract_events_from_text(
        &self,
        article_id: &str,
        title: &str,
        text: &str,
        timestamp: Option<chrono::DateTime<chrono::Utc>>,
    ) -> AppResult<Vec<Event>> {
        let locations = self.extract_locations_from_text(text);
        
        let event_types = self.classify_event_types(text);
        
        let mut events = Vec::new();
        
        for (event_type, confidence) in event_types {
            let location = locations.first().cloned();
            
            let event = Event {
                id: format!("event_{}_{}", article_id, events.len()),
                title: self.extract_event_title(text, &event_type),
                description: self.extract_event_description(text, &event_type),
                event_type: event_type.clone(),
                timestamp,
                location,
                source_article_id: article_id.to_string(),
                confidence,
                entities: self.extract_entities_from_text_simple(text),
            };
            
            events.push(event);
        }
        
        if events.is_empty() {
            let default_event = Event {
                id: format!("event_{}_0", article_id),
                title: title.to_string(),
                description: text.chars().take(500).collect(),
                event_type: "news".to_string(),
                timestamp,
                location: locations.first().cloned(),
                source_article_id: article_id.to_string(),
                confidence: 0.5,
                entities: self.extract_entities_from_text_simple(text),
            };
            events.push(default_event);
        }
        
        Ok(events)
    }
    
    /// Legacy; worker uses entities from extract_events.
    #[allow(dead_code)]
    pub async fn extract_entities_from_text(&self, text: &str) -> AppResult<Vec<Entity>> {
        Ok(self.extract_entities_from_text_simple(text))
    }
    
    fn extract_locations_from_text(&self, text: &str) -> Vec<EventLocation> {
        let mut locations = Vec::new();
        
        let location_names = self.extract_location_names(text);
        for location_name in location_names {
            if let Some(coords) = self.geojson_index.find_location(&location_name) {
                if !locations.iter().any(|l: &EventLocation| l.name == coords.name.as_deref().unwrap_or(location_name.as_str())) {
                    locations.push(EventLocation {
                        name: coords.name.unwrap_or_else(|| location_name.clone()),
                        latitude: coords.latitude,
                        longitude: coords.longitude,
                        address: None,
                        geojson_properties: Some(serde_json::to_value(&coords.properties).unwrap_or_default()),
                    });
                }
            }
        }

        for venue_name in self.extract_venue_names_from_text(text) {
            if let Some(coords) = self.geojson_index.find_location(&venue_name) {
                if !locations.iter().any(|l| l.name == coords.name.as_deref().unwrap_or(venue_name.as_str())) {
                    locations.push(EventLocation {
                        name: coords.name.unwrap_or_else(|| venue_name.clone()),
                        latitude: coords.latitude,
                        longitude: coords.longitude,
                        address: None,
                        geojson_properties: Some(serde_json::to_value(&coords.properties).unwrap_or_default()),
                    });
                }
            }
        }

        locations
    }

    fn extract_entities_from_text_simple(&self, text: &str) -> Vec<Entity> {
        let mut entities = Vec::new();
        let text_lower = text.to_lowercase();
        
        let common_orgs = vec![
            "police", "fire department", "city council", "school board",
            "norfolk", "virginia beach", "chesapeake", "portsmouth",
            "hampton", "newport news", "suffolk", "williamsburg",
        ];
        
        for org in common_orgs {
            if text_lower.contains(org) {
                entities.push(Entity {
                    name: org.to_string(),
                    entity_type: if org.contains("police") || org.contains("fire") || org.contains("council") || org.contains("board") {
                        EntityType::Organization
                    } else {
                        EntityType::Location
                    },
                    mentions: vec![org.to_string()],
                });
            }
        }
        
        entities
    }

    /// Article-based pipeline: uses text_override (e.g. Lightpanda) when Some, else article.full_text().
    pub async fn extract_events(&self, article: &Article, text_override: Option<&str>) -> AppResult<Vec<Event>> {
        let text_owned = text_override.map(String::from).unwrap_or_else(|| article.full_text());
        let text = text_owned.as_str();
        let article_id = format!("article_{}", article.content_hash);
        
        let mut events = Vec::new();
        
        let locations = self.extract_locations(text, article);
        let event_types = self.classify_event_types(text);
        let entities = self.extract_entities(text);
        
        for (event_type, confidence) in event_types {
            let location = locations.first().cloned();
            
            let event = Event {
                id: format!("event_{}_{}", article_id, events.len()),
                title: self.extract_event_title(text, &event_type),
                description: self.extract_event_description(text, &event_type),
                event_type: event_type.clone(),
                timestamp: article.pub_date,
                location,
                source_article_id: article_id.clone(),
                confidence,
                entities: entities.clone(),
            };
            
            events.push(event);
        }
        
        if events.is_empty() {
            let default_event = Event {
                id: format!("event_{}_0", article_id),
                title: article.title.clone(),
                description: article.description.clone().unwrap_or_default(),
                event_type: "news".to_string(),
                timestamp: article.pub_date,
                location: locations.first().cloned(),
                source_article_id: article_id,
                confidence: 0.5,
                entities,
            };
            events.push(default_event);
        }
        
        Ok(events)
    }
    
    /// Used by extract_events; uses article.geo_tags and venue/location extraction from text.
    fn extract_locations(&self, text: &str, article: &Article) -> Vec<EventLocation> {
        let mut locations = Vec::new();
        
        for geo_tag in &article.geo_tags {
            if let Some(coords) = self.geojson_index.find_location(geo_tag) {
                locations.push(EventLocation {
                    name: coords.name.unwrap_or_else(|| geo_tag.clone()),
                    latitude: coords.latitude,
                    longitude: coords.longitude,
                    address: None,
                    geojson_properties: Some(serde_json::to_value(&coords.properties).unwrap_or_default()),
                });
            }
        }
        
        let location_names = self.extract_location_names(text);
        for location_name in location_names {
            if let Some(coords) = self.geojson_index.find_location(&location_name) {
                if !locations.iter().any(|l: &EventLocation| l.name == coords.name.as_deref().unwrap_or(location_name.as_str())) {
                    locations.push(EventLocation {
                        name: coords.name.unwrap_or_else(|| location_name.clone()),
                        latitude: coords.latitude,
                        longitude: coords.longitude,
                        address: None,
                        geojson_properties: Some(serde_json::to_value(&coords.properties).unwrap_or_default()),
                    });
                }
            }
        }

        for venue_name in self.extract_venue_names_from_text(text) {
            if let Some(coords) = self.geojson_index.find_location(&venue_name) {
                if !locations.iter().any(|l| l.name == coords.name.as_deref().unwrap_or(venue_name.as_str())) {
                    locations.push(EventLocation {
                        name: coords.name.unwrap_or_else(|| venue_name.clone()),
                        latitude: coords.latitude,
                        longitude: coords.longitude,
                        address: None,
                        geojson_properties: Some(serde_json::to_value(&coords.properties).unwrap_or_default()),
                    });
                }
            }
        }

        locations
    }

    fn extract_location_names(&self, text: &str) -> Vec<String> {
        let common_757_locations = vec![
            "Norfolk", "Virginia Beach", "Chesapeake", "Portsmouth", "Hampton",
            "Newport News", "Suffolk", "Williamsburg", "Yorktown", "Poquoson",
            "Newport News", "Hampton Roads", "757", "Tidewater", "Peninsula",
        ];
        
        let mut found = Vec::new();
        let text_lower = text.to_lowercase();
        
        for location in common_757_locations {
            if text_lower.contains(&location.to_lowercase()) {
                found.push(location.to_string());
            }
        }
        
        found
    }

    /// Venue/place names from GeoJSON that appear in the text (e.g. Hampton Coliseum, Scope Arena).
    pub fn extract_venue_names_from_text(&self, text: &str) -> Vec<String> {
        let names = self.geojson_index.all_location_names();
        let text_lower = text.to_lowercase();
        let mut out: Vec<String> = names
            .into_iter()
            .filter(|n| n.len() >= 3 && text_lower.contains(&n.to_lowercase()))
            .collect();
        out.sort_by(|a, b| b.len().cmp(&a.len())); // prefer longer/more specific first
        out
    }

    fn classify_event_types(&self, text: &str) -> Vec<(String, f64)> {
        let text_lower = text.to_lowercase();
        let mut types = Vec::new();
        
        let event_keywords = vec![
            ("accident", "accident", 0.8),
            ("crash", "accident", 0.9),
            ("collision", "accident", 0.85),
            ("fire", "emergency", 0.9),
            ("emergency", "emergency", 0.7),
            ("arrest", "crime", 0.85),
            ("charged", "crime", 0.8),
            ("robbery", "crime", 0.9),
            ("shooting", "crime", 0.95),
            ("meeting", "government", 0.7),
            ("council", "government", 0.8),
            ("election", "government", 0.9),
            ("vote", "government", 0.75),
            ("weather", "weather", 0.8),
            ("storm", "weather", 0.9),
            ("hurricane", "weather", 0.95),
            ("flood", "weather", 0.85),
            ("opening", "business", 0.7),
            ("closing", "business", 0.7),
            ("construction", "infrastructure", 0.8),
            ("road", "infrastructure", 0.6),
            ("bridge", "infrastructure", 0.8),
        ];
        
        for (keyword, event_type, confidence) in event_keywords {
            if text_lower.contains(keyword) {
                if let Some(existing) = types.iter_mut().find(|(t, _)| t == event_type) {
                    if existing.1 < confidence {
                        existing.1 = confidence;
                    }
                } else {
                    types.push((event_type.to_string(), confidence));
                }
            }
        }
        
        if types.is_empty() {
            types.push(("news".to_string(), 0.5));
        }
        
        types.sort_by(|a, b| b.1.partial_cmp(&a.1).unwrap_or(std::cmp::Ordering::Equal));
        types
    }
    
    fn extract_event_title(&self, text: &str, event_type: &str) -> String {
        let lines: Vec<&str> = text.lines().collect();
        if let Some(first_line) = lines.first() {
            if first_line.len() > 10 && first_line.len() < 200 {
                return first_line.trim().to_string();
            }
        }
        format!("{} event", event_type)
    }
    
    fn extract_event_description(&self, text: &str, _event_type: &str) -> String {
        let lines: Vec<&str> = text.lines().take(5).collect();
        lines.join(" ").chars().take(500).collect()
    }
    
    /// Used by extract_events for event-level entities.
    fn extract_entities(&self, text: &str) -> Vec<Entity> {
        let mut entities = Vec::new();
        
        let common_orgs = vec!["Police", "Fire Department", "City Council", "School Board"];
        for org in common_orgs {
            if text.contains(org) {
                entities.push(Entity {
                    name: org.to_string(),
                    entity_type: EntityType::Organization,
                    mentions: vec![org.to_string()],
                });
            }
        }
        
        entities
    }
}

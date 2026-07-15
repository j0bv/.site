use serde::{Deserialize, Serialize};
use serde_json::Value as JsonValue;
use std::collections::HashMap;
use anyhow::{Context, Result};

// ------ feed_id / Hampton Roads helpers for Location resolution ------

/// Map feed_id to a canonical location name for city/regional feeds. Media feeds -> None.
pub fn feed_id_to_location(feed_id: &str) -> Option<String> {
    match feed_id.to_lowercase().as_str() {
        "portsmouth-city" => Some("portsmouth".into()),
        "norfolk-city" => Some("norfolk".into()),
        "hampton-city" => Some("hampton".into()),
        "chesapeake-city" => Some("chesapeake".into()),
        "newport-news-city" => Some("newport-news".into()),
        "suffolk-city" => Some("suffolk".into()),
        "williamsburg-yorktown-daily" => Some("williamsburg".into()),
        "hrpdc" | "hrtpo" => Some("hampton-roads".into()),
        _ => None,
    }
}

/// Static (lat, lon) for Hampton Roads names. Normalize: lowercase, treat "-" as part of name.
pub fn hampton_roads_coords(name: &str) -> Option<(f64, f64)> {
    let n = name.to_lowercase();
    let n = n.trim();
    match n {
        "portsmouth" => Some((36.8354, -76.2983)),
        "norfolk" => Some((36.8508, -76.2852)),
        "hampton" => Some((37.0299, -76.3452)),
        "chesapeake" => Some((36.7682, -76.2875)),
        "virginia-beach" | "virginia beach" => Some((36.8529, -75.9780)),
        "newport-news" | "newport news" => Some((37.0834, -76.4696)),
        "suffolk" => Some((36.7282, -76.5836)),
        "williamsburg" => Some((37.2707, -76.7075)),
        "yorktown" => Some((37.2388, -76.5097)),
        "hampton-roads" | "hampton roads" => Some((36.85, -76.29)),
        "virginia" => Some((37.5, -79.0)),
        _ => None,
    }
}

/// Resolve a location name to (lat, lon). Tries GeoJsonIndex first, then hampton_roads_coords.
pub fn resolve_location(name: &str, geojson: Option<&GeoJsonIndex>) -> Option<(f64, f64)> {
    if let Some(g) = geojson {
        if let Some(c) = g.find_location(name) {
            return Some((c.latitude, c.longitude));
        }
    }
    hampton_roads_coords(name)
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LocationCoordinates {
    pub latitude: f64,
    pub longitude: f64,
    pub name: Option<String>,
    pub properties: HashMap<String, JsonValue>,
}

/// First non-empty of name, Name, NAME, LocName, Locality, City in properties; else fallback.
fn feature_name_from_properties(properties: Option<&JsonValue>, fallback: String) -> String {
    let p = match properties.and_then(|p| p.as_object()) {
        Some(o) => o,
        None => return fallback,
    };
    for key in ["name", "Name", "NAME", "LocName", "Locality", "City"] {
        if let Some(v) = p.get(key).and_then(|v| v.as_str()) {
            if !v.trim().is_empty() {
                return v.trim().to_string();
            }
        }
    }
    fallback
}

#[derive(Debug, Clone)]
struct GeoJsonFeature {
    name: String,
    geometry: JsonValue,
    properties: HashMap<String, JsonValue>,
}

#[derive(Debug, Clone)]
pub struct GeoJsonIndex {
    features: Vec<GeoJsonFeature>,
}

impl GeoJsonIndex {
    pub fn from_file(path: &std::path::Path) -> Result<Self> {
        let content = std::fs::read_to_string(path)
            .context(format!("Failed to read GeoJSON file: {:?}", path))?;
        
        Self::from_str(&content)
    }
    
    pub fn from_str(content: &str) -> Result<Self> {
        let json: JsonValue = serde_json::from_str(content)
            .context("Failed to parse GeoJSON")?;
        
        let mut features = Vec::new();
        
        if let Some(JsonValue::String(type_str)) = json.get("type") {
            if type_str == "FeatureCollection" {
                if let Some(JsonValue::Array(features_array)) = json.get("features") {
                    for (idx, feature_json) in features_array.iter().enumerate() {
                        if let Some(geometry) = feature_json.get("geometry") {
                            let props = feature_json.get("properties");
                            let name = feature_name_from_properties(props, format!("feature_{}", idx));
                            let properties = props
                                .and_then(|p| p.as_object())
                                .map(|obj| obj.iter().map(|(k, v)| (k.clone(), v.clone())).collect())
                                .unwrap_or_default();
                            features.push(GeoJsonFeature {
                                name,
                                geometry: geometry.clone(),
                                properties,
                            });
                        }
                    }
                }
            } else if type_str == "Feature" {
                if let Some(geometry) = json.get("geometry") {
                    let props = json.get("properties");
                    let name = feature_name_from_properties(props, "feature_0".to_string());
                    let properties = props
                        .and_then(|p| p.as_object())
                        .map(|obj| obj.iter().map(|(k, v)| (k.clone(), v.clone())).collect())
                        .unwrap_or_default();
                    features.push(GeoJsonFeature {
                        name,
                        geometry: geometry.clone(),
                        properties,
                    });
                }
            }
        }
        
        Ok(Self { features })
    }
    
    pub fn find_location(&self, location_name: &str) -> Option<LocationCoordinates> {
        let search_name = location_name.to_lowercase();
        if search_name.is_empty() {
            return None;
        }
        for feature in &self.features {
            if feature.name.to_lowercase().contains(&search_name) {
                return self.extract_coordinates(&feature.geometry, &feature.name, &feature.properties);
            }
            for key in ["name", "Name", "NAME", "LocName", "Locality", "City"] {
                if let Some(v) = feature.properties.get(key).and_then(|v| v.as_str()) {
                    if v.to_lowercase().contains(&search_name) {
                        return self.extract_coordinates(&feature.geometry, &feature.name, &feature.properties);
                    }
                }
            }
        }
        None
    }

    /// Returns all location names from the GeoJSON index.
    /// Used by BitNetAgent to match article text against venues (e.g. Hampton Coliseum).
    pub fn all_location_names(&self) -> Vec<String> {
        self.features.iter().map(|f| f.name.clone()).collect()
    }

    /// Nearest locations within max_distance_km. Reserved for proximity / map features.
    #[allow(dead_code)]
    pub fn find_nearest(&self, lat: f64, lon: f64, max_distance_km: f64) -> Vec<LocationCoordinates> {
        let mut results = Vec::new();
        
        for feature in &self.features {
            if let Some(coords) = self.extract_coordinates(&feature.geometry, &feature.name, &feature.properties) {
                let distance = haversine_distance(lat, lon, coords.latitude, coords.longitude);
                if distance <= max_distance_km {
                    results.push(coords);
                }
            }
        }
        
        results.sort_by(|a, b| {
            let dist_a = haversine_distance(lat, lon, a.latitude, a.longitude);
            let dist_b = haversine_distance(lat, lon, b.latitude, b.longitude);
            dist_a.partial_cmp(&dist_b).unwrap_or(std::cmp::Ordering::Equal)
        });
        
        results
    }
    
    fn extract_coordinates(
        &self,
        geometry: &JsonValue,
        name: &str,
        properties: &HashMap<String, JsonValue>,
    ) -> Option<LocationCoordinates> {
        let geom_type = geometry.get("type")?.as_str()?;
        
        match geom_type {
            "Point" => {
                if let Some(JsonValue::Array(coords)) = geometry.get("coordinates") {
                    if coords.len() >= 2 {
                        let lon = coords[0].as_f64()?;
                        let lat = coords[1].as_f64()?;
                        return Some(LocationCoordinates {
                            longitude: lon,
                            latitude: lat,
                            name: Some(name.to_string()),
                            properties: properties.clone(),
                        });
                    }
                }
            }
            "Polygon" => {
                if let Some(JsonValue::Array(rings)) = geometry.get("coordinates") {
                    if let Some(JsonValue::Array(ring)) = rings.first() {
                        let center = calculate_polygon_center(ring)?;
                        return Some(LocationCoordinates {
                            longitude: center.0,
                            latitude: center.1,
                            name: Some(name.to_string()),
                            properties: properties.clone(),
                        });
                    }
                }
            }
            "MultiPolygon" => {
                if let Some(JsonValue::Array(polygons)) = geometry.get("coordinates") {
                    if let Some(JsonValue::Array(polygon)) = polygons.first() {
                        if let Some(JsonValue::Array(ring)) = polygon.first() {
                            let center = calculate_polygon_center(ring)?;
                            return Some(LocationCoordinates {
                                longitude: center.0,
                                latitude: center.1,
                                name: Some(name.to_string()),
                                properties: properties.clone(),
                            });
                        }
                    }
                }
            }
            _ => {}
        }
        
        None
    }
}

fn calculate_polygon_center(ring: &[JsonValue]) -> Option<(f64, f64)> {
    let mut sum_lon = 0.0;
    let mut sum_lat = 0.0;
    let mut count = 0;
    
    for point in ring {
        if let Some(coords) = point.as_array() {
            if coords.len() >= 2 {
                if let (Some(lon), Some(lat)) = (coords[0].as_f64(), coords[1].as_f64()) {
                    sum_lon += lon;
                    sum_lat += lat;
                    count += 1;
                }
            }
        }
    }
    
    if count > 0 {
        Some((sum_lon / count as f64, sum_lat / count as f64))
    } else {
        None
    }
}

fn haversine_distance(lat1: f64, lon1: f64, lat2: f64, lon2: f64) -> f64 {
    let r = 6371.0;
    let d_lat = (lat2 - lat1).to_radians();
    let d_lon = (lon2 - lon1).to_radians();
    
    let a = (d_lat / 2.0).sin().powi(2) +
            lat1.to_radians().cos() * lat2.to_radians().cos() *
            (d_lon / 2.0).sin().powi(2);
    let c = 2.0 * a.sqrt().asin();
    
    r * c
}

pub fn load_geojson_directory(dir: &std::path::Path) -> Result<GeoJsonIndex> {
    let mut all_features = Vec::new();
    
    if !dir.exists() {
        tracing::warn!("GeoJSON directory does not exist: {:?}", dir);
        return Ok(GeoJsonIndex { features: all_features });
    }
    
    for entry in std::fs::read_dir(dir)
        .context(format!("Failed to read directory: {:?}", dir))? {
        let entry = entry?;
        let path = entry.path();
        
        if path.extension().and_then(|s| s.to_str()) == Some("geojson") ||
           path.extension().and_then(|s| s.to_str()) == Some("json") {
            match GeoJsonIndex::from_file(&path) {
                Ok(index) => {
                    all_features.extend(index.features);
                }
                Err(e) => {
                    tracing::warn!("Failed to load GeoJSON file {:?}: {}", path, e);
                }
            }
        }
    }
    
    Ok(GeoJsonIndex { features: all_features })
}

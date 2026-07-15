use ::geojson::{GeoJson, Geometry, Value as GeoValue};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use anyhow::{Context, Result};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LocationCoordinates {
    pub latitude: f64,
    pub longitude: f64,
    pub name: Option<String>,
    pub properties: HashMap<String, serde_json::Value>,
}

#[derive(Debug, Clone)]
pub struct GeoJsonIndex {
    features: Vec<GeoJsonFeature>,
}

#[derive(Debug, Clone)]
struct GeoJsonFeature {
    name: String,
    geometry: Geometry,
    properties: HashMap<String, serde_json::Value>,
}

impl GeoJsonIndex {
    pub fn from_file(path: &std::path::Path) -> Result<Self> {
        let content = std::fs::read_to_string(path)
            .context(format!("Failed to read GeoJSON file: {:?}", path))?;
        
        Self::from_str(&content)
    }
    
    pub fn from_str(content: &str) -> Result<Self> {
        let geojson: GeoJson = content.parse()
            .context("Failed to parse GeoJSON")?;
        
        let mut features = Vec::new();
        
        match geojson {
            GeoJson::FeatureCollection(collection) => {
                for feature in collection.features {
                    if let Some(geometry) = feature.geometry {
                        let name = feature.properties
                            .as_ref()
                            .and_then(|p| p.get("name"))
                            .and_then(|v| v.as_str())
                            .map(|s| s.to_string())
                            .unwrap_or_else(|| format!("feature_{}", features.len()));
                        
                        let properties = feature.properties
                            .unwrap_or_default()
                            .iter()
                            .map(|(k, v)| (k.clone(), v.clone()))
                            .collect();
                        
                        features.push(GeoJsonFeature {
                            name,
                            geometry,
                            properties,
                        });
                    }
                }
            }
            GeoJson::Feature(feature) => {
                if let Some(geometry) = feature.geometry {
                    let name = feature.properties
                        .as_ref()
                        .and_then(|p| p.get("name"))
                        .and_then(|v| v.as_str())
                        .map(|s| s.to_string())
                        .unwrap_or_else(|| "feature_0".to_string());
                    
                    let properties = feature.properties
                        .unwrap_or_default()
                        .iter()
                        .map(|(k, v)| (k.clone(), v.clone()))
                        .collect();
                    
                    features.push(GeoJsonFeature {
                        name,
                        geometry,
                        properties,
                    });
                }
            }
            GeoJson::Geometry(geometry) => {
                features.push(GeoJsonFeature {
                    name: "geometry_0".to_string(),
                    geometry,
                    properties: HashMap::new(),
                });
            }
        }
        
        Ok(Self { features })
    }
    
    pub fn find_location(&self, location_name: &str) -> Option<LocationCoordinates> {
        let search_name = location_name.to_lowercase();
        
        for feature in &self.features {
            if feature.name.to_lowercase().contains(&search_name) {
                return self.extract_coordinates(&feature.geometry, &feature.name, &feature.properties);
            }
            
            if let Some(prop_name) = feature.properties.get("name") {
                if let Some(name_str) = prop_name.as_str() {
                    if name_str.to_lowercase().contains(&search_name) {
                        return self.extract_coordinates(&feature.geometry, &feature.name, &feature.properties);
                    }
                }
            }
        }
        
        None
    }
    
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
        geometry: &Geometry,
        name: &str,
        properties: &HashMap<String, serde_json::Value>,
    ) -> Option<LocationCoordinates> {
        match &geometry.value {
            GeoValue::Point(point) => {
                Some(LocationCoordinates {
                    longitude: point[0],
                    latitude: point[1],
                    name: Some(name.to_string()),
                    properties: properties.clone(),
                })
            }
            GeoValue::Polygon(polygon) => {
                if let Some(ring) = polygon.first() {
                    if !ring.is_empty() {
                        let center = calculate_polygon_center(ring);
                        Some(LocationCoordinates {
                            longitude: center.0,
                            latitude: center.1,
                            name: Some(name.to_string()),
                            properties: properties.clone(),
                        })
                    } else {
                        None
                    }
                } else {
                    None
                }
            }
            GeoValue::MultiPolygon(multi_polygon) => {
                if let Some(polygon) = multi_polygon.first() {
                    if let Some(ring) = polygon.first() {
                        if !ring.is_empty() {
                            let center = calculate_polygon_center(ring);
                            Some(LocationCoordinates {
                                longitude: center.0,
                                latitude: center.1,
                                name: Some(name.to_string()),
                                properties: properties.clone(),
                            })
                        } else {
                            None
                        }
                    } else {
                        None
                    }
                } else {
                    None
                }
            }
            _ => None,
        }
    }
}

fn calculate_polygon_center(ring: &[[f64; 2]]) -> (f64, f64) {
    let mut sum_lon = 0.0;
    let mut sum_lat = 0.0;
    let count = ring.len() as f64;
    
    for point in ring {
        sum_lon += point[0];
        sum_lat += point[1];
    }
    
    (sum_lon / count, sum_lat / count)
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
        return Ok(GeoJsonIndex { features: all_features });
    }
    
    for entry in std::fs::read_dir(dir)
        .context(format!("Failed to read directory: {:?}", dir))? {
        let entry = entry?;
        let path = entry.path();
        
        if path.extension().and_then(|s| s.to_str()) == Some("geojson") ||
           path.extension().and_then(|s| s.to_str()) == Some("json") {
            let index = GeoJsonIndex::from_file(&path)?;
            all_features.extend(index.features);
        }
    }
    
    Ok(GeoJsonIndex { features: all_features })
}

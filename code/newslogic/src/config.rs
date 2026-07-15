use serde::Deserialize;
use anyhow::Context;
use crate::error::{AppResult, wrap_yaml_error};

#[derive(Debug, Deserialize)]
pub struct Config {
    #[allow(dead_code)] // reserved: config file schema
    pub version: String,
    #[allow(dead_code)] // reserved: config file schema
    pub description: Option<String>,
    pub feeds: FeedsConfig,
}

#[derive(Debug, Deserialize)]
pub struct FeedsConfig {
    pub local_news: Option<FeedCategory>,
    pub government: Option<FeedCategory>,
    #[serde(default)]
    pub transit: Option<FeedCategory>,
    #[serde(default)]
    pub education: Option<FeedCategory>,
    pub real_time: Option<FeedCategory>,
}

#[derive(Debug, Deserialize)]
pub struct FeedCategory {
    pub enabled: bool,
    pub sources: Vec<FeedSource>,
}

#[derive(Debug, Deserialize, Clone)]
pub struct FeedSource {
    pub id: String,
    pub name: String,
    #[serde(rename = "type")]
    pub feed_type: String,
    pub url: String,
    #[allow(dead_code)] // reserved: feed metadata / filtering
    pub category: Option<String>,
    #[allow(dead_code)] // reserved: feed metadata / filtering
    pub priority: Option<String>,
    #[allow(dead_code)] // reserved: feed metadata / filtering
    pub refresh_interval: Option<u64>,
    #[allow(dead_code)] // reserved: feed metadata / filtering
    pub tags: Option<Vec<String>>,
    pub geo_tags: Option<Vec<String>>,
    pub enabled: bool,
}

pub fn load_config(path: &str) -> AppResult<Config> {
    let content = std::fs::read_to_string(path)
        .context(format!("Failed to read config file: {}", path))?;
    
    let config: Config = serde_yaml::from_str(&content)
        .map_err(wrap_yaml_error)?;
    
    Ok(config)
}

pub fn get_rss_feeds(config: &Config) -> Vec<&FeedSource> {
    let mut feeds = Vec::new();
    
    if let Some(ref local_news) = config.feeds.local_news {
        if local_news.enabled {
            feeds.extend(
                local_news.sources.iter()
                    .filter(|s| s.feed_type == "rss" && s.enabled)
            );
        }
    }
    
    if let Some(ref government) = config.feeds.government {
        if government.enabled {
            feeds.extend(
                government.sources.iter()
                    .filter(|s| s.feed_type == "rss" && s.enabled)
            );
        }
    }
    
    if let Some(ref transit) = config.feeds.transit {
        if transit.enabled {
            feeds.extend(
                transit.sources.iter()
                    .filter(|s| s.feed_type == "rss" && s.enabled)
            );
        }
    }

    if let Some(ref education) = config.feeds.education {
        if education.enabled {
            feeds.extend(
                education.sources.iter()
                    .filter(|s| s.feed_type == "rss" && s.enabled)
            );
        }
    }
    
    if let Some(ref real_time) = config.feeds.real_time {
        if real_time.enabled {
            feeds.extend(
                real_time.sources.iter()
                    .filter(|s| s.feed_type == "rss" && s.enabled)
            );
        }
    }
    
    feeds
}

/// Redis URL for truth-engine. Default: redis://127.0.0.1:6379
pub fn redis_url() -> String {
    std::env::var("REDIS_URL").unwrap_or_else(|_| "redis://127.0.0.1:6379".to_string())
}

#[derive(Debug, Clone)]
pub struct LightpandaConfig {
    pub host: String,
    pub port: u16,
    pub timeout_seconds: u64,
}

impl Default for LightpandaConfig {
    fn default() -> Self {
        Self {
            host: "127.0.0.1".to_string(),
            port: 9222,
            timeout_seconds: 30,
        }
    }
}

impl LightpandaConfig {
    pub fn from_env() -> Self {
        Self {
            host: std::env::var("LIGHTPANDA_HOST")
                .unwrap_or_else(|_| "127.0.0.1".to_string()),
            port: std::env::var("LIGHTPANDA_PORT")
                .ok()
                .and_then(|s| s.parse().ok())
                .unwrap_or(9222),
            timeout_seconds: std::env::var("LIGHTPANDA_TIMEOUT")
                .ok()
                .and_then(|s| s.parse().ok())
                .unwrap_or(30),
        }
    }
}


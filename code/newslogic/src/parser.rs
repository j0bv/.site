use rss::Channel;
use anyhow::Context;
use crate::error::AppResult;
use chrono::{DateTime, Utc};
use std::collections::HashSet;
use sha2::{Sha256, Digest};

#[derive(Debug, Clone)]
pub struct Article {
    pub title: String,
    pub description: Option<String>,
    pub link: Option<String>,
    pub pub_date: Option<DateTime<Utc>>,
    pub author: Option<String>,
    pub content: Option<String>,
    pub categories: Vec<String>,
    pub guid: Option<String>,
    pub feed_id: String,
    pub feed_name: String,
    pub geo_tags: Vec<String>,
    pub content_hash: String,
}

impl Article {
    pub fn compute_hash(&self) -> String {
        let mut hasher = Sha256::new();
        
        if let Some(ref link) = self.link {
            hasher.update(link.as_bytes());
        } else if let Some(ref guid) = self.guid {
            hasher.update(guid.as_bytes());
        } else {
            hasher.update(self.title.as_bytes());
            if let Some(ref desc) = self.description {
                hasher.update(desc.as_bytes());
            }
        }
        
        format!("{:x}", hasher.finalize())
    }
    
    pub fn full_text(&self) -> String {
        let mut text = self.title.clone();
        
        if let Some(ref desc) = self.description {
            text.push_str("\n\n");
            text.push_str(desc);
        }
        
        if let Some(ref content) = self.content {
            text.push_str("\n\n");
            text.push_str(content);
        }
        
        text
    }
}

pub fn parse_rss_feed(
    xml_content: &str,
    feed_id: &str,
    feed_name: &str,
    geo_tags: &[String],
) -> AppResult<Vec<Article>> {
    let channel = Channel::read_from(xml_content.as_bytes())
        .context("Failed to parse RSS XML")?;
    
    let mut articles = Vec::new();
    let mut seen_hashes = HashSet::new();
    
    for item in channel.items() {
        let title = item.title()
            .unwrap_or("Untitled")
            .to_string();
        
        let description = item.description().map(|s| s.to_string());
        let link = item.link().map(|s| s.to_string());
        
        let pub_date = item.pub_date()
            .and_then(|date_str| {
                DateTime::parse_from_rfc2822(date_str)
                    .or_else(|_| DateTime::parse_from_rfc3339(date_str))
                    .ok()
            })
            .map(|dt| dt.with_timezone(&Utc));
        
        let author = item.author().map(|s| s.to_string());
        let content = item.content().map(|s| s.to_string());
        let guid = item.guid().map(|g| g.value().to_string());
        
        let categories: Vec<String> = item.categories()
            .iter()
            .map(|cat| cat.name().to_string())
            .collect();
        
        let mut article = Article {
            title,
            description,
            link,
            pub_date,
            author,
            content,
            categories,
            guid,
            feed_id: feed_id.to_string(),
            feed_name: feed_name.to_string(),
            geo_tags: geo_tags.to_vec(),
            content_hash: String::new(),
        };
        
        article.content_hash = article.compute_hash();
        
        if !seen_hashes.contains(&article.content_hash) {
            seen_hashes.insert(article.content_hash.clone());
            articles.push(article);
        }
    }
    
    Ok(articles)
}

pub fn parse_all_feeds_from_directory(
    output_dir: &std::path::Path,
    config: &crate::config::Config,
) -> AppResult<Vec<Article>> {
    use std::fs;
    use crate::config::get_rss_feeds;
    
    let feeds = get_rss_feeds(config);
    let mut all_articles = Vec::new();
    let mut global_seen_hashes = HashSet::new();
    
    for feed in feeds {
        let xml_path = output_dir.join(format!("{}.xml", feed.id));
        
        if !xml_path.exists() {
            tracing::warn!("XML file not found: {:?}", xml_path);
            continue;
        }
        
        let xml_content = fs::read_to_string(&xml_path)
            .context(format!("Failed to read XML file: {:?}", xml_path))?;
        
        let geo_tags = feed.geo_tags.clone().unwrap_or_default();
        
        match parse_rss_feed(&xml_content, &feed.id, &feed.name, &geo_tags) {
            Ok(articles) => {
                let mut deduplicated = Vec::new();
                for article in articles {
                    if !global_seen_hashes.contains(&article.content_hash) {
                        global_seen_hashes.insert(article.content_hash.clone());
                        deduplicated.push(article);
                    }
                }
                all_articles.extend(deduplicated);
            }
            Err(e) => {
                tracing::error!("Failed to parse feed {}: {}", feed.id, e);
            }
        }
    }
    
    Ok(all_articles)
}

pub fn extract_article_links(
    output_dir: &std::path::Path,
    config: &crate::config::Config,
    max_links: Option<usize>,
) -> crate::error::AppResult<Vec<String>> {
    use std::fs;
    use crate::config::get_rss_feeds;
    
    let feeds = get_rss_feeds(config);
    let mut all_links = Vec::new();
    
    for feed in feeds {
        let xml_path = output_dir.join(format!("{}.xml", feed.id));
        
        if !xml_path.exists() {
            tracing::warn!("XML file not found: {:?}", xml_path);
            continue;
        }
        
        let xml_content = fs::read_to_string(&xml_path)
            .context(format!("Failed to read XML file: {:?}", xml_path))?;
        
        let channel = Channel::read_from(xml_content.as_bytes())
            .context("Failed to parse RSS XML")?;
        
        for item in channel.items() {
            if let Some(link) = item.link() {
                if !link.is_empty() {
                    all_links.push(link.to_string());
                    
                    if let Some(max) = max_links {
                        if all_links.len() >= max {
                            return Ok(all_links);
                        }
                    }
                }
            }
        }
    }
    
    Ok(all_links)
}


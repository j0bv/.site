mod config;
mod fetcher;
mod error;
mod parser;
mod memory_graph;
mod lightpanda_client;
mod models;
mod geo_data;
mod bitnet_agent;
mod bitnet;
mod nlp_processor;
mod api_server;
mod worker;
mod agent1_extractor;
mod agent2_verifier;

use std::path::Path;
use anyhow::Context;
use tracing::{info, error, warn};
use config::{load_config, get_rss_feeds};
use fetcher::{fetch_feed, save_xml, get_output_path};
use parser::parse_all_feeds_from_directory;
use memory_graph::{Neo4jClient, DocumentWithMemories};

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt::init();
    
    let args: Vec<String> = std::env::args().collect();
    
    if args.len() > 1 && args[1] == "memory-graph" {
        return process_memory_graph().await;
    }
    
    if args.len() > 1 && args[1] == "validate-lightpanda" {
        return validate_lightpanda().await;
    }
    
    if args.len() > 1 && args[1] == "api-server" {
        return start_api_server().await;
    }

    if args.len() > 1 && args[1] == "truth-engine" {
        return run_truth_engine().await;
    }

    if args.len() > 1 && args[1] == "nearby-venues" {
        return run_nearby_venues().await;
    }

    if args.len() > 1 && args[1] == "events-with-locations" {
        return run_events_with_locations().await;
    }
    
    if args.len() > 1 && args[1] == "bitnet-worker" {
        return run_bitnet_worker().await;
    }
    
    info!("Starting RSS feed fetcher");
    
    let config_path = "config/feeds.yaml";
    let config = load_config(config_path)
        .context("Failed to load configuration")?;
    
    info!("Loaded configuration from {}", config_path);
    
    let feeds = get_rss_feeds(&config);
    info!("Found {} RSS feeds to fetch", feeds.len());
    
    if feeds.is_empty() {
        warn!("No enabled RSS feeds found in configuration");
        return Ok(());
    }
    
    let output_dir = Path::new("output");
    
    let mut success_count = 0;
    let mut failure_count = 0;
    
    let fetch_tasks: Vec<_> = feeds.iter().map(|feed| {
        let feed_id = feed.id.clone();
        let feed_name = feed.name.clone();
        let feed_url = feed.url.clone();
        let output_path = get_output_path(&feed.id, output_dir);
        
        tokio::spawn(async move {
            info!("Fetching feed: {} ({})", feed_name, feed_id);
            
            match fetch_feed(&feed_url).await {
                Ok(xml_content) => {
                    match save_xml(&xml_content, &output_path) {
                        Ok(()) => {
                            info!("Successfully saved feed: {} to {:?}", feed_name, output_path);
                            Ok(())
                        }
                        Err(e) => {
                            error!("Failed to save feed {}: {}", feed_name, e);
                            Err(e)
                        }
                    }
                }
                Err(e) => {
                    error!("Failed to fetch feed {} ({}): {}", feed_name, feed_url, e);
                    Err(e)
                }
            }
        })
    }).collect();
    
    for task in fetch_tasks {
        match task.await {
            Ok(Ok(())) => success_count += 1,
            Ok(Err(_)) => failure_count += 1,
            Err(e) => {
                error!("Task join error: {}", e);
                failure_count += 1;
            }
        }
    }
    
    info!(
        "Completed fetching feeds. Success: {}, Failed: {}, Total: {}",
        success_count,
        failure_count,
        feeds.len()
    );
    
    Ok(())
}

async fn process_memory_graph() -> anyhow::Result<()> {
    info!("Processing XML files into memory-graph");
    
    let config_path = "config/feeds.yaml";
    let config = load_config(config_path)
        .context("Failed to load configuration")?;
    
    let output_dir = Path::new("output");
    
    info!("Parsing all RSS feeds from {:?}", output_dir);
    let articles = parse_all_feeds_from_directory(output_dir, &config)
        .context("Failed to parse RSS feeds")?;
    
    info!("Found {} unique articles after deduplication", articles.len());
    
    let org_id = std::env::var("NEO4J_ORG_ID")
        .unwrap_or_else(|_| "default-org".to_string());
    let user_id = std::env::var("NEO4J_USER_ID")
        .unwrap_or_else(|_| "default-user".to_string());
    
    let client = Neo4jClient::new(org_id.clone(), user_id.clone()).await
        .context("Failed to create Neo4j client. Ensure NEO4J_PASSWORD is set.")?;
    
    let documents: Vec<DocumentWithMemories> = articles.iter()
        .map(|article| DocumentWithMemories::from_article(
            article,
            &client.org_id,
            &client.user_id,
        ))
        .collect();
    
    info!("Converted {} articles to memory-graph documents", documents.len());
    
    let json_output = Path::new("output").join("memory-graph-documents.json");
    client.export_to_json_file(&documents, &json_output)
        .context("Failed to export documents to JSON")?;
    
    info!("Exported documents to {:?}", json_output);
    
    info!("Uploading articles to Neo4j...");
    client.add_articles_batch(&articles).await
        .context("Failed to upload articles to Neo4j")?;
    info!("Successfully uploaded {} articles to Neo4j", articles.len());
    
    info!("Memory-graph processing complete!");
    Ok(())
}

async fn run_bitnet_worker() -> anyhow::Result<()> {
    use std::sync::Arc;
    use config::LightpandaConfig;
    use geo_data::load_geojson_directory;
    use lightpanda_client::LightpandaClient;
    use bitnet_agent::BitNetAgent;
    use worker::ArticleWorker;
    info!("Starting BitNet worker (RSS → Lightpanda → BitNet → NLP → Neo4j)");
    let config_path = "config/feeds.yaml";
    let config = load_config(config_path)
        .context("Failed to load configuration")?;
    let output_dir = Path::new("output");
    info!("Parsing all RSS feeds from {:?}", output_dir);
    let articles = parse_all_feeds_from_directory(output_dir, &config)
        .context("Failed to parse RSS feeds")?;
    info!("Found {} unique articles after deduplication", articles.len());
    if articles.is_empty() {
        info!("No articles to process");
        return Ok(());
    }
    let lp_config = LightpandaConfig::from_env();
    let lightpanda = LightpandaClient::new(lp_config.host.clone(), lp_config.port)
        .with_timeout(lp_config.timeout_seconds);
    let geojson_dir = std::env::var("GEOJSON_DIR").unwrap_or_else(|_| "geojson".to_string());
    let geojson_path = Path::new(&geojson_dir);
    let bitnet = BitNetAgent::new(geojson_path)
        .context("Failed to create BitNet agent (ensure GEOJSON_DIR/geojson exists with .geojson files)")?;
    let geojson_index = load_geojson_directory(geojson_path).ok();
    let org_id = std::env::var("NEO4J_ORG_ID")
        .unwrap_or_else(|_| "default-org".to_string());
    let user_id = std::env::var("NEO4J_USER_ID")
        .unwrap_or_else(|_| "default-user".to_string());
    let neo4j = Arc::new(
        Neo4jClient::new_with_geojson(org_id, user_id, geojson_index).await
            .context("Failed to create Neo4j client. Ensure NEO4J_PASSWORD is set.")?
    );
    let nlp = nlp_processor::NlpProcessor::default();
    let worker = ArticleWorker::new(lightpanda, bitnet, neo4j, nlp);
    worker.process_articles_batch(&articles).await
        .context("BitNet worker process_articles_batch failed")?;
    info!("BitNet worker complete");
    Ok(())
}

async fn run_nearby_venues() -> anyhow::Result<()> {
    use geo_data::load_geojson_directory;
    info!("CLI: nearby-venues");
    let org_id = std::env::var("NEO4J_ORG_ID")
        .unwrap_or_else(|_| "default-org".to_string());
    let user_id = std::env::var("NEO4J_USER_ID")
        .unwrap_or_else(|_| "default-user".to_string());
    let geojson_dir = std::env::var("GEOJSON_DIR").unwrap_or_else(|_| "geojson".to_string());
    let geojson_index = load_geojson_directory(Path::new(&geojson_dir)).ok();
    if geojson_index.is_none() {
        tracing::warn!("No GeoJSON data loaded from {}", geojson_dir);
    }
    let neo4j = Neo4jClient::new_with_geojson(org_id, user_id, geojson_index).await
        .context("Failed to create Neo4j client. Ensure NEO4J_PASSWORD is set.")?;
    let lat = 36.85;
    let lon = -76.29;
    let max_km = 10.0;
    let venues = neo4j.get_nearby_venues(lat, lon, max_km);
    info!("Found {} venues within {}km of ({}, {})", venues.len(), max_km, lat, lon);
    for v in &venues {
        info!("  {} ({:.4}, {:.4})", v.name.as_deref().unwrap_or("?"), v.latitude, v.longitude);
    }
    Ok(())
}

async fn run_events_with_locations() -> anyhow::Result<()> {
    info!("CLI: events-with-locations");
    let org_id = std::env::var("NEO4J_ORG_ID")
        .unwrap_or_else(|_| "default-org".to_string());
    let user_id = std::env::var("NEO4J_USER_ID")
        .unwrap_or_else(|_| "default-user".to_string());
    let neo4j = Neo4jClient::new(org_id, user_id).await
        .context("Failed to create Neo4j client. Ensure NEO4J_PASSWORD is set.")?;
    let events = neo4j.get_events_with_locations().await?;
    info!("Retrieved {} events with location data", events.len());
    std::fs::create_dir_all("output")?;
    let path = Path::new("output/events-with-locations.json");
    std::fs::write(path, serde_json::to_string_pretty(&events)?)?;
    info!("Events written to {:?}", path);
    for e in events.iter().take(5) {
        info!("  {:?}", e);
    }
    Ok(())
}

async fn validate_lightpanda() -> anyhow::Result<()> {
    use lightpanda_client::LightpandaClient;
    use config::LightpandaConfig;
    use parser::extract_article_links;
    
    info!("Starting Lightpanda validation");
    
    let lp_config = LightpandaConfig::from_env();
    info!("Lightpanda config: {}:{} (timeout: {}s)", lp_config.host, lp_config.port, lp_config.timeout_seconds);
    
    let client = LightpandaClient::new(lp_config.host.clone(), lp_config.port)
        .with_timeout(lp_config.timeout_seconds);
    
    info!("Step 1: Validating Lightpanda connection...");
    match client.validate_connection().await {
        Ok(true) => {
            info!("✓ Lightpanda connection validated successfully");
        }
        Ok(false) => {
            error!("✗ Lightpanda connection validation failed");
            return Err(anyhow::anyhow!("Lightpanda connection validation failed"));
        }
        Err(e) => {
            error!("✗ Failed to connect to Lightpanda: {}", e);
            return Err(e);
        }
    }
    
    info!("Step 2: Reading XML files and extracting article links...");
    let config_path = "config/feeds.yaml";
    let config = load_config(config_path)
        .context("Failed to load configuration")?;
    
    let output_dir = Path::new("output");
    let article_links = extract_article_links(output_dir, &config, Some(5))
        .context("Failed to extract article links")?;
    
    if article_links.is_empty() {
        warn!("No article links found in XML files");
        return Ok(());
    }
    
    info!("Found {} article links (testing first {} links)", article_links.len(), article_links.len().min(5));
    
    let links_to_test = article_links.iter().take(5);
    let mut success_count = 0;
    let mut failure_count = 0;
    
    for (idx, link) in links_to_test.enumerate() {
        info!("Step 3.{}: Testing navigation to: {}", idx + 1, link);
        
        match client.test_navigation(link).await {
            Ok(content) => {
                info!("✓ Successfully extracted content from: {}", link);
                info!("  Title: {}", content.title);
                info!("  Text length: {} characters", content.main_text.len());
                info!("  HTML length: {} characters", content.html_content.len());
                info!("  Images found: {}", content.images.len());
                if let Some(date) = content.published_date {
                    info!("  Published: {}", date);
                }
                success_count += 1;
            }
            Err(e) => {
                error!("✗ Failed to extract content from {}: {}", link, e);
                failure_count += 1;
            }
        }
    }
    
    info!("Validation complete!");
    info!("  Success: {}", success_count);
    info!("  Failed: {}", failure_count);
    info!("  Total tested: {}", success_count + failure_count);
    
    if failure_count > 0 {
        warn!("Some articles failed to extract. Check Lightpanda logs for details.");
    }
    
    Ok(())
}

async fn run_truth_engine() -> anyhow::Result<()> {
    use std::sync::Arc;
    use agent1_extractor::TruthExtractor;
    use agent2_verifier::Verifier;
    use config::{get_rss_feeds, load_config, redis_url, LightpandaConfig};
    use fetcher::fetch_feed;
    use crate::error::RedisError;
    use lightpanda_client::LightpandaClient;
    use models::EnrichedArticle;
    use parser::parse_rss_feed;

    info!("Starting Truth Engine (RSS → Lightpanda → Agent1 → Neo4j → Agent2)");

    let config_path = "config/feeds.yaml";
    let config = load_config(config_path).context("Failed to load config")?;
    let feeds = get_rss_feeds(&config);
    if feeds.is_empty() {
        anyhow::bail!("No RSS feeds enabled");
    }

    let redis_client = redis::Client::open(redis_url().as_str())
        .context("Failed to create Redis client")?;
    let redis = redis_client.get_connection_manager().await
        .map_err(|e| RedisError::ConnectionFailed(e.to_string()))?;

    let org_id = std::env::var("NEO4J_ORG_ID").unwrap_or_else(|_| "default-org".to_string());
    let user_id = std::env::var("NEO4J_USER_ID").unwrap_or_else(|_| "default-user".to_string());
    let neo4j = Arc::new(
        Neo4jClient::new(org_id, user_id).await
            .context("Failed to create Neo4j client. Set NEO4J_PASSWORD.")?,
    );

    let lp_cfg = LightpandaConfig::from_env();
    let lightpanda = Arc::new(
        LightpandaClient::new(lp_cfg.host.clone(), lp_cfg.port).with_timeout(lp_cfg.timeout_seconds),
    );

    let bitnet = Arc::new(bitnet::BitNetInference::new());

    let redis_fetcher = redis.clone();
    let lightpanda_fetcher = lightpanda.clone();
    let config_path = config_path.to_string();
    let fetcher = tokio::spawn(async move {
        use crate::error::RedisError;
        let cfg = load_config(&config_path).expect("config");
        let feeds = get_rss_feeds(&cfg);
        let mut conn = redis_fetcher.clone();
        loop {
            for feed in &feeds {
                if let Ok(xml) = fetch_feed(&feed.url).await {
                    let geo = feed.geo_tags.as_deref().unwrap_or(&[]);
                    if let Ok(articles) = parse_rss_feed(&xml, &feed.id, &feed.name, geo) {
                        for a in articles.into_iter().take(5) {
                            let link = match &a.link {
                                Some(l) if !l.is_empty() => l.clone(),
                                _ => continue,
                            };
                            let published = a.pub_date.unwrap_or_else(chrono::Utc::now);
                            match lightpanda_fetcher.extract_article_content(&link).await {
                                Ok(content) => {
                                    let en = EnrichedArticle {
                                        id: a.content_hash.clone(),
                                        title: a.title,
                                        url: link,
                                        source: a.feed_name,
                                        published,
                                        text: content.main_text,
                                        html: content.html_content,
                                    };
                                    let j = serde_json::to_string(&en).unwrap();
                                    let _: Result<(), _> = redis::cmd("RPUSH")
                                        .arg("articles:extracted")
                                        .arg(j)
                                        .query_async(&mut conn)
                                        .await
                                        .map_err(|e| RedisError::CommandFailed(e.to_string()));
                                }
                                Err(e) => tracing::warn!("Lightpanda extract {}: {}", link, e),
                            }
                        }
                    }
                }
            }
            tokio::time::sleep(tokio::time::Duration::from_secs(60)).await;
        }
    });

    let a1 = TruthExtractor::new(redis.clone(), neo4j.clone(), bitnet.clone());
    let a2 = Verifier::new(redis.clone(), neo4j.clone(), bitnet.clone());
    let task1 = tokio::spawn(async move { a1.run().await });
    let task2 = tokio::spawn(async move { a2.run().await });

    let _ = tokio::join!(fetcher, task1, task2);
    Ok(())
}

async fn start_api_server() -> anyhow::Result<()> {
    use std::sync::Arc;
    use api_server::start_server;
    
    info!("Starting API server for BitNet frontend");
    
    let org_id = std::env::var("NEO4J_ORG_ID")
        .unwrap_or_else(|_| "default-org".to_string());
    let user_id = std::env::var("NEO4J_USER_ID")
        .unwrap_or_else(|_| "default-user".to_string());
    
    let neo4j_client = Arc::new(
        Neo4jClient::new(org_id, user_id).await
            .context("Failed to create Neo4j client. Ensure NEO4J_PASSWORD is set.")?
    );
    
    let port = std::env::var("API_PORT")
        .ok()
        .and_then(|s| s.parse().ok())
        .unwrap_or(3001);
    
    info!("API server will listen on port {}", port);
    info!("Endpoints:");
    info!("  GET /api/map/events - Get map events");
    info!("  GET /api/map/geojson - Get map events as GeoJSON");
    info!("  GET /api/memory/graph - Get knowledge graph data");
    
    start_server(neo4j_client, port).await
        .context("API server error")?;
    
    Ok(())
}

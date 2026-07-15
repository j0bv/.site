use crate::parser::Article;
use crate::lightpanda_client::LightpandaClient;
use crate::bitnet_agent::{BitNetAgent, Entity};
use crate::memory_graph::Neo4jClient;
use crate::error::AppResult;
use crate::nlp_processor::{NlpProcessor, ProcessedArticle};
use anyhow::Context;
use std::sync::Arc;
use tracing::{info, warn, error};

pub struct ArticleWorker {
    lightpanda: LightpandaClient,
    bitnet: BitNetAgent,
    neo4j: Arc<Neo4jClient>,
    nlp: NlpProcessor,
}

impl ArticleWorker {
    pub fn new(
        lightpanda: LightpandaClient,
        bitnet: BitNetAgent,
        neo4j: Arc<Neo4jClient>,
        nlp: NlpProcessor,
    ) -> Self {
        Self {
            lightpanda,
            bitnet,
            neo4j,
            nlp,
        }
    }

    pub async fn process_article(&self, article: &Article) -> AppResult<ProcessedArticle> {
        let article_id = format!("article_{}", article.content_hash);
        
        info!("Processing article: {} ({})", article.title, article_id);
        
        let full_content = if let Some(ref link) = article.link {
            match self.lightpanda.extract_article_content(link).await {
                Ok(content) => {
                    info!("Extracted {} chars from {}", content.main_text.len(), link);
                    Some(content.main_text)
                }
                Err(e) => {
                    warn!("Failed to extract content from {}: {}, using RSS content", link, e);
                    None
                }
            }
        } else {
            None
        };
        
        let text_to_analyze: String = full_content
            .unwrap_or_else(|| article.full_text());
        
        let events = self.bitnet.extract_events(article, Some(&text_to_analyze)).await
            .context("Failed to extract events")?;
        
        info!("Extracted {} events from article", events.len());
        
        let entities: Vec<Entity> = events.iter().flat_map(|e| &e.entities).cloned().collect();
        let mut seen_names = std::collections::HashSet::new();
        let unique_entities: Vec<Entity> = entities
            .into_iter()
            .filter(|e| seen_names.insert(e.name.clone()))
            .collect();
        info!("Extracted {} unique entities from events", unique_entities.len());
        
        let quality = self.nlp.analyze(&text_to_analyze);
        self.neo4j.add_document_quality(&article_id, &quality).await
            .context("Failed to store document quality in Neo4j")?;
        
        for event in &events {
            self.neo4j.add_event(event).await
                .context("Failed to store event in Neo4j")?;
        }
        
        for entity in &unique_entities {
            self.neo4j.add_entity(&article_id, entity).await
                .context("Failed to store entity in Neo4j")?;
        }

        let venue_names = self.bitnet.extract_venue_names_from_text(&text_to_analyze);
        self.neo4j.add_article(article, Some(&venue_names)).await
            .context("Failed to store article in Neo4j")?;
        
        info!("Successfully processed article: {}", article_id);
        
        Ok(ProcessedArticle { article_id, quality })
    }

    pub async fn process_articles_batch(&self, articles: &[Article]) -> AppResult<()> {
        info!("Processing batch of {} articles", articles.len());
        
        let mut success_count = 0;
        let mut failure_count = 0;
        let mut processed: Vec<ProcessedArticle> = Vec::new();
        
        for (idx, article) in articles.iter().enumerate() {
            match self.process_article(article).await {
                Ok(pa) => {
                    success_count += 1;
                    processed.push(pa);
                    if (idx + 1) % 10 == 0 {
                        info!("Processed {}/{} articles", idx + 1, articles.len());
                    }
                }
                Err(e) => {
                    error!("Failed to process article {}: {}", article.title, e);
                    failure_count += 1;
                }
            }
            
            tokio::time::sleep(tokio::time::Duration::from_millis(100)).await;
        }
        
        if processed.len() >= 2 {
            let edges = self.nlp.batch_similarity(&processed, 0.15, 10);
            self.neo4j.add_similarity_batch(&edges).await
                .context("Failed to store similarity edges in Neo4j")?;
        }
        
        info!("Batch complete: {} succeeded, {} failed", success_count, failure_count);
        
        Ok(())
    }
}

use anyhow::Result;
use serde_json::{json, Value};
use tokio_tungstenite::{connect_async, tungstenite::Message};
use futures::{SinkExt, StreamExt};
use std::time::Duration;
use tokio::time::timeout;
use crate::error::LightpandaError;
use crate::models::{ArticleContent, ImageInfo};
use chrono::Utc;

pub struct LightpandaClient {
    host: String,
    port: u16,
    timeout_seconds: u64,
}

impl LightpandaClient {
    pub fn new(host: String, port: u16) -> Self {
        Self {
            host,
            port,
            timeout_seconds: 30,
        }
    }

    pub fn with_timeout(mut self, timeout_seconds: u64) -> Self {
        self.timeout_seconds = timeout_seconds;
        self
    }

    fn ws_url(&self) -> String {
        format!("ws://{}:{}/json", self.host, self.port)
    }

    pub async fn validate_connection(&self) -> Result<bool> {
        tracing::info!("Validating Lightpanda connection at {}:{}", self.host, self.port);
        
        let ws_url = format!("ws://{}:{}/json", self.host, self.port);
        
        match timeout(Duration::from_secs(5), connect_async(&ws_url)).await {
            Ok(Ok((_ws_stream, _response))) => {
                tracing::info!("Successfully connected to Lightpanda WebSocket");
                
                let version_result = self.get_version().await;
                match version_result {
                    Ok(version_info) => {
                        tracing::info!("Lightpanda version: {:?}", version_info);
                        Ok(true)
                    }
                    Err(e) => {
                        tracing::warn!("Connected but failed to get version: {}", e);
                        Ok(true)
                    }
                }
            }
            Ok(Err(e)) => {
                tracing::error!("Failed to connect to Lightpanda: {}", e);
                Err(anyhow::anyhow!("Connection failed: {}", e))
            }
            Err(_) => {
                tracing::error!("Connection timeout to Lightpanda");
                Err(anyhow::anyhow!("Connection timeout"))
            }
        }
    }

    async fn get_version(&self) -> Result<Value> {
        let ws_url = self.ws_url();
        let (ws_stream, _) = connect_async(&ws_url)
            .await
            .map_err(|e| LightpandaError::ConnectionFailed(e.to_string()))?;
        
        let (mut write, mut read) = ws_stream.split();
        
        let version_cmd = json!({
            "jsonrpc": "2.0",
            "id": 1,
            "method": "Browser.getVersion"
        });
        
        write.send(Message::Text(version_cmd.to_string())).await
            .map_err(|e| LightpandaError::WebSocketError(e.to_string()))?;
        
        let timeout_duration = Duration::from_secs(self.timeout_seconds);
        loop {
            match timeout(timeout_duration, read.next()).await {
                Ok(Some(Ok(Message::Text(text)))) => {
                    let json: Value = serde_json::from_str(&text)
                        .map_err(|e| LightpandaError::JsonError(e.to_string()))?;
                    return Ok(json);
                }
                Ok(Some(Ok(Message::Ping(_) | Message::Pong(_) | Message::Binary(_) | Message::Close(_) | Message::Frame(_)))) => {
                    continue;
                }
                Ok(Some(Err(e))) => return Err(LightpandaError::WebSocketError(e.to_string()).into()),
                Ok(None) => return Err(LightpandaError::ConnectionFailed("Connection closed".to_string()).into()),
                Err(_) => return Err(LightpandaError::NavigationTimeout("Version request timeout".to_string()).into()),
            }
        }
    }

    pub async fn test_navigation(&self, url: &str) -> Result<ArticleContent> {
        tracing::info!("Testing navigation to: {}", url);
        self.extract_article_content(url).await
    }

    pub async fn extract_article_content(&self, url: &str) -> Result<ArticleContent> {
        let ws_url = self.ws_url();
        let (ws_stream, _) = connect_async(&ws_url)
            .await
            .map_err(|e| LightpandaError::ConnectionFailed(e.to_string()))?;
        
        let (mut write, mut read) = ws_stream.split();
        
        let timeout_duration = Duration::from_secs(self.timeout_seconds);
        
        let target_id = self.create_target(&mut write, &mut read, timeout_duration).await?;
        
        self.navigate_to_url(&mut write, &mut read, &target_id, url, timeout_duration).await?;
        
        let content = self.extract_content(&mut write, &mut read, timeout_duration).await?;
        
        Ok(content)
    }

    async fn create_target(
        &self,
        write: &mut futures::stream::SplitSink<tokio_tungstenite::WebSocketStream<tokio_tungstenite::MaybeTlsStream<tokio::net::TcpStream>>, Message>,
        read: &mut futures::stream::SplitStream<tokio_tungstenite::WebSocketStream<tokio_tungstenite::MaybeTlsStream<tokio::net::TcpStream>>>,
        timeout_duration: Duration,
    ) -> Result<String> {
        let target_cmd = json!({
            "jsonrpc": "2.0",
            "id": 2,
            "method": "Target.createTarget",
            "params": { "url": "about:blank" }
        });
        
        write.send(Message::Text(target_cmd.to_string())).await
            .map_err(|e| LightpandaError::WebSocketError(e.to_string()))?;
        
        loop {
            match timeout(timeout_duration, read.next()).await {
                Ok(Some(Ok(Message::Text(text)))) => {
                    let json: Value = serde_json::from_str(&text)
                        .map_err(|e| LightpandaError::JsonError(e.to_string()))?;
                    
                    if let Some(result) = json.get("result") {
                        if let Some(target_id) = result.get("targetId") {
                            if let Some(id) = target_id.as_str() {
                                return Ok(id.to_string());
                            }
                        }
                    }
                    
                    return Err(LightpandaError::CDPError("Failed to get target ID from response".to_string()).into());
                }
                Ok(Some(Ok(Message::Ping(_) | Message::Pong(_) | Message::Binary(_) | Message::Close(_) | Message::Frame(_)))) => {
                    continue;
                }
                Ok(Some(Err(e))) => return Err(LightpandaError::WebSocketError(e.to_string()).into()),
                Ok(None) => return Err(LightpandaError::ConnectionFailed("Connection closed".to_string()).into()),
                Err(_) => return Err(LightpandaError::NavigationTimeout("Target creation timeout".to_string()).into()),
            }
        }
    }

    async fn navigate_to_url(
        &self,
        write: &mut futures::stream::SplitSink<tokio_tungstenite::WebSocketStream<tokio_tungstenite::MaybeTlsStream<tokio::net::TcpStream>>, Message>,
        read: &mut futures::stream::SplitStream<tokio_tungstenite::WebSocketStream<tokio_tungstenite::MaybeTlsStream<tokio::net::TcpStream>>>,
        _target_id: &str,
        url: &str,
        timeout_duration: Duration,
    ) -> Result<()> {
        let navigate_cmd = json!({
            "jsonrpc": "2.0",
            "id": 3,
            "method": "Page.navigate",
            "params": { "url": url }
        });
        
        write.send(Message::Text(navigate_cmd.to_string())).await
            .map_err(|e| LightpandaError::WebSocketError(e.to_string()))?;
        
        let mut load_event_received = false;
        let start_time = std::time::Instant::now();
        
        while start_time.elapsed() < timeout_duration {
            match timeout(Duration::from_secs(1), read.next()).await {
                Ok(Some(Ok(Message::Text(text)))) => {
                    if text.contains("Page.loadEventFired") || text.contains("Page.frameNavigated") {
                        load_event_received = true;
                        break;
                    }
                }
                Ok(Some(Ok(Message::Ping(_) | Message::Pong(_) | Message::Binary(_) | Message::Close(_) | Message::Frame(_)))) => {
                    continue;
                }
                Ok(Some(Err(e))) => {
                    return Err(LightpandaError::WebSocketError(e.to_string()).into());
                }
                Ok(None) => {
                    return Err(LightpandaError::ConnectionFailed("Connection closed".to_string()).into());
                }
                Err(_) => {
                    continue;
                }
            }
        }
        
        if !load_event_received {
            return Err(LightpandaError::NavigationTimeout(format!("Page load timeout for {}", url)).into());
        }
        
        tokio::time::sleep(Duration::from_millis(500)).await;
        
        Ok(())
    }

    async fn extract_content(
        &self,
        write: &mut futures::stream::SplitSink<tokio_tungstenite::WebSocketStream<tokio_tungstenite::MaybeTlsStream<tokio::net::TcpStream>>, Message>,
        read: &mut futures::stream::SplitStream<tokio_tungstenite::WebSocketStream<tokio_tungstenite::MaybeTlsStream<tokio::net::TcpStream>>>,
        timeout_duration: Duration,
    ) -> Result<ArticleContent> {
        let extraction_script = r#"
        (function() {
            const title = document.title || '';
            const url = window.location.href || '';
            
            let article = document.querySelector('article, main, .article-body, .post-content, .entry-content, [role="article"]');
            if (!article) {
                article = document.body;
            }
            
            const clone = article.cloneNode(true);
            
            const selectorsToRemove = [
                'nav', 'header', 'footer', 'aside', 'script', 'style',
                '.ad', '.advertisement', '[class*="ad"]', '[id*="ad"]',
                '.sidebar', '.social-share', '.comments', '.related-articles',
                'iframe', 'noscript'
            ];
            
            selectorsToRemove.forEach(selector => {
                clone.querySelectorAll(selector).forEach(el => el.remove());
            });
            
            const mainText = clone.innerText || clone.textContent || '';
            const htmlContent = clone.innerHTML || '';
            
            const images = Array.from(clone.querySelectorAll('img')).map(img => ({
                src: img.src || img.getAttribute('data-src') || '',
                alt: img.alt || '',
                caption: img.getAttribute('data-caption') || img.title || null
            }));
            
            let publishedDate = null;
            const timeEl = clone.querySelector('time[datetime], time[pubdate]') || 
                          document.querySelector('time[datetime], time[pubdate]');
            if (timeEl) {
                const dateStr = timeEl.getAttribute('datetime') || timeEl.getAttribute('pubdate');
                if (dateStr) {
                    publishedDate = dateStr;
                }
            }
            
            return {
                title: title,
                mainText: mainText.trim(),
                htmlContent: htmlContent,
                images: images,
                publishedDate: publishedDate,
                url: url
            };
        })()
        "#;
        
        let eval_cmd = json!({
            "jsonrpc": "2.0",
            "id": 4,
            "method": "Runtime.evaluate",
            "params": {
                "expression": extraction_script,
                "returnByValue": true
            }
        });
        
        write.send(Message::Text(eval_cmd.to_string())).await
            .map_err(|e| LightpandaError::WebSocketError(e.to_string()))?;
        
        loop {
            match timeout(timeout_duration, read.next()).await {
                Ok(Some(Ok(Message::Text(text)))) => {
                    let json: Value = serde_json::from_str(&text)
                        .map_err(|e| LightpandaError::JsonError(e.to_string()))?;
                    
                    if let Some(error) = json.get("error") {
                        return Err(LightpandaError::CDPError(format!("CDP error: {}", error)).into());
                    }
                    
                    let result = json.get("result")
                        .and_then(|r| r.get("value"))
                        .ok_or_else(|| LightpandaError::ContentExtractionFailed("No result value in CDP response".to_string()))?;
                
                let title = result.get("title")
                    .and_then(|v| v.as_str())
                    .unwrap_or("")
                    .to_string();
                
                let main_text = result.get("mainText")
                    .and_then(|v| v.as_str())
                    .unwrap_or("")
                    .to_string();
                
                let html_content = result.get("htmlContent")
                    .and_then(|v| v.as_str())
                    .unwrap_or("")
                    .to_string();
                
                let url = result.get("url")
                    .and_then(|v| v.as_str())
                    .unwrap_or("")
                    .to_string();
                
                let images: Vec<ImageInfo> = result.get("images")
                    .and_then(|v| v.as_array())
                    .map(|arr| {
                        arr.iter().filter_map(|img| {
                            Some(ImageInfo {
                                src: img.get("src")?.as_str()?.to_string(),
                                alt: img.get("alt")?.as_str()?.to_string(),
                                caption: img.get("caption").and_then(|v| v.as_str()).map(|s| s.to_string()),
                            })
                        }).collect()
                    })
                    .unwrap_or_default();
                
                let published_date = result.get("publishedDate")
                    .and_then(|v| v.as_str())
                    .and_then(|s| {
                        chrono::DateTime::parse_from_rfc3339(s)
                            .or_else(|_| chrono::DateTime::parse_from_rfc2822(s))
                            .ok()
                            .map(|dt| dt.with_timezone(&Utc))
                    });
                
                    return Ok(ArticleContent {
                        title,
                        main_text,
                        html_content,
                        images,
                        published_date,
                        url,
                        extraction_timestamp: Utc::now(),
                    });
                }
                Ok(Some(Ok(Message::Ping(_) | Message::Pong(_) | Message::Binary(_) | Message::Close(_) | Message::Frame(_)))) => {
                    continue;
                }
                Ok(Some(Err(e))) => return Err(LightpandaError::WebSocketError(e.to_string()).into()),
                Ok(None) => return Err(LightpandaError::ConnectionFailed("Connection closed".to_string()).into()),
                Err(_) => return Err(LightpandaError::ContentExtractionFailed("Content extraction timeout".to_string()).into()),
            }
        }
    }
}


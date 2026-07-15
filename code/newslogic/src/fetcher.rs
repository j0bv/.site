use std::path::{Path, PathBuf};
use anyhow::Context;
use crate::error::{AppResult, wrap_network_error, wrap_io_error};

pub async fn fetch_feed(url: &str) -> AppResult<String> {
    let client = reqwest::Client::builder()
        .user_agent("newslogic-rss-parser/0.1.0")
        .timeout(std::time::Duration::from_secs(30))
        .build()
        .context("Failed to create HTTP client")?;
    
    let response = client
        .get(url)
        .send()
        .await
        .map_err(|e| wrap_network_error(e).context(format!("Network error fetching: {}", url)))?;
    
    let status = response.status();
    let response = response
        .error_for_status()
        .map_err(|e| {
            wrap_network_error(e).context(format!(
                "HTTP {} error when fetching: {}",
                status.as_u16(),
                url
            ))
        })?;
    
    let content = response
        .text()
        .await
        .map_err(wrap_network_error)
        .context(format!("Failed to read response body from: {}", url))?;
    
    Ok(content)
}

pub fn save_xml(content: &str, output_path: &Path) -> AppResult<()> {
    if let Some(parent) = output_path.parent() {
        std::fs::create_dir_all(parent)
            .map_err(wrap_io_error)
            .context(format!("Failed to create output directory: {:?}", parent))?;
    }
    
    std::fs::write(output_path, content)
        .map_err(wrap_io_error)
        .context(format!("Failed to write XML file: {:?}", output_path))?;
    
    Ok(())
}

pub fn get_output_path(feed_id: &str, output_dir: &Path) -> PathBuf {
    let sanitized_id = sanitize_filename(feed_id);
    output_dir.join(format!("{}.xml", sanitized_id))
}

fn sanitize_filename(name: &str) -> String {
    name.chars()
        .map(|c| match c {
            '/' | '\\' | ':' | '*' | '?' | '"' | '<' | '>' | '|' => '-',
            _ => c,
        })
        .collect()
}


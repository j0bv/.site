use anyhow::Result;
use thiserror::Error;

pub type AppResult<T> = Result<T>;

pub fn wrap_network_error(err: reqwest::Error) -> anyhow::Error {
    anyhow::Error::from(err).context("Network request failed")
}

pub fn wrap_io_error(err: std::io::Error) -> anyhow::Error {
    anyhow::Error::from(err).context("File I/O operation failed")
}

pub fn wrap_yaml_error(err: serde_yaml::Error) -> anyhow::Error {
    anyhow::Error::from(err).context("YAML parsing failed")
}

#[derive(Debug, Error)]
pub enum LightpandaError {
    #[error("Connection failed: {0}")]
    ConnectionFailed(String),
    
    #[error("Navigation timeout: {0}")]
    NavigationTimeout(String),
    
    #[error("CDP error: {0}")]
    CDPError(String),
    
    #[error("Content extraction failed: {0}")]
    ContentExtractionFailed(String),
    
    #[error("WebSocket error: {0}")]
    WebSocketError(String),
    
    #[error("JSON parsing error: {0}")]
    JsonError(String),
}

#[derive(Debug, Error)]
pub enum Neo4jError {
    #[error("Neo4j connection failed: {0}")]
    ConnectionFailed(String),
    #[error("Neo4j query failed: {0}")]
    QueryFailed(String),
    #[error("Neo4j transaction error: {0}")]
    TransactionError(String),
}

#[derive(Debug, Error)]
pub enum BitNetError {
    #[error("BitNet model load failed: {0}")]
    ModelLoadFailed(String),
    #[error("BitNet inference failed: {0}")]
    InferenceFailed(String),
}

#[derive(Debug, Error)]
pub enum RedisError {
    #[error("Redis connection failed: {0}")]
    ConnectionFailed(String),
    #[error("Redis command failed: {0}")]
    CommandFailed(String),
}

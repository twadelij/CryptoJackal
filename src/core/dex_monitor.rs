//! DEX Monitoring Module
//! 
//! This module handles WebSocket connections to Uniswap subgraphs for real-time
//! monitoring of liquidity events and new pair creation.

use crate::error::{CryptoJackalError, Result};
use futures_util::{SinkExt, StreamExt};
use serde::{Deserialize, Serialize};
use std::time::Duration;
use tokio::time::{sleep, Instant};
use tokio_tungstenite::{connect_async, tungstenite::Message};
use tracing::{debug, error, info, warn};
use url::Url;

/// Configuration for DEX monitoring
#[derive(Debug, Clone)]
pub struct DexMonitorConfig {
    /// Uniswap subgraph WebSocket URL
    pub subgraph_url: String,
    /// Reconnection timeout in seconds
    pub reconnect_timeout: u64,
    /// Maximum reconnection attempts
    pub max_reconnect_attempts: u32,
    /// Heartbeat interval in seconds
    pub heartbeat_interval: u64,
}

impl Default for DexMonitorConfig {
    fn default() -> Self {
        Self {
            subgraph_url: "wss://api.thegraph.com/subgraphs/name/uniswap/uniswap-v2".to_string(),
            reconnect_timeout: 5,
            max_reconnect_attempts: 10,
            heartbeat_interval: 30,
        }
    }
}

/// Represents a liquidity event from Uniswap
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LiquidityEvent {
    /// Event type (mint, burn, swap)
    pub event_type: String,
    /// Transaction hash
    pub transaction_hash: String,
    /// Block number
    pub block_number: u64,
    /// Timestamp
    pub timestamp: u64,
    /// Pair address
    pub pair_address: String,
    /// Token0 address
    pub token0: String,
    /// Token1 address
    pub token1: String,
    /// Amount0
    pub amount0: String,
    /// Amount1
    pub amount1: String,
    /// Liquidity amount (for mint/burn events)
    pub liquidity: Option<String>,
}

/// GraphQL subscription query for new liquidity events
const LIQUIDITY_SUBSCRIPTION: &str = r#"
subscription {
  mints(first: 10, orderBy: timestamp, orderDirection: desc) {
    id
    transaction {
      id
      blockNumber
      timestamp
    }
    pair {
      id
      token0 {
        id
        symbol
      }
      token1 {
        id
        symbol
      }
    }
    amount0
    amount1
    liquidity
  }
  burns(first: 10, orderBy: timestamp, orderDirection: desc) {
    id
    transaction {
      id
      blockNumber
      timestamp
    }
    pair {
      id
      token0 {
        id
        symbol
      }
      token1 {
        id
        symbol
      }
    }
    amount0
    amount1
    liquidity
  }
}
"#;

/// WebSocket message types
#[derive(Debug, Serialize, Deserialize)]
#[serde(tag = "type")]
enum WsMessage {
    #[serde(rename = "connection_init")]
    ConnectionInit,
    #[serde(rename = "start")]
    Start { id: String, payload: WsPayload },
    #[serde(rename = "stop")]
    Stop { id: String },
    #[serde(rename = "connection_ack")]
    ConnectionAck,
    #[serde(rename = "data")]
    Data { id: String, payload: serde_json::Value },
    #[serde(rename = "error")]
    Error { id: String, payload: serde_json::Value },
    #[serde(rename = "complete")]
    Complete { id: String },
}

#[derive(Debug, Serialize, Deserialize)]
struct WsPayload {
    query: String,
    variables: Option<serde_json::Value>,
}

/// DEX Monitor for WebSocket connections
pub struct DexMonitor {
    config: DexMonitorConfig,
    reconnect_attempts: u32,
    last_heartbeat: Option<Instant>,
}

impl DexMonitor {
    /// Creates a new DEX monitor instance
    pub fn new(config: DexMonitorConfig) -> Self {
        Self {
            config,
            reconnect_attempts: 0,
            last_heartbeat: None,
        }
    }

    /// Starts monitoring DEX events
    pub async fn start_monitoring(&mut self) -> Result<()> {
        info!("Starting DEX monitoring...");
        
        loop {
            match self.connect_and_monitor().await {
                Ok(_) => {
                    info!("DEX monitoring completed successfully");
                    break;
                }
                Err(e) => {
                    error!("DEX monitoring error: {}", e);
                    
                    if self.reconnect_attempts >= self.config.max_reconnect_attempts {
                        return Err(CryptoJackalError::Network(format!(
                            "Max reconnection attempts ({}) exceeded",
                            self.config.max_reconnect_attempts
                        )));
                    }
                    
                    self.reconnect_attempts += 1;
                    let delay = self.calculate_backoff_delay();
                    
                    warn!(
                        "Reconnecting in {} seconds (attempt {}/{})",
                        delay.as_secs(),
                        self.reconnect_attempts,
                        self.config.max_reconnect_attempts
                    );
                    
                    sleep(delay).await;
                }
            }
        }
        
        Ok(())
    }

    /// Establishes WebSocket connection and monitors events
    async fn connect_and_monitor(&mut self) -> Result<()> {
        let url = Url::parse(&self.config.subgraph_url)
            .map_err(|e| CryptoJackalError::Config(format!("Invalid WebSocket URL: {}", e)))?;

        info!("Connecting to Uniswap subgraph: {}", url);

        let (ws_stream, _) = connect_async(url).await
            .map_err(|e| CryptoJackalError::Network(format!("WebSocket connection failed: {}", e)))?;

        let (mut ws_sender, mut ws_receiver) = ws_stream.split();

        // Initialize connection
        let init_msg = WsMessage::ConnectionInit;
        let init_json = serde_json::to_string(&init_msg)
            .map_err(|e| CryptoJackalError::Internal(format!("JSON serialization error: {}", e)))?;
        
        ws_sender.send(Message::Text(init_json)).await
            .map_err(|e| CryptoJackalError::Network(format!("Failed to send init message: {}", e)))?;

        // Wait for connection acknowledgment
        if let Some(msg) = ws_receiver.next().await {
            let msg = msg.map_err(|e| CryptoJackalError::Network(format!("WebSocket error: {}", e)))?;
            
            if let Message::Text(text) = msg {
                let ws_msg: WsMessage = serde_json::from_str(&text)
                    .map_err(|e| CryptoJackalError::Internal(format!("JSON parsing error: {}", e)))?;
                
                match ws_msg {
                    WsMessage::ConnectionAck => {
                        info!("WebSocket connection acknowledged");
                    }
                    _ => {
                        return Err(CryptoJackalError::Network(
                            "Expected connection acknowledgment".to_string()
                        ));
                    }
                }
            }
        }

        // Start subscription
        let start_msg = WsMessage::Start {
            id: "liquidity_events".to_string(),
            payload: WsPayload {
                query: LIQUIDITY_SUBSCRIPTION.to_string(),
                variables: None,
            },
        };
        
        let start_json = serde_json::to_string(&start_msg)
            .map_err(|e| CryptoJackalError::Internal(format!("JSON serialization error: {}", e)))?;
        
        ws_sender.send(Message::Text(start_json)).await
            .map_err(|e| CryptoJackalError::Network(format!("Failed to start subscription: {}", e)))?;

        info!("Subscription started, monitoring liquidity events...");
        self.last_heartbeat = Some(Instant::now());
        self.reconnect_attempts = 0; // Reset on successful connection

        // Monitor incoming messages
        while let Some(msg) = ws_receiver.next().await {
            let msg = msg.map_err(|e| CryptoJackalError::Network(format!("WebSocket error: {}", e)))?;
            
            match msg {
                Message::Text(text) => {
                    if let Err(e) = self.handle_message(&text).await {
                        error!("Error handling message: {}", e);
                    }
                }
                Message::Close(_) => {
                    warn!("WebSocket connection closed by server");
                    break;
                }
                Message::Ping(data) => {
                    debug!("Received ping, sending pong");
                    ws_sender.send(Message::Pong(data)).await
                        .map_err(|e| CryptoJackalError::Network(format!("Failed to send pong: {}", e)))?;
                }
                _ => {
                    debug!("Received other message type: {:?}", msg);
                }
            }

            // Check heartbeat
            if let Some(last_heartbeat) = self.last_heartbeat {
                if last_heartbeat.elapsed() > Duration::from_secs(self.config.heartbeat_interval * 2) {
                    warn!("Heartbeat timeout, reconnecting...");
                    break;
                }
            }
        }

        Ok(())
    }

    /// Handles incoming WebSocket messages
    async fn handle_message(&mut self, text: &str) -> Result<()> {
        debug!("Received message: {}", text);

        let ws_msg: WsMessage = serde_json::from_str(text)
            .map_err(|e| CryptoJackalError::Internal(format!("JSON parsing error: {}", e)))?;

        match ws_msg {
            WsMessage::Data { id, payload } => {
                if id == "liquidity_events" {
                    self.process_liquidity_data(payload).await?;
                }
                self.last_heartbeat = Some(Instant::now());
            }
            WsMessage::Error { id, payload } => {
                error!("Subscription error for {}: {:?}", id, payload);
                return Err(CryptoJackalError::ExternalApi(format!(
                    "Subscription error: {:?}",
                    payload
                )));
            }
            WsMessage::Complete { id } => {
                info!("Subscription {} completed", id);
            }
            _ => {
                debug!("Unhandled message type: {:?}", ws_msg);
            }
        }

        Ok(())
    }

    /// Processes liquidity data from the subgraph
    async fn process_liquidity_data(&self, data: serde_json::Value) -> Result<()> {
        debug!("Processing liquidity data: {:?}", data);

        // Parse mints
        if let Some(mints) = data.get("data").and_then(|d| d.get("mints")) {
            if let Some(mints_array) = mints.as_array() {
                for mint in mints_array {
                    let event = self.parse_liquidity_event(mint, "mint")?;
                    self.handle_liquidity_event(event).await?;
                }
            }
        }

        // Parse burns
        if let Some(burns) = data.get("data").and_then(|d| d.get("burns")) {
            if let Some(burns_array) = burns.as_array() {
                for burn in burns_array {
                    let event = self.parse_liquidity_event(burn, "burn")?;
                    self.handle_liquidity_event(event).await?;
                }
            }
        }

        Ok(())
    }

    /// Parses a liquidity event from JSON data
    fn parse_liquidity_event(&self, data: &serde_json::Value, event_type: &str) -> Result<LiquidityEvent> {
        let transaction = data.get("transaction")
            .ok_or_else(|| CryptoJackalError::MarketData("Missing transaction data".to_string()))?;
        
        let pair = data.get("pair")
            .ok_or_else(|| CryptoJackalError::MarketData("Missing pair data".to_string()))?;

        let event = LiquidityEvent {
            event_type: event_type.to_string(),
            transaction_hash: transaction.get("id")
                .and_then(|v| v.as_str())
                .unwrap_or_default()
                .to_string(),
            block_number: transaction.get("blockNumber")
                .and_then(|v| v.as_str())
                .and_then(|s| s.parse().ok())
                .unwrap_or_default(),
            timestamp: transaction.get("timestamp")
                .and_then(|v| v.as_str())
                .and_then(|s| s.parse().ok())
                .unwrap_or_default(),
            pair_address: pair.get("id")
                .and_then(|v| v.as_str())
                .unwrap_or_default()
                .to_string(),
            token0: pair.get("token0")
                .and_then(|t| t.get("id"))
                .and_then(|v| v.as_str())
                .unwrap_or_default()
                .to_string(),
            token1: pair.get("token1")
                .and_then(|t| t.get("id"))
                .and_then(|v| v.as_str())
                .unwrap_or_default()
                .to_string(),
            amount0: data.get("amount0")
                .and_then(|v| v.as_str())
                .unwrap_or_default()
                .to_string(),
            amount1: data.get("amount1")
                .and_then(|v| v.as_str())
                .unwrap_or_default()
                .to_string(),
            liquidity: data.get("liquidity")
                .and_then(|v| v.as_str())
                .map(|s| s.to_string()),
        };

        Ok(event)
    }

    /// Handles a processed liquidity event
    async fn handle_liquidity_event(&self, event: LiquidityEvent) -> Result<()> {
        info!(
            "Liquidity event: {} on pair {} (tokens: {} / {})",
            event.event_type,
            event.pair_address,
            event.token0,
            event.token1
        );

        // TODO: Implement opportunity detection logic
        // This is where we would analyze the event for trading opportunities
        
        Ok(())
    }

    /// Calculates exponential backoff delay for reconnection
    fn calculate_backoff_delay(&self) -> Duration {
        let base_delay = Duration::from_secs(self.config.reconnect_timeout);
        let multiplier = 2_u64.pow(self.reconnect_attempts.min(6)); // Cap at 2^6 = 64
        let max_delay = Duration::from_secs(300); // Max 5 minutes
        
        std::cmp::min(base_delay * multiplier, max_delay)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_dex_monitor_config_default() {
        let config = DexMonitorConfig::default();
        assert!(!config.subgraph_url.is_empty());
        assert_eq!(config.reconnect_timeout, 5);
        assert_eq!(config.max_reconnect_attempts, 10);
    }

    #[test]
    fn test_backoff_calculation() {
        let config = DexMonitorConfig::default();
        let mut monitor = DexMonitor::new(config);
        
        // Test exponential backoff
        monitor.reconnect_attempts = 0;
        assert_eq!(monitor.calculate_backoff_delay(), Duration::from_secs(5));
        
        monitor.reconnect_attempts = 1;
        assert_eq!(monitor.calculate_backoff_delay(), Duration::from_secs(10));
        
        monitor.reconnect_attempts = 2;
        assert_eq!(monitor.calculate_backoff_delay(), Duration::from_secs(20));
    }

    #[tokio::test]
    async fn test_liquidity_event_parsing() {
        let config = DexMonitorConfig::default();
        let monitor = DexMonitor::new(config);
        
        let test_data = serde_json::json!({
            "transaction": {
                "id": "0x123",
                "blockNumber": "12345",
                "timestamp": "1640995200"
            },
            "pair": {
                "id": "0xpair123",
                "token0": {"id": "0xtoken0"},
                "token1": {"id": "0xtoken1"}
            },
            "amount0": "1000",
            "amount1": "2000",
            "liquidity": "1500"
        });
        
        let event = monitor.parse_liquidity_event(&test_data, "mint").unwrap();
        assert_eq!(event.event_type, "mint");
        assert_eq!(event.transaction_hash, "0x123");
        assert_eq!(event.block_number, 12345);
    }
}

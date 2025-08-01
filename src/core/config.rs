use anyhow::Result;
use serde::{Deserialize, Serialize};
use std::env;

/// Comprehensive configuration for CryptoJackal bot
/// Integrates with order queue system and MetaMask wallet
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct Config {
    // Network configuration
    pub node_url: String,
    pub websocket_url: Option<String>,
    
    // Trading parameters
    pub scan_interval: u64,
    pub base_trade_amount: u128,
    pub max_slippage: f64,
    pub min_liquidity: f64,
    pub max_price_impact: f64,
    pub min_profit_threshold: f64,
    pub target_tokens: Vec<String>,
    
    // Order queue configuration
    pub max_concurrent_trades: Option<usize>,
    pub order_timeout_seconds: Option<u64>,
    pub max_retry_attempts: Option<u32>,
    
    // Gas configuration
    pub gas_limit: u64,
    pub max_gas_price_gwei: Option<u64>,
    pub priority_fee_gwei: Option<u64>,
    
    // Risk management
    pub max_position_size: Option<u128>,
    pub daily_loss_limit: Option<f64>,
    pub volatility_threshold: Option<f64>,
    
    // Price feed configuration
    pub price_feed_update_interval_ms: Option<u64>,
    pub price_feed_outlier_threshold: Option<f64>,
    pub price_feed_alert_threshold_percent: Option<f64>,
    pub price_feed_enabled_sources: Option<Vec<String>>,
}

impl Config {
    /// Loads configuration from environment variables with sensible defaults
    pub fn load() -> Result<Self> {
        let config = Self {
            // Network configuration
            node_url: get_env_var("ETH_RPC_URL")?,
            websocket_url: get_env_var_optional("WSS_URL"),
            
            // Trading parameters
            scan_interval: get_env_var_or_default("SCAN_INTERVAL", "1000")?.parse()?,
            base_trade_amount: get_env_var_or_default("BASE_TRADE_AMOUNT", "1000000000000000000")?.parse()?, // 1 ETH
            max_slippage: get_env_var_or_default("MAX_SLIPPAGE", "0.03")?.parse()?,
            min_liquidity: get_env_var_or_default("MIN_LIQUIDITY", "50000")?.parse()?,
            max_price_impact: get_env_var_or_default("MAX_PRICE_IMPACT", "0.05")?.parse()?,
            min_profit_threshold: get_env_var_or_default("MIN_PROFIT_THRESHOLD", "0.02")?.parse()?,
            target_tokens: get_env_var_or_default("TARGET_TOKENS", "")
                .split(',')
                .filter(|s| !s.trim().is_empty())
                .map(|s| s.trim().to_string())
                .collect(),
            
            // Order queue configuration
            max_concurrent_trades: get_env_var_optional("MAX_CONCURRENT_TRADES")
                .and_then(|v| v.parse().ok()),
            order_timeout_seconds: get_env_var_optional("ORDER_TIMEOUT_SECONDS")
                .and_then(|v| v.parse().ok()),
            max_retry_attempts: get_env_var_optional("MAX_RETRY_ATTEMPTS")
                .and_then(|v| v.parse().ok()),
            
            // Gas configuration
            gas_limit: get_env_var_or_default("GAS_LIMIT", "200000")?.parse()?,
            max_gas_price_gwei: get_env_var_optional("MAX_GAS_PRICE_GWEI")
                .and_then(|v| v.parse().ok()),
            priority_fee_gwei: get_env_var_optional("PRIORITY_FEE_GWEI")
                .and_then(|v| v.parse().ok()),
            
            // Risk management
            max_position_size: get_env_var_optional("MAX_POSITION_SIZE")
                .and_then(|v| v.parse().ok()),
            daily_loss_limit: get_env_var_optional("DAILY_LOSS_LIMIT")
                .and_then(|v| v.parse().ok()),
            volatility_threshold: get_env_var_optional("VOLATILITY_THRESHOLD")
                .and_then(|v| v.parse().ok()),
            
            // Price feed configuration
            price_feed_update_interval_ms: get_env_var_optional("PRICE_FEED_UPDATE_INTERVAL_MS")
                .and_then(|v| v.parse().ok()),
            price_feed_outlier_threshold: get_env_var_optional("PRICE_FEED_OUTLIER_THRESHOLD")
                .and_then(|v| v.parse().ok()),
            price_feed_alert_threshold_percent: get_env_var_optional("PRICE_FEED_ALERT_THRESHOLD_PERCENT")
                .and_then(|v| v.parse().ok()),
            price_feed_enabled_sources: get_env_var_optional("PRICE_FEED_ENABLED_SOURCES")
                .map(|s| s.split(',').map(|s| s.trim().to_string()).collect()),
        };
        
        // Validate configuration
        config.validate()?;
        Ok(config)
    }
    
    /// Validates configuration parameters
    fn validate(&self) -> Result<()> {
        if self.max_slippage < 0.0 || self.max_slippage > 1.0 {
            return Err(anyhow::anyhow!("max_slippage must be between 0.0 and 1.0"));
        }
        
        if self.max_price_impact < 0.0 || self.max_price_impact > 1.0 {
            return Err(anyhow::anyhow!("max_price_impact must be between 0.0 and 1.0"));
        }
        
        if self.min_profit_threshold < 0.0 {
            return Err(anyhow::anyhow!("min_profit_threshold must be positive"));
        }
        
        if self.base_trade_amount == 0 {
            return Err(anyhow::anyhow!("base_trade_amount must be greater than 0"));
        }
        
        if let Some(max_concurrent) = self.max_concurrent_trades {
            if max_concurrent == 0 || max_concurrent > 20 {
                return Err(anyhow::anyhow!("max_concurrent_trades must be between 1 and 20"));
            }
        }
        
        Ok(())
    }
    
    /// Creates a default configuration for testing
    pub fn default_for_testing() -> Self {
        Self {
            node_url: "http://localhost:8545".to_string(),
            websocket_url: Some("ws://localhost:8546".to_string()),
            scan_interval: 1000,
            base_trade_amount: 1000000000000000000, // 1 ETH
            max_slippage: 0.03,
            min_liquidity: 50000.0,
            max_price_impact: 0.05,
            min_profit_threshold: 0.02,
            target_tokens: vec![],
            max_concurrent_trades: Some(3),
            order_timeout_seconds: Some(30),
            max_retry_attempts: Some(2),
            gas_limit: 200000,
            max_gas_price_gwei: Some(100),
            priority_fee_gwei: Some(5),
            max_position_size: Some(5000000000000000000), // 5 ETH
            daily_loss_limit: Some(0.10), // 10%
            volatility_threshold: Some(0.50), // 50%
            price_feed_update_interval_ms: Some(5000), // 5 seconds
            price_feed_outlier_threshold: Some(3.0), // 3 standard deviations
            price_feed_alert_threshold_percent: Some(5.0), // 5%
            price_feed_enabled_sources: Some(vec!["coingecko".to_string(), "uniswap_v2".to_string(), "uniswap_v3".to_string()]),
        }
    }
}

impl Default for Config {
    fn default() -> Self {
        Self::default_for_testing()
    }
}

fn get_env_var(key: &str) -> Result<String> {
    env::var(key).map_err(|_| anyhow::anyhow!("Missing required environment variable: {}", key))
}

fn get_env_var_optional(key: &str) -> Option<String> {
    env::var(key).ok()
}

fn get_env_var_or_default(key: &str, default: &str) -> Result<String> {
    Ok(env::var(key).unwrap_or_else(|_| default.to_string()))
} 
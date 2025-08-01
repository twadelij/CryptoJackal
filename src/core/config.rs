use anyhow::{Result, Context};
use serde::{Deserialize, Serialize};
use std::env;
use std::fs;
use std::path::Path;
use std::str::FromStr;
use tracing::{info, warn, error};

/// Configuration for the CryptoJackal trading bot
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    // Network configuration
    pub node_url: String,
    pub chain_id: u64,
    
    // Wallet configuration
    pub private_key: String,
    pub wallet_address: String,
    
    // Trading parameters
    pub scan_interval: u64,
    pub gas_limit: u64,
    pub gas_price_gwei: u64,
    pub max_gas_price_gwei: u64,
    pub slippage_tolerance: f64,
    pub min_liquidity: f64,
    pub max_price_impact: f64,
    pub target_tokens: Vec<String>,
    
    // Risk management
    pub max_trade_size_eth: f64,
    pub min_profit_threshold: f64,
    pub max_daily_trades: u32,
    pub stop_loss_percentage: f64,
    pub take_profit_percentage: f64,
    
    // DEX configuration
    pub uniswap_v2_router: String,
    pub uniswap_v3_router: String,
    pub weth_address: String,
    
    // Monitoring and alerts
    pub enable_telegram_alerts: bool,
    pub telegram_bot_token: Option<String>,
    pub telegram_chat_id: Option<String>,
    
    // Performance settings
    pub enable_mev_protection: bool,
    pub flashbots_relay_url: Option<String>,
    pub max_priority_fee_gwei: u64,
    
    // Logging configuration
    pub log_level: String,
    pub log_file_path: Option<String>,
}

impl Config {
    /// Load configuration from file, environment variables, or create default
    pub fn load() -> Result<Self> {
        let config_path = env::var("CONFIG_PATH").unwrap_or_else(|_| "config.json".to_string());
        
        // Try to load from file first
        if Path::new(&config_path).exists() {
            match Self::load_from_file(&config_path) {
                Ok(mut config) => {
                    info!("Configuration loaded from {}", config_path);
                    config.override_from_env()?;
                    config.validate()?;
                    Ok(config)
                }
                Err(e) => {
                    warn!("Failed to load config from file: {}. Using environment variables.", e);
                    Self::load_from_env()
                }
            }
        } else {
            info!("No config file found. Loading from environment variables.");
            Self::load_from_env()
        }
    }
    
    /// Load configuration from JSON file
    pub fn load_from_file(path: &str) -> Result<Self> {
        let content = fs::read_to_string(path)
            .with_context(|| format!("Failed to read config file: {}", path))?;
        
        serde_json::from_str(&content)
            .with_context(|| format!("Failed to parse config file: {}", path))
    }
    
    /// Save configuration to JSON file
    pub fn save_to_file(&self, path: &str) -> Result<()> {
        let content = serde_json::to_string_pretty(self)
            .context("Failed to serialize config to JSON")?;
        
        fs::write(path, content)
            .with_context(|| format!("Failed to write config file: {}", path))?;
        
        info!("Configuration saved to {}", path);
        Ok(())
    }
    
    /// Load configuration from environment variables
    pub fn load_from_env() -> Result<Self> {
        let mut config = Self::default();
        config.override_from_env()?;
        config.validate()?;
        Ok(config)
    }
    
    /// Override configuration values from environment variables
    pub fn override_from_env(&mut self) -> Result<()> {
        // Network configuration
        if let Ok(val) = env::var("NODE_URL") {
            self.node_url = val;
        }
        if let Ok(val) = env::var("CHAIN_ID") {
            self.chain_id = val.parse().context("Invalid CHAIN_ID")?;
        }
        
        // Wallet configuration
        if let Ok(val) = env::var("PRIVATE_KEY") {
            self.private_key = val;
        }
        if let Ok(val) = env::var("WALLET_ADDRESS") {
            self.wallet_address = val;
        }
        
        // Trading parameters
        if let Ok(val) = env::var("SCAN_INTERVAL") {
            self.scan_interval = val.parse().context("Invalid SCAN_INTERVAL")?;
        }
        if let Ok(val) = env::var("GAS_LIMIT") {
            self.gas_limit = val.parse().context("Invalid GAS_LIMIT")?;
        }
        if let Ok(val) = env::var("GAS_PRICE_GWEI") {
            self.gas_price_gwei = val.parse().context("Invalid GAS_PRICE_GWEI")?;
        }
        if let Ok(val) = env::var("MAX_GAS_PRICE_GWEI") {
            self.max_gas_price_gwei = val.parse().context("Invalid MAX_GAS_PRICE_GWEI")?;
        }
        if let Ok(val) = env::var("SLIPPAGE_TOLERANCE") {
            self.slippage_tolerance = val.parse().context("Invalid SLIPPAGE_TOLERANCE")?;
        }
        if let Ok(val) = env::var("MIN_LIQUIDITY") {
            self.min_liquidity = val.parse().context("Invalid MIN_LIQUIDITY")?;
        }
        if let Ok(val) = env::var("MAX_PRICE_IMPACT") {
            self.max_price_impact = val.parse().context("Invalid MAX_PRICE_IMPACT")?;
        }
        if let Ok(val) = env::var("TARGET_TOKENS") {
            self.target_tokens = val.split(',').map(|s| s.trim().to_string()).collect();
        }
        
        // Risk management
        if let Ok(val) = env::var("MAX_TRADE_SIZE_ETH") {
            self.max_trade_size_eth = val.parse().context("Invalid MAX_TRADE_SIZE_ETH")?;
        }
        if let Ok(val) = env::var("MIN_PROFIT_THRESHOLD") {
            self.min_profit_threshold = val.parse().context("Invalid MIN_PROFIT_THRESHOLD")?;
        }
        if let Ok(val) = env::var("MAX_DAILY_TRADES") {
            self.max_daily_trades = val.parse().context("Invalid MAX_DAILY_TRADES")?;
        }
        if let Ok(val) = env::var("STOP_LOSS_PERCENTAGE") {
            self.stop_loss_percentage = val.parse().context("Invalid STOP_LOSS_PERCENTAGE")?;
        }
        if let Ok(val) = env::var("TAKE_PROFIT_PERCENTAGE") {
            self.take_profit_percentage = val.parse().context("Invalid TAKE_PROFIT_PERCENTAGE")?;
        }
        
        // DEX configuration
        if let Ok(val) = env::var("UNISWAP_V2_ROUTER") {
            self.uniswap_v2_router = val;
        }
        if let Ok(val) = env::var("UNISWAP_V3_ROUTER") {
            self.uniswap_v3_router = val;
        }
        if let Ok(val) = env::var("WETH_ADDRESS") {
            self.weth_address = val;
        }
        
        // Monitoring and alerts
        if let Ok(val) = env::var("ENABLE_TELEGRAM_ALERTS") {
            self.enable_telegram_alerts = val.parse().unwrap_or(false);
        }
        if let Ok(val) = env::var("TELEGRAM_BOT_TOKEN") {
            self.telegram_bot_token = Some(val);
        }
        if let Ok(val) = env::var("TELEGRAM_CHAT_ID") {
            self.telegram_chat_id = Some(val);
        }
        
        // Performance settings
        if let Ok(val) = env::var("ENABLE_MEV_PROTECTION") {
            self.enable_mev_protection = val.parse().unwrap_or(false);
        }
        if let Ok(val) = env::var("FLASHBOTS_RELAY_URL") {
            self.flashbots_relay_url = Some(val);
        }
        if let Ok(val) = env::var("MAX_PRIORITY_FEE_GWEI") {
            self.max_priority_fee_gwei = val.parse().context("Invalid MAX_PRIORITY_FEE_GWEI")?;
        }
        
        // Logging configuration
        if let Ok(val) = env::var("LOG_LEVEL") {
            self.log_level = val;
        }
        if let Ok(val) = env::var("LOG_FILE_PATH") {
            self.log_file_path = Some(val);
        }
        
        Ok(())
    }
    
    /// Validate configuration values
    pub fn validate(&self) -> Result<()> {
        // Validate network configuration
        if self.node_url.is_empty() {
            return Err(anyhow::anyhow!("NODE_URL is required"));
        }
        if self.chain_id == 0 {
            return Err(anyhow::anyhow!("CHAIN_ID must be greater than 0"));
        }
        
        // Validate wallet configuration
        if self.private_key.is_empty() {
            return Err(anyhow::anyhow!("PRIVATE_KEY is required"));
        }
        if self.wallet_address.is_empty() {
            return Err(anyhow::anyhow!("WALLET_ADDRESS is required"));
        }
        
        // Validate trading parameters
        if self.scan_interval == 0 {
            return Err(anyhow::anyhow!("SCAN_INTERVAL must be greater than 0"));
        }
        if self.gas_limit == 0 {
            return Err(anyhow::anyhow!("GAS_LIMIT must be greater than 0"));
        }
        if self.gas_price_gwei == 0 {
            return Err(anyhow::anyhow!("GAS_PRICE_GWEI must be greater than 0"));
        }
        if self.max_gas_price_gwei < self.gas_price_gwei {
            return Err(anyhow::anyhow!("MAX_GAS_PRICE_GWEI must be greater than or equal to GAS_PRICE_GWEI"));
        }
        if !(0.0..=100.0).contains(&self.slippage_tolerance) {
            return Err(anyhow::anyhow!("SLIPPAGE_TOLERANCE must be between 0 and 100"));
        }
        if self.min_liquidity <= 0.0 {
            return Err(anyhow::anyhow!("MIN_LIQUIDITY must be greater than 0"));
        }
        if !(0.0..=100.0).contains(&self.max_price_impact) {
            return Err(anyhow::anyhow!("MAX_PRICE_IMPACT must be between 0 and 100"));
        }
        if self.target_tokens.is_empty() {
            return Err(anyhow::anyhow!("At least one TARGET_TOKEN is required"));
        }
        
        // Validate risk management
        if self.max_trade_size_eth <= 0.0 {
            return Err(anyhow::anyhow!("MAX_TRADE_SIZE_ETH must be greater than 0"));
        }
        if self.min_profit_threshold < 0.0 {
            return Err(anyhow::anyhow!("MIN_PROFIT_THRESHOLD must be greater than or equal to 0"));
        }
        if self.max_daily_trades == 0 {
            return Err(anyhow::anyhow!("MAX_DAILY_TRADES must be greater than 0"));
        }
        if !(0.0..=100.0).contains(&self.stop_loss_percentage) {
            return Err(anyhow::anyhow!("STOP_LOSS_PERCENTAGE must be between 0 and 100"));
        }
        if !(0.0..=100.0).contains(&self.take_profit_percentage) {
            return Err(anyhow::anyhow!("TAKE_PROFIT_PERCENTAGE must be between 0 and 100"));
        }
        
        // Validate DEX configuration
        if self.uniswap_v2_router.is_empty() {
            return Err(anyhow::anyhow!("UNISWAP_V2_ROUTER is required"));
        }
        if self.uniswap_v3_router.is_empty() {
            return Err(anyhow::anyhow!("UNISWAP_V3_ROUTER is required"));
        }
        if self.weth_address.is_empty() {
            return Err(anyhow::anyhow!("WETH_ADDRESS is required"));
        }
        
        // Validate monitoring configuration
        if self.enable_telegram_alerts {
            if self.telegram_bot_token.is_none() {
                return Err(anyhow::anyhow!("TELEGRAM_BOT_TOKEN is required when ENABLE_TELEGRAM_ALERTS is true"));
            }
            if self.telegram_chat_id.is_none() {
                return Err(anyhow::anyhow!("TELEGRAM_CHAT_ID is required when ENABLE_TELEGRAM_ALERTS is true"));
            }
        }
        
        // Validate performance settings
        if self.enable_mev_protection && self.flashbots_relay_url.is_none() {
            return Err(anyhow::anyhow!("FLASHBOTS_RELAY_URL is required when ENABLE_MEV_PROTECTION is true"));
        }
        if self.max_priority_fee_gwei > self.max_gas_price_gwei {
            return Err(anyhow::anyhow!("MAX_PRIORITY_FEE_GWEI cannot exceed MAX_GAS_PRICE_GWEI"));
        }
        
        // Validate logging configuration
        if !["trace", "debug", "info", "warn", "error"].contains(&self.log_level.as_str()) {
            return Err(anyhow::anyhow!("LOG_LEVEL must be one of: trace, debug, info, warn, error"));
        }
        
        info!("Configuration validation passed");
        Ok(())
    }
    
    /// Create a default configuration
    pub fn create_default_config(path: &str) -> Result<()> {
        let config = Self::default();
        config.save_to_file(path)
    }
}

impl Default for Config {
    fn default() -> Self {
        Self {
            // Network configuration
            node_url: "https://eth-mainnet.alchemyapi.io/v2/your-api-key".to_string(),
            chain_id: 1,
            
            // Wallet configuration
            private_key: "".to_string(),
            wallet_address: "".to_string(),
            
            // Trading parameters
            scan_interval: 1000, // 1 second
            gas_limit: 300000,
            gas_price_gwei: 20,
            max_gas_price_gwei: 100,
            slippage_tolerance: 2.0, // 2%
            min_liquidity: 10.0, // 10 ETH
            max_price_impact: 5.0, // 5%
            target_tokens: vec!["0xA0b86a33E6441b8c4C8C1C1B8c4C8C1C1B8c4C8C1".to_string()],
            
            // Risk management
            max_trade_size_eth: 0.1,
            min_profit_threshold: 0.5, // 0.5%
            max_daily_trades: 10,
            stop_loss_percentage: 10.0, // 10%
            take_profit_percentage: 20.0, // 20%
            
            // DEX configuration
            uniswap_v2_router: "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D".to_string(),
            uniswap_v3_router: "0xE592427A0AEce92De3Edee1F18E0157C05861564".to_string(),
            weth_address: "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2".to_string(),
            
            // Monitoring and alerts
            enable_telegram_alerts: false,
            telegram_bot_token: None,
            telegram_chat_id: None,
            
            // Performance settings
            enable_mev_protection: false,
            flashbots_relay_url: None,
            max_priority_fee_gwei: 2,
            
            // Logging configuration
            log_level: "info".to_string(),
            log_file_path: None,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::NamedTempFile;
    
    #[test]
    fn test_config_validation() {
        let mut config = Config::default();
        
        // Should fail validation due to empty required fields
        assert!(config.validate().is_err());
        
        // Set required fields
        config.private_key = "0x1234567890abcdef".to_string();
        config.wallet_address = "0x1234567890abcdef".to_string();
        
        // Should pass validation
        assert!(config.validate().is_ok());
    }
    
    #[test]
    fn test_config_serialization() {
        let config = Config::default();
        let json = serde_json::to_string(&config).unwrap();
        let deserialized: Config = serde_json::from_str(&json).unwrap();
        
        assert_eq!(config.scan_interval, deserialized.scan_interval);
        assert_eq!(config.gas_limit, deserialized.gas_limit);
    }
    
    #[test]
    fn test_config_file_operations() {
        let temp_file = NamedTempFile::new().unwrap();
        let config = Config::default();
        
        // Test save
        assert!(config.save_to_file(temp_file.path().to_str().unwrap()).is_ok());
        
        // Test load
        let loaded_config = Config::load_from_file(temp_file.path().to_str().unwrap()).unwrap();
        assert_eq!(config.scan_interval, loaded_config.scan_interval);
    }
    
    #[test]
    fn test_invalid_values() {
        let mut config = Config::default();
        config.private_key = "0x1234567890abcdef".to_string();
        config.wallet_address = "0x1234567890abcdef".to_string();
        
        // Test invalid slippage tolerance
        config.slippage_tolerance = 150.0;
        assert!(config.validate().is_err());
        
        // Test invalid gas price
        config.slippage_tolerance = 2.0;
        config.max_gas_price_gwei = 10;
        config.gas_price_gwei = 20;
        assert!(config.validate().is_err());
    }
} 
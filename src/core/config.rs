use anyhow::Result;
use serde::Deserialize;
use std::env;

#[derive(Debug, Clone, Deserialize)]
pub struct Config {
    // Node Configuration
    pub node_url: String,
    pub chain_id: u64,
    pub network_name: String,
    
    // Trading Parameters
    pub scan_interval: u64,
    pub gas_limit: u64,
    pub slippage_tolerance: f64,
    pub min_liquidity: f64,
    pub max_price_impact: f64,
    pub trade_amount: u128,
    pub target_tokens: Vec<String>,
    
    // Web API Configuration
    pub api_host: String,
    pub api_port: u16,
    pub api_base_url: String,
    pub cors_origins: Vec<String>,
    
    // Database Configuration
    pub database_url: String,
    pub redis_url: String,
    
    // Security Configuration
    pub jwt_secret: String,
    pub session_timeout: u64,
    pub max_login_attempts: u32,
    
    // Logging Configuration
    pub log_level: String,
    pub log_format: String,
    pub log_file_enabled: bool,
    pub log_file_path: String,
    
    // Monitoring & Metrics
    pub metrics_enabled: bool,
    pub metrics_port: u16,
    pub health_check_enabled: bool,
    pub health_check_port: u16,
    
    // Paper Trading Configuration
    pub paper_trading_mode: bool,
    pub paper_trading_balance: f64,
    pub paper_trading_data_source: String,
    
    // Token Discovery Configuration
    pub dexscreener_api_url: String,
    pub coingecko_api_url: String,
    pub discovery_scan_interval: u64,
    pub max_new_tokens_per_scan: usize,
    pub token_security_check_enabled: bool,
    
    // Development Configuration
    pub environment: String,
    pub debug_mode: bool,
    pub hot_reload: bool,
    pub enable_profiling: bool,
    
    // External Service API Keys
    pub coingecko_api_key: Option<String>,
    pub dexscreener_api_key: Option<String>,
    pub telegram_bot_token: Option<String>,
    pub discord_webhook_url: Option<String>,
}

impl Config {
    pub fn load() -> Result<Self> {
        Ok(Self {
            // Node Configuration
            node_url: get_env_var("NODE_URL")?,
            chain_id: get_env_var("CHAIN_ID").unwrap_or_else(|_| "1".to_string()).parse()?,
            network_name: get_env_var("NETWORK_NAME").unwrap_or_else(|_| "ethereum".to_string()),
            
            // Trading Parameters
            scan_interval: get_env_var("SCAN_INTERVAL").unwrap_or_else(|_| "1000".to_string()).parse()?,
            gas_limit: get_env_var("GAS_LIMIT").unwrap_or_else(|_| "300000".to_string()).parse()?,
            slippage_tolerance: get_env_var("SLIPPAGE_TOLERANCE").unwrap_or_else(|_| "0.005".to_string()).parse()?,
            min_liquidity: get_env_var("MIN_LIQUIDITY").unwrap_or_else(|_| "10.0".to_string()).parse()?,
            max_price_impact: get_env_var("MAX_PRICE_IMPACT").unwrap_or_else(|_| "0.02".to_string()).parse()?,
            trade_amount: get_env_var("TRADE_AMOUNT").unwrap_or_else(|_| "100000000000000000".to_string()).parse()?,
            target_tokens: get_env_var("TARGET_TOKENS")
                .unwrap_or_else(|_| "".to_string())
                .split(',')
                .filter(|s| !s.trim().is_empty())
                .map(|s| s.trim().to_string())
                .collect(),
            
            // Web API Configuration
            api_host: get_env_var("API_HOST").unwrap_or_else(|_| "0.0.0.0".to_string()),
            api_port: get_env_var("API_PORT").unwrap_or_else(|_| "8080".to_string()).parse()?,
            api_base_url: get_env_var("API_BASE_URL").unwrap_or_else(|_| "http://localhost:8080".to_string()),
            cors_origins: get_env_var("CORS_ORIGINS")
                .unwrap_or_else(|_| "http://localhost:3000".to_string())
                .split(',')
                .map(|s| s.trim().to_string())
                .collect(),
            
            // Database Configuration
            database_url: get_env_var("DATABASE_URL").unwrap_or_else(|_| "sqlite://cryptojackal.db".to_string()),
            redis_url: get_env_var("REDIS_URL").unwrap_or_else(|_| "redis://localhost:6379".to_string()),
            
            // Security Configuration
            jwt_secret: get_env_var("JWT_SECRET").unwrap_or_else(|_| "default-secret-change-in-production".to_string()),
            session_timeout: get_env_var("SESSION_TIMEOUT").unwrap_or_else(|_| "3600".to_string()).parse()?,
            max_login_attempts: get_env_var("MAX_LOGIN_ATTEMPTS").unwrap_or_else(|_| "5".to_string()).parse()?,
            
            // Logging Configuration
            log_level: get_env_var("LOG_LEVEL").unwrap_or_else(|_| "info".to_string()),
            log_format: get_env_var("LOG_FORMAT").unwrap_or_else(|_| "json".to_string()),
            log_file_enabled: get_env_var("LOG_FILE_ENABLED").unwrap_or_else(|_| "true".to_string()).parse()?,
            log_file_path: get_env_var("LOG_FILE_PATH").unwrap_or_else(|_| "/var/log/cryptojackal/app.log".to_string()),
            
            // Monitoring & Metrics
            metrics_enabled: get_env_var("METRICS_ENABLED").unwrap_or_else(|_| "true".to_string()).parse()?,
            metrics_port: get_env_var("METRICS_PORT").unwrap_or_else(|_| "9090".to_string()).parse()?,
            health_check_enabled: get_env_var("HEALTH_CHECK_ENABLED").unwrap_or_else(|_| "true".to_string()).parse()?,
            health_check_port: get_env_var("HEALTH_CHECK_PORT").unwrap_or_else(|_| "8081".to_string()).parse()?,
            
            // Paper Trading Configuration
            paper_trading_mode: get_env_var("PAPER_TRADING_MODE").unwrap_or_else(|_| "false".to_string()).parse()?,
            paper_trading_balance: get_env_var("PAPER_TRADING_BALANCE").unwrap_or_else(|_| "10.0".to_string()).parse()?,
            paper_trading_data_source: get_env_var("PAPER_TRADING_DATA_SOURCE").unwrap_or_else(|_| "historical".to_string()),
            
            // Token Discovery Configuration
            dexscreener_api_url: get_env_var("DEXSCREENER_API_URL")
                .unwrap_or_else(|_| "https://api.dexscreener.com/latest/dex".to_string()),
            coingecko_api_url: get_env_var("COINGECKO_API_URL")
                .unwrap_or_else(|_| "https://api.coingecko.com/api/v3".to_string()),
            discovery_scan_interval: get_env_var("DISCOVERY_SCAN_INTERVAL").unwrap_or_else(|_| "30000".to_string()).parse()?,
            max_new_tokens_per_scan: get_env_var("MAX_NEW_TOKENS_PER_SCAN").unwrap_or_else(|_| "10".to_string()).parse()?,
            token_security_check_enabled: get_env_var("TOKEN_SECURITY_CHECK_ENABLED")
                .unwrap_or_else(|_| "true".to_string())
                .parse()?,
            
            // Development Configuration
            environment: get_env_var("ENVIRONMENT").unwrap_or_else(|_| "development".to_string()),
            debug_mode: get_env_var("DEBUG_MODE").unwrap_or_else(|_| "false".to_string()).parse()?,
            hot_reload: get_env_var("HOT_RELOAD").unwrap_or_else(|_| "true".to_string()).parse()?,
            enable_profiling: get_env_var("ENABLE_PROFILING").unwrap_or_else(|_| "false".to_string()).parse()?,
            
            // External Service API Keys
            coingecko_api_key: get_env_var_opt("COINGECKO_API_KEY"),
            dexscreener_api_key: get_env_var_opt("DEXSCREENER_API_KEY"),
            telegram_bot_token: get_env_var_opt("TELEGRAM_BOT_TOKEN"),
            discord_webhook_url: get_env_var_opt("DISCORD_WEBHOOK_URL"),
        })
    }
    
    /// Check if running in development mode
    pub fn is_development(&self) -> bool {
        self.environment == "development"
    }
    
    /// Check if paper trading mode is enabled
    pub fn is_paper_trading(&self) -> bool {
        self.paper_trading_mode
    }
    
    /// Get API server bind address
    pub fn api_bind_address(&self) -> String {
        format!("{}:{}", self.api_host, self.api_port)
    }
    
    /// Get metrics server bind address
    pub fn metrics_bind_address(&self) -> String {
        format!("0.0.0.0:{}", self.metrics_port)
    }
    
    /// Get health check server bind address
    pub fn health_check_bind_address(&self) -> String {
        format!("0.0.0.0:{}", self.health_check_port)
    }
}

pub fn get_env_var(key: &str) -> Result<String> {
    env::var(key).map_err(|_| anyhow::anyhow!("Missing required environment variable: {}", key))
}

pub fn get_env_var_opt(key: &str) -> Option<String> {
    env::var(key).ok()
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_config_defaults() {
        // Test that config can be loaded with minimal required vars
        std::env::set_var("NODE_URL", "https://example.com");
        std::env::set_var("TARGET_TOKENS", "0x123,0x456");
        std::env::set_var("TRADE_AMOUNT", "1000000000000000000");
        
        let config = Config::load().unwrap();
        
        assert_eq!(config.chain_id, 1);
        assert_eq!(config.scan_interval, 1000);
        assert_eq!(config.environment, "development");
        assert_eq!(config.target_tokens.len(), 2);
    }
    
    #[test]
    fn test_paper_trading_detection() {
        std::env::set_var("NODE_URL", "https://example.com");
        std::env::set_var("TARGET_TOKENS", "0x123");
        std::env::set_var("TRADE_AMOUNT", "1000000000000000000");
        std::env::set_var("PAPER_TRADING_MODE", "true");
        
        let config = Config::load().unwrap();
        assert!(config.is_paper_trading());
    }
} 
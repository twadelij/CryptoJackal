use anyhow::Result;
use serde::Deserialize;
use std::env;

#[derive(Debug, Clone, Deserialize)]
pub struct Config {
    pub node_url: String,
    pub scan_interval: u64,
    pub gas_limit: u64,
    pub slippage_tolerance: f64,
    pub min_liquidity: f64,
    pub max_price_impact: f64,
    pub trade_amount: u128,
    pub target_tokens: Vec<String>,
}

impl Config {
    pub fn load() -> Result<Self> {
        Ok(Self {
            node_url: get_env_var("NODE_URL")?,
            scan_interval: get_env_var("SCAN_INTERVAL")?.parse()?,
            gas_limit: get_env_var("GAS_LIMIT")?.parse()?,
            slippage_tolerance: get_env_var("SLIPPAGE_TOLERANCE")?.parse()?,
            min_liquidity: get_env_var("MIN_LIQUIDITY")?.parse()?,
            max_price_impact: get_env_var("MAX_PRICE_IMPACT")?.parse()?,
            trade_amount: get_env_var("TRADE_AMOUNT")?.parse()?,
            target_tokens: get_env_var("TARGET_TOKENS")?
                .split(',')
                .map(|s| s.trim().to_string())
                .collect(),
        })
    }
}

pub fn get_env_var(key: &str) -> Result<String> {
    env::var(key).map_err(|_| anyhow::anyhow!("Missing environment variable: {}", key))
} 
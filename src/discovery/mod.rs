use anyhow::Result;
use async_trait::async_trait;
use reqwest::Client;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::time::{Duration, SystemTime, UNIX_EPOCH};
use tracing::{info, warn, error};

use crate::core::config::Config;

pub mod dexscreener;
pub mod coingecko;
pub mod security;

use dexscreener::DexScreenerClient;
use coingecko::CoinGeckoClient;
use security::SecurityAnalyzer;

/// Discovered token information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DiscoveredToken {
    pub address: String,
    pub symbol: String,
    pub name: String,
    pub price: f64,
    pub market_cap: Option<f64>,
    pub liquidity: f64,
    pub volume_24h: Option<f64>,
    pub price_change_24h: Option<f64>,
    pub security_score: f64,
    pub discovered_at: SystemTime,
    pub tags: Vec<String>,
    pub dex_info: DexInfo,
}

/// DEX-specific information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DexInfo {
    pub pair_address: String,
    pub base_token: TokenInfo,
    pub quote_token: TokenInfo,
    pub dex_id: String,
    pub chain_id: String,
}

/// Basic token information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenInfo {
    pub address: String,
    pub name: String,
    pub symbol: String,
}

/// Token discovery trait
#[async_trait]
pub trait TokenDiscovery: Send + Sync {
    async fn discover_new_tokens(&self, limit: usize) -> Result<Vec<DiscoveredToken>>;
    async fn get_trending_tokens(&self, time_window: &str) -> Result<Vec<DiscoveredToken>>;
    async fn analyze_token(&self, address: &str) -> Result<DiscoveredToken>;
}

/// Main token discovery service
pub struct TokenDiscoveryService {
    dexscreener: DexScreenerClient,
    coingecko: CoinGeckoClient,
    security: SecurityAnalyzer,
    config: Config,
    http_client: Client,
}

impl TokenDiscoveryService {
    pub fn new(config: Config) -> Self {
        Self {
            dexscreener: DexScreenerClient::new(config.dexscreener_api_url.clone()),
            coingecko: CoinGeckoClient::new(
                config.coingecko_api_url.clone(),
                config.coingecko_api_key.clone()
            ),
            security: SecurityAnalyzer::new(),
            config,
            http_client: Client::builder()
                .timeout(Duration::from_secs(30))
                .build()
                .expect("Failed to create HTTP client"),
        }
    }

    /// Discover new tokens with security analysis
    pub async fn discover_new_tokens(&self) -> Result<Vec<DiscoveredToken>> {
        info!("Starting token discovery scan");

        // Get new pairs from DexScreener
        let mut discovered = Vec::new();
        
        // Fetch from multiple sources
        let dexscreener_tokens = self.dexscreener.get_new_pairs(50).await?;
        info!("Fetched {} tokens from DexScreener", dexscreener_tokens.len());

        // Process each token
        for token in dexscreener_tokens {
            // Skip if we already have too many
            if discovered.len() >= self.config.max_new_tokens_per_scan {
                break;
            }

            // Perform security analysis
            let security_score = self.security.analyze_token(&token).await?;

            // Skip low-security tokens
            if security_score < 0.3 && self.config.token_security_check_enabled {
                warn!("Skipping token {} due to low security score: {}", token.symbol, security_score);
                continue;
            }

            // Enhance with CoinGecko data if available
            let enhanced_token = self.enhance_with_coingecko_data(token, security_score).await?;
            
            discovered.push(enhanced_token);
        }

        info!("Discovered {} new tokens with security analysis", discovered.len());
        Ok(discovered)
    }

    /// Get trending tokens
    pub async fn get_trending_tokens(&self, time_window: &str) -> Result<Vec<DiscoveredToken>> {
        info!("Fetching trending tokens for time window: {}", time_window);

        let mut trending = Vec::new();

        // Get trending from DexScreener
        let dexscreener_trending = self.dexscreener.get_trending_pairs(time_window).await?;
        
        for token in dexscreener_trending {
            let security_score = self.security.analyze_token(&token).await?;
            let enhanced_token = self.enhance_with_coingecko_data(token, security_score).await?;
            trending.push(enhanced_token);
        }

        // Sort by volume and security score
        trending.sort_by(|a, b| {
            let score_a = a.volume_24h.unwrap_or(0.0) * a.security_score;
            let score_b = b.volume_24h.unwrap_or(0.0) * b.security_score;
            score_b.partial_cmp(&score_a).unwrap_or(std::cmp::Ordering::Equal)
        });

        info!("Found {} trending tokens", trending.len());
        Ok(trending)
    }

    /// Analyze a specific token
    pub async fn analyze_token(&self, address: &str) -> Result<DiscoveredToken> {
        info!("Analyzing token: {}", address);

        // Get token info from DexScreener
        let token = self.dexscreener.get_token_info(address).await?;
        
        // Security analysis
        let security_score = self.security.analyze_token(&token).await?;
        
        // Enhance with additional data
        let enhanced_token = self.enhance_with_coingecko_data(token, security_score).await?;
        
        Ok(enhanced_token)
    }

    /// Enhance token data with CoinGecko information
    async fn enhance_with_coingecko_data(
        &self, 
        mut token: DiscoveredToken, 
        security_score: f64
    ) -> Result<DiscoveredToken> {
        // Try to get CoinGecko data
        if let Ok(coingecko_data) = self.coingecko.get_token_info(&token.address).await {
            token.market_cap = coingecko_data.market_cap;
            token.volume_24h = coingecko_data.volume_24h;
            token.price_change_24h = coingecko_data.price_change_24h;
            
            // Add CoinGecko tags
            token.tags.extend(coingecko_data.categories);
        }

        token.security_score = security_score;
        token.discovered_at = SystemTime::now();
        
        Ok(token)
    }

    /// Continuous discovery task
    pub async fn start_discovery_task(&self) -> Result<()> {
        let interval = Duration::from_millis(self.config.discovery_scan_interval);
        
        loop {
            match self.discover_new_tokens().await {
                Ok(tokens) => {
                    if !tokens.is_empty() {
                        info!("Discovery cycle completed: {} new tokens found", tokens.len());
                        // TODO: Store tokens in database or send notifications
                    }
                }
                Err(e) => {
                    error!("Discovery cycle failed: {}", e);
                }
            }
            
            tokio::time::sleep(interval).await;
        }
    }
}

#[async_trait]
impl TokenDiscovery for TokenDiscoveryService {
    async fn discover_new_tokens(&self, limit: usize) -> Result<Vec<DiscoveredToken>> {
        let mut tokens = self.discover_new_tokens().await?;
        tokens.truncate(limit);
        Ok(tokens)
    }

    async fn get_trending_tokens(&self, time_window: &str) -> Result<Vec<DiscoveredToken>> {
        self.get_trending_tokens(time_window).await
    }

    async fn analyze_token(&self, address: &str) -> Result<DiscoveredToken> {
        self.analyze_token(address).await
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_token_discovery() {
        // TODO: Add integration tests
    }
}

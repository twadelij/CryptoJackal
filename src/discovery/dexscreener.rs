use anyhow::Result;
use reqwest::Client;
use serde::{Deserialize, Serialize};
use std::time::SystemTime;
use tracing::{info, warn};

use super::{DiscoveredToken, DexInfo, TokenInfo};

/// DexScreener API client
pub struct DexScreenerClient {
    base_url: String,
    client: Client,
}

impl DexScreenerClient {
    pub fn new(base_url: String) -> Self {
        Self {
            base_url,
            client: Client::builder()
                .timeout(std::time::Duration::from_secs(30))
                .build()
                .expect("Failed to create HTTP client"),
        }
    }

    /// Get new pairs from DexScreener
    pub async fn get_new_pairs(&self, limit: usize) -> Result<Vec<DiscoveredToken>> {
        let url = format!("{}/dex/pairs/new", self.base_url);
        
        let response: DexScreenerResponse = self.client
            .get(&url)
            .send()
            .await?
            .json()
            .await?;

        let mut tokens = Vec::new();
        
        for pair in response.pairs.into_iter().take(limit) {
            if let Some(token) = self.convert_pair_to_token(pair) {
                tokens.push(token);
            }
        }

        Ok(tokens)
    }

    /// Get trending pairs
    pub async fn get_trending_pairs(&self, time_window: &str) -> Result<Vec<DiscoveredToken>> {
        let url = format!("{}/dex/pairs/trending/{}", self.base_url, time_window);
        
        let response: DexScreenerResponse = self.client
            .get(&url)
            .send()
            .await?
            .json()
            .await?;

        let mut tokens = Vec::new();
        
        for pair in response.pairs {
            if let Some(token) = self.convert_pair_to_token(pair) {
                tokens.push(token);
            }
        }

        Ok(tokens)
    }

    /// Get specific token info
    pub async fn get_token_info(&self, address: &str) -> Result<DiscoveredToken> {
        let url = format!("{}/dex/tokens/{}", self.base_url, address);
        
        let response: DexScreenerResponse = self.client
            .get(&url)
            .send()
            .await?
            .json()
            .await?;

        if let Some(pair) = response.pairs.into_iter().next() {
            if let Some(token) = self.convert_pair_to_token(pair) {
                return Ok(token);
            }
        }

        Err(anyhow::anyhow!("Token not found: {}", address))
    }

    /// Convert DexScreener pair to our DiscoveredToken format
    fn convert_pair_to_token(&self, pair: DexScreenerPair) -> Option<DiscoveredToken> {
        // Use the base token (not quote token like USDT/WETH)
        let base_token = &pair.base_token;
        
        // Skip stablecoins and wrapped tokens
        if self.is_stablecoin_or_wrapped(&base_token.symbol) {
            return None;
        }

        // Calculate liquidity from pair liquidity
        let liquidity = pair.liquidity?.usd.unwrap_or(0.0);

        // Skip very low liquidity pairs
        if liquidity < 1000.0 {
            warn!("Skipping low liquidity pair: {} (${})", base_token.symbol, liquidity);
            return None;
        }

        let dex_info = DexInfo {
            pair_address: pair.pair_address,
            base_token: TokenInfo {
                address: base_token.address.clone(),
                name: base_token.name.clone(),
                symbol: base_token.symbol.clone(),
            },
            quote_token: TokenInfo {
                address: pair.quote_token.address.clone(),
                name: pair.quote_token.name.clone(),
                symbol: pair.quote_token.symbol.clone(),
            },
            dex_id: pair.dex_id,
            chain_id: pair.chain_id,
        };

        Some(DiscoveredToken {
            address: base_token.address.clone(),
            symbol: base_token.symbol.clone(),
            name: base_token.name.clone(),
            price: pair.price_usd.unwrap_or(0.0),
            market_cap: None, // Will be filled by CoinGecko
            liquidity,
            volume_24h: pair.volume.as_ref().and_then(|v| v.h24),
            price_change_24h: pair.price_change.as_ref().and_then(|v| v.h24),
            security_score: 0.5, // Will be calculated by security analyzer
            discovered_at: SystemTime::now(),
            tags: vec![],
            dex_info,
        })
    }

    /// Check if token is a stablecoin or wrapped token
    fn is_stablecoin_or_wrapped(&self, symbol: &str) -> bool {
        let symbol = symbol.to_uppercase();
        matches!(
            symbol.as_str(),
            "USDT" | "USDC" | "DAI" | "BUSD" | "TUSD" | "USDP" | "FDUSD" |
            "WETH" | "WBTC" | "WBNB" | "WMATIC" | "WAVAX" | "WFTM"
        )
    }
}

/// DexScreener API response structures
#[derive(Debug, Deserialize)]
struct DexScreenerResponse {
    pairs: Vec<DexScreenerPair>,
}

#[derive(Debug, Deserialize)]
struct DexScreenerPair {
    chain_id: String,
    dex_id: String,
    pair_address: String,
    base_token: DexToken,
    quote_token: DexToken,
    price_usd: Option<f64>,
    liquidity: Option<Liquidity>,
    volume: Option<Volume>,
    price_change: Option<PriceChange>,
}

#[derive(Debug, Deserialize)]
struct DexToken {
    address: String,
    name: String,
    symbol: String,
}

#[derive(Debug, Deserialize)]
struct Liquidity {
    usd: Option<f64>,
    base: Option<f64>,
    quote: Option<f64>,
}

#[derive(Debug, Deserialize)]
struct Volume {
    h24: Option<f64>,
    h6: Option<f64>,
    h1: Option<f64>,
    m5: Option<f64>,
}

#[derive(Debug, Deserialize)]
struct PriceChange {
    h24: Option<f64>,
    h6: Option<f64>,
    h1: Option<f64>,
    m5: Option<f64>,
}

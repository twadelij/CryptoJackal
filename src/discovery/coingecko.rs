use anyhow::Result;
use reqwest::Client;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use tracing::{info, warn};

/// CoinGecko API client
pub struct CoinGeckoClient {
    base_url: String,
    api_key: Option<String>,
    client: Client,
}

impl CoinGeckoClient {
    pub fn new(base_url: String, api_key: Option<String>) -> Self {
        let mut headers = reqwest::header::HeaderMap::new();
        
        // Add API key if provided
        if let Some(api_key) = &api_key {
            headers.insert(
                "x-cg-demo-api-key",
                api_key.parse().expect("Invalid API key")
            );
        }

        Self {
            base_url,
            api_key,
            client: Client::builder()
                .default_headers(headers)
                .timeout(std::time::Duration::from_secs(30))
                .build()
                .expect("Failed to create HTTP client"),
        }
    }

    /// Get token information by contract address
    pub async fn get_token_info(&self, contract_address: &str) -> Result<CoinGeckoTokenInfo> {
        let url = format!(
            "{}/coins/ethereum/contract/{}",
            self.base_url,
            contract_address
        );

        let response: CoinGeckoContractResponse = self.client
            .get(&url)
            .send()
            .await?
            .json()
            .await?;

        Ok(response.into())
    }

    /// Get trending tokens
    pub async fn get_trending(&self) -> Result<Vec<CoinGeckoTrendingItem>> {
        let url = format!("{}/search/trending", self.base_url);

        let response: CoinGeckoTrendingResponse = self.client
            .get(&url)
            .send()
            .await?
            .json()
            .await?;

        Ok(response.coins)
    }

    /// Get market data for multiple coins
    pub async fn get_market_data(&self, coin_ids: &[String]) -> Result<Vec<CoinGeckoMarketData>> {
        let ids = coin_ids.join(",");
        let url = format!(
            "{}/coins/markets?vs_currency=usd&ids={}&order=market_cap_desc&per_page=250&page=1",
            self.base_url,
            ids
        );

        let response: Vec<CoinGeckoMarketData> = self.client
            .get(&url)
            .send()
            .await?
            .json()
            .await?;

        Ok(response)
    }
}

/// CoinGecko contract response
#[derive(Debug, Deserialize)]
struct CoinGeckoContractResponse {
    id: String,
    name: String,
    symbol: String,
    asset_platform_id: Option<String>,
    platforms: Option<HashMap<String, String>>,
    detail_platforms: Option<HashMap<String, CoinGeckoPlatformDetail>>,
    market_data: Option<CoinGeckoMarketData>,
    categories: Option<Vec<String>>,
}

#[derive(Debug, Deserialize)]
struct CoinGeckoPlatformDetail {
    decimal_place: Option<u32>,
    contract_address: String,
}

/// CoinGecko token info (simplified)
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CoinGeckoTokenInfo {
    pub id: String,
    pub name: String,
    pub symbol: String,
    pub market_cap: Option<f64>,
    pub volume_24h: Option<f64>,
    pub price_change_24h: Option<f64>,
    pub categories: Vec<String>,
}

impl From<CoinGeckoContractResponse> for CoinGeckoTokenInfo {
    fn from(response: CoinGeckoContractResponse) -> Self {
        let market_data = response.market_data;

        Self {
            id: response.id,
            name: response.name,
            symbol: response.symbol.to_uppercase(),
            market_cap: market_data.as_ref().and_then(|m| m.market_cap),
            volume_24h: market_data.as_ref().and_then(|m| m.total_volume.usd),
            price_change_24h: market_data.as_ref().and_then(|m| m.price_change_percentage_24h),
            categories: response.categories.unwrap_or_default(),
        }
    }
}

/// CoinGecko market data
#[derive(Debug, Deserialize)]
struct CoinGeckoMarketData {
    current_price: Option<HashMap<String, f64>>,
    market_cap: Option<f64>,
    total_volume: TotalVolume,
    price_change_percentage_24h: Option<f64>,
}

#[derive(Debug, Deserialize)]
struct TotalVolume {
    usd: Option<f64>,
}

/// CoinGecko trending response
#[derive(Debug, Deserialize)]
struct CoinGeckoTrendingResponse {
    coins: Vec<CoinGeckoTrendingItem>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct CoinGeckoTrendingItem {
    pub item: CoinGeckoTrendingCoin,
}

#[derive(Debug, Clone, Deserialize)]
pub struct CoinGeckoTrendingCoin {
    pub id: String,
    pub name: String,
    pub symbol: String,
    pub market_cap_rank: Option<u32>,
    pub thumb: String,
    pub small: String,
    pub large: String,
    pub slug: String,
    pub price_btc: Option<f64>,
    pub score: Option<u32>,
    pub data: Option<CoinGeckoTrendingData>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct CoinGeckoTrendingData {
    pub price: Option<String>,
    pub price_change_percentage_24h: Option<CoinGeckoPriceChange>,
    pub market_cap: Option<String>,
    pub market_cap_change_percentage_24h: Option<CoinGeckoPriceChange>,
    pub sparkline: Option<String>,
    pub content: Option<String>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct CoinGeckoPriceChange {
    pub usd: Option<f64>,
}

/// CoinGecko market data for list view
#[derive(Debug, Clone, Deserialize)]
pub struct CoinGeckoMarketData {
    pub id: String,
    pub symbol: String,
    pub name: String,
    pub current_price: Option<f64>,
    pub market_cap: Option<f64>,
    pub market_cap_rank: Option<u32>,
    pub total_volume: Option<f64>,
    pub price_change_percentage_24h: Option<f64>,
    pub circulating_supply: Option<f64>,
    pub total_supply: Option<f64>,
    pub max_supply: Option<f64>,
    pub ath: Option<f64>,
    pub ath_change_percentage: Option<f64>,
    pub ath_date: Option<String>,
    pub atl: Option<f64>,
    pub atl_change_percentage: Option<f64>,
    pub atl_date: Option<String>,
    pub last_updated: Option<String>,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_coingecko_client() {
        let client = CoinGeckoClient::new(
            "https://api.coingecko.com/api/v3".to_string(),
            None,
        );

        // Test with a known token (WETH)
        match client.get_token_info("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2").await {
            Ok(token_info) => {
                assert_eq!(token_info.symbol, "WETH");
                println!("Successfully fetched WETH info: {}", token_info.name);
            }
            Err(e) => {
                println!("Failed to fetch token info: {}", e);
            }
        }
    }
}

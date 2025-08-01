use anyhow::Result;
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::RwLock;
use tokio::time::{interval, sleep};
use tracing::{debug, error, info, warn};

use super::config::Config;
use super::types::Token;

/// Price data from a single source
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PriceData {
    pub token_address: String,
    pub symbol: String,
    pub price_usd: f64,
    pub volume_24h: f64,
    pub market_cap: Option<f64>,
    pub price_change_24h: f64,
    pub source: PriceSource,
    pub timestamp: DateTime<Utc>,
    pub confidence: f64, // 0.0 to 1.0
}

/// Price feed sources
#[derive(Debug, Clone, Copy, PartialEq, Serialize, Deserialize)]
pub enum PriceSource {
    CoinGecko,
    CoinMarketCap,
    UniswapV2,
    UniswapV3,
    DexScreener,
    Custom,
}

/// Aggregated price data from multiple sources
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AggregatedPrice {
    pub token_address: String,
    pub symbol: String,
    pub price_usd: f64,
    pub volume_24h: f64,
    pub market_cap: Option<f64>,
    pub price_change_24h: f64,
    pub sources_count: usize,
    pub confidence: f64,
    pub volatility: f64,
    pub outlier_detected: bool,
    pub aggregated_at: DateTime<Utc>,
}

/// Price change alert
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PriceAlert {
    pub token_address: String,
    pub symbol: String,
    pub alert_type: AlertType,
    pub old_price: f64,
    pub new_price: f64,
    pub change_percentage: f64,
    pub threshold: f64,
    pub timestamp: DateTime<Utc>,
}

/// Alert types
#[derive(Debug, Clone, Copy, PartialEq, Serialize, Deserialize)]
pub enum AlertType {
    PriceSpike,      // Sudden price increase
    PriceDrop,       // Sudden price decrease
    VolumeSpike,     // Unusual volume increase
    VolatilityAlert, // High volatility detected
    OutlierDetected, // Price outlier from aggregation
}

/// Price feed configuration
#[derive(Debug, Clone)]
pub struct PriceFeedConfig {
    pub update_interval_ms: u64,
    pub aggregation_timeout_ms: u64,
    pub outlier_threshold: f64,
    pub alert_thresholds: AlertThresholds,
    pub enabled_sources: Vec<PriceSource>,
    pub max_retry_attempts: u32,
    pub retry_delay_ms: u64,
}

/// Alert thresholds
#[derive(Debug, Clone)]
pub struct AlertThresholds {
    pub price_change_percent: f64,
    pub volume_spike_percent: f64,
    pub volatility_threshold: f64,
    pub outlier_threshold: f64,
}

impl Default for PriceFeedConfig {
    fn default() -> Self {
        Self {
            update_interval_ms: 5000, // 5 seconds
            aggregation_timeout_ms: 2000, // 2 seconds
            outlier_threshold: 3.0, // 3 standard deviations
            alert_thresholds: AlertThresholds {
                price_change_percent: 5.0, // 5%
                volume_spike_percent: 200.0, // 200%
                volatility_threshold: 0.5, // 50%
                outlier_threshold: 2.0, // 2 standard deviations
            },
            enabled_sources: vec![
                PriceSource::CoinGecko,
                PriceSource::UniswapV2,
                PriceSource::UniswapV3,
            ],
            max_retry_attempts: 3,
            retry_delay_ms: 1000,
        }
    }
}

/// Price feed monitoring system
pub struct PriceFeedMonitor {
    config: PriceFeedConfig,
    price_cache: Arc<RwLock<HashMap<String, AggregatedPrice>>>,
    price_history: Arc<RwLock<HashMap<String, Vec<PriceData>>>>,
    alerts: Arc<RwLock<Vec<PriceAlert>>>,
    metrics: Arc<RwLock<PriceFeedMetrics>>,
    shutdown_signal: Arc<RwLock<bool>>,
}

/// Price feed performance metrics
#[derive(Debug, Clone, Default)]
pub struct PriceFeedMetrics {
    pub total_updates: u64,
    pub successful_updates: u64,
    pub failed_updates: u64,
    pub alerts_generated: u64,
    pub average_update_time_ms: f64,
    pub cache_hit_rate: f64,
    pub last_update_time: Option<DateTime<Utc>>,
}

impl PriceFeedMonitor {
    /// Creates a new price feed monitor
    pub fn new(config: PriceFeedConfig) -> Self {
        info!("Initializing price feed monitor with config: {:?}", config);
        
        Self {
            config,
            price_cache: Arc::new(RwLock::new(HashMap::new())),
            price_history: Arc::new(RwLock::new(HashMap::new())),
            alerts: Arc::new(RwLock::new(Vec::new())),
            metrics: Arc::new(RwLock::new(PriceFeedMetrics::default())),
            shutdown_signal: Arc::new(RwLock::new(false)),
        }
    }

    /// Starts the price feed monitoring
    pub async fn start_monitoring(&self, target_tokens: Vec<String>) -> Result<()> {
        info!("Starting price feed monitoring for {} tokens", target_tokens.len());
        
        let mut interval_timer = interval(Duration::from_millis(self.config.update_interval_ms));
        
        loop {
            // Check shutdown signal
            if *self.shutdown_signal.read().await {
                info!("Shutdown signal received, stopping price feed monitoring");
                break;
            }
            
            let update_start = Instant::now();
            
            // Update prices for all target tokens
            for token_address in &target_tokens {
                if let Err(e) = self.update_token_price(token_address).await {
                    error!("Failed to update price for {}: {}", token_address, e);
                    self.increment_failed_updates().await;
                } else {
                    self.increment_successful_updates().await;
                }
            }
            
            // Update metrics
            let update_duration = update_start.elapsed();
            self.update_metrics(update_duration).await;
            
            // Wait for next interval
            interval_timer.tick().await;
        }
        
        Ok(())
    }

    /// Updates price for a specific token
    async fn update_token_price(&self, token_address: &str) -> Result<()> {
        let mut price_data_vec = Vec::new();
        
        // Fetch prices from all enabled sources
        for source in &self.config.enabled_sources {
            if let Ok(price_data) = self.fetch_price_from_source(token_address, *source).await {
                price_data_vec.push(price_data);
            }
        }
        
        if price_data_vec.is_empty() {
            return Err(anyhow::anyhow!("No price data available for token {}", token_address));
        }
        
        // Aggregate prices
        let aggregated_price = self.aggregate_prices(&price_data_vec).await?;
        
        // Check for alerts
        self.check_price_alerts(&aggregated_price).await;
        
        // Update cache and history
        self.update_price_cache(&aggregated_price).await;
        self.update_price_history(&price_data_vec).await;
        
        debug!("Updated price for {}: ${:.6}", aggregated_price.symbol, aggregated_price.price_usd);
        
        Ok(())
    }

    /// Fetches price from a specific source
    async fn fetch_price_from_source(&self, token_address: &str, source: PriceSource) -> Result<PriceData> {
        let mut attempts = 0;
        
        while attempts < self.config.max_retry_attempts {
            match self.fetch_price_with_retry(token_address, source).await {
                Ok(price_data) => return Ok(price_data),
                Err(e) => {
                    attempts += 1;
                    warn!("Attempt {} failed for {} from {:?}: {}", attempts, token_address, source, e);
                    
                    if attempts < self.config.max_retry_attempts {
                        sleep(Duration::from_millis(self.config.retry_delay_ms)).await;
                    }
                }
            }
        }
        
        Err(anyhow::anyhow!("Failed to fetch price from {:?} after {} attempts", source, self.config.max_retry_attempts))
    }

    /// Fetches price with retry logic
    async fn fetch_price_with_retry(&self, token_address: &str, source: PriceSource) -> Result<PriceData> {
        match source {
            PriceSource::CoinGecko => self.fetch_coingecko_price(token_address).await,
            PriceSource::UniswapV2 => self.fetch_uniswap_v2_price(token_address).await,
            PriceSource::UniswapV3 => self.fetch_uniswap_v3_price(token_address).await,
            PriceSource::DexScreener => self.fetch_dexscreener_price(token_address).await,
            _ => Err(anyhow::anyhow!("Unsupported price source: {:?}", source)),
        }
    }

    /// Fetches price from CoinGecko API
    async fn fetch_coingecko_price(&self, token_address: &str) -> Result<PriceData> {
        // This would be implemented with actual CoinGecko API calls
        // For now, return mock data
        Ok(PriceData {
            token_address: token_address.to_string(),
            symbol: "TOKEN".to_string(),
            price_usd: 1.0 + (rand::random::<f64>() * 0.1), // Random price around $1
            volume_24h: 1000000.0,
            market_cap: Some(10000000.0),
            price_change_24h: 0.05,
            source: PriceSource::CoinGecko,
            timestamp: Utc::now(),
            confidence: 0.9,
        })
    }

    /// Fetches price from Uniswap V2
    async fn fetch_uniswap_v2_price(&self, token_address: &str) -> Result<PriceData> {
        // This would be implemented with actual Uniswap V2 contract calls
        // For now, return mock data
        Ok(PriceData {
            token_address: token_address.to_string(),
            symbol: "TOKEN".to_string(),
            price_usd: 1.0 + (rand::random::<f64>() * 0.05), // Random price around $1
            volume_24h: 500000.0,
            market_cap: None,
            price_change_24h: 0.03,
            source: PriceSource::UniswapV2,
            timestamp: Utc::now(),
            confidence: 0.8,
        })
    }

    /// Fetches price from Uniswap V3
    async fn fetch_uniswap_v3_price(&self, token_address: &str) -> Result<PriceData> {
        // This would be implemented with actual Uniswap V3 contract calls
        // For now, return mock data
        Ok(PriceData {
            token_address: token_address.to_string(),
            symbol: "TOKEN".to_string(),
            price_usd: 1.0 + (rand::random::<f64>() * 0.03), // Random price around $1
            volume_24h: 750000.0,
            market_cap: None,
            price_change_24h: 0.02,
            source: PriceSource::UniswapV3,
            timestamp: Utc::now(),
            confidence: 0.85,
        })
    }

    /// Fetches price from DexScreener
    async fn fetch_dexscreener_price(&self, token_address: &str) -> Result<PriceData> {
        // This would be implemented with actual DexScreener API calls
        // For now, return mock data
        Ok(PriceData {
            token_address: token_address.to_string(),
            symbol: "TOKEN".to_string(),
            price_usd: 1.0 + (rand::random::<f64>() * 0.08), // Random price around $1
            volume_24h: 800000.0,
            market_cap: Some(8000000.0),
            price_change_24h: 0.04,
            source: PriceSource::DexScreener,
            timestamp: Utc::now(),
            confidence: 0.75,
        })
    }

    /// Aggregates prices from multiple sources
    async fn aggregate_prices(&self, price_data_vec: &[PriceData]) -> Result<AggregatedPrice> {
        if price_data_vec.is_empty() {
            return Err(anyhow::anyhow!("No price data to aggregate"));
        }
        
        let token_address = price_data_vec[0].token_address.clone();
        let symbol = price_data_vec[0].symbol.clone();
        
        // Calculate weighted average price based on confidence
        let total_weight: f64 = price_data_vec.iter().map(|p| p.confidence).sum();
        let weighted_price: f64 = price_data_vec.iter()
            .map(|p| p.price_usd * p.confidence)
            .sum::<f64>() / total_weight;
        
        // Calculate average volume and market cap
        let avg_volume: f64 = price_data_vec.iter()
            .map(|p| p.volume_24h)
            .sum::<f64>() / price_data_vec.len() as f64;
        
        let market_cap = if price_data_vec.iter().any(|p| p.market_cap.is_some()) {
            let sum: f64 = price_data_vec.iter()
                .filter_map(|p| p.market_cap)
                .sum();
            let count = price_data_vec.iter().filter(|p| p.market_cap.is_some()).count();
            Some(sum / count as f64)
        } else {
            None
        };
        
        // Calculate average price change
        let avg_price_change: f64 = price_data_vec.iter()
            .map(|p| p.price_change_24h)
            .sum::<f64>() / price_data_vec.len() as f64;
        
        // Calculate volatility
        let prices: Vec<f64> = price_data_vec.iter().map(|p| p.price_usd).collect();
        let volatility = self.calculate_volatility(&prices);
        
        // Detect outliers
        let outlier_detected = self.detect_outliers(&prices);
        
        // Calculate overall confidence
        let confidence = total_weight / price_data_vec.len() as f64;
        
        Ok(AggregatedPrice {
            token_address,
            symbol,
            price_usd: weighted_price,
            volume_24h: avg_volume,
            market_cap,
            price_change_24h: avg_price_change,
            sources_count: price_data_vec.len(),
            confidence,
            volatility,
            outlier_detected,
            aggregated_at: Utc::now(),
        })
    }

    /// Calculates volatility from price data
    fn calculate_volatility(&self, prices: &[f64]) -> f64 {
        if prices.len() < 2 {
            return 0.0;
        }
        
        let mean = prices.iter().sum::<f64>() / prices.len() as f64;
        let variance = prices.iter()
            .map(|p| (p - mean).powi(2))
            .sum::<f64>() / (prices.len() - 1) as f64;
        
        variance.sqrt() / mean // Coefficient of variation
    }

    /// Detects outliers using z-score method
    fn detect_outliers(&self, prices: &[f64]) -> bool {
        if prices.len() < 3 {
            return false;
        }
        
        let mean = prices.iter().sum::<f64>() / prices.len() as f64;
        let variance = prices.iter()
            .map(|p| (p - mean).powi(2))
            .sum::<f64>() / (prices.len() - 1) as f64;
        let std_dev = variance.sqrt();
        
        if std_dev == 0.0 {
            return false;
        }
        
        // Check if any price is more than threshold standard deviations from mean
        prices.iter().any(|&price| {
            let z_score = (price - mean).abs() / std_dev;
            z_score > self.config.outlier_threshold
        })
    }

    /// Checks for price alerts
    async fn check_price_alerts(&self, aggregated_price: &AggregatedPrice) {
        let mut alerts = Vec::new();
        
        // Get previous price from cache
        if let Some(previous_price) = self.get_cached_price(&aggregated_price.token_address).await {
            let price_change = ((aggregated_price.price_usd - previous_price.price_usd) / previous_price.price_usd).abs() * 100.0;
            
            // Check for price spike/drop
            if price_change > self.config.alert_thresholds.price_change_percent {
                let alert_type = if aggregated_price.price_usd > previous_price.price_usd {
                    AlertType::PriceSpike
                } else {
                    AlertType::PriceDrop
                };
                
                alerts.push(PriceAlert {
                    token_address: aggregated_price.token_address.clone(),
                    symbol: aggregated_price.symbol.clone(),
                    alert_type,
                    old_price: previous_price.price_usd,
                    new_price: aggregated_price.price_usd,
                    change_percentage: price_change,
                    threshold: self.config.alert_thresholds.price_change_percent,
                    timestamp: Utc::now(),
                });
            }
            
            // Check for volume spike
            let volume_change = ((aggregated_price.volume_24h - previous_price.volume_24h) / previous_price.volume_24h).abs() * 100.0;
            if volume_change > self.config.alert_thresholds.volume_spike_percent {
                alerts.push(PriceAlert {
                    token_address: aggregated_price.token_address.clone(),
                    symbol: aggregated_price.symbol.clone(),
                    alert_type: AlertType::VolumeSpike,
                    old_price: previous_price.volume_24h,
                    new_price: aggregated_price.volume_24h,
                    change_percentage: volume_change,
                    threshold: self.config.alert_thresholds.volume_spike_percent,
                    timestamp: Utc::now(),
                });
            }
        }
        
        // Check for high volatility
        if aggregated_price.volatility > self.config.alert_thresholds.volatility_threshold {
            alerts.push(PriceAlert {
                token_address: aggregated_price.token_address.clone(),
                symbol: aggregated_price.symbol.clone(),
                alert_type: AlertType::VolatilityAlert,
                old_price: 0.0,
                new_price: aggregated_price.volatility,
                change_percentage: aggregated_price.volatility * 100.0,
                threshold: self.config.alert_thresholds.volatility_threshold * 100.0,
                timestamp: Utc::now(),
            });
        }
        
        // Check for outlier detection
        if aggregated_price.outlier_detected {
            alerts.push(PriceAlert {
                token_address: aggregated_price.token_address.clone(),
                symbol: aggregated_price.symbol.clone(),
                alert_type: AlertType::OutlierDetected,
                old_price: 0.0,
                new_price: aggregated_price.price_usd,
                change_percentage: 0.0,
                threshold: self.config.outlier_threshold,
                timestamp: Utc::now(),
            });
        }
        
        // Store alerts
        if !alerts.is_empty() {
            let mut alerts_store = self.alerts.write().await;
            alerts_store.extend(alerts.clone());
            
            // Keep only last 100 alerts
            if alerts_store.len() > 100 {
                alerts_store.drain(0..alerts_store.len() - 100);
            }
            
            // Log alerts
            for alert in alerts {
                info!("🚨 Price alert: {} - {:?} - {:.2}% change", 
                      alert.symbol, alert.alert_type, alert.change_percentage);
            }
            
            self.increment_alerts_generated().await;
        }
    }

    /// Updates price cache
    async fn update_price_cache(&self, aggregated_price: &AggregatedPrice) {
        let mut cache = self.price_cache.write().await;
        cache.insert(aggregated_price.token_address.clone(), aggregated_price.clone());
    }

    /// Updates price history
    async fn update_price_history(&self, price_data_vec: &[PriceData]) {
        for price_data in price_data_vec {
            let mut history = self.price_history.write().await;
            let entry = history.entry(price_data.token_address.clone()).or_insert_with(Vec::new);
            entry.push(price_data.clone());
            
            // Keep only last 1000 price points per token
            if entry.len() > 1000 {
                entry.drain(0..entry.len() - 1000);
            }
        }
    }

    /// Gets cached price for a token
    async fn get_cached_price(&self, token_address: &str) -> Option<AggregatedPrice> {
        self.price_cache.read().await.get(token_address).cloned()
    }

    /// Gets current price for a token
    pub async fn get_current_price(&self, token_address: &str) -> Option<AggregatedPrice> {
        self.get_cached_price(token_address).await
    }

    /// Gets price history for a token
    pub async fn get_price_history(&self, token_address: &str) -> Vec<PriceData> {
        self.price_history.read().await
            .get(token_address)
            .cloned()
            .unwrap_or_default()
    }

    /// Gets recent alerts
    pub async fn get_recent_alerts(&self, limit: usize) -> Vec<PriceAlert> {
        let alerts = self.alerts.read().await;
        alerts.iter()
            .rev()
            .take(limit)
            .cloned()
            .collect()
    }

    /// Gets price feed metrics
    pub async fn get_metrics(&self) -> PriceFeedMetrics {
        self.metrics.read().await.clone()
    }

    /// Increments successful updates
    async fn increment_successful_updates(&self) {
        let mut metrics = self.metrics.write().await;
        metrics.successful_updates += 1;
        metrics.total_updates += 1;
        metrics.last_update_time = Some(Utc::now());
    }

    /// Increments failed updates
    async fn increment_failed_updates(&self) {
        let mut metrics = self.metrics.write().await;
        metrics.failed_updates += 1;
        metrics.total_updates += 1;
    }

    /// Increments alerts generated
    async fn increment_alerts_generated(&self) {
        let mut metrics = self.metrics.write().await;
        metrics.alerts_generated += 1;
    }

    /// Updates metrics
    async fn update_metrics(&self, update_duration: Duration) {
        let mut metrics = self.metrics.write().await;
        
        // Update average update time
        let total_time = metrics.average_update_time_ms * (metrics.total_updates - 1) as f64;
        let new_total_time = total_time + update_duration.as_millis() as f64;
        metrics.average_update_time_ms = new_total_time / metrics.total_updates as f64;
        
        // Update cache hit rate (simplified)
        metrics.cache_hit_rate = metrics.successful_updates as f64 / metrics.total_updates as f64;
    }

    /// Shuts down the price feed monitor
    pub async fn shutdown(&self) {
        info!("Shutting down price feed monitor");
        *self.shutdown_signal.write().await = true;
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_price_feed_monitor_creation() {
        let config = PriceFeedConfig::default();
        let monitor = PriceFeedMonitor::new(config);
        assert!(monitor.get_metrics().await.total_updates == 0);
    }

    #[test]
    fn test_volatility_calculation() {
        let prices = vec![1.0, 1.1, 0.9, 1.05, 0.95];
        let monitor = PriceFeedMonitor::new(PriceFeedConfig::default());
        let volatility = monitor.calculate_volatility(&prices);
        assert!(volatility > 0.0);
    }

    #[test]
    fn test_outlier_detection() {
        let prices = vec![1.0, 1.1, 1.05, 5.0, 1.02]; // 5.0 is an outlier
        let monitor = PriceFeedMonitor::new(PriceFeedConfig::default());
        let is_outlier = monitor.detect_outliers(&prices);
        assert!(is_outlier);
    }
} 
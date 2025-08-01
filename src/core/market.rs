use crate::error::{CryptoJackalError, ErrorContext, Result};
use ethers::prelude::*;
use std::collections::HashMap;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tracing::{debug, info, warn, error};

use super::config::Config;
use super::types::{TokenInfo, TradeParams, TradeResult};

/// Liquidity pair information
#[derive(Debug, Clone)]
pub struct LiquidityPair {
    pub address: Address,
    pub token0: Address,
    pub token1: Address,
    pub token0_symbol: String,
    pub token1_symbol: String,
    pub reserve0: u128,
    pub reserve1: u128,
    pub total_supply: u128,
    pub fee: u32, // 0.3% = 3000, 0.05% = 500, etc.
    pub version: UniswapVersion,
    pub created_at: u64,
    pub liquidity_usd: f64,
    pub volume_24h: f64,
    pub price_change_24h: f64,
}

/// Uniswap version
#[derive(Debug, Clone, Copy, PartialEq)]
pub enum UniswapVersion {
    V2,
    V3,
}

/// Trading opportunity with detailed analysis
#[derive(Debug, Clone)]
pub struct Opportunity {
    pub token: TokenInfo,
    pub pair: LiquidityPair,
    pub price: f64,
    pub liquidity: f64,
    pub volatility: f64,
    pub price_impact: f64,
    pub volume_24h: f64,
    pub market_cap: f64,
    pub confidence_score: f64,
    pub risk_level: RiskLevel,
    pub detected_at: Instant,
    pub opportunity_type: OpportunityType,
}

/// Risk assessment levels
#[derive(Debug, Clone, Copy, PartialEq)]
pub enum RiskLevel {
    Low,
    Medium,
    High,
    Extreme,
}

/// Types of trading opportunities
#[derive(Debug, Clone, Copy, PartialEq)]
pub enum OpportunityType {
    NewToken,           // Newly listed token
    LiquidityAddition,  // Large liquidity addition
    PriceSpike,         // Significant price increase
    Arbitrage,          // Price difference between DEXs
    Momentum,           // Strong upward momentum
}

/// Market scanner configuration
#[derive(Debug, Clone)]
pub struct ScannerConfig {
    pub scan_interval_ms: u64,
    pub max_pairs_per_scan: usize,
    pub min_liquidity_usd: f64,
    pub max_price_impact: f64,
    pub min_volume_24h: f64,
    pub enable_v2: bool,
    pub enable_v3: bool,
    pub gas_price_threshold: u64,
    pub max_slippage: f64,
}

impl Default for ScannerConfig {
    fn default() -> Self {
        Self {
            scan_interval_ms: 1000,
            max_pairs_per_scan: 1000,
            min_liquidity_usd: 10000.0,
            max_price_impact: 5.0,
            min_volume_24h: 1000.0,
            enable_v2: true,
            enable_v3: true,
            gas_price_threshold: 50,
            max_slippage: 2.0,
        }
    }
}

/// Market monitoring and opportunity detection
pub struct Market {
    provider: Arc<Provider<Http>>,
    config: Arc<Config>,
    scanner_config: ScannerConfig,
    pair_cache: Arc<tokio::sync::RwLock<HashMap<Address, LiquidityPair>>>,
    opportunity_history: Arc<tokio::sync::RwLock<Vec<Opportunity>>>,
    last_scan_time: Arc<tokio::sync::RwLock<Instant>>,
    scan_metrics: Arc<tokio::sync::RwLock<ScanMetrics>>,
}

/// Scanning performance metrics
#[derive(Debug, Clone)]
pub struct ScanMetrics {
    pub total_scans: u64,
    pub pairs_scanned: u64,
    pub opportunities_found: u64,
    pub average_scan_time_ms: u64,
    pub last_scan_duration_ms: u64,
    pub cache_hit_rate: f64,
    pub error_count: u64,
}

impl Default for ScanMetrics {
    fn default() -> Self {
        Self {
            total_scans: 0,
            pairs_scanned: 0,
            opportunities_found: 0,
            average_scan_time_ms: 0,
            last_scan_duration_ms: 0,
            cache_hit_rate: 0.0,
            error_count: 0,
        }
    }
}

impl Market {
    pub async fn new(provider: &Provider<Http>, config: &Config) -> Result<Self> {
        let scanner_config = ScannerConfig::default();
        
        info!("Initializing market scanner with config: {:?}", scanner_config);
        
        Ok(Self {
            provider: Arc::new(provider.clone()),
            config: Arc::new(config.clone()),
            scanner_config,
            pair_cache: Arc::new(tokio::sync::RwLock::new(HashMap::new())),
            opportunity_history: Arc::new(tokio::sync::RwLock::new(Vec::new())),
            last_scan_time: Arc::new(tokio::sync::RwLock::new(Instant::now())),
            scan_metrics: Arc::new(tokio::sync::RwLock::new(ScanMetrics::default())),
        })
    }

    /// Scan for trading opportunities
    pub async fn scan_opportunities(&self) -> Result<Vec<Opportunity>> {
        let context = ErrorContext::new("market_opportunity_scan");
        let scan_start = Instant::now();
        
        info!("Starting market opportunity scan");
        
        let mut opportunities = Vec::new();
        let mut pairs_scanned = 0;
        
        // Scan Uniswap V2 pairs
        if self.scanner_config.enable_v2 {
            match self.scan_uniswap_v2_pairs().await {
                Ok(v2_opportunities) => {
                    opportunities.extend(v2_opportunities);
                    pairs_scanned += self.get_cached_pairs_count().await;
                }
                Err(e) => {
                    error!(error = %e, "Failed to scan Uniswap V2 pairs");
                    self.increment_error_count().await;
                }
            }
        }
        
        // Scan Uniswap V3 pairs
        if self.scanner_config.enable_v3 {
            match self.scan_uniswap_v3_pairs().await {
                Ok(v3_opportunities) => {
                    opportunities.extend(v3_opportunities);
                    pairs_scanned += self.get_cached_pairs_count().await;
                }
                Err(e) => {
                    error!(error = %e, "Failed to scan Uniswap V3 pairs");
                    self.increment_error_count().await;
                }
            }
        }
        
        // Filter and rank opportunities
        let filtered_opportunities = self.filter_and_rank_opportunities(opportunities).await;
        
        // Update metrics
        let scan_duration = scan_start.elapsed();
        self.update_scan_metrics(scan_duration, pairs_scanned, filtered_opportunities.len()).await;
        
        // Update last scan time
        *self.last_scan_time.write().await = Instant::now();
        
        info!(
            scan_duration_ms = scan_duration.as_millis(),
            pairs_scanned,
            opportunities_found = filtered_opportunities.len(),
            "Market scan completed"
        );
        
        Ok(filtered_opportunities)
    }

    /// Scan Uniswap V2 pairs for opportunities
    async fn scan_uniswap_v2_pairs(&self) -> Result<Vec<Opportunity>> {
        let context = ErrorContext::new("uniswap_v2_scan");
        let mut opportunities = Vec::new();
        
        // Get factory contract
        let factory_address = Address::from_str(&self.config.uniswap_v2_router)
            .map_err(|e| CryptoJackalError::Validation(format!("Invalid V2 factory address: {}", e)))?;
        
        // Get all pairs from factory (this would need to be implemented with actual contract calls)
        let pairs = self.get_uniswap_v2_pairs(factory_address).await?;
        
        for pair in pairs {
            if let Ok(opportunity) = self.analyze_pair_opportunity(&pair).await {
                opportunities.push(opportunity);
            }
        }
        
        Ok(opportunities)
    }

    /// Scan Uniswap V3 pairs for opportunities
    async fn scan_uniswap_v3_pairs(&self) -> Result<Vec<Opportunity>> {
        let context = ErrorContext::new("uniswap_v3_scan");
        let mut opportunities = Vec::new();
        
        // Get factory contract
        let factory_address = Address::from_str(&self.config.uniswap_v3_router)
            .map_err(|e| CryptoJackalError::Validation(format!("Invalid V3 factory address: {}", e)))?;
        
        // Get all pools from factory
        let pools = self.get_uniswap_v3_pools(factory_address).await?;
        
        for pool in pools {
            if let Ok(opportunity) = self.analyze_pool_opportunity(&pool).await {
                opportunities.push(opportunity);
            }
        }
        
        Ok(opportunities)
    }

    /// Get Uniswap V2 pairs from factory
    async fn get_uniswap_v2_pairs(&self, factory_address: Address) -> Result<Vec<LiquidityPair>> {
        // This would need actual contract interaction
        // For now, return empty vector as placeholder
        Ok(Vec::new())
    }

    /// Get Uniswap V3 pools from factory
    async fn get_uniswap_v3_pools(&self, factory_address: Address) -> Result<Vec<LiquidityPair>> {
        // This would need actual contract interaction
        // For now, return empty vector as placeholder
        Ok(Vec::new())
    }

    /// Analyze a pair for trading opportunities
    async fn analyze_pair_opportunity(&self, pair: &LiquidityPair) -> Result<Opportunity> {
        let context = ErrorContext::new("pair_analysis");
        
        // Get token information
        let token_info = self.get_token_info(&pair.token0).await?;
        
        // Calculate current price
        let price = self.calculate_token_price(pair).await?;
        
        // Calculate liquidity in USD
        let liquidity_usd = self.calculate_liquidity_usd(pair).await?;
        
        // Calculate volatility
        let volatility = self.calculate_volatility(pair).await?;
        
        // Estimate price impact
        let price_impact = self.estimate_price_impact(pair, price).await?;
        
        // Calculate volume and market cap
        let volume_24h = pair.volume_24h;
        let market_cap = self.calculate_market_cap(&token_info, price).await?;
        
        // Determine opportunity type
        let opportunity_type = self.determine_opportunity_type(pair, price, volume_24h).await?;
        
        // Calculate confidence score
        let confidence_score = self.calculate_confidence_score(pair, price_impact, volatility).await?;
        
        // Determine risk level
        let risk_level = self.assess_risk_level(volatility, price_impact, liquidity_usd).await?;
        
        // Check if this is a valid opportunity
        if !self.is_valid_opportunity(liquidity_usd, price_impact, volume_24h) {
            return Err(CryptoJackalError::Validation("Opportunity does not meet criteria".to_string()));
        }
        
        Ok(Opportunity {
            token: token_info,
            pair: pair.clone(),
            price,
            liquidity: liquidity_usd,
            volatility,
            price_impact,
            volume_24h,
            market_cap,
            confidence_score,
            risk_level,
            detected_at: Instant::now(),
            opportunity_type,
        })
    }

    /// Analyze a V3 pool for trading opportunities
    async fn analyze_pool_opportunity(&self, pool: &LiquidityPair) -> Result<Opportunity> {
        // Similar to V2 analysis but with V3-specific logic
        self.analyze_pair_opportunity(pool).await
    }

    /// Get token information from blockchain
    async fn get_token_info(&self, address: &Address) -> Result<TokenInfo> {
        let context = ErrorContext::new("token_info_fetch");
        
        // This would need actual ERC20 contract calls
        // For now, return placeholder data
        Ok(TokenInfo::new(
            &format!("{:?}", address),
            "TOKEN".to_string(),
            18,
            1000000000000000000000000, // 1M tokens
        )?)
    }

    /// Calculate token price from pair reserves
    async fn calculate_token_price(&self, pair: &LiquidityPair) -> Result<f64> {
        if pair.reserve1 == 0 {
            return Err(CryptoJackalError::Validation("Zero reserve in pair".to_string()));
        }
        
        let price = pair.reserve0 as f64 / pair.reserve1 as f64;
        Ok(price)
    }

    /// Calculate liquidity in USD
    async fn calculate_liquidity_usd(&self, pair: &LiquidityPair) -> Result<f64> {
        // This would need price oracle integration
        // For now, return the stored value
        Ok(pair.liquidity_usd)
    }

    /// Calculate token volatility
    async fn calculate_volatility(&self, pair: &LiquidityPair) -> Result<f64> {
        // This would need historical price data
        // For now, return a placeholder value
        Ok(0.15) // 15% volatility
    }

    /// Estimate price impact of a trade
    async fn estimate_price_impact(&self, pair: &LiquidityPair, current_price: f64) -> Result<f64> {
        // Calculate price impact based on trade size and liquidity
        let trade_size_usd = self.config.max_trade_size_eth * 2000.0; // Assume ETH = $2000
        let liquidity_usd = pair.liquidity_usd;
        
        if liquidity_usd == 0.0 {
            return Err(CryptoJackalError::Validation("Zero liquidity".to_string()));
        }
        
        let price_impact = (trade_size_usd / liquidity_usd) * 100.0;
        Ok(price_impact)
    }

    /// Calculate market cap
    async fn calculate_market_cap(&self, token: &TokenInfo, price: f64) -> Result<f64> {
        let supply = token.total_supply as f64;
        let market_cap = supply * price;
        Ok(market_cap)
    }

    /// Determine opportunity type
    async fn determine_opportunity_type(
        &self,
        pair: &LiquidityPair,
        price: f64,
        volume_24h: f64,
    ) -> Result<OpportunityType> {
        // Simple logic to determine opportunity type
        if pair.created_at > (chrono::Utc::now().timestamp() as u64 - 3600) {
            Ok(OpportunityType::NewToken)
        } else if volume_24h > 100000.0 {
            Ok(OpportunityType::Momentum)
        } else {
            Ok(OpportunityType::LiquidityAddition)
        }
    }

    /// Calculate confidence score for opportunity
    async fn calculate_confidence_score(
        &self,
        pair: &LiquidityPair,
        price_impact: f64,
        volatility: f64,
    ) -> Result<f64> {
        let mut score = 1.0;
        
        // Reduce score for high price impact
        if price_impact > 2.0 {
            score -= 0.3;
        }
        
        // Reduce score for high volatility
        if volatility > 0.5 {
            score -= 0.2;
        }
        
        // Increase score for high liquidity
        if pair.liquidity_usd > 100000.0 {
            score += 0.2;
        }
        
        Ok(score.max(0.0).min(1.0))
    }

    /// Assess risk level
    async fn assess_risk_level(
        &self,
        volatility: f64,
        price_impact: f64,
        liquidity_usd: f64,
    ) -> Result<RiskLevel> {
        let mut risk_score = 0;
        
        if volatility > 0.8 {
            risk_score += 3;
        } else if volatility > 0.5 {
            risk_score += 2;
        } else if volatility > 0.2 {
            risk_score += 1;
        }
        
        if price_impact > 10.0 {
            risk_score += 3;
        } else if price_impact > 5.0 {
            risk_score += 2;
        } else if price_impact > 2.0 {
            risk_score += 1;
        }
        
        if liquidity_usd < 10000.0 {
            risk_score += 3;
        } else if liquidity_usd < 50000.0 {
            risk_score += 2;
        } else if liquidity_usd < 100000.0 {
            risk_score += 1;
        }
        
        match risk_score {
            0..=2 => Ok(RiskLevel::Low),
            3..=4 => Ok(RiskLevel::Medium),
            5..=6 => Ok(RiskLevel::High),
            _ => Ok(RiskLevel::Extreme),
        }
    }

    /// Check if opportunity meets criteria
    fn is_valid_opportunity(&self, liquidity: f64, price_impact: f64, volume_24h: f64) -> bool {
        liquidity >= self.scanner_config.min_liquidity_usd
            && price_impact <= self.scanner_config.max_price_impact
            && volume_24h >= self.scanner_config.min_volume_24h
    }

    /// Filter and rank opportunities
    async fn filter_and_rank_opportunities(&self, mut opportunities: Vec<Opportunity>) -> Vec<Opportunity> {
        // Filter out invalid opportunities
        opportunities.retain(|opp| {
            opp.liquidity >= self.scanner_config.min_liquidity_usd
                && opp.price_impact <= self.scanner_config.max_price_impact
                && opp.volume_24h >= self.scanner_config.min_volume_24h
        });
        
        // Sort by confidence score (highest first)
        opportunities.sort_by(|a, b| b.confidence_score.partial_cmp(&a.confidence_score).unwrap());
        
        // Limit to maximum opportunities
        opportunities.truncate(10);
        
        // Store in history
        let mut history = self.opportunity_history.write().await;
        history.extend(opportunities.clone());
        
        // Keep only last 100 opportunities
        if history.len() > 100 {
            history.drain(0..history.len() - 100);
        }
        
        opportunities
    }

    /// Get cached pairs count
    async fn get_cached_pairs_count(&self) -> usize {
        self.pair_cache.read().await.len()
    }

    /// Increment error count
    async fn increment_error_count(&self) {
        let mut metrics = self.scan_metrics.write().await;
        metrics.error_count += 1;
    }

    /// Update scan metrics
    async fn update_scan_metrics(&self, duration: Duration, pairs_scanned: usize, opportunities_found: usize) {
        let mut metrics = self.scan_metrics.write().await;
        metrics.total_scans += 1;
        metrics.pairs_scanned += pairs_scanned as u64;
        metrics.opportunities_found += opportunities_found as u64;
        metrics.last_scan_duration_ms = duration.as_millis() as u64;
        
        // Update average scan time
        let total_time = metrics.average_scan_time_ms * (metrics.total_scans - 1) + metrics.last_scan_duration_ms;
        metrics.average_scan_time_ms = total_time / metrics.total_scans;
    }

    /// Get scan metrics
    pub async fn get_scan_metrics(&self) -> ScanMetrics {
        self.scan_metrics.read().await.clone()
    }

    /// Get opportunity history
    pub async fn get_opportunity_history(&self) -> Vec<Opportunity> {
        self.opportunity_history.read().await.clone()
    }

    /// Check if enough time has passed since last scan
    pub async fn should_scan(&self) -> bool {
        let last_scan = *self.last_scan_time.read().await;
        last_scan.elapsed() >= Duration::from_millis(self.scanner_config.scan_interval_ms)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::str::FromStr;

    #[tokio::test]
    async fn test_market_initialization() {
        let provider = Provider::<Http>::try_from("http://localhost:8545").unwrap();
        let config = Config::default();
        
        let market = Market::new(&provider, &config).await;
        assert!(market.is_ok());
    }

    #[tokio::test]
    async fn test_opportunity_validation() {
        let scanner_config = ScannerConfig::default();
        
        // Test valid opportunity
        let is_valid = scanner_config.min_liquidity_usd <= 50000.0
            && scanner_config.max_price_impact >= 2.0
            && scanner_config.min_volume_24h <= 5000.0;
        
        assert!(is_valid);
    }

    #[test]
    fn test_risk_assessment() {
        // Test low risk
        let risk = RiskLevel::Low;
        assert_eq!(risk, RiskLevel::Low);
        
        // Test extreme risk
        let risk = RiskLevel::Extreme;
        assert_eq!(risk, RiskLevel::Extreme);
    }
} 
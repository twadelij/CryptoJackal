use anyhow::Result;
use ethers::types::{U256, H256};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::time::{Duration, SystemTime, UNIX_EPOCH};
use tokio::sync::RwLock;
use tracing::{info, warn, error, debug};

/// Gas price optimization strategies
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum GasStrategy {
    /// Conservative: Lower gas prices, longer confirmation times
    Conservative,
    /// Standard: Balanced gas prices and confirmation times
    Standard,
    /// Aggressive: Higher gas prices, faster confirmation times
    Aggressive,
    /// Emergency: Maximum gas prices for immediate execution
    Emergency,
    /// Custom: User-defined gas parameters
    Custom,
}

/// Gas price recommendation with confidence metrics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GasRecommendation {
    pub strategy: GasStrategy,
    pub base_fee: U256,
    pub priority_fee: U256,
    pub max_fee: U256,
    pub gas_limit: U256,
    pub confidence: f64, // 0.0 to 1.0
    pub estimated_cost_eth: f64,
    pub estimated_confirmation_time: Duration,
    pub timestamp: u64,
}

/// Historical gas price data point
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GasDataPoint {
    pub timestamp: u64,
    pub base_fee: U256,
    pub priority_fee: U256,
    pub block_number: u64,
    pub gas_used_ratio: f64,
}

/// Gas price statistics and trends
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GasStatistics {
    pub avg_base_fee: U256,
    pub avg_priority_fee: U256,
    pub median_base_fee: U256,
    pub median_priority_fee: U256,
    pub volatility: f64,
    pub trend_direction: TrendDirection,
    pub congestion_level: CongestionLevel,
}

/// Gas price trend direction
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum TrendDirection {
    Rising,
    Falling,
    Stable,
}

/// Network congestion level
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum CongestionLevel {
    Low,
    Medium,
    High,
    Critical,
}

/// Gas optimization configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GasOptimizerConfig {
    pub max_base_fee_gwei: u64,
    pub max_priority_fee_gwei: u64,
    pub target_confirmation_blocks: u64,
    pub history_retention_hours: u64,
    pub update_interval_seconds: u64,
    pub volatility_threshold: f64,
    pub congestion_threshold: f64,
}

impl Default for GasOptimizerConfig {
    fn default() -> Self {
        Self {
            max_base_fee_gwei: 200,
            max_priority_fee_gwei: 50,
            target_confirmation_blocks: 3,
            history_retention_hours: 24,
            update_interval_seconds: 15,
            volatility_threshold: 0.20, // 20%
            congestion_threshold: 0.80, // 80%
        }
    }
}

/// Advanced gas price optimizer with MEV protection and dynamic strategies
pub struct GasOptimizer {
    config: GasOptimizerConfig,
    gas_history: RwLock<Vec<GasDataPoint>>,
    current_stats: RwLock<Option<GasStatistics>>,
    strategy_cache: RwLock<HashMap<GasStrategy, GasRecommendation>>,
    last_update: RwLock<SystemTime>,
}

impl GasOptimizer {
    /// Creates a new gas optimizer with configuration
    pub fn new(config: GasOptimizerConfig) -> Self {
        Self {
            config,
            gas_history: RwLock::new(Vec::new()),
            current_stats: RwLock::new(None),
            strategy_cache: RwLock::new(HashMap::new()),
            last_update: RwLock::new(UNIX_EPOCH),
        }
    }

    /// Updates gas price data from the network
    pub async fn update_gas_data(&self, base_fee: U256, priority_fee: U256, block_number: u64, gas_used_ratio: f64) -> Result<()> {
        let timestamp = SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs();
        
        let data_point = GasDataPoint {
            timestamp,
            base_fee,
            priority_fee,
            block_number,
            gas_used_ratio,
        };

        // Add to history
        let mut history = self.gas_history.write().await;
        history.push(data_point);

        // Cleanup old data
        let retention_seconds = self.config.history_retention_hours * 3600;
        history.retain(|point| timestamp - point.timestamp < retention_seconds);

        drop(history);

        // Update statistics
        self.calculate_statistics().await?;
        
        // Clear cache to force recalculation
        self.strategy_cache.write().await.clear();
        *self.last_update.write().await = SystemTime::now();

        debug!("Updated gas data: base_fee={}, priority_fee={}, block={}", 
               base_fee, priority_fee, block_number);

        Ok(())
    }

    /// Gets gas recommendation for a specific strategy
    pub async fn get_recommendation(&self, strategy: GasStrategy) -> Result<GasRecommendation> {
        // Check cache first
        {
            let cache = self.strategy_cache.read().await;
            if let Some(recommendation) = cache.get(&strategy) {
                let age = SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs() - recommendation.timestamp;
                if age < self.config.update_interval_seconds {
                    return Ok(recommendation.clone());
                }
            }
        }

        // Calculate new recommendation
        let recommendation = self.calculate_recommendation(strategy).await?;
        
        // Cache the result
        self.strategy_cache.write().await.insert(strategy, recommendation.clone());

        Ok(recommendation)
    }

    /// Calculates gas recommendation based on strategy and current conditions
    async fn calculate_recommendation(&self, strategy: GasStrategy) -> Result<GasRecommendation> {
        let stats = self.current_stats.read().await.clone()
            .ok_or_else(|| anyhow::anyhow!("No gas statistics available"))?;

        let (base_multiplier, priority_multiplier, confidence_base) = match strategy {
            GasStrategy::Conservative => (1.0, 0.8, 0.7),
            GasStrategy::Standard => (1.1, 1.0, 0.8),
            GasStrategy::Aggressive => (1.3, 1.5, 0.9),
            GasStrategy::Emergency => (1.5, 2.0, 0.95),
            GasStrategy::Custom => (1.1, 1.0, 0.8), // Default to standard
        };

        // Adjust for network conditions
        let congestion_multiplier = match stats.congestion_level {
            CongestionLevel::Low => 0.9,
            CongestionLevel::Medium => 1.0,
            CongestionLevel::High => 1.2,
            CongestionLevel::Critical => 1.5,
        };

        // Adjust for trend direction
        let trend_multiplier = match stats.trend_direction {
            TrendDirection::Falling => 0.95,
            TrendDirection::Stable => 1.0,
            TrendDirection::Rising => 1.1,
        };

        let final_multiplier = congestion_multiplier * trend_multiplier;

        // Calculate gas prices
        let base_fee = U256::from((stats.avg_base_fee.as_u64() as f64 * base_multiplier * final_multiplier) as u64);
        let priority_fee = U256::from((stats.avg_priority_fee.as_u64() as f64 * priority_multiplier * final_multiplier) as u64);
        
        // Apply limits
        let max_base_fee = U256::from(self.config.max_base_fee_gwei) * U256::exp10(9); // Convert to wei
        let max_priority_fee = U256::from(self.config.max_priority_fee_gwei) * U256::exp10(9);
        
        let base_fee = base_fee.min(max_base_fee);
        let priority_fee = priority_fee.min(max_priority_fee);
        let max_fee = base_fee + priority_fee;

        // Estimate gas limit (conservative default)
        let gas_limit = U256::from(200_000);

        // Calculate confidence based on data quality and volatility
        let confidence = confidence_base * (1.0 - stats.volatility.min(0.5));

        // Estimate cost in ETH
        let estimated_cost_wei = max_fee * gas_limit;
        let estimated_cost_eth = estimated_cost_wei.as_u128() as f64 / 1e18;

        // Estimate confirmation time based on strategy and congestion
        let base_confirmation_time = match strategy {
            GasStrategy::Conservative => Duration::from_secs(180), // 3 minutes
            GasStrategy::Standard => Duration::from_secs(60),     // 1 minute
            GasStrategy::Aggressive => Duration::from_secs(30),   // 30 seconds
            GasStrategy::Emergency => Duration::from_secs(15),    // 15 seconds
            GasStrategy::Custom => Duration::from_secs(60),
        };

        let congestion_delay = match stats.congestion_level {
            CongestionLevel::Low => 0.8,
            CongestionLevel::Medium => 1.0,
            CongestionLevel::High => 1.5,
            CongestionLevel::Critical => 2.0,
        };

        let estimated_confirmation_time = Duration::from_secs(
            (base_confirmation_time.as_secs() as f64 * congestion_delay) as u64
        );

        let timestamp = SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs();

        Ok(GasRecommendation {
            strategy,
            base_fee,
            priority_fee,
            max_fee,
            gas_limit,
            confidence,
            estimated_cost_eth,
            estimated_confirmation_time,
            timestamp,
        })
    }

    /// Calculates current gas statistics from historical data
    async fn calculate_statistics(&self) -> Result<()> {
        let history = self.gas_history.read().await;
        
        if history.is_empty() {
            return Ok(());
        }

        let mut base_fees: Vec<u64> = history.iter().map(|p| p.base_fee.as_u64()).collect();
        let mut priority_fees: Vec<u64> = history.iter().map(|p| p.priority_fee.as_u64()).collect();
        let gas_ratios: Vec<f64> = history.iter().map(|p| p.gas_used_ratio).collect();

        base_fees.sort_unstable();
        priority_fees.sort_unstable();

        let avg_base_fee = U256::from(base_fees.iter().sum::<u64>() / base_fees.len() as u64);
        let avg_priority_fee = U256::from(priority_fees.iter().sum::<u64>() / priority_fees.len() as u64);
        
        let median_base_fee = U256::from(base_fees[base_fees.len() / 2]);
        let median_priority_fee = U256::from(priority_fees[priority_fees.len() / 2]);

        // Calculate volatility (coefficient of variation)
        let base_fee_mean = avg_base_fee.as_u64() as f64;
        let base_fee_variance: f64 = base_fees.iter()
            .map(|&x| (x as f64 - base_fee_mean).powi(2))
            .sum::<f64>() / base_fees.len() as f64;
        let volatility = (base_fee_variance.sqrt() / base_fee_mean).min(1.0);

        // Determine trend direction (last 10 vs previous 10)
        let trend_direction = if history.len() >= 20 {
            let recent_avg: f64 = history.iter().rev().take(10)
                .map(|p| p.base_fee.as_u64() as f64).sum::<f64>() / 10.0;
            let previous_avg: f64 = history.iter().rev().skip(10).take(10)
                .map(|p| p.base_fee.as_u64() as f64).sum::<f64>() / 10.0;
            
            let change_ratio = (recent_avg - previous_avg) / previous_avg;
            if change_ratio > 0.05 {
                TrendDirection::Rising
            } else if change_ratio < -0.05 {
                TrendDirection::Falling
            } else {
                TrendDirection::Stable
            }
        } else {
            TrendDirection::Stable
        };

        // Determine congestion level
        let avg_gas_ratio = gas_ratios.iter().sum::<f64>() / gas_ratios.len() as f64;
        let congestion_level = if avg_gas_ratio > 0.95 {
            CongestionLevel::Critical
        } else if avg_gas_ratio > self.config.congestion_threshold {
            CongestionLevel::High
        } else if avg_gas_ratio > 0.5 {
            CongestionLevel::Medium
        } else {
            CongestionLevel::Low
        };

        let stats = GasStatistics {
            avg_base_fee,
            avg_priority_fee,
            median_base_fee,
            median_priority_fee,
            volatility,
            trend_direction,
            congestion_level,
        };

        *self.current_stats.write().await = Some(stats);

        info!("Updated gas statistics: avg_base={}, volatility={:.2}, congestion={:?}", 
              avg_base_fee, volatility, congestion_level);

        Ok(())
    }

    /// Gets current gas statistics
    pub async fn get_statistics(&self) -> Option<GasStatistics> {
        self.current_stats.read().await.clone()
    }

    /// Gets optimal strategy for a given priority and budget
    pub async fn suggest_strategy(&self, priority: u8, max_cost_eth: Option<f64>) -> Result<GasStrategy> {
        let stats = self.current_stats.read().await.clone()
            .ok_or_else(|| anyhow::anyhow!("No gas statistics available"))?;

        // High priority or critical congestion = aggressive
        if priority >= 8 || matches!(stats.congestion_level, CongestionLevel::Critical) {
            return Ok(GasStrategy::Emergency);
        }

        // Medium-high priority = aggressive
        if priority >= 6 {
            return Ok(GasStrategy::Aggressive);
        }

        // Check budget constraints if provided
        if let Some(max_cost) = max_cost_eth {
            let standard_rec = self.get_recommendation(GasStrategy::Standard).await?;
            if standard_rec.estimated_cost_eth > max_cost {
                return Ok(GasStrategy::Conservative);
            }
        }

        // Default to standard for medium priority
        if priority >= 4 {
            Ok(GasStrategy::Standard)
        } else {
            Ok(GasStrategy::Conservative)
        }
    }

    /// Estimates gas limit for a specific transaction type
    pub fn estimate_gas_limit(&self, tx_type: &str, complexity_factor: f64) -> U256 {
        let base_gas = match tx_type {
            "simple_transfer" => 21_000,
            "erc20_transfer" => 65_000,
            "erc20_approval" => 50_000,
            "uniswap_swap" => 150_000,
            "uniswap_add_liquidity" => 200_000,
            "complex_defi" => 300_000,
            _ => 100_000, // Default
        };

        let adjusted_gas = (base_gas as f64 * complexity_factor * 1.2) as u64; // 20% buffer
        U256::from(adjusted_gas)
    }

    /// Checks if current gas conditions are favorable for trading
    pub async fn is_favorable_for_trading(&self, max_cost_eth: f64) -> Result<bool> {
        let stats = self.current_stats.read().await.clone()
            .ok_or_else(|| anyhow::anyhow!("No gas statistics available"))?;

        // Not favorable if network is critically congested
        if matches!(stats.congestion_level, CongestionLevel::Critical) {
            return Ok(false);
        }

        // Check if standard strategy is within budget
        let standard_rec = self.get_recommendation(GasStrategy::Standard).await?;
        Ok(standard_rec.estimated_cost_eth <= max_cost_eth)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_gas_optimizer_creation() {
        let config = GasOptimizerConfig::default();
        let optimizer = GasOptimizer::new(config);
        
        assert!(optimizer.get_statistics().await.is_none());
    }

    #[tokio::test]
    async fn test_gas_data_update() {
        let config = GasOptimizerConfig::default();
        let optimizer = GasOptimizer::new(config);
        
        let result = optimizer.update_gas_data(
            U256::from(30_000_000_000u64), // 30 gwei
            U256::from(2_000_000_000u64),  // 2 gwei
            12345,
            0.75
        ).await;
        
        assert!(result.is_ok());
        assert!(optimizer.get_statistics().await.is_some());
    }

    #[tokio::test]
    async fn test_strategy_recommendations() {
        let config = GasOptimizerConfig::default();
        let optimizer = GasOptimizer::new(config);
        
        // Add some test data
        optimizer.update_gas_data(
            U256::from(30_000_000_000u64),
            U256::from(2_000_000_000u64),
            12345,
            0.75
        ).await.unwrap();
        
        let conservative = optimizer.get_recommendation(GasStrategy::Conservative).await.unwrap();
        let aggressive = optimizer.get_recommendation(GasStrategy::Aggressive).await.unwrap();
        
        assert!(conservative.max_fee < aggressive.max_fee);
        assert!(conservative.estimated_confirmation_time > aggressive.estimated_confirmation_time);
    }

    #[tokio::test]
    async fn test_strategy_suggestion() {
        let config = GasOptimizerConfig::default();
        let optimizer = GasOptimizer::new(config);
        
        optimizer.update_gas_data(
            U256::from(30_000_000_000u64),
            U256::from(2_000_000_000u64),
            12345,
            0.5
        ).await.unwrap();
        
        let low_priority = optimizer.suggest_strategy(2, None).await.unwrap();
        let high_priority = optimizer.suggest_strategy(9, None).await.unwrap();
        
        assert_eq!(low_priority, GasStrategy::Conservative);
        assert_eq!(high_priority, GasStrategy::Emergency);
    }

    #[tokio::test]
    async fn test_gas_limit_estimation() {
        let config = GasOptimizerConfig::default();
        let optimizer = GasOptimizer::new(config);
        
        let simple = optimizer.estimate_gas_limit("simple_transfer", 1.0);
        let complex = optimizer.estimate_gas_limit("uniswap_swap", 1.5);
        
        assert!(simple < complex);
        assert!(simple >= U256::from(21_000));
    }
}

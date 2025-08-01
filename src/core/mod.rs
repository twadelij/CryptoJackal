use anyhow::Result;
use ethers::prelude::*;
use std::sync::Arc;
use tokio::sync::RwLock;
use tracing::{info, warn, error};

mod config;
pub mod order_queue;
pub mod dex_monitor;
pub mod gas_optimizer;
mod market;
mod price_feed;
mod types;

use crate::trading::Trading;
use crate::wallet::Wallet;
use order_queue::{OrderQueue, OrderQueueConfig, OrderPriority, OrderStrategy, OrderContext};
use dex_monitor::DexMonitor;
use gas_optimizer::{GasOptimizer, GasOptimizerConfig, GasStrategy};
use price_feed::{PriceFeedMonitor, PriceFeedConfig, AlertThresholds, PriceSource};

pub struct Bot {
    wallet: Arc<RwLock<Wallet>>,
    trading: Arc<RwLock<Trading>>,
    provider: Provider<Http>,
    config: config::Config,
    dex_monitor: Arc<RwLock<DexMonitor>>,
    order_queue: Arc<OrderQueue>,
    gas_optimizer: Arc<GasOptimizer>,
    price_feed_monitor: Arc<PriceFeedMonitor>,
}

impl Bot {
    pub async fn new() -> Result<Self> {
        let config = config::Config::load()?;
        let provider = Provider::<Http>::try_from(&config.node_url)?;
        
        let wallet = Arc::new(RwLock::new(Wallet::new(&config).await?));
        let trading = Arc::new(RwLock::new(Trading::new(&config).await?));
        
        // Initialize DEX monitor
        let dex_monitor = Arc::new(RwLock::new(DexMonitor::new(&config).await?));
        
        // Initialize order queue with configuration
        let queue_config = OrderQueueConfig {
            max_concurrent_orders: config.max_concurrent_trades.unwrap_or(5),
            default_timeout_seconds: config.order_timeout_seconds.unwrap_or(30),
            max_retry_attempts: config.max_retry_attempts.unwrap_or(3),
            cleanup_interval_seconds: 300, // 5 minutes
        };
        let order_queue = Arc::new(OrderQueue::new(queue_config));
        
        // Initialize gas optimizer
        let gas_config = GasOptimizerConfig {
            max_base_fee_gwei: config.max_gas_price_gwei.unwrap_or(200),
            max_priority_fee_gwei: config.priority_fee_gwei.unwrap_or(50),
            target_confirmation_blocks: 3,
            history_retention_hours: 24,
            update_interval_seconds: 15,
            volatility_threshold: 0.20,
            congestion_threshold: 0.80,
        };
        let gas_optimizer = Arc::new(GasOptimizer::new(gas_config));
        
        // Initialize price feed monitor
        let price_feed_config = PriceFeedConfig {
            update_interval_ms: config.price_feed_update_interval_ms.unwrap_or(5000),
            aggregation_timeout_ms: 2000,
            outlier_threshold: config.price_feed_outlier_threshold.unwrap_or(3.0),
            alert_thresholds: AlertThresholds {
                price_change_percent: config.price_feed_alert_threshold_percent.unwrap_or(5.0),
                volume_spike_percent: 200.0,
                volatility_threshold: 0.5,
                outlier_threshold: 2.0,
            },
            enabled_sources: Self::parse_price_sources(&config.price_feed_enabled_sources),
            max_retry_attempts: 3,
            retry_delay_ms: 1000,
        };
        let price_feed_monitor = Arc::new(PriceFeedMonitor::new(price_feed_config));

        Ok(Self {
            wallet,
            trading,
            provider,
            config,
            dex_monitor,
            order_queue,
            gas_optimizer,
            price_feed_monitor,
        })
    }

    pub async fn run(&self) -> Result<()> {
        info!("🚀 CryptoJackal Bot starting with gas optimization and order queue system...");
        
        // Start gas monitoring task
        let gas_optimizer_clone = Arc::clone(&self.gas_optimizer);
        let provider_clone = self.provider.clone();
        let gas_monitoring_task = tokio::spawn(async move {
            loop {
                if let Err(e) = Self::update_gas_data(&gas_optimizer_clone, &provider_clone).await {
                    error!("Gas monitoring error: {}", e);
                }
                tokio::time::sleep(std::time::Duration::from_secs(15)).await;
            }
        });
        
        // Start order queue processing
        let trading_clone = Arc::clone(&self.trading);
        let wallet_clone = Arc::clone(&self.wallet);
        let order_queue_clone = Arc::clone(&self.order_queue);
        let queue_task = tokio::spawn(async move {
            if let Err(e) = order_queue_clone.start(trading_clone, wallet_clone).await {
                error!("Order queue error: {}", e);
            }
        });
        
        // Start DEX monitoring
        let dex_monitor = Arc::clone(&self.dex_monitor);
        let monitoring_task = tokio::spawn(async move {
            let mut monitor = dex_monitor.write().await;
            if let Err(e) = monitor.start_monitoring().await {
                error!("DEX monitoring failed: {}", e);
            }
        });
        
        // Start opportunity processing
        let processing_task = self.start_opportunity_processing().await?;
        
        info!("✅ All systems started - monitoring for opportunities with gas optimization...");
        
        tokio::select! {
            _ = gas_monitoring_task => {
                warn!("Gas monitoring task completed");
            }
            _ = queue_task => {
                warn!("Order queue task completed");
            }
            _ = monitoring_task => {
                warn!("DEX monitoring task completed");
            }
            _ = processing_task => {
                warn!("Opportunity processing completed");
            }
        }
        
        self.order_queue.shutdown().await;
        info!("🛑 Bot shutdown complete");
        Ok(())
    }

    async fn process_market_opportunities(&self, market: &market::Market) -> Result<()> {
        let opportunities = market.scan_opportunities().await?;
        
        for opportunity in opportunities {
            if self.trading.read().await.should_execute(&opportunity) {
                info!("Found trading opportunity: {:?}", opportunity);
                
                match self.execute_trade(&opportunity).await {
                    Ok(_) => info!("Successfully executed trade for {:?}", opportunity),
                    Err(e) => warn!("Failed to execute trade: {}", e),
                }
            }
        }
        
        Ok(())
    }

    async fn execute_trade(&self, opportunity: &market::Opportunity) -> Result<()> {
        let wallet = self.wallet.read().await;
        let trading = self.trading.read().await;
        
        // Perform the trade
        trading.execute(opportunity, &wallet).await
    }
    
    /// Updates gas data from the network for the gas optimizer
    async fn update_gas_data(gas_optimizer: &Arc<GasOptimizer>, provider: &Provider<Http>) -> Result<()> {
        // Get latest block
        let latest_block = provider.get_block(BlockNumber::Latest).await?
            .ok_or_else(|| anyhow::anyhow!("Failed to get latest block"))?;
        
        if let (Some(base_fee), Some(gas_used), Some(gas_limit)) = 
            (latest_block.base_fee_per_gas, latest_block.gas_used, latest_block.gas_limit) {
            
            let gas_used_ratio = gas_used.as_u64() as f64 / gas_limit.as_u64() as f64;
            
            // Estimate priority fee from recent transactions (simplified)
            let priority_fee = U256::from(2_000_000_000u64); // 2 gwei default
            
            gas_optimizer.update_gas_data(
                base_fee,
                priority_fee,
                latest_block.number.unwrap_or_default().as_u64(),
                gas_used_ratio
            ).await?;
        }
        
        Ok(())
    }
    
    /// Starts the opportunity processing loop with gas optimization
    async fn start_opportunity_processing(&self) -> Result<tokio::task::JoinHandle<()>> {
        let dex_monitor = Arc::clone(&self.dex_monitor);
        let order_queue = Arc::clone(&self.order_queue);
        let gas_optimizer = Arc::clone(&self.gas_optimizer);
        let config = self.config.clone();
        
        let task = tokio::spawn(async move {
            let mut interval = tokio::time::interval(std::time::Duration::from_millis(config.scan_interval));
            
            loop {
                interval.tick().await;
                
                // Check if gas conditions are favorable
                let max_cost_eth = 0.01; // 0.01 ETH max gas cost
                let gas_favorable = match gas_optimizer.is_favorable_for_trading(max_cost_eth).await {
                    Ok(favorable) => favorable,
                    Err(e) => {
                        warn!("Failed to check gas conditions: {}", e);
                        true // Default to allowing trades
                    }
                };
                
                if !gas_favorable {
                    debug!("Skipping opportunity scan due to unfavorable gas conditions");
                    continue;
                }
                
                // Scan for opportunities
                let monitor = dex_monitor.read().await;
                if let Ok(opportunities) = monitor.get_latest_opportunities().await {
                    for opportunity in opportunities {
                        // Calculate priority based on profit potential and urgency
                        let priority = Self::calculate_order_priority(&opportunity);
                        
                        // Suggest optimal gas strategy
                        let gas_strategy = match gas_optimizer.suggest_strategy(priority as u8, Some(max_cost_eth)).await {
                            Ok(strategy) => strategy,
                            Err(_) => GasStrategy::Standard,
                        };
                        
                        // Create order context with gas optimization
                        let context = OrderContext {
                            opportunity: opportunity.clone(),
                            gas_strategy,
                            max_gas_cost_eth: max_cost_eth,
                            created_at: std::time::SystemTime::now(),
                        };
                        
                        // Submit to order queue
                        if let Err(e) = order_queue.submit_order(priority, OrderStrategy::Immediate, context).await {
                            error!("Failed to submit order: {}", e);
                        }
                    }
                }
            }
        });
        
        Ok(task)
    }
    
    /// Calculates order priority based on opportunity characteristics
    fn calculate_order_priority(opportunity: &market::Opportunity) -> OrderPriority {
        // High priority for high profit opportunities
        if opportunity.profit_percentage > 0.10 { // 10%+
            OrderPriority::High
        } else if opportunity.profit_percentage > 0.05 { // 5%+
            OrderPriority::Medium
        } else if opportunity.profit_percentage > 0.02 { // 2%+
            OrderPriority::Low
        } else {
            OrderPriority::VeryLow
        }
    }
    
    /// Gets current gas optimization metrics
    pub async fn get_gas_metrics(&self) -> Option<gas_optimizer::GasStatistics> {
        self.gas_optimizer.get_statistics().await
    }
    
    /// Gets order queue status
    pub async fn get_order_queue_status(&self) -> order_queue::QueueMetrics {
        self.order_queue.get_metrics().await
    }
    
    /// Checks if the bot should execute trades based on gas conditions
    pub async fn should_trade(&self, max_cost_eth: f64) -> bool {
        match self.gas_optimizer.is_favorable_for_trading(max_cost_eth).await {
            Ok(favorable) => favorable,
            Err(_) => true, // Default to allowing trades if gas check fails
        }
    }
    
    /// Parses price source strings into PriceSource enum
    fn parse_price_sources(sources: &Option<Vec<String>>) -> Vec<PriceSource> {
        let default_sources = vec![
            PriceSource::CoinGecko,
            PriceSource::UniswapV2,
            PriceSource::UniswapV3,
        ];
        
        match sources {
            Some(source_strings) => {
                let mut parsed_sources = Vec::new();
                for source_str in source_strings {
                    match source_str.to_lowercase().as_str() {
                        "coingecko" => parsed_sources.push(PriceSource::CoinGecko),
                        "coinmarketcap" => parsed_sources.push(PriceSource::CoinMarketCap),
                        "uniswap_v2" | "uniswapv2" => parsed_sources.push(PriceSource::UniswapV2),
                        "uniswap_v3" | "uniswapv3" => parsed_sources.push(PriceSource::UniswapV3),
                        "dexscreener" => parsed_sources.push(PriceSource::DexScreener),
                        _ => warn!("Unknown price source: {}", source_str),
                    }
                }
                if parsed_sources.is_empty() {
                    default_sources
                } else {
                    parsed_sources
                }
            }
            None => default_sources,
        }
    }
    
    /// Gets price feed metrics
    pub async fn get_price_feed_metrics(&self) -> price_feed::PriceFeedMetrics {
        self.price_feed_monitor.get_metrics().await
    }
    
    /// Gets current price for a token
    pub async fn get_token_price(&self, token_address: &str) -> Option<price_feed::AggregatedPrice> {
        self.price_feed_monitor.get_current_price(token_address).await
    }
    
    /// Gets recent price alerts
    pub async fn get_recent_price_alerts(&self, limit: usize) -> Vec<price_feed::PriceAlert> {
        self.price_feed_monitor.get_recent_alerts(limit).await
    }
} 
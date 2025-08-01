use anyhow::Result;
use ethers::prelude::*;
use std::sync::Arc;
use tokio::sync::RwLock;
use tracing::{info, warn, error};

pub mod config;
pub mod dex_monitor;
pub mod market;
pub mod order_queue;
pub mod types;

use crate::trading::Trading;
use crate::wallet::Wallet;
use self::order_queue::{OrderQueue, QueueConfig, Order, OrderPriority, ExecutionStrategy, OrderContext};
use self::dex_monitor::DexMonitor;

pub struct Bot {
    wallet: Arc<RwLock<Wallet>>,
    trading: Arc<RwLock<Trading>>,
    provider: Provider<Http>,
    config: config::Config,
    dex_monitor: Arc<RwLock<DexMonitor>>,
    order_queue: Arc<OrderQueue>,
}

impl Bot {
    pub async fn new() -> Result<Self> {
        let config = config::Config::load()?;
        let provider = Provider::<Http>::try_from(&config.node_url)?;
        
        let wallet = Arc::new(RwLock::new(Wallet::new(&config).await?));
        let trading = Arc::new(RwLock::new(Trading::new(&config).await?));
        let dex_monitor = Arc::new(RwLock::new(DexMonitor::new(&config).await?));
        
        // Configure order queue with optimal settings
        let queue_config = QueueConfig {
            max_concurrent_executions: config.max_concurrent_trades.unwrap_or(3),
            default_timeout_seconds: 30,
            max_retry_attempts: 2,
            cleanup_interval_seconds: 300, // 5 minutes
            metrics_update_interval_seconds: 10,
        };
        let order_queue = Arc::new(OrderQueue::new(queue_config));

        Ok(Self {
            wallet,
            trading,
            provider,
            config,
            dex_monitor,
            order_queue,
        })
    }

    pub async fn run(&self) -> Result<()> {
        info!("🚀 CryptoJackal Bot starting with order queue system...");
        
        // Start order queue processing
        let trading_clone = Arc::new(self.trading.read().await.clone());
        let wallet_clone = Arc::new(self.wallet.read().await.clone());
        let order_queue_clone = Arc::clone(&self.order_queue);
        
        // Start order queue in background
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
        
        info!("✅ All systems started - monitoring for opportunities...");
        
        // Wait for any task to complete (shouldn't happen in normal operation)
        tokio::select! {
            _ = queue_task => warn!("Order queue task completed"),
            _ = monitoring_task => warn!("DEX monitoring task completed"),
            _ = processing_task => warn!("Opportunity processing completed"),
        }
        
        // Cleanup
        self.order_queue.shutdown().await;
        info!("🛑 Bot shutdown complete");
        Ok(())
    }

    async fn start_opportunity_processing(&self) -> Result<()> {
        info!("Bot is running...");
        
        // Initialize market monitoring
        let market = market::Market::new(&self.provider, &self.config).await?;
        
        loop {
            if let Err(e) = self.process_market_opportunities(&market).await {
                error!("Error processing market opportunities: {}", e);
            }
            
            tokio::time::sleep(std::time::Duration::from_millis(self.config.scan_interval)).await;
        }
    }

    async fn process_market_opportunities(&self, market: &market::Market) -> Result<()> {
        let opportunities = market.scan_opportunities().await?;
        
        for opportunity in opportunities {
            if self.trading.read().await.should_execute(&opportunity) {
                info!("🎯 Found trading opportunity: {} (profit: {:.2}%)", 
                      opportunity.token.symbol, opportunity.expected_profit * 100.0);
                
                match self.queue_trade_order(&opportunity).await {
                    Ok(order_id) => {
                        info!("📋 Order {} queued for execution", order_id);
                        
                        // Log queue metrics
                        let metrics = self.order_queue.get_metrics().await;
                        debug!("Queue status: {} pending, {} active, {:.1}% success rate", 
                               metrics.queue_size, metrics.concurrent_executions, 
                               metrics.success_rate * 100.0);
                    }
                    Err(e) => error!("❌ Failed to queue order: {}", e),
                }
            }
        }
        
        Ok(())
    }

    /// Queues a trading opportunity for execution via the order queue
    async fn queue_trade_order(&self, opportunity: &market::Opportunity) -> Result<String> {
        // Determine order priority based on opportunity characteristics
        let priority = self.calculate_order_priority(opportunity);
        
        // Create execution strategy based on opportunity urgency
        let strategy = if opportunity.volatility > 0.15 {
            ExecutionStrategy::Immediate // High volatility = immediate execution
        } else if opportunity.expected_profit > 0.10 {
            ExecutionStrategy::Immediate // High profit = immediate execution
        } else {
            // Normal opportunities can wait a bit for optimal timing
            ExecutionStrategy::Delayed(std::time::Duration::from_millis(500))
        };
        
        // Create order context
        let context = OrderContext {
            opportunity: opportunity.clone(),
            max_slippage: self.config.max_slippage,
            timeout_seconds: 30,
            retry_count: 2,
            gas_price_multiplier: if priority == OrderPriority::Critical { 1.5 } else { 1.2 },
        };
        
        // Create and submit order
        let order = Order::new(priority, strategy, context);
        let order_id = self.order_queue.submit_order(order).await?;
        
        info!("📊 Order {} submitted with priority {:?}", order_id, priority);
        Ok(order_id)
    }
    
    /// Calculates order priority based on opportunity characteristics
    fn calculate_order_priority(&self, opportunity: &market::Opportunity) -> OrderPriority {
        // Emergency: Very high profit with low risk
        if opportunity.expected_profit > 0.20 && opportunity.price_impact < 0.02 {
            return OrderPriority::Emergency;
        }
        
        // Critical: High profit or very low price impact
        if opportunity.expected_profit > 0.15 || opportunity.price_impact < 0.01 {
            return OrderPriority::Critical;
        }
        
        // High: Good profit with reasonable risk
        if opportunity.expected_profit > 0.08 && opportunity.volatility < 0.20 {
            return OrderPriority::High;
        }
        
        // Normal: Standard opportunities
        if opportunity.expected_profit > 0.05 {
            return OrderPriority::Normal;
        }
        
        // Low: Marginal opportunities
        OrderPriority::Low
    }
    
    /// Gets current order queue metrics for monitoring
    pub async fn get_queue_metrics(&self) -> order_queue::ExecutionMetrics {
        self.order_queue.get_metrics().await
    }
    
    /// Gets the status of a specific order
    pub async fn get_order_status(&self, order_id: &str) -> Option<order_queue::OrderStatus> {
        self.order_queue.get_order_status(order_id).await
    }
}
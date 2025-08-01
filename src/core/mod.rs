use crate::error::Result;
use ethers::prelude::*;
use std::sync::Arc;
use tokio::sync::RwLock;
use tracing::{info, warn, error};

mod config;
mod dex_monitor;
mod market;
mod types;

use crate::trading::Trading;
use crate::wallet::Wallet;
use crate::utils;

// Re-export public types
pub use config::Config;
pub use dex_monitor::{DexMonitor, DexMonitorConfig, LiquidityEvent};
pub use types::*;

pub struct Bot {
    wallet: Arc<RwLock<Wallet>>,
    trading: Arc<RwLock<Trading>>,
    provider: Provider<Http>,
    config: Config,
    dex_monitor: Arc<RwLock<DexMonitor>>,
}

impl Bot {
    pub async fn new() -> Result<Self> {
        let config = Config::load()?;
        let provider = Provider::<Http>::try_from(&config.node_url)?;
        
        let wallet = Arc::new(RwLock::new(Wallet::new(&config).await?));
        let trading = Arc::new(RwLock::new(Trading::new(&config).await?));
        
        // Initialize DEX monitor with configuration
        let dex_config = DexMonitorConfig::default();
        let dex_monitor = Arc::new(RwLock::new(DexMonitor::new(dex_config)));

        Ok(Self {
            wallet,
            trading,
            provider,
            config,
            dex_monitor,
        })
    }

    pub async fn run(&self) -> Result<()> {
        info!("Bot is running...");
        
        // Start WebSocket DEX monitoring in a separate task
        let dex_monitor = Arc::clone(&self.dex_monitor);
        let monitoring_task = tokio::spawn(async move {
            let mut monitor = dex_monitor.write().await;
            if let Err(e) = monitor.start_monitoring().await {
                error!("DEX monitoring failed: {}", e);
            }
        });
        
        // Initialize traditional market monitoring as fallback
        let market = market::Market::new(&self.provider, &self.config).await?;
        
        // Run both monitoring systems concurrently
        tokio::select! {
            _ = monitoring_task => {
                warn!("WebSocket monitoring task completed");
            }
            _ = self.run_traditional_monitoring(&market) => {
                warn!("Traditional monitoring completed");
            }
        }
        
        Ok(())
    }
    
    /// Runs traditional polling-based market monitoring as fallback
    async fn run_traditional_monitoring(&self, market: &market::Market) -> Result<()> {
        loop {
            if let Err(e) = self.process_market_opportunities(market).await {
                error!("Error processing market opportunities: {}", e);
            }
            
            tokio::time::sleep(std::time::Duration::from_millis(self.config.scan_interval)).await;
        }
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
}

// Re-export the Bot struct and related types for external use
pub use self::Bot; 
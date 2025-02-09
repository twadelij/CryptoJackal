use anyhow::Result;
use ethers::prelude::*;
use std::sync::Arc;
use tokio::sync::RwLock;
use tracing::{info, warn, error};

mod config;
mod market;
mod types;

use crate::trading::Trading;
use crate::wallet::Wallet;

pub struct Bot {
    wallet: Arc<RwLock<Wallet>>,
    trading: Arc<RwLock<Trading>>,
    provider: Provider<Http>,
    config: config::Config,
}

impl Bot {
    pub async fn new() -> Result<Self> {
        let config = config::Config::load()?;
        let provider = Provider::<Http>::try_from(&config.node_url)?;
        
        let wallet = Arc::new(RwLock::new(Wallet::new(&config).await?));
        let trading = Arc::new(RwLock::new(Trading::new(&config).await?));

        Ok(Self {
            wallet,
            trading,
            provider,
            config,
        })
    }

    pub async fn run(&self) -> Result<()> {
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
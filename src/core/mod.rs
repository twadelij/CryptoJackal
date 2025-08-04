use anyhow::Result;
use ethers::prelude::*;
use std::sync::Arc;
use tokio::sync::RwLock;
use tracing::{info, warn, error};

pub mod config;
pub mod market;
pub mod transaction_signing;
pub mod types;

use crate::trading::Trading;
use crate::wallet::Wallet;
use transaction_signing::{TransactionSigningWorkflow, TransactionSigningConfig, GasStrategy};

pub struct Bot {
    wallet: Arc<RwLock<Wallet>>,
    trading: Arc<RwLock<Trading>>,
    provider: Provider<Http>,
    config: config::Config,
    transaction_signing: Arc<TransactionSigningWorkflow>,
}

impl Bot {
    pub async fn new(config: &config::Config) -> Result<Self> {
        let config = config.clone();
        let provider = Provider::<Http>::try_from(&config.node_url)?;
        
        let wallet = Arc::new(RwLock::new(Wallet::new(&config).await?));
        let trading = Arc::new(RwLock::new(Trading::new(&config).await?));
        
        // Initialize transaction signing workflow
        let transaction_signing_config = TransactionSigningConfig::default();
        let transaction_signing = Arc::new(TransactionSigningWorkflow::new(transaction_signing_config));

        Ok(Self {
            wallet,
            trading,
            provider,
            config,
            transaction_signing,
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
        
        // Prepare trade parameters
        // Get wallet reference from RwLock
        let trade_params = trading.prepare_trade_params(opportunity, &wallet)?;
        
        // Prepare transaction for signing
        let transaction_request = self.transaction_signing
            .prepare_swap_transaction(&trade_params, GasStrategy::Standard, &self.provider)
            .await?;
        
        info!("Prepared transaction for signing: {}", transaction_request.id);
        
        // Sign transaction using MetaMask
        let signed_transaction = self.transaction_signing
            .sign_transaction(&transaction_request.id, &wallet)
            .await?;
        
        info!("Transaction signed: {}", transaction_request.id);
        
        // Submit transaction to network
        let tx_hash = self.transaction_signing
            .submit_transaction(&transaction_request.id, &signed_transaction, &self.provider)
            .await?;
        
        info!("Transaction submitted: {} -> {}", transaction_request.id, tx_hash);
        
        // Wait for confirmation
        let confirmed_transaction = self.transaction_signing
            .wait_for_confirmation(&transaction_request.id, &tx_hash, &self.provider)
            .await?;
        
        match confirmed_transaction.status {
            transaction_signing::TransactionStatus::Confirmed => {
                info!("Transaction confirmed: {} -> {}", transaction_request.id, tx_hash);
                Ok(())
            }
            transaction_signing::TransactionStatus::Failed(error_msg) => {
                error!("Transaction failed: {} -> {}", transaction_request.id, error_msg);
                Err(anyhow::anyhow!("Transaction failed: {}", error_msg))
            }
            _ => {
                error!("Transaction in unexpected state: {:?}", confirmed_transaction.status);
                Err(anyhow::anyhow!("Transaction in unexpected state"))
            }
        }
    }
    
    /// Gets transaction signing metrics
    pub async fn get_transaction_metrics(&self) -> transaction_signing::TransactionMetrics {
        self.transaction_signing.get_metrics().await
    }
    
    /// Gets transaction status
    pub async fn get_transaction_status(&self, transaction_id: &str) -> Option<transaction_signing::TransactionStatus> {
        self.transaction_signing.get_transaction_status(transaction_id).await
    }
    
    /// Cancels a pending transaction
    pub async fn cancel_transaction(&self, transaction_id: &str) -> Result<()> {
        self.transaction_signing.cancel_transaction(transaction_id).await
    }
} 
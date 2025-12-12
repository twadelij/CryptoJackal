use anyhow::Result;
use chrono::{DateTime, Utc};
use ethers::prelude::*;
use serde::{Deserialize, Serialize};
use std::time::{SystemTime, UNIX_EPOCH};
use tracing::{info, warn};

use crate::core::{
    config::Config,
    market::Opportunity,
    types::{TradeParams, TradeResult},
};
use crate::wallet::Wallet;

/// Trading opportunity for paper trading and backtesting
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TradingOpportunity {
    pub id: String,
    pub token_address: String,
    pub token_symbol: String,
    pub token_name: String,
    pub current_price: f64,
    pub expected_profit: f64,
    pub liquidity: f64,
    pub volume_24h: f64,
    pub price_impact: f64,
    pub confidence_score: f64,
    pub discovered_at: DateTime<Utc>,
    pub expires_at: DateTime<Utc>,
}

pub struct Trading {
    config: Config,
    router_address: Address,
}

impl Trading {
    pub async fn new(config: &Config) -> Result<Self> {
        Ok(Self {
            config: config.clone(),
            router_address: "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D".parse()?, // Uniswap V2 Router
        })
    }

    pub fn should_execute(&self, opportunity: &Opportunity) -> bool {
        // Implement trading strategy logic here
        opportunity.volatility > 0.05 && // More than 5% volatility
        opportunity.price_impact < self.config.max_price_impact &&
        opportunity.liquidity >= self.config.min_liquidity
    }

    pub async fn execute(&self, opportunity: &Opportunity, wallet: &Wallet) -> Result<TradeResult> {
        info!("Executing trade for token: {}", opportunity.token.symbol);

        let trade_params = self.prepare_trade_params(opportunity, wallet)?;
        
        // Execute the trade
        match self.execute_swap(&trade_params, wallet).await {
            Ok(result) => {
                info!("Trade executed successfully: {:?}", result);
                Ok(result)
            }
            Err(e) => {
                warn!("Trade execution failed: {}", e);
                Err(e)
            }
        }
    }

    pub fn prepare_trade_params(&self, opportunity: &Opportunity, wallet: &Wallet) -> Result<TradeParams> {
        let deadline = SystemTime::now()
            .duration_since(UNIX_EPOCH)?
            .as_secs()
            + 300; // 5 minutes deadline

        Ok(TradeParams {
            token_in: opportunity.token.address,
            token_out: self.get_base_token_address()?,
            amount_in: self.calculate_trade_amount(opportunity)?,
            min_amount_out: self.calculate_min_amount_out(opportunity)?,
            deadline,
            recipient: wallet.address(),
        })
    }

    async fn execute_swap(&self, params: &TradeParams, wallet: &Wallet) -> Result<TradeResult> {
        // This is now handled by the transaction signing workflow in the Bot
        // The actual swap execution is delegated to the transaction signing module
        Ok(TradeResult {
            success: true,
            tx_hash: None, // Will be set by transaction signing workflow
            amount_in: params.amount_in,
            amount_out: params.min_amount_out,
            gas_used: 0, // Will be updated by transaction signing workflow
        })
    }

    fn calculate_trade_amount(&self, opportunity: &Opportunity) -> Result<u128> {
        // Calculate optimal trade amount based on opportunity and config
        let base_amount = self.config.trade_amount;
        let volatility_factor = (opportunity.volatility * 100.0).min(50.0) / 100.0; // Cap at 50%
        let liquidity_factor = (opportunity.liquidity / self.config.min_liquidity).min(2.0); // Cap at 2x
        
        let adjusted_amount = (base_amount as f64 * volatility_factor * liquidity_factor) as u128;
        Ok(adjusted_amount.max(1000)) // Minimum 1000 wei
    }

    fn calculate_min_amount_out(&self, opportunity: &Opportunity) -> Result<u128> {
        // Calculate minimum amount out with slippage protection
        let slippage_tolerance = self.config.slippage_tolerance;
        let expected_amount = opportunity.expected_profit as u128;
        
        let min_amount = expected_amount - (expected_amount * slippage_tolerance as u128 / 10000);
        Ok(min_amount.max(1)) // Minimum 1 wei
    }

    fn get_base_token_address(&self) -> Result<Address> {
        // WETH address on Ethereum mainnet
        Ok("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2".parse()?)
    }
} 
use anyhow::Result;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::time::{SystemTime, UNIX_EPOCH};
use tokio::sync::RwLock;
use tracing::{info, warn, error};

use crate::core::config::Config;
use crate::trading::TradingOpportunity;

pub mod portfolio;
pub mod simulator;
pub mod historical_data;

use portfolio::PaperPortfolio;
use simulator::TradeSimulator;
use historical_data::HistoricalDataProvider;

/// Paper trading service
pub struct PaperTradingService {
    config: Config,
    portfolio: RwLock<PaperPortfolio>,
    simulator: TradeSimulator,
    data_provider: HistoricalDataProvider,
}

impl PaperTradingService {
    pub fn new(config: Config) -> Self {
        Self {
            portfolio: RwLock::new(PaperPortfolio::new(config.paper_trading_balance)),
            simulator: TradeSimulator::new(),
            data_provider: HistoricalDataProvider::new(config.paper_trading_data_source.clone()),
            config,
        }
    }

    /// Execute a paper trade
    pub async fn execute_trade(&self, opportunity: &TradingOpportunity, amount: f64) -> Result<PaperTrade> {
        info!("Executing paper trade: {} {} at ${}", amount, opportunity.token_symbol, opportunity.current_price);

        let mut portfolio = self.portfolio.write().await;

        // Check if we have sufficient ETH balance
        if portfolio.eth_balance < amount {
            return Err(anyhow::anyhow!("Insufficient ETH balance. Available: {}, Required: {}", portfolio.eth_balance, amount));
        }

        // Simulate the trade
        let simulated_result = self.simulator.simulate_trade(opportunity, amount).await?;

        // Update portfolio
        portfolio.execute_trade(&simulated_result).await?;

        // Create trade record
        let trade = PaperTrade {
            id: format!("paper-{}", uuid::Uuid::new_v4()),
            token_address: opportunity.token_address.clone(),
            token_symbol: opportunity.token_symbol.clone(),
            trade_type: simulated_result.trade_type,
            amount_in: simulated_result.amount_in,
            amount_out: simulated_result.amount_out,
            price_per_token: simulated_result.price_per_token,
            gas_used: simulated_result.gas_used,
            gas_cost_eth: simulated_result.gas_cost_eth,
            slippage: simulated_result.slippage,
            status: PaperTradeStatus::Executed,
            executed_at: SystemTime::now(),
            block_number: simulated_result.block_number,
            transaction_hash: format!("paper-{}", simulated_result.block_number),
        };

        info!("Paper trade executed successfully: {} {} -> {} {}", 
            trade.amount_in, "ETH", trade.amount_out, trade.token_symbol);

        Ok(trade)
    }

    /// Get current portfolio balance
    pub async fn get_portfolio_balance(&self) -> Result<PaperBalance> {
        let portfolio = self.portfolio.read().await;
        let balance = portfolio.get_current_balance().await?;
        Ok(balance)
    }

    /// Get trading history
    pub async fn get_trading_history(&self, limit: Option<usize>) -> Result<Vec<PaperTrade>> {
        let portfolio = self.portfolio.read().await;
        let trades = portfolio.get_trade_history(limit).await?;
        Ok(trades)
    }

    /// Reset portfolio to initial state
    pub async fn reset_portfolio(&self) -> Result<()> {
        let mut portfolio = self.portfolio.write().await;
        portfolio.reset(self.config.paper_trading_balance).await?;
        info!("Paper trading portfolio reset to {} ETH", self.config.paper_trading_balance);
        Ok(())
    }

    /// Get portfolio performance metrics
    pub async fn get_performance_metrics(&self) -> Result<PaperPerformanceMetrics> {
        let portfolio = self.portfolio.read().await;
        let metrics = portfolio.calculate_performance_metrics().await?;
        Ok(metrics)
    }

    /// Simulate historical trading
    pub async fn simulate_historical_trading(&self, start_date: SystemTime, end_date: SystemTime) -> Result<Vec<PaperTrade>> {
        info!("Starting historical trading simulation");

        let historical_data = self.data_provider.get_historical_opportunities(start_date, end_date).await?;
        let mut simulated_trades = Vec::new();

        // Reset portfolio for clean simulation
        self.reset_portfolio().await?;

        for opportunity in historical_data {
            // Use 10% of current portfolio value per trade
            let current_balance = self.get_portfolio_balance().await?.total_value_usd;
            let trade_amount_usd = current_balance * 0.1;
            let trade_amount_eth = trade_amount_usd / 2000.0; // Assuming $2000 ETH price

            if let Ok(trade) = self.execute_trade(&opportunity, trade_amount_eth).await {
                simulated_trades.push(trade);
            }
        }

        info!("Historical simulation completed: {} trades executed", simulated_trades.len());
        Ok(simulated_trades)
    }

    /// Get paper trading statistics
    pub async fn get_statistics(&self) -> Result<PaperTradingStats> {
        let portfolio = self.portfolio.read().await;
        let balance = portfolio.get_current_balance().await?;
        let metrics = portfolio.calculate_performance_metrics().await?;
        let trades = portfolio.get_trade_history(None).await?;

        let stats = PaperTradingStats {
            total_trades: trades.len(),
            successful_trades: trades.iter().filter(|t| t.status == PaperTradeStatus::Executed).count(),
            failed_trades: trades.iter().filter(|t| t.status == PaperTradeStatus::Failed).count(),
            current_balance_eth: balance.eth_balance,
            current_balance_usd: balance.total_value_usd,
            initial_balance_eth: self.config.paper_trading_balance,
            initial_balance_usd: self.config.paper_trading_balance * 2000.0, // Assuming $2000 ETH price
            total_pnl_usd: metrics.total_pnl_usd,
            total_pnl_percentage: metrics.total_pnl_percentage,
            win_rate: metrics.win_rate,
            average_trade_return: metrics.average_trade_return,
            sharpe_ratio: metrics.sharpe_ratio,
            max_drawdown: metrics.max_drawdown,
            current_drawdown: metrics.current_drawdown,
        };

        Ok(stats)
    }
}

/// Paper trade record
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PaperTrade {
    pub id: String,
    pub token_address: String,
    pub token_symbol: String,
    pub trade_type: PaperTradeType,
    pub amount_in: f64,
    pub amount_out: f64,
    pub price_per_token: f64,
    pub gas_used: u64,
    pub gas_cost_eth: f64,
    pub slippage: f64,
    pub status: PaperTradeStatus,
    pub executed_at: SystemTime,
    pub block_number: u64,
    pub transaction_hash: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum PaperTradeType {
    Buy,
    Sell,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum PaperTradeStatus {
    Pending,
    Executed,
    Failed,
}

/// Paper portfolio balance
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PaperBalance {
    pub eth_balance: f64,
    pub token_balances: HashMap<String, f64>,
    pub total_value_usd: f64,
    pub total_value_eth: f64,
    pub last_updated: SystemTime,
}

/// Paper performance metrics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PaperPerformanceMetrics {
    pub total_pnl_usd: f64,
    pub total_pnl_percentage: f64,
    pub win_rate: f64,
    pub average_trade_return: f64,
    pub sharpe_ratio: f64,
    pub max_drawdown: f64,
    pub current_drawdown: f64,
    pub total_trades: usize,
    pub profitable_trades: usize,
    pub losing_trades: usize,
}

/// Paper trading statistics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PaperTradingStats {
    pub total_trades: usize,
    pub successful_trades: usize,
    pub failed_trades: usize,
    pub current_balance_eth: f64,
    pub current_balance_usd: f64,
    pub initial_balance_eth: f64,
    pub initial_balance_usd: f64,
    pub total_pnl_usd: f64,
    pub total_pnl_percentage: f64,
    pub win_rate: f64,
    pub average_trade_return: f64,
    pub sharpe_ratio: f64,
    pub max_drawdown: f64,
    pub current_drawdown: f64,
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::trading::TradingOpportunity;

    #[tokio::test]
    async fn test_paper_trade_execution() {
        let config = Config {
            paper_trading_balance: 10.0,
            paper_trading_data_source: "simulation".to_string(),
            ..Default::default()
        };

        let service = PaperTradingService::new(config);

        let opportunity = TradingOpportunity {
            id: "test-1".to_string(),
            token_address: "0x1234567890123456789012345678901234567890".to_string(),
            token_symbol: "TEST".to_string(),
            token_name: "Test Token".to_string(),
            current_price: 0.001,
            expected_profit: 0.01,
            liquidity: 10000.0,
            volume_24h: 50000.0,
            price_impact: 0.01,
            confidence_score: 0.85,
            discovered_at: SystemTime::now(),
            expires_at: SystemTime::now() + std::time::Duration::from_secs(300),
        };

        let trade = service.execute_trade(&opportunity, 1.0).await;
        assert!(trade.is_ok());

        let balance = service.get_portfolio_balance().await;
        assert!(balance.is_ok());
    }
}

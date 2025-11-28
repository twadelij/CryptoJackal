use anyhow::Result;
use std::time::{SystemTime, Duration};
use tracing::{info, warn};

use crate::core::config::Config;
use crate::paper_trading::PaperTradingService;
use crate::discovery::TokenDiscoveryService;
use super::BacktestResults;

/// Backtesting engine for strategy validation
pub struct BacktestingEngine {
    config: Config,
    paper_service: PaperTradingService,
    discovery_service: TokenDiscoveryService,
}

impl BacktestingEngine {
    pub fn new(config: Config) -> Self {
        Self {
            paper_service: PaperTradingService::new(config.clone()),
            discovery_service: TokenDiscoveryService::new(config.clone()),
            config,
        }
    }

    /// Run comprehensive backtest
    pub async fn run_comprehensive_backtest(&self) -> Result<BacktestResults> {
        info!("Starting comprehensive backtesting");

        let mut results = BacktestResults::new();
        
        // Test different market scenarios
        let scenarios = vec![
            ("Bull Market", 30, 1.5, 0.8),
            ("Bear Market", 30, -0.8, 1.2),
            ("Sideways Market", 30, 0.1, 1.0),
            ("High Volatility", 7, 2.0, 1.5),
            ("Low Volume", 30, 0.3, 0.5),
        ];

        for (scenario_name, days, price_trend, volatility_multiplier) in scenarios {
            info!("Testing scenario: {}", scenario_name);
            
            let scenario_result = self.run_scenario(days, price_trend, volatility_multiplier).await?;
            self.merge_scenario_results(&mut results, &scenario_result);
            results.scenarios_tested += 1;
        }

        // Calculate final metrics
        self.calculate_final_metrics(&mut results);

        info!("Backtesting completed. Win rate: {:.1}%, Total return: {:.2}%", 
              results.win_rate, results.total_return * 100.0);

        Ok(results)
    }

    /// Run individual market scenario
    async fn run_scenario(&self, days: u32, price_trend: f64, volatility_multiplier: f64) -> Result<BacktestResults> {
        let mut scenario_results = BacktestResults::new();
        scenario_results.backtest_period_days = days;

        // Reset paper trading portfolio
        self.paper_service.reset_portfolio().await?;

        // Simulate trading over the period
        let start_time = SystemTime::now() - Duration::from_secs(days as u64 * 24 * 3600);
        let end_time = SystemTime::now();

        // Generate mock trading opportunities based on scenario
        let opportunities = self.generate_scenario_opportunities(start_time, end_time, price_trend, volatility_multiplier).await?;

        let mut trades_executed = 0;
        let mut profitable_trades = 0;
        let mut losing_trades = 0;
        let mut total_pnl = 0.0;
        let mut max_drawdown = 0.0;
        let mut peak_value = self.config.paper_trading_balance;
        let mut running_returns = Vec::new();

        for opportunity in opportunities {
            // Risk 5% of current portfolio per trade
            let current_balance = self.paper_service.get_portfolio_balance().await?.total_value_usd;
            let trade_amount = current_balance * 0.05;
            let trade_amount_eth = trade_amount / 2000.0; // Assuming $2000 ETH price

            if trade_amount_eth < 0.001 {
                continue; // Skip if amount too small
            }

            // Execute trade
            match self.paper_service.execute_trade(&opportunity, trade_amount_eth).await {
                Ok(_) => {
                    trades_executed += 1;
                    
                    // Calculate P&L for this trade (simplified)
                    let expected_return = opportunity.expected_profit;
                    let actual_return = expected_return * (0.8 + (rand::random::<f64>() * 0.4)); // Add some randomness
                    
                    if actual_return > 0.0 {
                        profitable_trades += 1;
                    } else {
                        losing_trades += 1;
                    }
                    
                    total_pnl += actual_return;
                    running_returns.push(actual_return);
                    
                    // Track drawdown
                    let current_value = self.paper_service.get_portfolio_balance().await?.total_value_usd;
                    if current_value > peak_value {
                        peak_value = current_value;
                    }
                    let current_drawdown = (peak_value - current_value) / peak_value;
                    if current_drawdown > max_drawdown {
                        max_drawdown = current_drawdown;
                    }
                }
                Err(e) => {
                    warn!("Trade execution failed in backtest: {}", e);
                }
            }
        }

        // Calculate metrics
        scenario_results.total_trades = trades_executed;
        scenario_results.profitable_trades = profitable_trades;
        scenario_results.losing_trades = losing_trades;
        scenario_results.win_rate = if trades_executed > 0 {
            profitable_trades as f64 / trades_executed as f64
        } else {
            0.0
        };
        scenario_results.total_return = total_pnl / self.config.paper_trading_balance;
        scenario_results.max_drawdown = max_drawdown;
        
        // Calculate Sharpe ratio (simplified)
        if running_returns.len() > 1 {
            let mean_return = running_returns.iter().sum::<f64>() / running_returns.len() as f64;
            let variance = running_returns.iter()
                .map(|r| (r - mean_return).powi(2))
                .sum::<f64>() / (running_returns.len() - 1) as f64;
            let std_dev = variance.sqrt();
            
            if std_dev > 0.0 {
                scenario_results.sharpe_ratio = mean_return / std_dev * (365.0_f64).sqrt(); // Annualized
            }
        }

        Ok(scenario_results)
    }

    /// Generate mock trading opportunities for scenario
    async fn generate_scenario_opportunities(
        &self,
        start_time: SystemTime,
        end_time: SystemTime,
        price_trend: f64,
        volatility_multiplier: f64,
    ) -> Result<Vec<crate::trading::TradingOpportunity>> {
        let mut opportunities = Vec::new();
        let duration = end_time.duration_since(start_time)?;
        let hours = duration.as_secs() / 3600;
        
        // Generate opportunities every few hours
        for i in 0..(hours / 6) { // Every 6 hours
            let timestamp = start_time + Duration::from_secs(i * 6 * 3600);
            
            // Base price with trend and volatility
            let base_price = 0.001;
            let trend_component = price_trend * 0.001 * (i as f64 / 24.0); // Daily trend
            let volatility_component = (rand::random::<f64>() - 0.5) * 0.0005 * volatility_multiplier;
            let price = base_price + trend_component + volatility_component;
            
            // Generate opportunity with random characteristics
            let opportunity = crate::trading::TradingOpportunity {
                id: format!("backtest-{}", i),
                token_address: format!("0x{:040x}", rand::random::<u64>()),
                token_symbol: format!("TOKEN{}", i),
                token_name: format!("Backtest Token {}", i),
                current_price: price,
                expected_profit: (rand::random::<f64>() - 0.3) * 0.05, // -15% to +20%
                liquidity: 10000.0 + rand::random::<f64>() * 90000.0,
                volume_24h: 50000.0 + rand::random::<f64>() * 450000.0,
                price_impact: 0.005 + rand::random::<f64>() * 0.025,
                confidence_score: 0.6 + rand::random::<f64>() * 0.35,
                discovered_at: timestamp,
                expires_at: timestamp + Duration::from_secs(300),
            };
            
            opportunities.push(opportunity);
        }
        
        Ok(opportunities)
    }

    /// Merge scenario results into overall results
    fn merge_scenario_results(&self, overall: &mut BacktestResults, scenario: &BacktestResults) {
        overall.total_trades += scenario.total_trades;
        overall.profitable_trades += scenario.profitable_trades;
        overall.losing_trades += scenario.losing_trades;
        overall.total_return += scenario.total_return;
        overall.max_drawdown = overall.max_drawdown.max(scenario.max_drawdown);
    }

    /// Calculate final backtesting metrics
    fn calculate_final_metrics(&self, results: &mut BacktestResults) {
        if results.total_trades > 0 {
            results.win_rate = results.profitable_trades as f64 / results.total_trades as f64;
        }
        
        // Average return across scenarios
        if results.scenarios_tested > 0 {
            results.total_return /= results.scenarios_tested as f64;
        }
        
        // Simplified Sharpe ratio calculation
        if results.max_drawdown > 0.0 {
            results.sharpe_ratio = results.total_return / results.max_drawdown;
        }
    }
}

// Add rand dependency for backtesting randomness
fn rand() -> rand::Rng {
    rand::thread_rng()
}

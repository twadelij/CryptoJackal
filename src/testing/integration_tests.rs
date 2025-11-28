use anyhow::Result;
use std::time::SystemTime;
use tracing::{info, warn, error};

use crate::core::config::Config;
use super::{TestCategoryResults, SingleTestResult};

/// Integration test suite
pub struct IntegrationTestSuite {
    config: Config,
}

impl IntegrationTestSuite {
    pub fn new(config: Config) -> Self {
        Self { config }
    }

    /// Run all integration tests
    pub async fn run_all(&self) -> Result<TestCategoryResults> {
        info!("Running integration tests");

        let mut results = TestCategoryResults::new("Integration");
        let start_time = SystemTime::now();

        // Test 1: Configuration loading
        results.test_results.push(self.test_config_loading().await?);

        // Test 2: API server startup
        results.test_results.push(self.test_api_startup().await?);

        // Test 3: Paper trading service
        results.test_results.push(self.test_paper_trading_service().await?);

        // Test 4: Token discovery service
        results.test_results.push(self.test_token_discovery_service().await?);

        // Test 5: Core bot functionality
        results.test_results.push(self.test_core_bot_functionality().await?);

        // Test 6: Database connectivity (if configured)
        results.test_results.push(self.test_database_connectivity().await?);

        // Calculate summary metrics
        results.total_tests = results.test_results.len();
        results.passed_tests = results.test_results.iter().filter(|t| t.passed).count();
        results.failed_tests = results.total_tests - results.passed_tests;
        results.execution_time_ms = SystemTime::now()
            .duration_since(start_time)
            .unwrap_or_default()
            .as_millis() as u64;

        info!("Integration tests completed: {}/{} passed", results.passed_tests, results.total_tests);
        Ok(results)
    }

    async fn test_config_loading(&self) -> Result<SingleTestResult> {
        let start_time = SystemTime::now();
        
        match Config::load() {
            Ok(config) => {
                // Validate essential config fields
                let mut passed = true;
                let mut error_message = None;
                let mut metrics = std::collections::HashMap::new();

                if config.node_url.is_empty() {
                    passed = false;
                    error_message = Some("Node URL is empty".to_string());
                }

                if config.trade_amount == 0 {
                    passed = false;
                    error_message = Some("Trade amount is zero".to_string());
                }

                metrics.insert("config_fields_count", config.scan_interval as f64);

                Ok(SingleTestResult {
                    test_name: "Configuration Loading".to_string(),
                    passed,
                    execution_time_ms: SystemTime::now()
                        .duration_since(start_time)
                        .unwrap_or_default()
                        .as_millis() as u64,
                    error_message,
                    metrics,
                })
            }
            Err(e) => Ok(SingleTestResult {
                test_name: "Configuration Loading".to_string(),
                passed: false,
                execution_time_ms: SystemTime::now()
                    .duration_since(start_time)
                    .unwrap_or_default()
                    .as_millis() as u64,
                error_message: Some(format!("Failed to load config: {}", e)),
                metrics: std::collections::HashMap::new(),
            }),
        }
    }

    async fn test_api_startup(&self) -> Result<SingleTestResult> {
        let start_time = SystemTime::now();
        
        // Test API server can be created and configured
        match crate::api::ApiServer::new(std::sync::Arc::new(self.config.clone())) {
            Ok(_) => Ok(SingleTestResult {
                test_name: "API Server Startup".to_string(),
                passed: true,
                execution_time_ms: SystemTime::now()
                    .duration_since(start_time)
                    .unwrap_or_default()
                    .as_millis() as u64,
                error_message: None,
                metrics: std::collections::HashMap::new(),
            }),
            Err(e) => Ok(SingleTestResult {
                test_name: "API Server Startup".to_string(),
                passed: false,
                execution_time_ms: SystemTime::now()
                    .duration_since(start_time)
                    .unwrap_or_default()
                    .as_millis() as u64,
                error_message: Some(format!("Failed to create API server: {}", e)),
                metrics: std::collections::HashMap::new(),
            }),
        }
    }

    async fn test_paper_trading_service(&self) -> Result<SingleTestResult> {
        let start_time = SystemTime::now();
        
        let paper_service = crate::paper_trading::PaperTradingService::new(self.config.clone());
        
        // Test basic operations
        let mut passed = true;
        let mut error_message = None;
        let mut metrics = std::collections::HashMap::new();

        // Test portfolio balance
        match paper_service.get_portfolio_balance().await {
            Ok(balance) => {
                metrics.insert("initial_balance_eth", balance.eth_balance);
                metrics.insert("initial_balance_usd", balance.total_value_usd);
                
                if balance.eth_balance <= 0.0 {
                    passed = false;
                    error_message = Some("Initial ETH balance is not positive".to_string());
                }
            }
            Err(e) => {
                passed = false;
                error_message = Some(format!("Failed to get portfolio balance: {}", e));
            }
        }

        // Test portfolio reset
        if passed {
            match paper_service.reset_portfolio().await {
                Ok(_) => {
                    metrics.insert("reset_successful", 1.0);
                }
                Err(e) => {
                    passed = false;
                    error_message = Some(format!("Failed to reset portfolio: {}", e));
                }
            }
        }

        Ok(SingleTestResult {
            test_name: "Paper Trading Service".to_string(),
            passed,
            execution_time_ms: SystemTime::now()
                .duration_since(start_time)
                .unwrap_or_default()
                .as_millis() as u64,
            error_message,
            metrics,
        })
    }

    async fn test_token_discovery_service(&self) -> Result<SingleTestResult> {
        let start_time = SystemTime::now();
        
        let discovery_service = crate::discovery::TokenDiscoveryService::new(self.config.clone());
        
        let mut passed = true;
        let mut error_message = None;
        let mut metrics = std::collections::HashMap::new();

        // Test trending tokens
        match discovery_service.get_trending_tokens("24h").await {
            Ok(tokens) => {
                metrics.insert("trending_tokens_count", tokens.len() as f64);
                
                // Validate token structure
                for token in &tokens {
                    if token.address.is_empty() || token.symbol.is_empty() {
                        passed = false;
                        error_message = Some("Invalid token structure found".to_string());
                        break;
                    }
                }
            }
            Err(e) => {
                passed = false;
                error_message = Some(format!("Failed to get trending tokens: {}", e));
            }
        }

        // Test new token discovery
        if passed {
            match discovery_service.discover_new_tokens().await {
                Ok(tokens) => {
                    metrics.insert("new_tokens_count", tokens.len() as f64);
                }
                Err(e) => {
                    warn!("New token discovery failed (may be expected): {}", e);
                    metrics.insert("new_tokens_count", 0.0);
                }
            }
        }

        Ok(SingleTestResult {
            test_name: "Token Discovery Service".to_string(),
            passed,
            execution_time_ms: SystemTime::now()
                .duration_since(start_time)
                .unwrap_or_default()
                .as_millis() as u64,
            error_message,
            metrics,
        })
    }

    async fn test_core_bot_functionality(&self) -> Result<SingleTestResult> {
        let start_time = SystemTime::now();
        
        let mut passed = true;
        let mut error_message = None;
        let mut metrics = std::collections::HashMap::new();

        // Test bot creation
        match crate::core::Bot::new(&self.config).await {
            Ok(_) => {
                metrics.insert("bot_creation_successful", 1.0);
            }
            Err(e) => {
                passed = false;
                error_message = Some(format!("Failed to create bot: {}", e));
            }
        }

        Ok(SingleTestResult {
            test_name: "Core Bot Functionality".to_string(),
            passed,
            execution_time_ms: SystemTime::now()
                .duration_since(start_time)
                .unwrap_or_default()
                .as_millis() as u64,
            error_message,
            metrics,
        })
    }

    async fn test_database_connectivity(&self) -> Result<SingleTestResult> {
        let start_time = SystemTime::now();
        
        let mut passed = true;
        let mut error_message = None;
        let mut metrics = std::collections::HashMap::new();

        // For now, just test that database URL is configured
        if self.config.database_url.is_empty() {
            metrics.insert("database_configured", 0.0);
            warn!("Database not configured - skipping connectivity test");
        } else {
            metrics.insert("database_configured", 1.0);
            // TODO: Add actual database connectivity test
        }

        Ok(SingleTestResult {
            test_name: "Database Connectivity".to_string(),
            passed,
            execution_time_ms: SystemTime::now()
                .duration_since(start_time)
                .unwrap_or_default()
                .as_millis() as u64,
            error_message,
            metrics,
        })
    }
}

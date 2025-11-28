use anyhow::Result;
use std::time::SystemTime;
use tracing::{info, warn};

use crate::core::config::Config;
use super::{TestCategoryResults, SingleTestResult};

/// Security test suite
pub struct SecurityTestSuite {
    config: Config,
}

impl SecurityTestSuite {
    pub fn new(config: Config) -> Self {
        Self { config }
    }

    /// Run all security tests
    pub async fn run_all(&self) -> Result<TestCategoryResults> {
        info!("Running security tests");

        let mut results = TestCategoryResults::new("Security");
        let start_time = SystemTime::now();

        // Test 1: Configuration security
        results.test_results.push(self.test_configuration_security().await?);

        // Test 2: API security
        results.test_results.push(self.test_api_security().await?);

        // Test 3: Paper trading security
        results.test_results.push(self.test_paper_trading_security().await?);

        // Test 4: Token discovery security
        results.test_results.push(self.test_token_discovery_security().await?);

        // Test 5: Input validation
        results.test_results.push(self.test_input_validation().await?);

        // Calculate summary metrics
        results.total_tests = results.test_results.len();
        results.passed_tests = results.test_results.iter().filter(|t| t.passed).count();
        results.failed_tests = results.total_tests - results.passed_tests;
        results.execution_time_ms = SystemTime::now()
            .duration_since(start_time)
            .unwrap_or_default()
            .as_millis() as u64;

        info!("Security tests completed: {}/{} passed", results.passed_tests, results.total_tests);
        Ok(results)
    }

    async fn test_configuration_security(&self) -> Result<SingleTestResult> {
        let start_time = SystemTime::now();
        let mut metrics = std::collections::HashMap::new();
        let mut passed = true;
        let mut error_message = None;

        // Check for default secrets
        if self.config.jwt_secret == "default-secret-change-in-production" {
            passed = false;
            error_message = Some("Default JWT secret detected".to_string());
        }

        // Check for empty API keys in production
        if self.config.environment == "production" {
            if self.config.coingecko_api_key.is_none() {
                warn!("No CoinGecko API key in production");
            }
        }

        metrics.insert("jwt_secret_length", self.config.jwt_secret.len() as f64);
        metrics.insert("paper_trading_enabled", self.config.paper_trading_mode as u8 as f64);

        Ok(SingleTestResult {
            test_name: "Configuration Security".to_string(),
            passed,
            execution_time_ms: SystemTime::now()
                .duration_since(start_time)
                .unwrap_or_default()
                .as_millis() as u64,
            error_message,
            metrics,
        })
    }

    async fn test_api_security(&self) -> Result<SingleTestResult> {
        let start_time = SystemTime::now();
        let mut metrics = std::collections::HashMap::new();

        // Test API server configuration for security
        let api_server = crate::api::ApiServer::new(std::sync::Arc::new(self.config.clone()));
        
        let mut passed = true;
        let mut error_message = None;

        // Check CORS configuration
        if self.config.cors_origins.is_empty() {
            passed = false;
            error_message = Some("No CORS origins configured".to_string());
        }

        metrics.insert("cors_origins_count", self.config.cors_origins.len() as f64);
        metrics.insert("api_port_configured", self.config.api_port as f64);

        Ok(SingleTestResult {
            test_name: "API Security".to_string(),
            passed,
            execution_time_ms: SystemTime::now()
                .duration_since(start_time)
                .unwrap_or_default()
                .as_millis() as u64,
            error_message,
            metrics,
        })
    }

    async fn test_paper_trading_security(&self) -> Result<SingleTestResult> {
        let start_time = SystemTime::now();
        let mut metrics = std::collections::HashMap::new();

        let paper_service = crate::paper_trading::PaperTradingService::new(self.config.clone());
        
        let mut passed = true;
        let mut error_message = None;

        // Test paper trading isolation
        let balance = paper_service.get_portfolio_balance().await?;
        
        // Ensure paper trading doesn't use real money
        if self.config.paper_trading_mode {
            metrics.insert("paper_trading_enabled", 1.0);
            
            // Check that balance is reasonable
            if balance.eth_balance > 1000.0 {
                passed = false;
                error_message = Some("Paper trading balance seems too high".to_string());
            }
        } else {
            metrics.insert("paper_trading_enabled", 0.0);
        }

        metrics.insert("paper_balance_eth", balance.eth_balance);

        Ok(SingleTestResult {
            test_name: "Paper Trading Security".to_string(),
            passed,
            execution_time_ms: SystemTime::now()
                .duration_since(start_time)
                .unwrap_or_default()
                .as_millis() as u64,
            error_message,
            metrics,
        })
    }

    async fn test_token_discovery_security(&self) -> Result<SingleTestResult> {
        let start_time = SystemTime::now();
        let mut metrics = std::collections::HashMap::new();

        let discovery_service = crate::discovery::TokenDiscoveryService::new(self.config.clone());
        
        let mut passed = true;
        let mut error_message = None;

        // Test security analysis
        let mock_token = crate::discovery::DiscoveredToken {
            address: "0x1234567890123456789012345678901234567890".to_string(),
            symbol: "TEST".to_string(),
            name: "Test Token".to_string(),
            price: 0.001,
            market_cap: None,
            liquidity: 10000.0,
            volume_24h: Some(50000.0),
            price_change_24h: Some(0.05),
            security_score: 0.5,
            discovered_at: SystemTime::now(),
            tags: vec![],
            dex_info: crate::discovery::DexInfo {
                pair_address: "0x1234567890123456789012345678901234567890".to_string(),
                base_token: crate::discovery::TokenInfo {
                    address: "0x1234567890123456789012345678901234567890".to_string(),
                    name: "Test Token".to_string(),
                    symbol: "TEST".to_string(),
                },
                quote_token: crate::discovery::TokenInfo {
                    address: "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2".to_string(),
                    name: "Wrapped Ether".to_string(),
                    symbol: "WETH".to_string(),
                },
                dex_id: "uniswap_v2".to_string(),
                chain_id: "1".to_string(),
            },
        };

        let security_score = discovery_service.security.analyze_token(&mock_token).await?;
        metrics.insert("security_score", security_score);

        // Security score should be between 0 and 1
        if security_score < 0.0 || security_score > 1.0 {
            passed = false;
            error_message = Some("Security score out of valid range".to_string());
        }

        Ok(SingleTestResult {
            test_name: "Token Discovery Security".to_string(),
            passed,
            execution_time_ms: SystemTime::now()
                .duration_since(start_time)
                .unwrap_or_default()
                .as_millis() as u64,
            error_message,
            metrics,
        })
    }

    async fn test_input_validation(&self) -> Result<SingleTestResult> {
        let start_time = SystemTime::now();
        let mut metrics = std::collections::HashMap::new();
        let mut passed = true;
        let mut error_message = None;

        // Test various input validation scenarios
        
        // 1. Test invalid token addresses
        let invalid_addresses = vec![
            "", // Empty
            "0x", // Incomplete
            "invalid", // Invalid format
            "0x123", // Too short
        ];

        let mut invalid_address_count = 0;
        for address in invalid_addresses {
            if !self.is_valid_address(address) {
                invalid_address_count += 1;
            }
        }

        metrics.insert("invalid_addresses_detected", invalid_address_count as f64);

        // 2. Test configuration bounds
        let config_bounds_ok = self.config.scan_interval > 0 && 
                             self.config.gas_limit > 0 &&
                             self.config.trade_amount > 0;

        if !config_bounds_ok {
            passed = false;
            error_message = Some("Invalid configuration bounds detected".to_string());
        }

        metrics.insert("config_bounds_valid", config_bounds_ok as u8 as f64);

        Ok(SingleTestResult {
            test_name: "Input Validation".to_string(),
            passed,
            execution_time_ms: SystemTime::now()
                .duration_since(start_time)
                .unwrap_or_default()
                .as_millis() as u64,
            error_message,
            metrics,
        })
    }

    /// Simple address validation (simplified)
    fn is_valid_address(&self, address: &str) -> bool {
        if address.len() != 42 {
            return false;
        }
        
        if !address.starts_with("0x") {
            return false;
        }
        
        address[2..].chars().all(|c| c.is_ascii_hexdigit())
    }
}

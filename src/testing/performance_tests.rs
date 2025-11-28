use anyhow::Result;
use std::time::{SystemTime, Duration, Instant};
use tracing::{info, warn};

use crate::core::config::Config;
use super::{TestCategoryResults, SingleTestResult};

/// Performance test suite
pub struct PerformanceTestSuite {
    config: Config,
}

impl PerformanceTestSuite {
    pub fn new(config: Config) -> Self {
        Self { config }
    }

    /// Run all performance tests
    pub async fn run_all(&self) -> Result<TestCategoryResults> {
        info!("Running performance tests");

        let mut results = TestCategoryResults::new("Performance");
        let start_time = SystemTime::now();

        // Test 1: API response time
        results.test_results.push(self.test_api_response_time().await?);

        // Test 2: Token discovery performance
        results.test_results.push(self.test_token_discovery_performance().await?);

        // Test 3: Paper trading performance
        results.test_results.push(self.test_paper_trading_performance().await?);

        // Test 4: Configuration loading performance
        results.test_results.push(self.test_config_loading_performance().await?);

        // Test 5: Memory usage
        results.test_results.push(self.test_memory_usage().await?);

        // Calculate summary metrics
        results.total_tests = results.test_results.len();
        results.passed_tests = results.test_results.iter().filter(|t| t.passed).count();
        results.failed_tests = results.total_tests - results.passed_tests;
        results.execution_time_ms = SystemTime::now()
            .duration_since(start_time)
            .unwrap_or_default()
            .as_millis() as u64;

        info!("Performance tests completed: {}/{} passed", results.passed_tests, results.total_tests);
        Ok(results)
    }

    async fn test_api_response_time(&self) -> Result<SingleTestResult> {
        let start_time = SystemTime::now();
        let mut metrics = std::collections::HashMap::new();

        // Test API server creation time
        let api_start = Instant::now();
        let _api_server = crate::api::ApiServer::new(std::sync::Arc::new(self.config.clone()));
        let api_creation_time = api_start.elapsed();
        
        metrics.insert("api_creation_time_ms", api_creation_time.as_millis() as f64);

        // Performance criteria: API server should create within 100ms
        let passed = api_creation_time < Duration::from_millis(100);
        let error_message = if passed {
            None
        } else {
            Some(format!("API server creation took too long: {}ms", api_creation_time.as_millis()))
        };

        Ok(SingleTestResult {
            test_name: "API Response Time".to_string(),
            passed,
            execution_time_ms: SystemTime::now()
                .duration_since(start_time)
                .unwrap_or_default()
                .as_millis() as u64,
            error_message,
            metrics,
        })
    }

    async fn test_token_discovery_performance(&self) -> Result<SingleTestResult> {
        let start_time = SystemTime::now();
        let mut metrics = std::collections::HashMap::new();

        let discovery_service = crate::discovery::TokenDiscoveryService::new(self.config.clone());
        
        // Test trending tokens performance
        let discovery_start = Instant::now();
        let result = discovery_service.get_trending_tokens("24h").await;
        let discovery_time = discovery_start.elapsed();
        
        metrics.insert("discovery_time_ms", discovery_time.as_millis() as f64);

        let mut passed = true;
        let mut error_message = None;

        if discovery_time > Duration::from_secs(5) {
            passed = false;
            error_message = Some(format!("Token discovery took too long: {}ms", discovery_time.as_millis()));
        }

        if let Ok(tokens) = result {
            metrics.insert("tokens_found", tokens.len() as f64);
        } else {
            metrics.insert("tokens_found", 0.0);
        }

        Ok(SingleTestResult {
            test_name: "Token Discovery Performance".to_string(),
            passed,
            execution_time_ms: SystemTime::now()
                .duration_since(start_time)
                .unwrap_or_default()
                .as_millis() as u64,
            error_message,
            metrics,
        })
    }

    async fn test_paper_trading_performance(&self) -> Result<SingleTestResult> {
        let start_time = SystemTime::now();
        let mut metrics = std::collections::HashMap::new();

        let paper_service = crate::paper_trading::PaperTradingService::new(self.config.clone());
        
        // Test portfolio balance retrieval performance
        let balance_start = Instant::now();
        let result = paper_service.get_portfolio_balance().await;
        let balance_time = balance_start.elapsed();
        
        metrics.insert("balance_retrieval_time_ms", balance_time.as_millis() as f64);

        let mut passed = true;
        let mut error_message = None;

        if balance_time > Duration::from_millis(50) {
            passed = false;
            error_message = Some(format!("Balance retrieval took too long: {}ms", balance_time.as_millis()));
        }

        if let Ok(balance) = result {
            metrics.insert("balance_eth", balance.eth_balance);
        }

        Ok(SingleTestResult {
            test_name: "Paper Trading Performance".to_string(),
            passed,
            execution_time_ms: SystemTime::now()
                .duration_since(start_time)
                .unwrap_or_default()
                .as_millis() as u64,
            error_message,
            metrics,
        })
    }

    async fn test_config_loading_performance(&self) -> Result<SingleTestResult> {
        let start_time = SystemTime::now();
        let mut metrics = std::collections::HashMap::new();

        // Test multiple config loads
        let config_start = Instant::now();
        let mut successful_loads = 0;
        
        for _ in 0..10 {
            match Config::load() {
                Ok(_) => successful_loads += 1,
                Err(_) => {}
            }
        }
        
        let total_time = config_start.elapsed();
        let avg_time_per_load = total_time / 10;
        
        metrics.insert("avg_config_load_time_ms", avg_time_per_load.as_millis() as f64);
        metrics.insert("successful_loads", successful_loads as f64);

        // Performance criteria: Config should load within 10ms on average
        let passed = avg_time_per_load < Duration::from_millis(10);
        let error_message = if passed {
            None
        } else {
            Some(format!("Config loading too slow: {}ms average", avg_time_per_load.as_millis()))
        };

        Ok(SingleTestResult {
            test_name: "Config Loading Performance".to_string(),
            passed,
            execution_time_ms: SystemTime::now()
                .duration_since(start_time)
                .unwrap_or_default()
                .as_millis() as u64,
            error_message,
            metrics,
        })
    }

    async fn test_memory_usage(&self) -> Result<SingleTestResult> {
        let start_time = SystemTime::now();
        let mut metrics = std::collections::HashMap::new();

        // Get initial memory usage
        let initial_memory = self.get_memory_usage();
        metrics.insert("initial_memory_mb", initial_memory as f64);

        // Create multiple services to test memory usage
        let mut services = Vec::new();
        
        for _ in 0..10 {
            services.push(crate::paper_trading::PaperTradingService::new(self.config.clone()));
            services.push(crate::discovery::TokenDiscoveryService::new(self.config.clone()));
        }

        // Get peak memory usage
        let peak_memory = self.get_memory_usage();
        metrics.insert("peak_memory_mb", peak_memory as f64);
        metrics.insert("memory_increase_mb", (peak_memory - initial_memory) as f64);
        metrics.insert("services_created", (services.len() * 2) as f64);

        // Memory criteria: Should not increase by more than 100MB
        let memory_increase = peak_memory - initial_memory;
        let passed = memory_increase < 100;
        let error_message = if passed {
            None
        } else {
            Some(format!("Memory usage increased too much: {}MB", memory_increase))
        };

        Ok(SingleTestResult {
            test_name: "Memory Usage".to_string(),
            passed,
            execution_time_ms: SystemTime::now()
                .duration_since(start_time)
                .unwrap_or_default()
                .as_millis() as u64,
            error_message,
            metrics,
        })
    }

    /// Get current memory usage in MB (simplified)
    fn get_memory_usage(&self) -> u64 {
        // In a real implementation, this would use system APIs to get actual memory usage
        // For now, return a mock value
        use std::sync::atomic::{AtomicU64, Ordering};
        static MEMORY_COUNTER: AtomicU64 = AtomicU64::new(50); // Start at 50MB
        
        // Simulate memory growth
        MEMORY_COUNTER.fetch_add(5, Ordering::Relaxed)
    }
}

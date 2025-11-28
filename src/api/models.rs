use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};
use std::collections::HashMap;

/// Standard API response wrapper
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApiResponse<T> {
    pub success: bool,
    pub data: Option<T>,
    pub error: Option<String>,
    pub timestamp: DateTime<Utc>,
}

impl<T> ApiResponse<T> {
    pub fn success(data: T) -> Self {
        Self {
            success: true,
            data: Some(data),
            error: None,
            timestamp: Utc::now(),
        }
    }

    pub fn error(error: String) -> Self {
        Self {
            success: false,
            data: None,
            error: Some(error),
            timestamp: Utc::now(),
        }
    }
}

/// Health check response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HealthResponse {
    pub status: String,
    pub timestamp: String,
    pub version: String,
}

/// Bot status response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BotStatusResponse {
    pub running: bool,
    pub uptime_seconds: Option<u64>,
    pub last_trade: Option<DateTime<Utc>>,
    pub total_trades: u64,
    pub successful_trades: u64,
    pub failed_trades: u64,
    pub current_mode: String, // "live", "paper", "stopped"
}

/// Configuration response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConfigResponse {
    pub trading_parameters: TradingParameters,
    pub api_settings: ApiSettings,
    pub paper_trading: PaperTradingConfig,
    pub discovery_settings: DiscoverySettings,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TradingParameters {
    pub scan_interval: u64,
    pub gas_limit: u64,
    pub slippage_tolerance: f64,
    pub min_liquidity: f64,
    pub max_price_impact: f64,
    pub trade_amount: String,
    pub target_tokens: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApiSettings {
    pub api_host: String,
    pub api_port: u16,
    pub cors_origins: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PaperTradingConfig {
    pub enabled: bool,
    pub balance: f64,
    pub data_source: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DiscoverySettings {
    pub scan_interval: u64,
    pub max_new_tokens: usize,
    pub security_checks_enabled: bool,
}

/// Trading opportunity
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

/// Trade execution request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TradeExecutionRequest {
    pub token_address: String,
    pub amount: String,
    pub slippage_tolerance: Option<f64>,
    pub gas_limit: Option<u64>,
}

/// Trade execution response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TradeExecutionResponse {
    pub trade_id: String,
    pub status: String, // "pending", "executed", "failed"
    pub transaction_hash: Option<String>,
    pub amount_in: String,
    pub amount_out: Option<String>,
    pub gas_used: Option<u64>,
    pub gas_price: Option<String>,
    pub executed_at: Option<DateTime<Utc>>,
    pub error_message: Option<String>,
}

/// Trading history response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TradingHistoryResponse {
    pub trades: Vec<TradeExecutionResponse>,
    pub total_pages: u32,
    pub current_page: u32,
    pub total_trades: u64,
}

/// Paper trading balance
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PaperBalanceResponse {
    pub eth_balance: f64,
    pub token_balances: HashMap<String, f64>,
    pub total_value_usd: f64,
    pub pnl_24h: f64,
    pub pnl_total: f64,
    pub last_updated: DateTime<Utc>,
}

/// Discovered token
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DiscoveredToken {
    pub address: String,
    pub symbol: String,
    pub name: String,
    pub price: f64,
    pub market_cap: f64,
    pub liquidity: f64,
    pub volume_24h: f64,
    pub price_change_24h: f64,
    pub security_score: f64,
    pub discovered_at: DateTime<Utc>,
    pub tags: Vec<String>,
}

/// Trending tokens response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TrendingTokensResponse {
    pub tokens: Vec<DiscoveredToken>,
    pub time_window: String, // "1h", "24h", "7d"
    pub updated_at: DateTime<Utc>,
}

/// Metrics response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MetricsResponse {
    pub system_metrics: SystemMetrics,
    pub trading_metrics: TradingMetrics,
    pub performance_metrics: PerformanceMetrics,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SystemMetrics {
    pub uptime_seconds: u64,
    pub memory_usage_mb: f64,
    pub cpu_usage_percent: f64,
    pub active_connections: u32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TradingMetrics {
    pub total_trades: u64,
    pub successful_trades: u64,
    pub failed_trades: u64,
    pub success_rate: f64,
    pub average_execution_time_ms: f64,
    pub total_volume_eth: f64,
    pub total_profit_eth: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PerformanceMetrics {
    pub api_requests_per_second: f64,
    pub average_response_time_ms: f64,
    pub error_rate_percent: f64,
    pub cache_hit_rate: f64,
}

/// Configuration update request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConfigUpdateRequest {
    pub trading_parameters: Option<TradingParameters>,
    pub paper_trading: Option<PaperTradingConfig>,
    pub discovery_settings: Option<DiscoverySettings>,
}

use anyhow::Result;
use axum::{
    extract::State,
    http::StatusCode,
    response::Json,
    routing::{get, post},
    Router,
};
use serde::{Deserialize, Serialize};
use std::net::SocketAddr;
use std::sync::Arc;
use tokio::sync::RwLock;
use tower::ServiceBuilder;
use tower_http::{
    cors::{Any, CorsLayer},
    trace::TraceLayer,
};
use tracing::{info, error};

use crate::core::{config::Config, Bot};
use crate::discovery::TokenDiscoveryService;
use crate::paper_trading::PaperTradingService;

pub mod handlers;
pub mod models;
pub mod setup;
pub mod middleware;

use handlers::*;
use models::*;

/// API Server state
#[derive(Clone)]
pub struct ApiState {
    pub config: Arc<Config>,
    pub bot: Arc<RwLock<Option<Bot>>>,
    pub discovery_service: Arc<TokenDiscoveryService>,
    pub paper_trading_service: Arc<PaperTradingService>,
}

/// API Server
pub struct ApiServer {
    config: Arc<Config>,
    app: Router,
}

impl ApiServer {
    pub fn new(config: Arc<Config>) -> Self {
        let discovery_service = Arc::new(TokenDiscoveryService::new((*config).clone()));
        let paper_trading_service = Arc::new(PaperTradingService::new((*config).clone()));

        let state = ApiState {
            config: config.clone(),
            bot: Arc::new(RwLock::new(None)),
            discovery_service,
            paper_trading_service,
        };

        let app = Router::new()
            // Health check endpoint
            .route("/", get(health_check))
            .route("/health", get(health_check))
            .route("/api/health", get(health_check))
            
            // Bot control endpoints
            .route("/api/bot/status", get(get_bot_status))
            .route("/api/bot/start", post(start_bot))
            .route("/api/bot/stop", post(stop_bot))
            
            // Configuration endpoints
            .route("/api/config", get(get_config))
            .route("/api/config", post(update_config))
            
            // Trading endpoints
            .route("/api/trading/opportunities", get(get_opportunities))
            .route("/api/trading/execute", post(execute_trade))
            .route("/api/trading/history", get(get_trading_history))
            
            // Paper trading endpoints
            .route("/api/paper-trading/balance", get(get_paper_balance))
            .route("/api/paper-trading/execute", post(execute_paper_trade))
            .route("/api/paper-trading/reset", post(reset_paper_portfolio))
            .route("/api/paper-trading/history", get(get_paper_history))
            .route("/api/paper-trading/performance", get(get_paper_performance))
            
            // Token discovery endpoints
            .route("/api/discovery/trending", get(get_trending_tokens))
            .route("/api/discovery/new", get(get_new_tokens))
            .route("/api/discovery/analyze/:address", get(analyze_token))
            
            // Metrics and monitoring endpoints
            .route("/api/metrics", get(get_metrics))
            .route("/api/metrics/performance", get(get_performance_metrics))
            
            // Setup wizard endpoints
            .nest_service("/api/setup", setup::create_setup_routes())
            
            // Apply middleware
            .layer(
                ServiceBuilder::new()
                    .layer(TraceLayer::new_for_http())
                    .layer(CorsLayer::new().allow_origin(Any).allow_methods(Any).allow_headers(Any))
            )
            .with_state(state);

        Self { config, app }
    }

    pub async fn run(self) -> Result<()> {
        let addr: SocketAddr = format!("{}:{}", self.config.api_host, self.config.api_port)
            .parse()
            .map_err(|e| anyhow::anyhow!("Invalid bind address: {}", e))?;

        info!("Starting API server on {}", addr);

        let listener = tokio::net::TcpListener::bind(addr).await?;
        axum::serve(listener, self.app)
            .await
            .map_err(|e| anyhow::anyhow!("API server failed: {}", e))?;

        Ok(())
    }
}

// Health check handler
async fn health_check() -> Result<Json<HealthResponse>, StatusCode> {
    Ok(Json(HealthResponse {
        status: "healthy".to_string(),
        timestamp: chrono::Utc::now().to_rfc3339(),
        version: env!("CARGO_PKG_VERSION").to_string(),
    }))
}

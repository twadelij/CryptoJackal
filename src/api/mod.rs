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

pub mod handlers;
pub mod middleware;
pub mod models;

use handlers::*;
use models::*;

/// API Server state
#[derive(Clone)]
pub struct ApiState {
    pub config: Arc<Config>,
    pub bot: Arc<RwLock<Option<Bot>>>,
}

/// API Server
pub struct ApiServer {
    config: Arc<Config>,
    app: Router,
}

impl ApiServer {
    pub fn new(config: Arc<Config>) -> Self {
        let state = ApiState {
            config: config.clone(),
            bot: Arc::new(RwLock::new(None)),
        };

        let app = Router::new()
            // Health check endpoint
            .route("/health", get(health_check))
            
            // Bot control endpoints
            .route("/api/bot/status", get(bot_status))
            .route("/api/bot/start", post(start_bot))
            .route("/api/bot/stop", post(stop_bot))
            
            // Configuration endpoints
            .route("/api/config", get(get_config))
            .route("/api/config", post(update_config))
            
            // Trading endpoints
            .route("/api/trading/opportunities", get(get_opportunities))
            .route("/api/trading/execute", post(execute_trade))
            .route("/api/trading/history", get(trading_history))
            
            // Paper trading endpoints
            .route("/api/paper-trading/balance", get(paper_balance))
            .route("/api/paper-trading/reset", post(reset_paper_balance))
            
            // Token discovery endpoints
            .route("/api/discovery/tokens", get(discover_tokens))
            .route("/api/discovery/trending", get(trending_tokens))
            
            // Metrics endpoints
            .route("/api/metrics", get(get_metrics))
            
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

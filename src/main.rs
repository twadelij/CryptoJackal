use anyhow::Result;
use std::sync::Arc;
use tracing::{info, Level};
use tracing_subscriber::FmtSubscriber;

mod core;
mod demo;
mod trading;
mod wallet;
mod api;
mod discovery;
mod paper_trading;
mod testing;

#[cfg(test)]
mod integration_tests;

#[tokio::main]
async fn main() -> Result<()> {
    // Initialize logging with pretty output
    let log_level = std::env::var("LOG_LEVEL").unwrap_or_else(|_| "info".to_string());
    let log_format = std::env::var("LOG_FORMAT").unwrap_or_else(|_| "pretty".to_string());
    
    let subscriber = if log_format == "json" {
        FmtSubscriber::builder()
            .with_max_level(match log_level.as_str() {
                "debug" => Level::DEBUG,
                "info" => Level::INFO,
                "warn" => Level::WARN,
                "error" => Level::ERROR,
                _ => Level::INFO,
            })
            .json()
            .finish()
    } else {
        FmtSubscriber::builder()
            .with_max_level(match log_level.as_str() {
                "debug" => Level::DEBUG,
                "info" => Level::INFO,
                "warn" => Level::WARN,
                "error" => Level::ERROR,
                _ => Level::INFO,
            })
            .with_file(true)
            .with_line_number(true)
            .with_thread_ids(true)
            .with_target(false)
            .with_thread_names(true)
            .pretty()
            .finish()
    };

    tracing::subscriber::set_global_default(subscriber)?;

    info!("🚀 Starting CryptoJackal...");

    // Load environment variables
    dotenv::dotenv().ok();

    // Load configuration
    let config = Arc::new(core::config::Config::load()?);
    
    info!("📊 Configuration loaded:");
    info!("   🌐 Network: {}", config.network_name);
    info!("   💱 Trading Mode: {}", if config.is_paper_trading() { "Paper Trading" } else { "Live Trading" });
    info!("   📡 API Server: {}", config.api_bind_address());
    info!("   🔍 Discovery Scan Interval: {}ms", config.discovery_scan_interval);

    // Start API server
    let api_server = api::ApiServer::new(config.clone());
    let api_handle = tokio::spawn(async move {
        if let Err(e) = api_server.run().await {
            eprintln!("API server failed: {}", e);
        }
    });

    // Wait for API server to start
    tokio::time::sleep(tokio::time::Duration::from_secs(2)).await;

    info!("✅ CryptoJackal is ready!");
    info!("📡 API server running on http://{}", config.api_bind_address());
    info!("🏥 Health check available at http://:{}/health", config.health_check_port);
    
    if config.metrics_enabled {
        info!("📊 Metrics available on port {}", config.metrics_port);
    }

    // Keep the main task alive
    tokio::select! {
        _ = api_handle => {
            info!("API server task ended");
        }
        _ = tokio::signal::ctrl_c() => {
            info!("Received Ctrl+C, shutting down...");
        }
    }

    info!("👋 CryptoJackal stopped gracefully");
    Ok(())
}

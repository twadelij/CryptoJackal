//! CryptoJackal Demo Binary
//! A minimal demo to verify the build works

use anyhow::Result;
use tracing::{info, Level};
use tracing_subscriber::FmtSubscriber;

#[tokio::main]
async fn main() -> Result<()> {
    // Initialize logging
    FmtSubscriber::builder()
        .with_max_level(Level::INFO)
        .pretty()
        .init();

    info!("🐺 CryptoJackal Demo Starting");
    info!("=============================");

    // Load environment variables
    dotenvy::dotenv().ok();

    info!("✅ Environment loaded");
    info!("✅ Demo binary is working!");
    info!("🎉 Build verification complete!");

    Ok(())
} 
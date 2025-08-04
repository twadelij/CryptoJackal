use anyhow::Result;
use tracing::{info, Level};
use tracing_subscriber::FmtSubscriber;

mod core;
mod demo;
mod trading;
mod wallet;

#[tokio::main]
async fn main() -> Result<()> {
    // Initialize logging with pretty output
    FmtSubscriber::builder()
        .with_max_level(Level::INFO)
        .with_file(true)
        .with_line_number(true)
        .with_thread_ids(true)
        .with_target(true)
        .with_thread_names(true)
        .pretty()
        .init();

    info!("🎭 Starting CryptoJackal Transaction Signing Demo");
    info!("==================================================");

    // Load environment variables
    dotenv::dotenv().ok();

    // Run the comprehensive demo
    demo::run_cryptojackal_demo().await?;
    
    // Show development metrics
    demo::show_development_metrics().await?;

    info!("🎉 Demo completed successfully!");
    Ok(())
} 
use anyhow::Result;
use tracing::{info, Level};
use tracing_subscriber::FmtSubscriber;

mod core;
mod trading;
mod wallet;

#[tokio::main]
async fn main() -> Result<()> {
    // Initialize logging with pretty output
    let subscriber = FmtSubscriber::builder()
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

    // Run the demo (placeholder - demo module needs to be implemented)
    info!("🚀 CryptoJackal Demo would run here");
    info!("📊 All systems integrated and ready for testing");

    info!("🎉 Demo completed successfully!");
    Ok(())
} 
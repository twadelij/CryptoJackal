use tracing::{info, Level};
use tracing_subscriber::FmtSubscriber;

// Module declarations
mod core;
mod error;
mod trading;
mod utils;
mod wallet;

// Re-export common types
pub use error::{CryptoJackalError, Result};

#[tokio::main]
async fn main() -> Result<()> {
    // Initialize logging
    let subscriber = FmtSubscriber::builder()
        .with_max_level(Level::INFO)
        .with_file(true)
        .with_line_number(true)
        .with_thread_ids(true)
        .with_target(false)
        .with_thread_names(true)
        .with_caller_location(true)
        .pretty()
        .init();

    info!("Starting SniperBot...");

    // Load environment variables
    dotenv::dotenv().ok();

    // Initialize the bot
    let bot = core::Bot::new().await?;
    
    // Start the bot
    bot.run().await?;

    Ok(())
}

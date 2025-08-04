use anyhow::Result;
use tracing::{info, Level};
use tracing_subscriber::FmtSubscriber;

mod core;
mod demo;
mod trading;
mod wallet;

#[cfg(test)]
mod integration_tests;

#[tokio::main]
async fn main() -> Result<()> {
    // Initialize logging with pretty output
    FmtSubscriber::builder()
        .with_max_level(Level::INFO)
        .with_file(true)
        .with_line_number(true)
        .with_thread_ids(true)
        .with_target(false)
        .with_thread_names(true)
        .with_target(true)
        .pretty()
        .init();

    info!("Starting CryptoJackal...");

    // Load environment variables
    dotenv::dotenv().ok();

    // Load configuration
    let config = core::config::Config::load()?;
    
    // Initialize the bot
    let bot = core::Bot::new(&config).await?;
    
    // Start the bot
    bot.run().await?;

    Ok(())
}

//! CryptoJackal - High-Performance Cryptocurrency Sniper Bot
//! 
//! CryptoJackal is a Rust-based cryptocurrency sniper bot designed for high-frequency
//! trading on decentralized exchanges, with a focus on Uniswap V2/V3.
//! 
//! ## Features
//! 
//! - **High Performance**: Built in Rust for maximum speed and efficiency
//! - **MetaMask Integration**: Secure wallet connection without private key storage
//! - **Real-time Monitoring**: WebSocket connections to DEX subgraphs
//! - **Risk Management**: Built-in stop-loss and take-profit mechanisms
//! - **Gamification**: Achievement system and performance tracking
//! 
//! ## Architecture
//! 
//! The bot is organized into several core modules:
//! 
//! - [`core`]: Main bot logic and orchestration
//! - [`wallet`]: MetaMask integration and transaction signing
//! - [`trading`]: DEX interaction and trade execution
//! - [`utils`]: Common utilities and helper functions
//! - [`error`]: Error handling and custom error types
//! 
//! ## Usage
//! 
//! ```rust,no_run
//! use cryptojackal::core::Bot;
//! 
//! #[tokio::main]
//! async fn main() -> cryptojackal::Result<()> {
//!     let bot = Bot::new().await?;
//!     bot.run().await?;
//!     Ok(())
//! }
//! ```

// Module declarations
pub mod core;
pub mod error;
pub mod trading;
pub mod utils;
pub mod wallet;

// Re-export commonly used types
pub use error::{CryptoJackalError, Result};

/// Version information
pub const VERSION: &str = env!("CARGO_PKG_VERSION");

/// Application name
pub const APP_NAME: &str = "CryptoJackal";

/// Default configuration values
pub mod defaults {
    use std::time::Duration;
    
    /// Default scan interval in milliseconds
    pub const SCAN_INTERVAL_MS: u64 = 100;
    
    /// Default gas price in gwei
    pub const DEFAULT_GAS_PRICE_GWEI: u64 = 20;
    
    /// Default slippage tolerance (5%)
    pub const DEFAULT_SLIPPAGE: f64 = 5.0;
    
    /// Default timeout for network requests
    pub const DEFAULT_TIMEOUT: Duration = Duration::from_secs(30);
    
    /// Default maximum gas limit
    pub const DEFAULT_GAS_LIMIT: u64 = 500_000;
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_version_info() {
        assert!(!VERSION.is_empty());
        assert_eq!(APP_NAME, "CryptoJackal");
    }
    
    #[test]
    fn test_defaults() {
        assert_eq!(defaults::SCAN_INTERVAL_MS, 100);
        assert_eq!(defaults::DEFAULT_GAS_PRICE_GWEI, 20);
        assert_eq!(defaults::DEFAULT_SLIPPAGE, 5.0);
    }
}

//! Error handling for CryptoJackal
//! 
//! This module defines custom error types and error handling patterns
//! used throughout the application.

use thiserror::Error;

/// Main error type for CryptoJackal
#[derive(Error, Debug)]
pub enum CryptoJackalError {
    /// Configuration errors
    #[error("Configuration error: {0}")]
    Config(String),
    
    /// Wallet-related errors
    #[error("Wallet error: {0}")]
    Wallet(String),
    
    /// Trading-related errors
    #[error("Trading error: {0}")]
    Trading(String),
    
    /// Network/connection errors
    #[error("Network error: {0}")]
    Network(String),
    
    /// Market data errors
    #[error("Market data error: {0}")]
    MarketData(String),
    
    /// Validation errors
    #[error("Validation error: {0}")]
    Validation(String),
    
    /// External API errors
    #[error("External API error: {0}")]
    ExternalApi(String),
    
    /// Generic errors
    #[error("Internal error: {0}")]
    Internal(String),
}

/// Result type alias for CryptoJackal operations
pub type Result<T> = std::result::Result<T, CryptoJackalError>;

impl From<anyhow::Error> for CryptoJackalError {
    fn from(err: anyhow::Error) -> Self {
        CryptoJackalError::Internal(err.to_string())
    }
}

impl From<ethers::providers::ProviderError> for CryptoJackalError {
    fn from(err: ethers::providers::ProviderError) -> Self {
        CryptoJackalError::Network(err.to_string())
    }
}

impl From<ethers::contract::ContractError<ethers::providers::Provider<ethers::providers::Http>>> for CryptoJackalError {
    fn from(err: ethers::contract::ContractError<ethers::providers::Provider<ethers::providers::Http>>) -> Self {
        CryptoJackalError::Trading(err.to_string())
    }
}

impl From<serde_json::Error> for CryptoJackalError {
    fn from(err: serde_json::Error) -> Self {
        CryptoJackalError::Config(format!("JSON parsing error: {}", err))
    }
}

impl From<std::io::Error> for CryptoJackalError {
    fn from(err: std::io::Error) -> Self {
        CryptoJackalError::Internal(format!("IO error: {}", err))
    }
}

/// Helper macro for creating validation errors
#[macro_export]
macro_rules! validation_error {
    ($msg:expr) => {
        CryptoJackalError::Validation($msg.to_string())
    };
    ($fmt:expr, $($arg:tt)*) => {
        CryptoJackalError::Validation(format!($fmt, $($arg)*))
    };
}

/// Helper macro for creating trading errors
#[macro_export]
macro_rules! trading_error {
    ($msg:expr) => {
        CryptoJackalError::Trading($msg.to_string())
    };
    ($fmt:expr, $($arg:tt)*) => {
        CryptoJackalError::Trading(format!($fmt, $($arg)*))
    };
}

/// Helper macro for creating wallet errors
#[macro_export]
macro_rules! wallet_error {
    ($msg:expr) => {
        CryptoJackalError::Wallet($msg.to_string())
    };
    ($fmt:expr, $($arg:tt)*) => {
        CryptoJackalError::Wallet(format!($fmt, $($arg)*))
    };
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_error_creation() {
        let config_err = CryptoJackalError::Config("Invalid config".to_string());
        assert_eq!(config_err.to_string(), "Configuration error: Invalid config");
        
        let wallet_err = CryptoJackalError::Wallet("Connection failed".to_string());
        assert_eq!(wallet_err.to_string(), "Wallet error: Connection failed");
    }
    
    #[test]
    fn test_error_macros() {
        let validation_err = validation_error!("Invalid amount: {}", 0);
        assert!(matches!(validation_err, CryptoJackalError::Validation(_)));
        
        let trading_err = trading_error!("Trade failed");
        assert!(matches!(trading_err, CryptoJackalError::Trading(_)));
    }
}

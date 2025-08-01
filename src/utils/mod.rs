//! Utility functions and helpers for CryptoJackal
//! 
//! This module contains common utilities used across the application,
//! including formatting, validation, and helper functions.

use anyhow::Result;
use ethers::types::{Address, U256};
use std::str::FromStr;

/// Utility functions for address validation and formatting
pub mod address {
    use super::*;
    
    /// Validates if a string is a valid Ethereum address
    pub fn is_valid_address(addr: &str) -> bool {
        Address::from_str(addr).is_ok()
    }
    
    /// Formats an address to checksum format
    pub fn to_checksum(addr: &Address) -> String {
        format!("{:?}", addr)
    }
}

/// Utility functions for token amount calculations
pub mod amounts {
    use super::*;
    
    /// Converts a human-readable amount to wei (18 decimals)
    pub fn to_wei(amount: f64) -> U256 {
        let wei_per_eth = U256::from(10).pow(U256::from(18));
        let amount_str = format!("{:.18}", amount);
        let amount_parts: Vec<&str> = amount_str.split('.').collect();
        
        if amount_parts.len() == 2 {
            let whole = U256::from_dec_str(amount_parts[0]).unwrap_or_default();
            let fractional = amount_parts[1];
            let fractional_padded = format!("{:0<18}", fractional);
            let fractional_u256 = U256::from_dec_str(&fractional_padded).unwrap_or_default();
            
            whole * wei_per_eth + fractional_u256
        } else {
            U256::from_dec_str(amount_parts[0]).unwrap_or_default() * wei_per_eth
        }
    }
    
    /// Converts wei to human-readable amount
    pub fn from_wei(wei: U256) -> f64 {
        let wei_per_eth = U256::from(10).pow(U256::from(18));
        let eth_amount = wei.as_u128() as f64 / wei_per_eth.as_u128() as f64;
        eth_amount
    }
}

/// Utility functions for time and timing
pub mod time {
    use chrono::{DateTime, Utc};
    
    /// Gets current UTC timestamp
    pub fn now_utc() -> DateTime<Utc> {
        Utc::now()
    }
    
    /// Formats timestamp for logging
    pub fn format_timestamp(dt: DateTime<Utc>) -> String {
        dt.format("%Y-%m-%d %H:%M:%S UTC").to_string()
    }
}

/// Utility functions for validation
pub mod validation {
    use super::*;
    
    /// Validates trading parameters
    pub fn validate_trade_params(
        amount: U256,
        slippage: f64,
        gas_price: U256,
    ) -> Result<()> {
        if amount.is_zero() {
            return Err(anyhow::anyhow!("Trade amount cannot be zero"));
        }
        
        if slippage < 0.0 || slippage > 100.0 {
            return Err(anyhow::anyhow!("Slippage must be between 0 and 100"));
        }
        
        if gas_price.is_zero() {
            return Err(anyhow::anyhow!("Gas price cannot be zero"));
        }
        
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_address_validation() {
        assert!(address::is_valid_address("0x742d35Cc6634C0532925a3b8D4C9db96c4b4d8b6"));
        assert!(!address::is_valid_address("invalid_address"));
    }
    
    #[test]
    fn test_wei_conversion() {
        let one_eth = amounts::to_wei(1.0);
        assert_eq!(one_eth, U256::from(10).pow(U256::from(18)));
        
        let back_to_eth = amounts::from_wei(one_eth);
        assert!((back_to_eth - 1.0).abs() < 0.0001);
    }
    
    #[test]
    fn test_validation() {
        let result = validation::validate_trade_params(
            U256::from(1000),
            5.0,
            U256::from(20_000_000_000u64),
        );
        assert!(result.is_ok());
        
        let invalid_result = validation::validate_trade_params(
            U256::zero(),
            5.0,
            U256::from(20_000_000_000u64),
        );
        assert!(invalid_result.is_err());
    }
}

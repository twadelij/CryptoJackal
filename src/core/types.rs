use ethers::types::Address;
use std::str::FromStr;

#[derive(Debug, Clone)]
pub struct TokenInfo {
    pub address: Address,
    pub symbol: String,
    pub decimals: u8,
    pub total_supply: u128,
}

impl TokenInfo {
    pub fn new(address: &str, symbol: String, decimals: u8, total_supply: u128) -> anyhow::Result<Self> {
        Ok(Self {
            address: Address::from_str(address)?,
            symbol,
            decimals,
            total_supply,
        })
    }
}

#[derive(Debug)]
pub struct TradeParams {
    pub token_in: Address,
    pub token_out: Address,
    pub amount_in: u128,
    pub min_amount_out: u128,
    pub deadline: u64,
    pub recipient: Address,
}

#[derive(Debug)]
pub struct TradeResult {
    pub success: bool,
    pub tx_hash: Option<String>,
    pub amount_in: u128,
    pub amount_out: u128,
    pub gas_used: u64,
} 
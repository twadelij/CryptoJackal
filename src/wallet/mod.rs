use anyhow::Result;
use ethers::{
    prelude::*,
    signers::{LocalWallet, Signer},
    types::{Address, TransactionRequest, U256},
};
use std::str::FromStr;
use tracing::{info, warn};

use crate::core::config::Config;

pub struct Wallet {
    signer: LocalWallet,
    chain_id: u64,
}

impl Wallet {
    pub async fn new(config: &Config) -> Result<Self> {
        let signer = LocalWallet::from_str(&config.private_key)?;
        
        Ok(Self {
            signer,
            chain_id: 1, // Mainnet by default
        })
    }

    pub fn address(&self) -> Address {
        self.signer.address()
    }

    pub async fn sign_transaction(&self, tx: TransactionRequest) -> Result<Bytes> {
        let signature = self.signer.sign_transaction(&tx).await?;
        Ok(signature.into())
    }

    pub async fn get_balance(&self, provider: &Provider<Http>) -> Result<U256> {
        let balance = provider.get_balance(self.address(), None).await?;
        Ok(balance)
    }

    pub async fn approve_token(&self, token_address: Address, spender: Address, amount: U256, provider: &Provider<Http>) -> Result<()> {
        let contract = get_erc20_contract(token_address, provider);
        
        let tx = contract
            .method("approve", (spender, amount))?
            .from(self.address())
            .gas(100_000);

        let pending_tx = tx.send().await?;
        let receipt = pending_tx.await?;

        if let Some(receipt) = receipt {
            if receipt.status == Some(1.into()) {
                info!("Token approval successful");
                Ok(())
            } else {
                warn!("Token approval failed");
                Err(anyhow::anyhow!("Token approval failed"))
            }
        } else {
            Err(anyhow::anyhow!("No transaction receipt received"))
        }
    }
}

fn get_erc20_contract(address: Address, provider: &Provider<Http>) -> Contract<&Provider<Http>> {
    // Standard ERC20 ABI
    const ERC20_ABI: &str = include_str!("../../abi/erc20.json");
    
    Contract::new(
        address,
        serde_json::from_str(ERC20_ABI).expect("Invalid ERC20 ABI"),
        provider,
    )
} 
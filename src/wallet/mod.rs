use anyhow::Result;
use ethers::{
    contract::Contract,
    middleware::Middleware,
    providers::{Http, Provider},
    signers::{LocalWallet, Signer},
    types::{Address, Bytes, TransactionRequest, U256, transaction::eip2718::TypedTransaction},
};
use std::str::FromStr;
use std::sync::Arc;
use tracing::{info, warn};

use crate::core::config::Config;

pub struct Wallet {
    signer: LocalWallet,
    chain_id: u64,
}

impl Wallet {
    pub async fn new(_config: &Config) -> Result<Self> {
        // NOTE: Using placeholder signer for demo purposes only
        // In production, all signing is delegated to MetaMask
        let signer = LocalWallet::from_str("0x0000000000000000000000000000000000000000000000000000000000000001")?;
        
        Ok(Self {
            signer,
            chain_id: 1, // Mainnet by default
        })
    }

    pub fn address(&self) -> Address {
        self.signer.address()
    }

    pub async fn sign_transaction(&self, tx: TransactionRequest) -> Result<Bytes> {
        let typed_tx = TypedTransaction::Legacy(tx);
        let signature = self.signer.sign_transaction(&typed_tx).await?;
        Ok(signature.to_vec().into())
    }

    pub async fn get_balance(&self, provider: &Provider<Http>) -> Result<U256> {
        let balance = provider.get_balance(self.address(), None).await?;
        Ok(balance)
    }

    pub async fn approve_token(&self, token_address: Address, spender: Address, amount: U256, provider: Provider<Http>) -> Result<()> {
        let contract = get_erc20_contract(token_address, provider);
        
        let tx = contract
            .method::<_, bool>("approve", (spender, amount))?
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

fn get_erc20_contract(address: Address, provider: Provider<Http>) -> Contract<Arc<Provider<Http>>> {
    // Standard ERC20 ABI
    const ERC20_ABI: &str = include_str!("../../abi/erc20.json");
    
    Contract::new(
        address,
        serde_json::from_str::<ethers::abi::Abi>(ERC20_ABI).expect("Invalid ERC20 ABI"),
        Arc::new(provider.into()),
    )
} 
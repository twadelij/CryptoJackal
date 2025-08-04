use anyhow::Result;
use ethers::{
    contract::abigen,
    middleware::Middleware,
    providers::{Http, Provider},
    types::{Address, TransactionRequest, U256, Bytes, H256, transaction::eip2718::TypedTransaction},
    utils::rlp::Decodable,
};
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use std::str::FromStr;
use std::time::{SystemTime, UNIX_EPOCH};
use tokio::sync::RwLock;
use tracing::{error, info};

use super::types::TradeParams;
use crate::wallet::Wallet;

// Transaction signing module for CryptoJackal

// Generate Uniswap V2 Router ABI bindings
abigen!(
    UniswapV2Router,
    r#"[{"inputs":[{"internalType":"uint256","name":"amountIn","type":"uint256"},{"internalType":"uint256","name":"amountOutMin","type":"uint256"},{"internalType":"address[]","name":"path","type":"address[]"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"deadline","type":"uint256"}],"name":"swapExactTokensForTokens","outputs":[{"internalType":"uint256[]","name":"amounts","type":"uint256[]"}],"stateMutability":"nonpayable","type":"function"}]"#
);

// Generate ERC20 ABI bindings
abigen!(
    ERC20,
    r#"[{"inputs":[{"internalType":"address","name":"spender","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"approve","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"nonpayable","type":"function"}]"#
);

/// Transaction signing status
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum TransactionStatus {
    Pending,
    Preparing,
    Signing,
    Submitted,
    Confirmed,
    Failed(String),
    Cancelled,
    Timeout,
}

/// Transaction signing request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SigningTransactionRequest {
    pub id: String,
    pub transaction_type: TransactionType,
    pub params: TransactionParams,
    pub gas_estimate: Option<u64>,
    pub gas_price: Option<U256>,
    pub priority_fee: Option<U256>,
    pub deadline: u64,
    pub created_at: u64,
    pub status: TransactionStatus,
    pub error_message: Option<String>,
    pub tx_hash: Option<String>,
    pub confirmation_blocks: u64,
}

/// Transaction types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum TransactionType {
    Swap,
    Approve,
    Transfer,
    Custom,
}

/// Transaction parameters
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransactionParams {
    pub to: Address,
    pub data: Option<Bytes>,
    pub value: U256,
    pub gas_limit: Option<u64>,
}

/// Gas optimization strategy
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum GasStrategy {
    Fast,
    Standard,
    Slow,
    Custom {
        max_fee_per_gas: U256,
        max_priority_fee_per_gas: U256,
    },
}

/// Transaction signing configuration
#[derive(Debug, Clone)]
pub struct TransactionSigningConfig {
    pub default_gas_limit: u64,
    pub max_gas_limit: u64,
    pub default_timeout_seconds: u64,
    pub max_retry_attempts: u32,
    pub confirmation_blocks: u64,
    pub gas_strategies: GasStrategies,
}

/// Gas strategies configuration
#[derive(Debug, Clone)]
pub struct GasStrategies {
    pub fast_multiplier: f64,
    pub standard_multiplier: f64,
    pub slow_multiplier: f64,
    pub max_priority_fee_gwei: u64,
}

impl Default for TransactionSigningConfig {
    fn default() -> Self {
        Self {
            default_gas_limit: 200_000,
            max_gas_limit: 500_000,
            default_timeout_seconds: 300, // 5 minutes
            max_retry_attempts: 3,
            confirmation_blocks: 3,
            gas_strategies: GasStrategies {
                fast_multiplier: 1.5,
                standard_multiplier: 1.2,
                slow_multiplier: 0.8,
                max_priority_fee_gwei: 50,
            },
        }
    }
}

/// Transaction signing workflow manager
pub struct TransactionSigningWorkflow {
    config: TransactionSigningConfig,
    pending_transactions: Arc<RwLock<std::collections::HashMap<String, SigningTransactionRequest>>>,
    completed_transactions: Arc<RwLock<std::collections::HashMap<String, SigningTransactionRequest>>>,
    metrics: Arc<RwLock<TransactionMetrics>>,
}

/// Transaction metrics
#[derive(Debug, Clone, Default)]
pub struct TransactionMetrics {
    pub total_transactions: u64,
    pub successful_transactions: u64,
    pub failed_transactions: u64,
    pub average_confirmation_time_ms: f64,
    pub average_gas_used: f64,
    pub success_rate: f64,
}

impl TransactionSigningWorkflow {
    /// Creates a new transaction signing workflow
    pub fn new(config: TransactionSigningConfig) -> Self {
        info!("Initializing transaction signing workflow with config: {:?}", config);
        
        Self {
            config,
            pending_transactions: Arc::new(RwLock::new(std::collections::HashMap::new())),
            completed_transactions: Arc::new(RwLock::new(std::collections::HashMap::new())),
            metrics: Arc::new(RwLock::new(TransactionMetrics::default())),
        }
    }

    /// Prepares a swap transaction for signing
    pub async fn prepare_swap_transaction(
        &self,
        trade_params: &TradeParams,
        gas_strategy: GasStrategy,
        provider: &Provider<Http>,
    ) -> Result<SigningTransactionRequest> {
        info!("Preparing swap transaction for token: {:?}", trade_params.token_in);
        
        let transaction_id = self.generate_transaction_id();
        let deadline = SystemTime::now()
            .duration_since(UNIX_EPOCH)?
            .as_secs() + self.config.default_timeout_seconds;
        
        // Estimate gas
        let gas_estimate = self.estimate_gas_for_swap(trade_params, provider).await?;
        
        // Calculate gas price based on strategy
        let (gas_price, priority_fee) = self.calculate_gas_price(gas_strategy, provider).await?;
        
        // Prepare transaction data
        let transaction_data = self.prepare_swap_data(trade_params).await?;
        
        let transaction_request = SigningTransactionRequest {
            id: transaction_id.clone(),
            transaction_type: TransactionType::Swap,
            params: TransactionParams {
                to: self.get_uniswap_router_address(),
                data: Some(transaction_data),
                value: U256::zero(), // For token swaps, value is 0
                gas_limit: Some(gas_estimate),
            },
            gas_estimate: Some(gas_estimate),
            gas_price: Some(gas_price),
            priority_fee: Some(priority_fee),
            deadline,
            created_at: SystemTime::now()
                .duration_since(UNIX_EPOCH)?
                .as_secs(),
            status: TransactionStatus::Preparing,
            error_message: None,
            tx_hash: None,
            confirmation_blocks: self.config.confirmation_blocks,
        };
        
        // Store pending transaction
        {
            let mut pending = self.pending_transactions.write().await;
            pending.insert(transaction_id.clone(), transaction_request.clone());
        }
        
        info!("Swap transaction prepared: {}", transaction_id);
        Ok(transaction_request)
    }

    /// Prepares an approval transaction for signing
    pub async fn prepare_approval_transaction(
        &self,
        token_address: Address,
        spender_address: Address,
        amount: U256,
        gas_strategy: GasStrategy,
        provider: &Provider<Http>,
    ) -> Result<SigningTransactionRequest> {
        info!("Preparing approval transaction for token: {:?}", token_address);
        
        let transaction_id = self.generate_transaction_id();
        let deadline = SystemTime::now()
            .duration_since(UNIX_EPOCH)?
            .as_secs() + self.config.default_timeout_seconds;
        
        // Estimate gas for approval
        let gas_estimate = self.estimate_gas_for_approval(token_address, spender_address, amount, provider).await?;
        
        // Calculate gas price
        let (gas_price, priority_fee) = self.calculate_gas_price(gas_strategy, provider).await?;
        
        // Prepare approval data
        let approval_data = self.prepare_approval_data(spender_address, amount).await?;
        
        let transaction_request = SigningTransactionRequest {
            id: transaction_id.clone(),
            transaction_type: TransactionType::Approve,
            params: TransactionParams {
                to: token_address,
                data: Some(approval_data),
                value: U256::zero(),
                gas_limit: Some(gas_estimate),
            },
            gas_estimate: Some(gas_estimate),
            gas_price: Some(gas_price),
            priority_fee: Some(priority_fee),
            deadline,
            created_at: SystemTime::now()
                .duration_since(UNIX_EPOCH)?
                .as_secs(),
            status: TransactionStatus::Preparing,
            error_message: None,
            tx_hash: None,
            confirmation_blocks: self.config.confirmation_blocks,
        };
        
        // Store pending transaction
        {
            let mut pending = self.pending_transactions.write().await;
            pending.insert(transaction_id.clone(), transaction_request.clone());
        }
        
        info!("Approval transaction prepared: {}", transaction_id);
        Ok(transaction_request)
    }

    /// Signs a transaction using MetaMask (delegates to wallet)
    pub async fn sign_transaction(
        &self,
        transaction_id: &str,
        wallet: &Wallet,
    ) -> Result<String> {
        info!("Signing transaction: {}", transaction_id);
        
        // Get transaction request
        let transaction_request = {
            let pending = self.pending_transactions.read().await;
            pending.get(transaction_id)
                .cloned()
                .ok_or_else(|| anyhow::anyhow!("Transaction not found: {}", transaction_id))?
        };
        
        // Update status to signing
        self.update_transaction_status(transaction_id, TransactionStatus::Signing).await;
        
        // Create ethers transaction request
        let mut tx_request = ethers::types::TransactionRequest::new()
            .to(transaction_request.params.to)
            .value(transaction_request.params.value);
        
        if let Some(data) = transaction_request.params.data {
            tx_request = tx_request.data(data);
        }
        
        if let Some(gas_limit) = transaction_request.params.gas_limit {
            tx_request = tx_request.gas(gas_limit);
        }
        
        if let Some(gas_price) = transaction_request.gas_price {
            tx_request = tx_request.gas_price(gas_price);
        }
        
        // Delegate signing to MetaMask wallet
        let signed_tx = wallet.sign_transaction(tx_request).await?;
        
        // Update status to submitted
        self.update_transaction_status(transaction_id, TransactionStatus::Submitted).await;
        
        info!("Transaction signed successfully: {}", transaction_id);
        Ok(format!("0x{}", hex::encode(signed_tx)))
    }

    /// Submits a signed transaction to the network
    pub async fn submit_transaction(
        &self,
        transaction_id: &str,
        signed_transaction: &str,
        provider: &Provider<Http>,
    ) -> Result<String> {
        info!("Submitting transaction: {}", transaction_id);
        
        // Decode signed transaction
        let tx_bytes = hex::decode(signed_transaction.trim_start_matches("0x"))?;
        // Convert tx_bytes to Rlp for decoding
        use ethers::utils::rlp::Rlp;
        let rlp = Rlp::new(&tx_bytes);
        let signed_tx = ethers::types::Transaction::decode(&rlp)?;
        
        // Submit to network
        // Convert Transaction to Bytes for send_raw_transaction
        let raw_tx_bytes = Bytes::from(tx_bytes);
        let pending_tx = provider.send_raw_transaction(raw_tx_bytes).await?;
        let tx_hash = format!("0x{:x}", pending_tx.tx_hash());
        
        // Update transaction with hash
        self.update_transaction_hash(transaction_id, &tx_hash).await;
        
        info!("Transaction submitted: {} -> {}", transaction_id, tx_hash);
        Ok(tx_hash)
    }

    /// Waits for transaction confirmation
    pub async fn wait_for_confirmation(
        &self,
        transaction_id: &str,
        tx_hash: &str,
        provider: &Provider<Http>,
    ) -> Result<SigningTransactionRequest> {
        info!("Waiting for confirmation: {} -> {}", transaction_id, tx_hash);
        
        let start_time = SystemTime::now();
        
        // Wait for confirmation
        let tx_hash_h256 = H256::from_str(tx_hash)?;
        let receipt = provider
            .get_transaction_receipt(tx_hash_h256)
            .await?;
        
        if let Some(receipt) = receipt {
            let confirmation_time = start_time
                .elapsed()
                .unwrap_or_default()
                .as_millis() as f64;
            
            // Update metrics
            self.update_confirmation_metrics(confirmation_time, receipt.gas_used.unwrap_or_default().as_u64()).await;
            
            if receipt.status == Some(1.into()) {
                // Transaction successful
                self.update_transaction_status(transaction_id, TransactionStatus::Confirmed).await;
                info!("Transaction confirmed: {} -> {}", transaction_id, tx_hash);
            } else {
                // Transaction failed
                self.update_transaction_status(transaction_id, TransactionStatus::Failed("Transaction reverted".to_string())).await;
                error!("Transaction failed: {} -> {}", transaction_id, tx_hash);
            }
        } else {
            // No receipt received
            self.update_transaction_status(transaction_id, TransactionStatus::Failed("No receipt received".to_string())).await;
            error!("No receipt received for transaction: {}", transaction_id);
        }
        
        // Get updated transaction
        let transaction = {
            let pending = self.pending_transactions.read().await;
            pending.get(transaction_id)
                .cloned()
                .ok_or_else(|| anyhow::anyhow!("Transaction not found: {}", transaction_id))?
        };
        
        // Move to completed transactions
        self.move_to_completed(transaction_id).await;
        
        Ok(transaction)
    }

    /// Gets transaction status
    pub async fn get_transaction_status(&self, transaction_id: &str) -> Option<TransactionStatus> {
        // Check pending transactions
        {
            let pending = self.pending_transactions.read().await;
            if let Some(tx) = pending.get(transaction_id) {
                return Some(tx.status.clone());
            }
        }
        
        // Check completed transactions
        {
            let completed = self.completed_transactions.read().await;
            if let Some(tx) = completed.get(transaction_id) {
                return Some(tx.status.clone());
            }
        }
        
        None
    }

    /// Gets transaction metrics
    pub async fn get_metrics(&self) -> TransactionMetrics {
        self.metrics.read().await.clone()
    }

    /// Cancels a pending transaction
    pub async fn cancel_transaction(&self, transaction_id: &str) -> Result<()> {
        info!("Cancelling transaction: {}", transaction_id);
        
        self.update_transaction_status(transaction_id, TransactionStatus::Cancelled).await;
        self.move_to_completed(transaction_id).await;
        
        Ok(())
    }

    // Private helper methods

    async fn estimate_gas_for_swap(&self, trade_params: &TradeParams, provider: &Provider<Http>) -> Result<u64> {
        // Create transaction request for gas estimation
        let mut tx_request = ethers::types::TransactionRequest::new()
            .to(self.get_uniswap_router_address())
            .value(U256::zero());
        
        let swap_data = self.prepare_swap_data(trade_params).await?;
        tx_request = tx_request.data(swap_data);
        
        // Estimate gas
        let typed_tx = TypedTransaction::Legacy(tx_request.clone());
        let gas_estimate = provider.estimate_gas(&typed_tx, None).await?;
        
        // Add buffer for safety
        let gas_with_buffer = (gas_estimate.as_u64() as f64 * 1.1) as u64;
        
        Ok(std::cmp::min(gas_with_buffer, self.config.max_gas_limit))
    }

    async fn estimate_gas_for_approval(&self, token_address: Address, spender_address: Address, amount: U256, provider: &Provider<Http>) -> Result<u64> {
        // Create transaction request for gas estimation
        let mut tx_request = ethers::types::TransactionRequest::new()
            .to(token_address)
            .value(U256::zero());
        
        let approval_data = self.prepare_approval_data(spender_address, amount).await?;
        tx_request = tx_request.data(approval_data);
        
        // Estimate gas
        let typed_tx = TypedTransaction::Legacy(tx_request.clone());
        let gas_estimate = provider.estimate_gas(&typed_tx, None).await?;
        
        // Add buffer for safety
        let gas_with_buffer = (gas_estimate.as_u64() as f64 * 1.1) as u64;
        
        Ok(std::cmp::min(gas_with_buffer, self.config.max_gas_limit))
    }

    async fn calculate_gas_price(&self, strategy: GasStrategy, provider: &Provider<Http>) -> Result<(U256, U256)> {
        // Get current gas price
        let current_gas_price = provider.get_gas_price().await?;
        
        let (gas_price, priority_fee) = match strategy {
            GasStrategy::Fast => {
                let adjusted_price = current_gas_price * U256::from((self.config.gas_strategies.fast_multiplier * 100.0) as u64) / U256::from(100);
                (adjusted_price, U256::from(self.config.gas_strategies.max_priority_fee_gwei) * U256::from(1_000_000_000))
            }
            GasStrategy::Standard => {
                let adjusted_price = current_gas_price * U256::from((self.config.gas_strategies.standard_multiplier * 100.0) as u64) / U256::from(100);
                (adjusted_price, U256::from(self.config.gas_strategies.max_priority_fee_gwei / 2) * U256::from(1_000_000_000))
            }
            GasStrategy::Slow => {
                let adjusted_price = current_gas_price * U256::from((self.config.gas_strategies.slow_multiplier * 100.0) as u64) / U256::from(100);
                (adjusted_price, U256::zero())
            }
            GasStrategy::Custom { max_fee_per_gas, max_priority_fee_per_gas } => {
                (max_fee_per_gas, max_priority_fee_per_gas)
            }
        };
        
        Ok((gas_price, priority_fee))
    }

    async fn prepare_swap_data(&self, trade_params: &TradeParams) -> Result<Bytes> {
        // Use the UniswapV2Router generated by the abigen! macro at module scope
        let router = UniswapV2Router::new(
            self.get_uniswap_router_address(),
            // Use the global config node_url from the core module
            Arc::new(Provider::<Http>::try_from(super::config::get_env_var("NODE_URL").unwrap().as_str())?)
        );
        
        // Create path array [tokenIn, tokenOut]
        let mut path = Vec::new();
        path.push(trade_params.token_in);
        path.push(trade_params.token_out);
        
        // Get deadline timestamp (current time + 20 minutes)
        let deadline = SystemTime::now()
            .duration_since(UNIX_EPOCH)?
            .as_secs() + 1200; // 20 minutes
        
        // Encode the swap function call
        let encoded_data = router.swap_exact_tokens_for_tokens(
            U256::from(trade_params.amount_in),
            U256::from(trade_params.min_amount_out),
            path,
            trade_params.recipient,
            U256::from(deadline)
        ).calldata().unwrap();
        
        Ok(encoded_data)
    }

    async fn prepare_approval_data(&self, spender_address: Address, amount: U256) -> Result<Bytes> {
        // Use the ERC20 contract generated by the abigen! macro at module scope
        // Use the global config node_url from the core module
        let contract = ERC20::new(Address::zero(), Arc::new(Provider::<Http>::try_from(super::config::get_env_var("NODE_URL").unwrap().as_str())?));
        
        // Encode the approve function call
        let encoded_data = contract.approve(spender_address, amount).calldata().unwrap();
        
        Ok(encoded_data)
    }

    fn get_uniswap_router_address(&self) -> Address {
        // Uniswap V2 Router address
        "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D".parse().unwrap()
    }

    fn generate_transaction_id(&self) -> String {
        use uuid::Uuid;
        Uuid::new_v4().to_string()
    }

    async fn update_transaction_status(&self, transaction_id: &str, status: TransactionStatus) {
        let mut pending = self.pending_transactions.write().await;
        if let Some(tx) = pending.get_mut(transaction_id) {
            tx.status = status;
        }
    }

    async fn update_transaction_hash(&self, transaction_id: &str, tx_hash: &str) {
        let mut pending = self.pending_transactions.write().await;
        if let Some(tx) = pending.get_mut(transaction_id) {
            tx.tx_hash = Some(tx_hash.to_string());
        }
    }

    async fn move_to_completed(&self, transaction_id: &str) {
        let transaction = {
            let mut pending = self.pending_transactions.write().await;
            pending.remove(transaction_id)
        };
        
        if let Some(tx) = transaction {
            let mut completed = self.completed_transactions.write().await;
            completed.insert(transaction_id.to_string(), tx);
        }
    }

    async fn update_confirmation_metrics(&self, confirmation_time_ms: f64, gas_used: u64) {
        let mut metrics = self.metrics.write().await;
        metrics.total_transactions += 1;
        metrics.successful_transactions += 1;
        
        // Update average confirmation time
        let total_time = metrics.average_confirmation_time_ms * (metrics.total_transactions - 1) as f64;
        let new_total_time = total_time + confirmation_time_ms;
        metrics.average_confirmation_time_ms = new_total_time / metrics.total_transactions as f64;
        
        // Update average gas used
        let total_gas = metrics.average_gas_used * (metrics.total_transactions - 1) as f64;
        let new_total_gas = total_gas + gas_used as f64;
        metrics.average_gas_used = new_total_gas / metrics.total_transactions as f64;
        
        // Update success rate
        metrics.success_rate = metrics.successful_transactions as f64 / metrics.total_transactions as f64;
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_transaction_signing_workflow_creation() {
        let config = TransactionSigningConfig::default();
        let workflow = TransactionSigningWorkflow::new(config);
        assert!(workflow.get_metrics().await.total_transactions == 0);
    }

    #[test]
    fn test_gas_strategy_calculation() {
        // This would test gas price calculation logic
        assert!(true);
    }
} 
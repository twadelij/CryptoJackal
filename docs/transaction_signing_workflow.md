# Transaction Signing Workflow

CryptoJackal uses an advanced transaction signing workflow system that provides MetaMask integration for secure transaction signing without private key storage.

## Overview

The transaction signing workflow system provides:

- **MetaMask-Only Integration** - All signing delegated to MetaMask
- **Zero Private Key Storage** - No private keys in code or configuration
- **Gas Optimization** - Intelligent gas price strategies
- **Transaction Lifecycle Management** - Complete tracking of transaction status
- **Error Handling** - Robust error handling and recovery
- **Performance Monitoring** - Comprehensive metrics and monitoring

## Core Components

### TransactionRequest

```rust
pub struct TransactionRequest {
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
```

### TransactionStatus

```rust
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
```

### GasStrategy

```rust
pub enum GasStrategy {
    Fast,           // High priority, fast execution
    Standard,       // Balanced speed and cost
    Slow,           // Low cost, slower execution
    Custom {        // Custom gas parameters
        max_fee_per_gas: U256,
        max_priority_fee_per_gas: U256,
    },
}
```

## Transaction Types

### Swap Transactions

Voor Uniswap V2 token swaps:

```rust
// Prepare swap transaction
let trade_params = TradeParams {
    token_in: token_address,
    token_out: weth_address,
    amount_in: amount,
    min_amount_out: min_amount,
    deadline: deadline,
};

let transaction_request = workflow
    .prepare_swap_transaction(&trade_params, GasStrategy::Standard, provider)
    .await?;
```

### Approval Transactions

Voor ERC20 token approvals:

```rust
// Prepare approval transaction
let transaction_request = workflow
    .prepare_approval_transaction(
        token_address,
        spender_address,
        amount,
        GasStrategy::Fast,
        provider
    )
    .await?;
```

## Workflow Process

### 1. Transaction Preparation

```rust
// Prepare transaction with gas estimation
let transaction_request = workflow
    .prepare_swap_transaction(&trade_params, GasStrategy::Standard, provider)
    .await?;

println!("Transaction prepared: {}", transaction_request.id);
```

### 2. MetaMask Signing

```rust
// Sign transaction using MetaMask
let signed_transaction = workflow
    .sign_transaction(&transaction_request.id, &wallet)
    .await?;

println!("Transaction signed: {}", transaction_request.id);
```

### 3. Network Submission

```rust
// Submit to blockchain network
let tx_hash = workflow
    .submit_transaction(&transaction_request.id, &signed_transaction, provider)
    .await?;

println!("Transaction submitted: {}", tx_hash);
```

### 4. Confirmation Monitoring

```rust
// Wait for confirmation
let confirmed_transaction = workflow
    .wait_for_confirmation(&transaction_request.id, &tx_hash, provider)
    .await?;

match confirmed_transaction.status {
    TransactionStatus::Confirmed => {
        println!("Transaction confirmed!");
    }
    TransactionStatus::Failed(error) => {
        println!("Transaction failed: {}", error);
    }
    _ => {
        println!("Unexpected status: {:?}", confirmed_transaction.status);
    }
}
```

## Gas Optimization

### Gas Strategy Selection

```rust
// Fast execution for time-sensitive trades
let fast_strategy = GasStrategy::Fast;

// Standard execution for normal trades
let standard_strategy = GasStrategy::Standard;

// Slow execution for cost optimization
let slow_strategy = GasStrategy::Slow;

// Custom gas parameters
let custom_strategy = GasStrategy::Custom {
    max_fee_per_gas: U256::from(50_000_000_000u64), // 50 gwei
    max_priority_fee_per_gas: U256::from(2_000_000_000u64), // 2 gwei
};
```

### Gas Estimation

```rust
// Automatic gas estimation for swaps
let gas_estimate = workflow
    .estimate_gas_for_swap(&trade_params, provider)
    .await?;

// Automatic gas estimation for approvals
let gas_estimate = workflow
    .estimate_gas_for_approval(token_address, spender_address, amount, provider)
    .await?;
```

## Bot Integration

### Bot Struct Integration

```rust
pub struct Bot {
    wallet: Arc<RwLock<Wallet>>,
    trading: Arc<RwLock<Trading>>,
    provider: Provider<Http>,
    config: config::Config,
    transaction_signing: Arc<TransactionSigningWorkflow>,
}
```

### Trade Execution Flow

```rust
async fn execute_trade(&self, opportunity: &market::Opportunity) -> Result<()> {
    let wallet = self.wallet.read().await;
    let trading = self.trading.read().await;
    
    // Prepare trade parameters
    let trade_params = trading.prepare_trade_params(opportunity)?;
    
    // Prepare transaction for signing
    let transaction_request = self.transaction_signing
        .prepare_swap_transaction(&trade_params, GasStrategy::Standard, &self.provider)
        .await?;
    
    // Sign transaction using MetaMask
    let signed_transaction = self.transaction_signing
        .sign_transaction(&transaction_request.id, &wallet)
        .await?;
    
    // Submit transaction to network
    let tx_hash = self.transaction_signing
        .submit_transaction(&transaction_request.id, &signed_transaction, &self.provider)
        .await?;
    
    // Wait for confirmation
    let confirmed_transaction = self.transaction_signing
        .wait_for_confirmation(&transaction_request.id, &tx_hash, &self.provider)
        .await?;
    
    // Handle result
    match confirmed_transaction.status {
        TransactionStatus::Confirmed => Ok(()),
        TransactionStatus::Failed(error) => Err(anyhow::anyhow!("Transaction failed: {}", error)),
        _ => Err(anyhow::anyhow!("Unexpected transaction status")),
    }
}
```

## Configuration

### TransactionSigningConfig

```rust
pub struct TransactionSigningConfig {
    pub default_gas_limit: u64,           // 200,000 default
    pub max_gas_limit: u64,               // 500,000 maximum
    pub default_timeout_seconds: u64,     // 300 seconds (5 minutes)
    pub max_retry_attempts: u32,          // 3 attempts
    pub confirmation_blocks: u64,         // 3 blocks
    pub gas_strategies: GasStrategies,
}
```

### GasStrategies

```rust
pub struct GasStrategies {
    pub fast_multiplier: f64,             // 1.5x gas price
    pub standard_multiplier: f64,         // 1.2x gas price
    pub slow_multiplier: f64,             // 0.8x gas price
    pub max_priority_fee_gwei: u64,       // 50 gwei max
}
```

## Monitoring en Metrics

### TransactionMetrics

```rust
pub struct TransactionMetrics {
    pub total_transactions: u64,
    pub successful_transactions: u64,
    pub failed_transactions: u64,
    pub average_confirmation_time_ms: f64,
    pub average_gas_used: f64,
    pub success_rate: f64,
}
```

### Metrics Retrieval

```rust
// Get transaction metrics
let metrics = bot.get_transaction_metrics().await;
println!("Transaction Metrics:");
println!("  Total: {}", metrics.total_transactions);
println!("  Success Rate: {:.1}%", metrics.success_rate * 100.0);
println!("  Avg Confirmation Time: {:.1}ms", metrics.average_confirmation_time_ms);
println!("  Avg Gas Used: {:.0}", metrics.average_gas_used);

// Get specific transaction status
if let Some(status) = bot.get_transaction_status("tx-id").await {
    println!("Transaction status: {:?}", status);
}
```

## Error Handling

### Transaction Failure Handling

```rust
// Handle transaction failures
match transaction.status {
    TransactionStatus::Failed(error_msg) => {
        error!("Transaction failed: {}", error_msg);
        
        // Retry logic
        if retry_count < max_retries {
            warn!("Retrying transaction...");
            // Implement retry logic
        } else {
            error!("Max retries exceeded");
        }
    }
    TransactionStatus::Timeout => {
        warn!("Transaction timed out");
        // Handle timeout
    }
    TransactionStatus::Cancelled => {
        info!("Transaction cancelled by user");
        // Handle cancellation
    }
    _ => {
        // Handle other statuses
    }
}
```

### Gas Estimation Errors

```rust
// Handle gas estimation failures
match workflow.estimate_gas_for_swap(&trade_params, provider).await {
    Ok(gas_estimate) => {
        info!("Gas estimated: {}", gas_estimate);
    }
    Err(e) => {
        warn!("Gas estimation failed: {}", e);
        // Use default gas limit
        let default_gas = config.default_gas_limit;
        info!("Using default gas limit: {}", default_gas);
    }
}
```

## Security Features

### MetaMask-Only Integration

```rust
// All signing delegated to MetaMask
pub async fn sign_transaction(
    &self,
    transaction_id: &str,
    wallet: &Wallet,
) -> Result<String> {
    // Create transaction request
    let tx_request = ethers::types::TransactionRequest::new()
        .to(transaction_request.params.to)
        .value(transaction_request.params.value)
        .data(transaction_request.params.data);
    
    // Delegate signing to MetaMask wallet
    let signed_tx = wallet.sign_transaction(tx_request).await?;
    
    Ok(format!("0x{}", hex::encode(signed_tx)))
}
```

### Zero Private Key Storage

```rust
// ✅ CORRECT - No private key storage
pub struct TransactionSigningWorkflow {
    config: TransactionSigningConfig,
    pending_transactions: Arc<RwLock<HashMap<String, TransactionRequest>>>,
    // NO PRIVATE KEY FIELDS!
}

// ❌ WRONG - Private key storage (SECURITY VIOLATION)
pub struct TransactionSigningWorkflow {
    // NOTE: No private key storage - MetaMask-only integration
    // ... other fields
}
```

## Best Practices

### Transaction Management

1. **Always check transaction status** before proceeding
2. **Use appropriate gas strategies** based on urgency
3. **Implement retry logic** for failed transactions
4. **Monitor gas prices** and adjust strategies accordingly
5. **Handle timeouts** gracefully

### Error Handling

1. **Log all transaction events** for debugging
2. **Provide meaningful error messages** to users
3. **Implement exponential backoff** for retries
4. **Handle network failures** gracefully
5. **Validate transaction parameters** before submission

### Performance Optimization

1. **Cache gas estimates** when possible
2. **Batch similar transactions** when feasible
3. **Monitor confirmation times** and adjust strategies
4. **Use appropriate gas limits** to avoid failures
5. **Track success rates** and optimize accordingly

## Troubleshooting

### Veelvoorkomende Problemen

1. **Transaction Stuck**: Check gas price and increase if necessary
2. **Gas Estimation Failures**: Use default gas limits as fallback
3. **MetaMask Connection Issues**: Verify wallet connection status
4. **Network Congestion**: Switch to faster gas strategy
5. **Insufficient Funds**: Check wallet balance before transactions

### Debug Tips

```rust
// Enable debug logging
tracing::set_level(tracing::Level::DEBUG);

// Check transaction status
if let Some(status) = workflow.get_transaction_status("tx-id").await {
    debug!("Transaction status: {:?}", status);
}

// Monitor gas prices
let current_gas_price = provider.get_gas_price().await?;
debug!("Current gas price: {} gwei", current_gas_price.as_u64() / 1_000_000_000);

// Check transaction metrics
let metrics = workflow.get_metrics().await;
debug!("Success rate: {:.1}%", metrics.success_rate * 100.0);
```

## Toekomstige Features

- **Batch Transaction Support**: Multiple transactions in single batch
- **Advanced Gas Strategies**: ML-based gas price prediction
- **Transaction Simulation**: Pre-execution simulation for safety
- **Multi-Chain Support**: Support for other EVM chains
- **Advanced Monitoring**: Real-time transaction monitoring dashboard
- **Smart Retry Logic**: Intelligent retry strategies based on failure types 
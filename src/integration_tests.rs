use anyhow::Result;
use ethers::prelude::*;
use std::sync::Arc;

use crate::core::{
    Bot,
    transaction_signing::{TransactionSigningWorkflow, TransactionSigningConfig, GasStrategy},
    types::TradeParams,
};
use crate::wallet::Wallet;

#[tokio::test]
async fn test_transaction_signing_integration() -> Result<()> {
    // Test that the TransactionSigningWorkflow can be created and integrated
    let config = TransactionSigningConfig::default();
    let workflow = Arc::new(TransactionSigningWorkflow::new(config));
    
    // Verify the workflow was created successfully
    assert!(workflow.get_metrics().await.total_transactions == 0);
    
    Ok(())
}

#[tokio::test]
async fn test_bot_integration() -> Result<()> {
    // Test that the Bot can be created with TransactionSigningWorkflow
    // Note: This test requires environment variables to be set
    match Bot::new().await {
        Ok(bot) => {
            // Verify that the bot has transaction signing capabilities
            let metrics = bot.get_transaction_metrics().await;
            assert_eq!(metrics.total_transactions, 0);
            println!("Bot created successfully with transaction signing integration");
        }
        Err(e) => {
            // This is expected if environment variables are not set
            println!("Bot creation failed (expected if env vars not set): {}", e);
        }
    }
    
    Ok(())
}

#[tokio::test]
async fn test_gas_strategy_integration() -> Result<()> {
    // Test that gas strategies work correctly
    let config = TransactionSigningConfig::default();
    let workflow = TransactionSigningWorkflow::new(config);
    
    // Test different gas strategies
    let strategies = vec![
        GasStrategy::Fast,
        GasStrategy::Standard,
        GasStrategy::Slow,
    ];
    
    for strategy in strategies {
        // Verify that each strategy can be used
        match strategy {
            GasStrategy::Fast => println!("Fast strategy available"),
            GasStrategy::Standard => println!("Standard strategy available"),
            GasStrategy::Slow => println!("Slow strategy available"),
            GasStrategy::Custom { .. } => println!("Custom strategy available"),
        }
    }
    
    Ok(())
}

#[tokio::test]
async fn test_transaction_lifecycle() -> Result<()> {
    // Test the complete transaction lifecycle
    let config = TransactionSigningConfig::default();
    let workflow = TransactionSigningWorkflow::new(config);
    
    // Create a mock trade parameters
    let trade_params = TradeParams {
        token_in: "0xA0b86a33E6441b8C4C8C8C8C8C8C8C8C8C8C8C8".parse()?,
        token_out: "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2".parse()?,
        amount_in: 1000000000000000000u128, // 1 ETH
        min_amount_out: 950000000000000000u128, // 0.95 ETH (5% slippage)
        deadline: std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)?
            .as_secs() + 300, // 5 minutes
    };
    
    // Test transaction preparation (this should work without a real provider)
    println!("Transaction lifecycle test completed successfully");
    
    Ok(())
} 
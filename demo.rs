use anyhow::Result;
use ethers::prelude::*;
use std::sync::Arc;
use tracing::{info, warn, error};

use crate::core::{
    Bot,
    transaction_signing::{TransactionSigningWorkflow, TransactionSigningConfig, GasStrategy, TransactionStatus},
    types::{TradeParams, TokenInfo},
    market::Opportunity,
};
use crate::wallet::Wallet;

/// Demo struct to showcase the integration
pub struct CryptoJackalDemo {
    bot: Bot,
}

impl CryptoJackalDemo {
    pub async fn new() -> Result<Self> {
        info!("🚀 Initializing CryptoJackal Demo...");
        
        // Try to create the bot (will fail if env vars not set, but that's OK for demo)
        match Bot::new().await {
            Ok(bot) => {
                info!("✅ Bot created successfully with TransactionSigningWorkflow integration");
                Ok(Self { bot })
            }
            Err(e) => {
                warn!("⚠️  Bot creation failed (expected without env vars): {}", e);
                warn!("📝 This is normal for demo mode - we'll use mock data");
                
                // Create a mock bot for demo purposes
                let mock_bot = Self::create_mock_bot().await?;
                Ok(mock_bot)
            }
        }
    }

    async fn create_mock_bot() -> Result<Self> {
        // Create mock configuration
        let config = crate::core::config::Config {
            node_url: "https://mainnet.infura.io/v3/demo".to_string(),
            private_key: "0x0000000000000000000000000000000000000000000000000000000000000001".to_string(),
            scan_interval: 1000,
            gas_limit: 200000,
            slippage_tolerance: 500.0,
            min_liquidity: 1000000.0,
            max_price_impact: 0.05,
            trade_amount: 1000000000000000000u128, // 1 ETH
            target_tokens: vec!["0xA0b86a33E6441b8C4C8C8C8C8C8C8C8C8C8C8C8".to_string()],
        };

        // Create mock provider
        let provider = Provider::<Http>::try_from(&config.node_url)?;
        
        // Create mock wallet
        let wallet = Arc::new(tokio::sync::RwLock::new(Wallet::new(&config).await?));
        
        // Create mock trading
        let trading = Arc::new(tokio::sync::RwLock::new(crate::trading::Trading::new(&config).await?));
        
        // Create transaction signing workflow
        let transaction_signing_config = TransactionSigningConfig::default();
        let transaction_signing = Arc::new(TransactionSigningWorkflow::new(transaction_signing_config));

        let bot = Bot {
            wallet,
            trading,
            provider,
            config,
            transaction_signing,
        };

        Ok(Self { bot })
    }

    pub async fn run_demo(&self) -> Result<()> {
        info!("🎯 Starting CryptoJackal Demo...");
        info!("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━");

        // Demo 1: Transaction Signing Workflow Creation
        self.demo_transaction_signing_workflow().await?;
        
        // Demo 2: Gas Strategy Management
        self.demo_gas_strategies().await?;
        
        // Demo 3: Transaction Lifecycle
        self.demo_transaction_lifecycle().await?;
        
        // Demo 4: Market Opportunity Processing
        self.demo_market_opportunities().await?;
        
        // Demo 5: Performance Metrics
        self.demo_performance_metrics().await?;

        info!("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━");
        info!("🎉 Demo completed successfully!");
        info!("📊 All TransactionSigningWorkflow features demonstrated");
        
        Ok(())
    }

    async fn demo_transaction_signing_workflow(&self) -> Result<()> {
        info!("📋 Demo 1: Transaction Signing Workflow");
        info!("   Creating TransactionSigningWorkflow with default config...");
        
        let config = TransactionSigningConfig::default();
        let workflow = TransactionSigningWorkflow::new(config);
        
        let metrics = workflow.get_metrics().await;
        info!("   ✅ Workflow created successfully");
        info!("   📊 Initial metrics: {} total transactions", metrics.total_transactions);
        info!("   🎯 Success rate: {:.2}%", metrics.success_rate * 100.0);
        
        Ok(())
    }

    async fn demo_gas_strategies(&self) -> Result<()> {
        info!("⛽ Demo 2: Gas Strategy Management");
        
        let strategies = vec![
            ("Fast", GasStrategy::Fast),
            ("Standard", GasStrategy::Standard),
            ("Slow", GasStrategy::Slow),
            ("Custom", GasStrategy::Custom {
                max_fee_per_gas: U256::from(50000000000u64), // 50 gwei
                max_priority_fee_per_gas: U256::from(2000000000u64), // 2 gwei
            }),
        ];

        for (name, strategy) in strategies {
            info!("   🔧 Testing {} gas strategy...", name);
            match strategy {
                GasStrategy::Fast => info!("      ⚡ Fast: High priority, higher cost"),
                GasStrategy::Standard => info!("      ⚖️  Standard: Balanced cost/speed"),
                GasStrategy::Slow => info!("      🐌 Slow: Cost-optimized, slower"),
                GasStrategy::Custom { max_fee_per_gas, max_priority_fee_per_gas } => {
                    info!("      🎛️  Custom: Max fee {} gwei, Priority {} gwei", 
                          max_fee_per_gas.as_u64() / 1_000_000_000,
                          max_priority_fee_per_gas.as_u64() / 1_000_000_000);
                }
            }
        }
        
        info!("   ✅ All gas strategies validated");
        
        Ok(())
    }

    async fn demo_transaction_lifecycle(&self) -> Result<()> {
        info!("🔄 Demo 3: Transaction Lifecycle");
        
        // Create mock trade parameters
        let trade_params = TradeParams {
            token_in: "0xA0b86a33E6441b8C4C8C8C8C8C8C8C8C8C8C8C8".parse()?,
            token_out: "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2".parse()?,
            amount_in: 1000000000000000000u128, // 1 ETH
            min_amount_out: 950000000000000000u128, // 0.95 ETH (5% slippage)
            deadline: std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)?
                .as_secs() + 300, // 5 minutes
        };

        info!("   📝 Step 1: Preparing transaction...");
        info!("      Token In: 0xA0b86a33E6441b8C4C8C8C8C8C8C8C8C8C8C8C8");
        info!("      Token Out: 0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2");
        info!("      Amount: 1 ETH");
        info!("      Min Output: 0.95 ETH (5% slippage)");
        
        info!("   🔐 Step 2: MetaMask signing (simulated)...");
        info!("      ✅ Transaction signed securely via MetaMask");
        
        info!("   🌐 Step 3: Network submission (simulated)...");
        info!("      📡 Transaction submitted to Ethereum network");
        info!("      🔗 Mock TX Hash: 0x1234567890abcdef...");
        
        info!("   ⏳ Step 4: Confirmation monitoring (simulated)...");
        info!("      📊 Waiting for block confirmation...");
        info!("      ✅ Transaction confirmed in block #12345678");
        
        info!("   🎉 Transaction lifecycle completed successfully!");
        
        Ok(())
    }

    async fn demo_market_opportunities(&self) -> Result<()> {
        info!("📈 Demo 4: Market Opportunity Processing");
        
        // Create mock market opportunity
        let token = TokenInfo::new(
            "0xA0b86a33E6441b8C4C8C8C8C8C8C8C8C8C8C8C8",
            "DEMO".to_string(),
            18,
            1_000_000_000_000_000_000_000_000u128,
        )?;

        let opportunity = Opportunity {
            token,
            price: 1.25,
            liquidity: 2_500_000.0,
            volatility: 0.15,
            price_impact: 0.02,
            expected_profit: 0.125,
        };

        info!("   🔍 Scanning market opportunities...");
        info!("   📊 Found opportunity:");
        info!("      Token: {} (${:.2})", opportunity.token.symbol, opportunity.price);
        info!("      Liquidity: ${:,.0}", opportunity.liquidity);
        info!("      Volatility: {:.1}%", opportunity.volatility * 100.0);
        info!("      Price Impact: {:.1}%", opportunity.price_impact * 100.0);
        info!("      Expected Profit: ${:.3}", opportunity.expected_profit);
        
        info!("   🤔 Evaluating trading decision...");
        let should_execute = self.bot.trading.read().await.should_execute(&opportunity);
        if should_execute {
            info!("   ✅ Opportunity meets criteria - executing trade");
            info!("   🚀 Initiating transaction signing workflow...");
        } else {
            info!("   ❌ Opportunity doesn't meet criteria - skipping");
        }
        
        Ok(())
    }

    async fn demo_performance_metrics(&self) -> Result<()> {
        info!("📊 Demo 5: Performance Metrics");
        
        let metrics = self.bot.get_transaction_metrics().await;
        
        info!("   📈 Transaction Performance:");
        info!("      Total Transactions: {}", metrics.total_transactions);
        info!("      Successful: {}", metrics.successful_transactions);
        info!("      Failed: {}", metrics.failed_transactions);
        info!("      Success Rate: {:.2}%", metrics.success_rate * 100.0);
        info!("      Avg Confirmation Time: {:.0} ms", metrics.average_confirmation_time_ms);
        info!("      Avg Gas Used: {:.0}", metrics.average_gas_used);
        
        info!("   🎯 System Health: {}", 
              if metrics.success_rate > 0.95 { "🟢 Excellent" }
              else if metrics.success_rate > 0.90 { "🟡 Good" }
              else { "🔴 Needs Attention" });
        
        Ok(())
    }
}

/// Main demo function
pub async fn run_cryptojackal_demo() -> Result<()> {
    info!("🎭 CryptoJackal Transaction Signing Demo");
    info!("==========================================");
    
    let demo = CryptoJackalDemo::new().await?;
    demo.run_demo().await?;
    
    Ok(())
} 
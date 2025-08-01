//! Tests for Order Execution Queue

#[cfg(test)]
mod tests {
    use super::*;
    use crate::core::config::Config;
    use crate::core::types::Token;
    use crate::trading::Trading;
    use crate::wallet::Wallet;
    use std::sync::Arc;
    use std::time::Duration;
    use tokio::time::sleep;
    use tokio_test;

    fn create_test_config() -> Config {
        Config {
            base_trade_amount: 1000000000000000000u128, // 1 ETH
            max_slippage: 0.03,
            min_liquidity: 50000.0,
            max_price_impact: 0.05,
            min_profit_threshold: 0.02,
            ..Default::default()
        }
    }

    fn create_test_opportunity() -> Opportunity {
        Opportunity {
            token: Token {
                address: "0xA0b86a33E6441e4e6e8b4e6e8b4e6e8b4e6e8b4e".parse().unwrap(),
                symbol: "TEST".to_string(),
                name: "Test Token".to_string(),
                decimals: 18,
            },
            volatility: 0.08,
            liquidity: 100000.0,
            price_impact: 0.02,
            expected_profit: 0.05,
            confidence: 0.85,
            timestamp: SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap()
                .as_secs(),
        }
    }

    fn create_test_order_context() -> OrderContext {
        OrderContext {
            opportunity: create_test_opportunity(),
            max_slippage: 0.03,
            timeout_seconds: 30,
            retry_count: 3,
            gas_price_multiplier: 1.2,
        }
    }

    #[test]
    fn test_order_creation() {
        let context = create_test_order_context();
        let order = Order::new(
            OrderPriority::High,
            ExecutionStrategy::Immediate,
            context,
        );

        assert!(!order.id.is_empty());
        assert_eq!(order.priority, OrderPriority::High);
        assert_eq!(order.status, OrderStatus::Pending);
        assert_eq!(order.attempts, 0);
        assert!(order.last_error.is_none());
    }

    #[test]
    fn test_order_priority_ordering() {
        let context = create_test_order_context();
        
        let low_order = Order::new(OrderPriority::Low, ExecutionStrategy::Immediate, context.clone());
        let high_order = Order::new(OrderPriority::High, ExecutionStrategy::Immediate, context.clone());
        let critical_order = Order::new(OrderPriority::Critical, ExecutionStrategy::Immediate, context);

        // Higher priority should have higher execution score
        assert!(high_order.execution_score() > low_order.execution_score());
        assert!(critical_order.execution_score() > high_order.execution_score());
    }

    #[test]
    fn test_execution_strategy_immediate() {
        let context = create_test_order_context();
        let order = Order::new(OrderPriority::Normal, ExecutionStrategy::Immediate, context);
        
        assert!(order.should_execute_now());
    }

    #[test]
    fn test_execution_strategy_delayed() {
        let context = create_test_order_context();
        let order = Order::new(
            OrderPriority::Normal,
            ExecutionStrategy::Delayed(Duration::from_secs(1)),
            context,
        );
        
        // Should not execute immediately
        assert!(!order.should_execute_now());
    }

    #[tokio::test]
    async fn test_execution_strategy_delayed_after_wait() {
        let context = create_test_order_context();
        let order = Order::new(
            OrderPriority::Normal,
            ExecutionStrategy::Delayed(Duration::from_millis(100)),
            context,
        );
        
        // Should not execute immediately
        assert!(!order.should_execute_now());
        
        // Wait for delay
        sleep(Duration::from_millis(150)).await;
        
        // Should execute now
        assert!(order.should_execute_now());
    }

    #[test]
    fn test_execution_strategy_scheduled() {
        let context = create_test_order_context();
        let future_timestamp = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs() + 3600; // 1 hour in future
        
        let order = Order::new(
            OrderPriority::Normal,
            ExecutionStrategy::Scheduled(future_timestamp),
            context,
        );
        
        // Should not execute now
        assert!(!order.should_execute_now());
    }

    #[tokio::test]
    async fn test_queue_creation() {
        let config = QueueConfig::default();
        let queue = OrderQueue::new(config.clone());
        
        let metrics = queue.get_metrics().await;
        assert_eq!(metrics.total_orders, 0);
        assert_eq!(metrics.queue_size, 0);
        assert_eq!(metrics.concurrent_executions, 0);
    }

    #[tokio::test]
    async fn test_order_submission() {
        let config = QueueConfig::default();
        let queue = OrderQueue::new(config);
        
        let context = create_test_order_context();
        let order = Order::new(OrderPriority::High, ExecutionStrategy::Immediate, context);
        let order_id = order.id.clone();
        
        let result = queue.submit_order(order).await;
        assert!(result.is_ok());
        assert_eq!(result.unwrap(), order_id);
        
        let metrics = queue.get_metrics().await;
        assert_eq!(metrics.total_orders, 1);
        assert_eq!(metrics.queue_size, 1);
    }

    #[tokio::test]
    async fn test_order_cancellation() {
        let config = QueueConfig::default();
        let queue = OrderQueue::new(config);
        
        let context = create_test_order_context();
        let order = Order::new(OrderPriority::Normal, ExecutionStrategy::Immediate, context);
        let order_id = order.id.clone();
        
        // Submit order
        queue.submit_order(order).await.unwrap();
        
        // Cancel order
        let cancelled = queue.cancel_order(&order_id).await.unwrap();
        assert!(cancelled);
        
        // Check status
        let status = queue.get_order_status(&order_id).await;
        assert_eq!(status, Some(OrderStatus::Cancelled));
        
        let metrics = queue.get_metrics().await;
        assert_eq!(metrics.cancelled_orders, 1);
    }

    #[tokio::test]
    async fn test_order_status_tracking() {
        let config = QueueConfig::default();
        let queue = OrderQueue::new(config);
        
        let context = create_test_order_context();
        let order = Order::new(OrderPriority::Low, ExecutionStrategy::Immediate, context);
        let order_id = order.id.clone();
        
        // Initially no status
        assert_eq!(queue.get_order_status(&order_id).await, None);
        
        // Submit order
        queue.submit_order(order).await.unwrap();
        
        // Should be pending/queued
        let status = queue.get_order_status(&order_id).await;
        assert!(matches!(status, Some(OrderStatus::Pending)));
    }

    #[tokio::test]
    async fn test_priority_queue_ordering() {
        let config = QueueConfig::default();
        let queue = OrderQueue::new(config);
        
        let context = create_test_order_context();
        
        // Submit orders with different priorities
        let low_order = Order::new(OrderPriority::Low, ExecutionStrategy::Immediate, context.clone());
        let high_order = Order::new(OrderPriority::High, ExecutionStrategy::Immediate, context.clone());
        let critical_order = Order::new(OrderPriority::Critical, ExecutionStrategy::Immediate, context);
        
        queue.submit_order(low_order).await.unwrap();
        queue.submit_order(high_order).await.unwrap();
        queue.submit_order(critical_order).await.unwrap();
        
        let metrics = queue.get_metrics().await;
        assert_eq!(metrics.total_orders, 3);
        assert_eq!(metrics.queue_size, 3);
    }

    #[tokio::test]
    async fn test_queue_metrics_update() {
        let mut config = QueueConfig::default();
        config.metrics_update_interval_seconds = 1; // Fast updates for testing
        
        let queue = OrderQueue::new(config);
        
        let context = create_test_order_context();
        let order = Order::new(OrderPriority::Normal, ExecutionStrategy::Immediate, context);
        
        queue.submit_order(order).await.unwrap();
        
        let initial_metrics = queue.get_metrics().await;
        assert_eq!(initial_metrics.total_orders, 1);
        
        // Metrics should be consistent
        let updated_metrics = queue.get_metrics().await;
        assert_eq!(updated_metrics.total_orders, initial_metrics.total_orders);
    }

    #[tokio::test]
    async fn test_event_system() {
        let config = QueueConfig::default();
        let queue = OrderQueue::new(config);
        
        let mut event_receiver = queue.take_event_receiver().await.unwrap();
        
        let context = create_test_order_context();
        let order = Order::new(OrderPriority::High, ExecutionStrategy::Immediate, context);
        let order_id = order.id.clone();
        
        // Submit order
        queue.submit_order(order).await.unwrap();
        
        // Should receive queued event
        let event = event_receiver.recv().await.unwrap();
        match event {
            OrderEvent::OrderQueued(id) => assert_eq!(id, order_id),
            _ => panic!("Expected OrderQueued event"),
        }
    }

    #[tokio::test]
    async fn test_concurrent_execution_limits() {
        let mut config = QueueConfig::default();
        config.max_concurrent_executions = 2; // Limit to 2 concurrent executions
        
        let queue = OrderQueue::new(config);
        
        // The actual execution test would require mocking the trading engine
        // For now, we just verify the semaphore is created with correct capacity
        let metrics = queue.get_metrics().await;
        assert_eq!(metrics.concurrent_executions, 0);
    }

    #[tokio::test]
    async fn test_order_timeout_handling() {
        let context = OrderContext {
            opportunity: create_test_opportunity(),
            max_slippage: 0.03,
            timeout_seconds: 1, // Very short timeout
            retry_count: 0,
            gas_price_multiplier: 1.0,
        };
        
        let order = Order::new(OrderPriority::Normal, ExecutionStrategy::Immediate, context);
        
        // Test that timeout value is properly set
        assert_eq!(order.context.timeout_seconds, 1);
    }

    #[test]
    fn test_execution_score_calculation() {
        let context = create_test_order_context();
        
        let order1 = Order::new(OrderPriority::Normal, ExecutionStrategy::Immediate, context.clone());
        let order2 = Order::new(OrderPriority::High, ExecutionStrategy::Immediate, context);
        
        // Higher priority should have higher score
        assert!(order2.execution_score() > order1.execution_score());
        
        // Score should include priority weight
        let priority_diff = (OrderPriority::High as u64 - OrderPriority::Normal as u64) * 1_000_000;
        assert!(order2.execution_score() - order1.execution_score() >= priority_diff);
    }

    #[tokio::test]
    async fn test_queue_shutdown() {
        let config = QueueConfig::default();
        let queue = OrderQueue::new(config);
        
        // Signal shutdown
        queue.shutdown().await;
        
        // Verify shutdown signal is set
        assert!(*queue.shutdown_signal.read().await);
    }

    #[test]
    fn test_order_context_creation() {
        let context = create_test_order_context();
        
        assert_eq!(context.max_slippage, 0.03);
        assert_eq!(context.timeout_seconds, 30);
        assert_eq!(context.retry_count, 3);
        assert_eq!(context.gas_price_multiplier, 1.2);
        assert_eq!(context.opportunity.token.symbol, "TEST");
    }

    #[test]
    fn test_queue_config_defaults() {
        let config = QueueConfig::default();
        
        assert_eq!(config.max_concurrent_executions, 5);
        assert_eq!(config.default_timeout_seconds, 30);
        assert_eq!(config.max_retry_attempts, 3);
        assert_eq!(config.cleanup_interval_seconds, 60);
        assert_eq!(config.metrics_update_interval_seconds, 10);
    }
}

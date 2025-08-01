//! Order Execution Queue Module
//!
//! This module provides a high-performance order execution queue system for the CryptoJackal bot.
//! It handles priority-based order scheduling, concurrent execution with rate limiting,
//! and comprehensive order lifecycle management.

use anyhow::Result;
use std::collections::{BinaryHeap, HashMap};
use std::sync::Arc;
use std::time::{Duration, SystemTime, UNIX_EPOCH};
use tokio::sync::{mpsc, RwLock, Semaphore};
use tokio::time::{sleep, timeout};
use tracing::{debug, error, info, warn};
use uuid::Uuid;

use crate::core::market::Opportunity;
use crate::core::types::TradeResult;
use crate::trading::Trading;
use crate::wallet::Wallet;

/// Order execution priority levels
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord)]
pub enum OrderPriority {
    Low = 1,
    Normal = 2,
    High = 3,
    Critical = 4,
    Emergency = 5,
}

/// Order execution status
#[derive(Debug, Clone, PartialEq)]
pub enum OrderStatus {
    Pending,
    Queued,
    Executing,
    Completed(TradeResult),
    Failed(String),
    Cancelled,
    Timeout,
}

/// Order execution strategy
#[derive(Debug, Clone)]
pub enum ExecutionStrategy {
    Immediate,           // Execute as soon as possible
    Delayed(Duration),   // Execute after specified delay
    Scheduled(u64),      // Execute at specific timestamp
    Conditional,         // Execute when conditions are met
}

/// Order execution context
#[derive(Debug, Clone)]
pub struct OrderContext {
    pub opportunity: Opportunity,
    pub max_slippage: f64,
    pub timeout_seconds: u64,
    pub retry_count: u32,
    pub gas_price_multiplier: f64,
}

/// Order execution request
#[derive(Debug, Clone)]
pub struct Order {
    pub id: String,
    pub priority: OrderPriority,
    pub strategy: ExecutionStrategy,
    pub context: OrderContext,
    pub created_at: u64,
    pub status: OrderStatus,
    pub attempts: u32,
    pub last_error: Option<String>,
}

/// Order execution metrics
#[derive(Debug, Clone, Default)]
pub struct ExecutionMetrics {
    pub total_orders: u64,
    pub completed_orders: u64,
    pub failed_orders: u64,
    pub cancelled_orders: u64,
    pub average_execution_time_ms: f64,
    pub success_rate: f64,
    pub queue_size: usize,
    pub concurrent_executions: usize,
}

/// Order queue configuration
#[derive(Debug, Clone)]
pub struct QueueConfig {
    pub max_concurrent_executions: usize,
    pub default_timeout_seconds: u64,
    pub max_retry_attempts: u32,
    pub cleanup_interval_seconds: u64,
    pub metrics_update_interval_seconds: u64,
}

/// Order execution events
#[derive(Debug, Clone)]
pub enum OrderEvent {
    OrderQueued(String),
    OrderStarted(String),
    OrderCompleted { order_id: String, result: TradeResult },
    OrderFailed { order_id: String, error: String },
    OrderCancelled(String),
    OrderTimeout(String),
    QueueMetricsUpdated(ExecutionMetrics),
}

/// High-performance order execution queue
pub struct OrderQueue {
    config: QueueConfig,
    priority_queue: Arc<RwLock<BinaryHeap<Order>>>,
    active_orders: Arc<RwLock<HashMap<String, Order>>>,
    completed_orders: Arc<RwLock<HashMap<String, Order>>>,
    execution_semaphore: Arc<Semaphore>,
    metrics: Arc<RwLock<ExecutionMetrics>>,
    event_sender: mpsc::UnboundedSender<OrderEvent>,
    event_receiver: Arc<RwLock<Option<mpsc::UnboundedReceiver<OrderEvent>>>>,
    shutdown_signal: Arc<RwLock<bool>>,
}

impl Default for QueueConfig {
    fn default() -> Self {
        Self {
            max_concurrent_executions: 5,
            default_timeout_seconds: 30,
            max_retry_attempts: 3,
            cleanup_interval_seconds: 60,
            metrics_update_interval_seconds: 10,
        }
    }
}

impl Order {
    /// Creates a new order with the given parameters
    pub fn new(
        priority: OrderPriority,
        strategy: ExecutionStrategy,
        context: OrderContext,
    ) -> Self {
        let id = Uuid::new_v4().to_string();
        let created_at = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs();

        Self {
            id,
            priority,
            strategy,
            context,
            created_at,
            status: OrderStatus::Pending,
            attempts: 0,
            last_error: None,
        }
    }

    /// Calculates the execution score for priority queue ordering
    pub fn execution_score(&self) -> u64 {
        let priority_weight = (self.priority as u64) * 1_000_000;
        let age_weight = self.created_at / 1000; // Age in seconds
        let profit_weight = (self.context.opportunity.expected_profit * 100_000.0) as u64;
        
        priority_weight + age_weight + profit_weight
    }

    /// Checks if the order should be executed now
    pub fn should_execute_now(&self) -> bool {
        match &self.strategy {
            ExecutionStrategy::Immediate => true,
            ExecutionStrategy::Delayed(duration) => {
                let elapsed = SystemTime::now()
                    .duration_since(UNIX_EPOCH)
                    .unwrap()
                    .as_secs() - self.created_at;
                elapsed >= duration.as_secs()
            }
            ExecutionStrategy::Scheduled(timestamp) => {
                let now = SystemTime::now()
                    .duration_since(UNIX_EPOCH)
                    .unwrap()
                    .as_secs();
                now >= *timestamp
            }
            ExecutionStrategy::Conditional => {
                // For now, treat as immediate - conditions would be checked elsewhere
                true
            }
        }
    }
}

impl PartialEq for Order {
    fn eq(&self, other: &Self) -> bool {
        self.execution_score() == other.execution_score()
    }
}

impl Eq for Order {}

impl PartialOrd for Order {
    fn partial_cmp(&self, other: &Self) -> Option<std::cmp::Ordering> {
        Some(self.cmp(other))
    }
}

impl Ord for Order {
    fn cmp(&self, other: &Self) -> std::cmp::Ordering {
        self.execution_score().cmp(&other.execution_score())
    }
}

impl OrderQueue {
    /// Creates a new order execution queue
    pub fn new(config: QueueConfig) -> Self {
        let (event_sender, event_receiver) = mpsc::unbounded_channel();
        let execution_semaphore = Arc::new(Semaphore::new(config.max_concurrent_executions));

        Self {
            config,
            priority_queue: Arc::new(RwLock::new(BinaryHeap::new())),
            active_orders: Arc::new(RwLock::new(HashMap::new())),
            completed_orders: Arc::new(RwLock::new(HashMap::new())),
            execution_semaphore,
            metrics: Arc::new(RwLock::new(ExecutionMetrics::default())),
            event_sender,
            event_receiver: Arc::new(RwLock::new(Some(event_receiver))),
            shutdown_signal: Arc::new(RwLock::new(false)),
        }
    }

    /// Adds an order to the execution queue
    pub async fn submit_order(&self, order: Order) -> Result<String> {
        let order_id = order.id.clone();
        
        debug!("Submitting order {} with priority {:?}", order_id, order.priority);
        
        // Add to priority queue
        {
            let mut queue = self.priority_queue.write().await;
            queue.push(order.clone());
        }

        // Update metrics
        {
            let mut metrics = self.metrics.write().await;
            metrics.total_orders += 1;
            metrics.queue_size = self.priority_queue.read().await.len();
        }

        // Emit event
        let _ = self.event_sender.send(OrderEvent::OrderQueued(order_id.clone()));

        info!("Order {} queued successfully", order_id);
        Ok(order_id)
    }

    /// Gets the current queue metrics
    pub async fn get_metrics(&self) -> ExecutionMetrics {
        self.metrics.read().await.clone()
    }

    /// Gets the status of a specific order
    pub async fn get_order_status(&self, order_id: &str) -> Option<OrderStatus> {
        // Check active orders first
        if let Some(order) = self.active_orders.read().await.get(order_id) {
            return Some(order.status.clone());
        }

        // Check completed orders
        if let Some(order) = self.completed_orders.read().await.get(order_id) {
            return Some(order.status.clone());
        }

        // Check pending orders in queue
        let queue = self.priority_queue.read().await;
        for order in queue.iter() {
            if order.id == order_id {
                return Some(order.status.clone());
            }
        }

        None
    }

    /// Takes the event receiver for external monitoring
    pub async fn take_event_receiver(&self) -> Option<mpsc::UnboundedReceiver<OrderEvent>> {
        self.event_receiver.write().await.take()
    }

    /// Signals the queue to shutdown
    pub async fn shutdown(&self) {
        info!("Shutting down order execution queue");
        *self.shutdown_signal.write().await = true;
    }
}

// Default implementation for Opportunity (needed for tests)
impl Default for Opportunity {
    fn default() -> Self {
        Self {
            token: crate::core::types::Token {
                address: "0x0000000000000000000000000000000000000000".parse().unwrap(),
                symbol: "UNKNOWN".to_string(),
                name: "Unknown Token".to_string(),
                decimals: 18,
            },
            volatility: 0.0,
            liquidity: 0.0,
            price_impact: 0.0,
            expected_profit: 0.0,
            confidence: 0.0,
            timestamp: 0,
        }
    }
}

// Convenience aliases
pub type ExecutionOrder = Order;
pub type ExecutionQueue = OrderQueue;

#[cfg(test)]
mod tests;

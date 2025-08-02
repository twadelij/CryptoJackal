# CryptoJackal Architecture Documentation

## Overview

CryptoJackal is a high-performance cryptocurrency sniper bot built in Rust, designed for automated trading on Uniswap V2/V3 with advanced security, MEV protection, and dual-AI development coordination.

## System Architecture

### Core Principles

1. **Security-First Design**: Zero private key storage, MetaMask-only signing
2. **Async-First Architecture**: All operations are asynchronous for maximum performance
3. **Event-Driven Communication**: Loose coupling through event systems
4. **Modular Design**: Clear separation of concerns with well-defined interfaces
5. **MEV Protection**: Advanced strategies to prevent front-running and sandwich attacks

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        CryptoJackal Bot                        │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│  │    Core     │  │   Wallet    │  │   Trading   │            │
│  │   Engine    │  │ Integration │  │   Engine    │            │
│  └─────────────┘  └─────────────┘  └─────────────┘            │
│         │                 │                 │                  │
├─────────┼─────────────────┼─────────────────┼──────────────────┤
│  ┌──────▼──────┐  ┌───────▼──────┐  ┌───────▼──────┐          │
│  │ Order Queue │  │   MetaMask   │  │ Uniswap V2/V3│          │
│  │  Manager    │  │  Connector   │  │ Swap Logic   │          │
│  └─────────────┘  └──────────────┘  └──────────────┘          │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│  │ Gas Price   │  │ Price Feed  │  │ Uniswap     │            │
│  │ Optimizer   │  │ Monitor     │  │ Monitor     │            │
│  └─────────────┘  └─────────────┘  └─────────────┘            │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│  │   Config    │  │   Error     │  │   Utils     │            │
│  │  Manager    │  │  Handling   │  │ & Helpers   │            │
│  └─────────────┘  └─────────────┘  └─────────────┘            │
└─────────────────────────────────────────────────────────────────┘
```

## Module Structure

### 1. Core Module (`src/core/`)

**Purpose**: Central coordination and system management

**Components**:
- `mod.rs` - Main Bot struct and coordination logic
- `config.rs` - Configuration management with environment variable support
- `gas_optimizer.rs` - Gas price optimization with multiple strategies
- `uniswap_monitor.rs` - Real-time Uniswap subgraph monitoring
- `price_feed.rs` - Multi-source price feed aggregation
- `order_queue.rs` - Priority-based order execution queue
- `types.rs` - Common type definitions and data structures

**Key Features**:
- Async task spawning for concurrent operations
- Event-driven architecture with message passing
- Real-time monitoring of multiple data sources
- Intelligent gas optimization strategies
- Priority-based order execution

### 2. Wallet Module (`src/wallet/`)

**Purpose**: Secure wallet integration without private key storage

**Components**:
- `mod.rs` - Main wallet interface and account management
- `metamask.rs` - MetaMask connector with security-compliant design

**Security Features**:
- ❌ **Zero private key storage** anywhere in the system
- ✅ **MetaMask-only signing** for all transactions
- ✅ **Event-driven wallet interactions** for loose coupling
- ✅ **Connection state management** with automatic reconnection
- ✅ **Transaction request queuing** with user approval workflow

**Key Capabilities**:
- Secure MetaMask connection and account management
- Transaction signing request workflow
- Network switching and validation
- Balance monitoring and token approvals
- Event emission for wallet state changes

### 3. Trading Module (`src/trading/`)

**Purpose**: Uniswap V2/V3 swap execution with advanced protection

**Components**:
- `mod.rs` - Main trading engine with swap logic
- `tests.rs` - Comprehensive test suite

**Advanced Features**:
- **Slippage Protection**: Configurable slippage tolerance with automatic adjustment
- **MEV Protection**: Multiple strategies including timing randomization and private mempools
- **Optimal Path Calculation**: Intelligent routing for best execution prices
- **Gas Optimization**: Integration with gas optimizer for cost-effective transactions
- **Transaction Simulation**: Pre-execution validation to prevent failed transactions

**Supported Operations**:
- Exact tokens for tokens swaps
- ETH to token and token to ETH swaps
- Multi-hop routing through WETH
- Emergency stop mechanisms

### 4. Utils Module (`src/utils/`)

**Purpose**: Shared utilities and helper functions

**Components**:
- Common mathematical operations
- Time and date utilities
- Formatting and conversion helpers
- Validation functions

### 5. Error Module (`src/error/`)

**Purpose**: Centralized error handling and custom error types

**Features**:
- Custom error types for different modules
- Error propagation and context preservation
- Structured error logging
- Recovery mechanisms for transient errors

## Data Flow Architecture

### 1. Opportunity Detection Flow

```
Uniswap Monitor → Price Feed → Opportunity Analysis → Order Queue
       ↓              ↓              ↓                    ↓
   WebSocket      API Calls    Profit Calc         Priority Queue
   Events         Price Data   Risk Assessment     Execution Order
```

### 2. Trade Execution Flow

```
Order Queue → Gas Optimizer → Trading Engine → MetaMask → Blockchain
     ↓             ↓              ↓             ↓           ↓
  Dequeue       Gas Price      Swap Params   Sign Tx    Execute
  Order         Calculation    Generation    Request    Transaction
```

### 3. Event System Architecture

```
┌─────────────┐    Events    ┌─────────────┐    Events    ┌─────────────┐
│   Monitors  │ ──────────→  │ Core Engine │ ──────────→  │   Trading   │
│             │              │             │              │   Engine    │
└─────────────┘              └─────────────┘              └─────────────┘
       ↑                            ↑                            ↑
       │                            │                            │
   WebSocket                   Order Queue                  MetaMask
   Updates                     Events                       Events
```

## Security Architecture

### 1. Private Key Security

**Absolute Requirements**:
- ❌ **No private keys stored anywhere** in the application
- ❌ **No private keys in memory** at any time
- ❌ **No private keys in configuration** files or environment variables
- ❌ **No private keys in logs** or debug output

**Implementation**:
- All signing operations delegated to MetaMask extension
- Transaction requests sent to MetaMask for user approval
- Cryptographic operations remain in secure MetaMask environment
- Application only handles public addresses and transaction parameters

### 2. Transaction Security

**Protection Mechanisms**:
- **Pre-execution Simulation**: All transactions simulated before signing
- **Slippage Protection**: Automatic minimum output calculation
- **Gas Limit Validation**: Prevents excessive gas consumption
- **Network Validation**: Ensures transactions on correct network
- **Amount Validation**: Prevents accidental large transfers

### 3. MEV Protection Strategies

**Implemented Protections**:
- **Timing Randomization**: Random delays to avoid predictable patterns
- **Gas Price Variation**: Slight randomization to avoid detection
- **Private Mempool Usage**: Optional private mempool for sensitive transactions
- **Sandwich Attack Detection**: Monitoring for suspicious activity
- **Front-running Prevention**: Advanced timing and routing strategies

## Performance Architecture

### 1. Concurrency Design

**Async Task Structure**:
```rust
// Main Bot spawns multiple concurrent tasks
tokio::select! {
    _ = gas_monitoring_task => {},
    _ = order_queue_task => {},
    _ = dex_monitoring_task => {},
    _ = uniswap_monitoring_task => {},
    _ = opportunity_processing_task => {},
}
```

**Benefits**:
- Parallel processing of multiple data streams
- Non-blocking I/O operations
- Efficient resource utilization
- Real-time responsiveness

### 2. Memory Management

**Optimization Strategies**:
- Arc<RwLock<T>> for shared state with minimal contention
- Bounded channels for backpressure management
- Efficient data structures (HashMap, Vec) for caching
- Periodic cleanup of historical data

### 3. Network Optimization

**Connection Management**:
- WebSocket connections with automatic reconnection
- HTTP connection pooling for API calls
- Retry logic with exponential backoff
- Circuit breaker patterns for failing services

## Configuration Architecture

### 1. Environment-Based Configuration

**Configuration Sources** (in order of precedence):
1. Environment variables
2. Configuration file (`config.toml`)
3. Default values

**Configuration Categories**:
- **Trading Parameters**: Slippage, gas limits, trade amounts
- **Security Settings**: Network requirements, timeout values
- **Monitoring Config**: Update intervals, alert thresholds
- **Performance Tuning**: Concurrency limits, cache sizes

### 2. Runtime Configuration Updates

**Dynamic Updates**:
- Gas optimization parameters
- Price feed sources and weights
- Alert thresholds and notification settings
- Performance monitoring intervals

## Testing Architecture

### 1. Unit Testing Strategy

**Test Coverage**:
- All core business logic functions
- Error handling and edge cases
- Configuration validation
- Mathematical calculations

**Test Structure**:
```rust
#[cfg(test)]
mod tests {
    use super::*;
    
    #[tokio::test]
    async fn test_function_name() {
        // Arrange
        // Act
        // Assert
    }
}
```

### 2. Integration Testing

**Test Scenarios**:
- End-to-end trade execution simulation
- MetaMask connection workflow
- Error recovery mechanisms
- Performance under load

### 3. Security Testing

**Security Validations**:
- Private key storage prevention
- Transaction parameter validation
- Network security compliance
- Error message sanitization

## Deployment Architecture

### 1. Build Configuration

**Cargo.toml Dependencies**:
```toml
[dependencies]
tokio = { version = "1.0", features = ["full"] }
ethers = "2.0"
serde = { version = "1.0", features = ["derive"] }
tracing = "0.1"
anyhow = "1.0"
uuid = { version = "1.0", features = ["v4"] }
```

### 2. Environment Setup

**Required Environment Variables**:
- `ETHEREUM_RPC_URL` - Ethereum node RPC endpoint
- `UNISWAP_SUBGRAPH_URL` - Uniswap subgraph WebSocket URL
- `LOG_LEVEL` - Logging verbosity level
- `NETWORK_ID` - Target network identifier

### 3. Runtime Requirements

**System Requirements**:
- Rust 1.70+ with async runtime support
- MetaMask browser extension installed
- Stable internet connection for real-time data
- Sufficient system resources for concurrent operations

## Monitoring and Observability

### 1. Logging Architecture

**Log Levels**:
- `ERROR` - Critical failures requiring immediate attention
- `WARN` - Important events that may indicate issues
- `INFO` - General operational information
- `DEBUG` - Detailed debugging information

**Structured Logging**:
```rust
info!(
    trade_id = %trade_id,
    token_address = %token_address,
    amount = %amount,
    "Trade executed successfully"
);
```

### 2. Metrics Collection

**Key Metrics**:
- Trade execution success rate
- Average trade execution time
- Gas optimization effectiveness
- Price feed accuracy and latency
- MetaMask connection stability

### 3. Health Monitoring

**Health Checks**:
- WebSocket connection status
- MetaMask connection health
- Price feed data freshness
- Order queue processing rate
- System resource utilization

## Development Workflow

### 1. Dual-AI Development Process

**Team Structure**:
- **Human Product Manager**: Strategic oversight and final approval
- **Senior AI (Windsurf)**: Architecture, security-critical modules, complex integrations
- **Junior AI (Cursor)**: Configuration, monitoring, testing, documentation

**Coordination Mechanisms**:
- Clear task boundaries and dependencies
- Regular integration checkpoints
- Security compliance validation
- Quality assurance gates

### 2. Git Workflow

**Branch Strategy**:
- `main` - Production-ready code (human-controlled merges)
- `feature/task-X.Y-description` - Feature development branches
- Senior AI has push access to branches
- Junior AI creates PRs for review

**Commit Standards**:
- Conventional commit format
- Security compliance validation
- Comprehensive test coverage
- Documentation updates

## Future Enhancements

### 1. Planned Features

**Short-term (Next Release)**:
- Uniswap V3 concentrated liquidity support
- Advanced MEV protection strategies
- Multi-DEX arbitrage opportunities
- Enhanced gas optimization algorithms

**Medium-term (3-6 months)**:
- Machine learning for price prediction
- Cross-chain trading support
- Advanced portfolio management
- Automated strategy optimization

**Long-term (6+ months)**:
- Custom AMM integration
- Institutional-grade risk management
- Advanced analytics dashboard
- Multi-wallet support

### 2. Scalability Considerations

**Performance Improvements**:
- Database integration for historical data
- Microservices architecture for horizontal scaling
- Advanced caching strategies
- Load balancing for high-frequency trading

**Security Enhancements**:
- Hardware security module integration
- Advanced threat detection
- Compliance framework integration
- Audit trail and reporting

---

## Conclusion

CryptoJackal represents a state-of-the-art cryptocurrency trading bot with enterprise-grade security, performance, and maintainability. The architecture prioritizes security through zero private key storage, performance through async-first design, and maintainability through modular structure and comprehensive testing.

The dual-AI development approach has proven highly effective, enabling rapid development while maintaining strict security compliance and code quality standards. The system is designed for extensibility and can be adapted for various trading strategies and market conditions.

**Key Architectural Strengths**:
- 🛡️ **Security-First**: Zero private key storage with MetaMask-only signing
- ⚡ **High Performance**: Async-first architecture with concurrent processing
- 🏗️ **Modular Design**: Clear separation of concerns with well-defined interfaces
- 🔄 **Event-Driven**: Loose coupling through comprehensive event systems
- 📊 **Observable**: Comprehensive logging, metrics, and health monitoring
- 🧪 **Well-Tested**: Extensive unit and integration test coverage

This architecture provides a solid foundation for continued development and enhancement of the CryptoJackal trading system.

---

*Architecture Version: 1.0*  
*Last Updated: January 2025*  
*Authors: Dual-AI Development Team (Windsurf + Cursor)*

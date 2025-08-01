# CryptoJackal Architecture

## Overview

CryptoJackal is a high-performance cryptocurrency sniper bot built in Rust, designed for automated trading on decentralized exchanges with a focus on Uniswap V2/V3.

## Module Structure

### Core Modules

```
src/
├── lib.rs              # Library entry point and common exports
├── main.rs             # Application entry point
├── error.rs            # Custom error types and error handling
├── core/               # Core bot logic and orchestration
│   ├── mod.rs          # Bot struct and main coordination
│   ├── config.rs       # Configuration management
│   ├── market.rs       # Market data and opportunity detection
│   └── types.rs        # Common data types
├── wallet/             # MetaMask integration and transaction signing
│   └── mod.rs          # Wallet connection and management
├── trading/            # DEX interaction and trade execution
│   └── mod.rs          # Trading logic and execution
└── utils/              # Common utilities and helper functions
    └── mod.rs          # Address validation, amount conversion, etc.
```

## Design Principles

### 1. Modular Architecture
- **Separation of Concerns**: Each module has a specific responsibility
- **Loose Coupling**: Modules communicate through well-defined interfaces
- **High Cohesion**: Related functionality is grouped together

### 2. Error Handling
- **Custom Error Types**: `CryptoJackalError` enum for different error categories
- **Result Pattern**: Consistent use of `Result<T>` throughout the codebase
- **Error Propagation**: Proper error context and chaining

### 3. Async-First Design
- **Tokio Runtime**: Full async/await support for high concurrency
- **Non-blocking Operations**: All I/O operations are asynchronous
- **Resource Sharing**: Arc<RwLock<T>> for shared state management

### 4. Security
- **No Private Key Storage**: MetaMask integration for secure signing
- **Input Validation**: Comprehensive validation of all user inputs
- **Safe Defaults**: Conservative default values for trading parameters

## Key Components

### Bot (core::Bot)
The main orchestrator that coordinates all other components:
- Manages wallet connections
- Monitors market opportunities
- Executes trading strategies
- Handles error recovery

### Wallet (wallet::Wallet)
Handles MetaMask integration:
- Connection management
- Transaction signing
- Balance queries
- Gas estimation

### Trading (trading::Trading)
Manages DEX interactions:
- Uniswap V2/V3 integration
- Trade execution
- Slippage protection
- MEV protection strategies

### Market (core::market::Market)
Monitors market conditions:
- Real-time price feeds
- Liquidity monitoring
- Opportunity detection
- Risk assessment

## Data Flow

```
Market Monitor → Opportunity Detection → Risk Assessment → Trade Execution
     ↓                    ↓                    ↓              ↓
WebSocket Feed → Filter Criteria → Validation → MetaMask Sign → DEX Contract
```

## Configuration

The bot uses a hierarchical configuration system:
1. Default values (hardcoded)
2. Configuration file (JSON)
3. Environment variables
4. Command-line arguments

## Testing Strategy

- **Unit Tests**: Each module has comprehensive unit tests
- **Integration Tests**: End-to-end testing of critical paths
- **Mock Services**: External dependencies are mocked for testing
- **Performance Tests**: Benchmarks for critical performance paths

## Deployment

The bot is designed to run as:
- **Standalone Binary**: Direct execution with configuration file
- **Docker Container**: Containerized deployment for production
- **Library Crate**: Integration into larger trading systems

## Future Enhancements

- Multi-DEX support (SushiSwap, PancakeSwap)
- Advanced MEV protection
- Machine learning price prediction
- Web dashboard for monitoring
- Mobile notifications

# CryptoJackal

A high-performance cryptocurrency sniper bot built in Rust, designed for rapid trading execution on Uniswap V2/V3 with MetaMask-only integration for maximum security.

## Features

- **High-speed trading execution** with optimized gas strategies
- **Zero private key storage** with MetaMask-only wallet integration
- **Real-time market monitoring** via Uniswap subgraph WebSocket
- **MEV protection** against front-running and sandwich attacks
- **Order execution queue** with prioritization and lifecycle management
- **Gas price optimization** for cost-effective transactions
- **Comprehensive testing suite** and performance monitoring

## Project Structure

```
src/
├── core/           # Core bot functionality
├── wallet/         # Wallet integration and management
├── trading/        # Trading logic and execution
└── utils/          # Utility functions and helpers
```

## Prerequisites

- Rust 1.70.0 or higher
- MetaMask wallet
- Ethereum/BSC node access (Infura or similar)

## Installation

1. Clone the repository:
```bash
git clone https://github.com/twadelij/CryptoJackal.git
cd CryptoJackal
```

2. Install dependencies:
```bash
cargo build
```

3. Configure your environment:
```bash
cp .env.example .env
# Edit .env with your configuration
```

## Usage

1. Configure your trading parameters in `config.toml`
2. Run the bot:
```bash
cargo run --release
```

## Security Considerations

- **Zero Private Key Storage**: All transaction signing is delegated to MetaMask
- **No Private Keys in Code or Config**: The system is designed to never require or store private keys
- **Environment Variables**: Use environment variables for node URLs and other sensitive data
- **Regular Dependency Audits**: All dependencies are regularly audited for security vulnerabilities
- **Testnet First**: Always test thoroughly on testnet before mainnet deployment

## Testing

Run the test suite:
```bash
cargo test
```

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Disclaimer

Trading cryptocurrencies carries significant risk. This bot is provided as-is with no guarantees. Use at your own risk.

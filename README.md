# 🐺 CryptoJackal

A fast, lightweight cryptocurrency trading bot built in Go.

## Features

- **Token Discovery** - Automatic discovery via CoinGecko and DexScreener
- **Paper Trading** - Risk-free simulation mode
- **Live Trading** - Execute real trades on Ethereum (when configured)
- **REST API** - Full API for integration and monitoring
- **Web Dashboard** - React-based UI for monitoring and control

## Quick Start

### Prerequisites

- Go 1.22+
- Docker (optional)

### Run Locally

```bash
# Clone and enter directory
cd CryptoJackal

# Copy environment file
cp .env.example .env

# Run the demo
make demo

# Or run the full bot
make run
```

### Run with Docker

```bash
# Build and run
docker compose up -d

# View logs
docker compose logs -f
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/health` | GET | Health check |
| `/api/bot/status` | GET | Get bot status |
| `/api/bot/start` | POST | Start the bot |
| `/api/bot/stop` | POST | Stop the bot |
| `/api/trading/opportunities` | GET | Get trading opportunities |
| `/api/trading/execute` | POST | Execute a trade |
| `/api/trading/history` | GET | Get trade history |
| `/api/discovery/trending` | GET | Get trending tokens |
| `/api/discovery/new` | GET | Get new tokens |
| `/api/discovery/analyze/:address` | GET | Analyze a token |
| `/api/paper/balance` | GET | Get paper trading balance |
| `/api/paper/reset` | POST | Reset paper trading |
| `/api/metrics` | GET | Get trading metrics |

## Configuration

See `.env.example` for all configuration options.

Key settings:
- `PAPER_TRADING_MODE=true` - Enable paper trading (recommended for testing)
- `ETH_NODE_URL` - Ethereum node URL (Infura, Alchemy, etc.)
- `PRIVATE_KEY` - Wallet private key (only for live trading)

## Project Structure

```
CryptoJackal/
├── cmd/
│   ├── cryptojackal/    # Main application
│   └── demo/            # Demo application
├── internal/
│   ├── api/             # HTTP API server
│   ├── config/          # Configuration
│   ├── discovery/       # Token discovery
│   ├── models/          # Data models
│   ├── paper/           # Paper trading
│   ├── trading/         # Trading engine
│   └── wallet/          # Wallet management
├── web/                 # React frontend
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

## Development

```bash
# Format code
make fmt

# Run tests
make test

# Build binaries
make build
```

## License

MIT

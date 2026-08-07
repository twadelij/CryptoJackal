# CryptoJackal

An autonomous cryptocurrency trading bot with multi-source data feeds, strategy engine, and ML-based trade evaluation. Built in Go with a React frontend.

## What It Does

- **Multi-Source Token Discovery** - Fetches tokens from DexScreener, GeckoTerminal, and CoinGecko with automatic failover
- **Rate Limiting & Caching** - Token bucket per source, 429 detection with exponential backoff, in-memory TTL cache
- **Strategy Engine** - 3 strategies (momentum breakout, dip buy, volume spike) with confidence scoring
- **Position Monitor** - Take profit / stop loss / trailing stop on open positions
- **Trade Journal & ML** - Records trade features and outcomes, trains logistic regression model to predict win probability
- **Paper Trading** - Practice trading with fake money (recommended for beginners)
- **Live Trading** - Execute real trades on Ethereum (requires setup)
- **Web Dashboard** - Control everything from your browser

## Quick Start

### Option 1: Docker (easiest)

```bash
git clone https://github.com/twadelij/CryptoJackal.git
cd CryptoJackal
cp .env.example .env
docker compose up --build
```

Open `http://localhost:8080` and log in with password from `.env` (default: `admin`).

### Option 2: Local Development

Requires: Go 1.22+, Node.js 20+, npm

```bash
git clone https://github.com/twadelij/CryptoJackal.git
cd CryptoJackal
cp .env.example .env
make dev
```

This starts the backend on `http://localhost:8080` and the frontend on `http://localhost:3000`.

### Option 3: Build and Run

```bash
make build
cd web && npm install && npm run build
cd ..
./bin/cryptojackal
```

Open `http://localhost:8080`.

## First Time Setup

**Paper Trading** works out of the box. No setup needed.

1. Log in with password from `.env` (or default `admin`)
2. Go to **Setup** and verify Paper Trading mode is enabled
3. Click **Start Bot** on the Dashboard
4. The bot will discover tokens and auto-trade in paper mode

**Live Trading** requires configuration:

1. Go to the **Setup** page in the dashboard (or edit `.env` directly)
2. Switch to **Live Trading** mode
3. Enter your Ethereum node URL (Infura, Alchemy, etc.)
4. Add your wallet private key (never share this)
5. Adjust trade amounts and stop loss settings

## Pages Explained

| Page | What It Does |
|------|-------------|
| **Dashboard** | Balance, P&L, total trades, win rate. Start/stop bot. API health indicators. |
| **Tokens** | Browse trending and new tokens. Search by name or symbol. Click a token to buy it. |
| **Portfolio** | Current holdings, average buy price, current value. Sell tokens here. |
| **History** | All past trades with timestamps and profit/loss per trade. |
| **Setup** | Trading mode, API keys, trade size, stop loss, scan interval. |

## Key Settings

| Setting | What It Does | Default |
|---------|-------------|---------|
| `PAPER_TRADING_MODE` | Use fake money instead of real money | `true` |
| `INITIAL_BALANCE` | Starting balance for paper trading | `10000` |
| `TRADE_AMOUNT` | How much to spend per trade | `100` |
| `STOP_LOSS` | Auto-sell if price drops this percentage | `5` |
| `MAX_SLIPPAGE` | Maximum price difference allowed during trade | `0.5` |
| `SCAN_INTERVAL_SECONDS` | How often to scan for opportunities | `180` (free tier) |
| `API_TIER` | API subscription tier: `free`, `basic`, `analyst` | `free` |
| `GECKOTERMINAL_ENABLED` | Enable GeckoTerminal as data source | `true` |
| `API_COOLDOWN_MINUTES` | Cooldown after 429 rate limit | `5` |
| `COINGECKO_API_KEY` | CoinGecko API key (optional, improves rate limits) | empty |
| `ETH_NODE_URL` | Your Ethereum RPC endpoint | empty |
| `PRIVATE_KEY` | Your wallet private key (live only) | empty |
| `ADMIN_PASSWORD` | Dashboard login password | `admin` |
| `JWT_SECRET` | Secret for session tokens | `change-me-in-production` |

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/health` | GET | Check if the server is running |
| `/api/health/external` | GET | Check CoinGecko/DexScreener connectivity |
| `/api/auth/login` | POST | Log in and get JWT token |
| `/api/config` | GET | View current configuration |
| `/api/config` | POST | Update configuration |
| `/api/bot/status` | GET | Check if bot is running |
| `/api/bot/start` | POST | Start auto-trading |
| `/api/bot/stop` | POST | Stop auto-trading |
| `/api/trading/opportunities` | GET | Get potential trades |
| `/api/trading/execute` | POST | Execute a manual trade |
| `/api/trading/history` | GET | Get trade history |
| `/api/discovery/trending` | GET | Get trending tokens |
| `/api/discovery/new` | GET | Get newly listed tokens |
| `/api/discovery/analyze/:address` | GET | Analyze a specific token |
| `/api/paper/balance` | GET | Get paper trading balance (real-time prices) |
| `/api/paper/reset` | POST | Reset paper trading to starting balance |
| `/api/paper/trade` | POST | Execute a paper trade |
| `/api/paper/history` | GET | Get paper trade history (filter: `?type=buy&status=executed&limit=50`) |
| `/api/paper/export` | GET | Export trades as JSON or CSV (`?format=csv`) |
| `/api/metrics` | GET | Get trading statistics |
| `/api/positions` | GET | Get open positions with live P&L |
| `/api/positions/:id/close` | POST | Manually close a position |
| `/api/strategies` | GET | List registered strategies |
| `/api/ml/status` | GET | ML model status (trained, samples, accuracy) |
| `/api/datasources/status` | GET | Status of all data providers |

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│                     Web Dashboard                         │
└──────────────────────────┬───────────────────────────────┘
                           │ HTTP (Gin)
                           ▼
┌──────────────────────────────────────────────────────────┐
│                   Trading Engine                          │
│                 internal/trading/engine.go                │
│                                                           │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐  │
│  │ Strategy    │  │ Position     │  │ Journal + ML    │  │
│  │ Engine      │  │ Monitor      │  │ Predictor       │  │
│  │ (3 strats)  │  │ (TP/SL/TS)   │  │ (logistic reg)  │  │
│  └──────┬──────┘  └──────────────┘  └─────────────────┘  │
│         │                                                 │
└─────────┼─────────────────────────────────────────────────┘
          ▼
┌──────────────────────────────────────────────────────────┐
│                 ProviderManager                           │
│              internal/datasource/manager.go               │
│                                                           │
│  Failover: DexScreener → GeckoTerminal → CoinGecko       │
│  Each with: RateLimiter + ResponseCache                   │
└──────────────────────────────────────────────────────────┘
```

### Internal Packages

| Package | Purpose |
|---------|---------|
| `internal/datasource` | Rate limiter (token bucket), TTL cache, GeckoTerminal client, ProviderManager with failover |
| `internal/discovery` | DexScreener + CoinGecko clients (implement Provider interface), discovery service |
| `internal/strategy` | Strategy interface + 3 strategies: momentum, dip buy, volume spike |
| `internal/portfolio` | Position monitor with take profit, stop loss, trailing stop |
| `internal/journal` | Trade journal recording features + outcomes for ML training |
| `internal/learning` | Logistic regression predictor (gradient descent), trains on trade history |
| `internal/trading` | Main engine: scan loop, monitor loop, safety rails, auto-execute |
| `internal/paper` | Paper trading service with portfolio management |
| `internal/config` | Config with API_TIER support (free/basic/analyst) |

### API Tiers

| Tier | Scan Interval | Rate Limits | Cost |
|------|--------------|------------|------|
| `free` | 180s | Conservative (DexScreener 300/min, CoinGecko 100/min, GeckoTerminal 30/min) | Free |
| `basic` | 60s | 3x limits, CoinGecko 300/min, GeckoTerminal 250/min | $29/mo |
| `analyst` | 30s | 5x limits, higher caps | $103/mo |

## Development

```bash
# Run backend + frontend in parallel (dev mode)
make dev

# Run tests
go test ./internal/...

# Build production binary
make build

# Run with Docker
make docker-up

# Stop Docker
make docker-down

# View logs
make logs
```

## Systemd Service (Linux)

To run on boot, copy the included service file:

```bash
sudo cp cryptojackal.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable cryptojackal
sudo systemctl start cryptojackal
```

## License

MIT

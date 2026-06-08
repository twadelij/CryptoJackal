# CryptoJackal

A cryptocurrency trading bot with a web dashboard. Built in Go with a React frontend.

## What It Does

- **Token Discovery** - Finds trending and new crypto tokens from CoinGecko and DexScreener
- **Paper Trading** - Practice trading with fake money (recommended for beginners)
- **Live Trading** - Execute real trades on Ethereum (requires setup)
- **Web Dashboard** - Control everything from your browser

## Quick Start

### Run the Backend

```bash
# Copy the example config
cp .env.example .env

# Paper trading is already enabled by default
go run ./cmd/cryptojackal
```

The backend runs on `http://localhost:8080`.

### Run the Web Dashboard (optional)

```bash
cd web
npm install
npm run dev
```

The dashboard opens on `http://localhost:3000`.

### Run Everything with Docker

```bash
docker compose up -d
```

The app is available at `http://localhost:8080`.

## First Time Setup

**Paper Trading** works out of the box. No setup needed.

**Live Trading** requires configuration:

1. Go to the **Setup** page in the dashboard (or edit `.env` directly)
2. Switch to **Live Trading** mode
3. Enter your Ethereum node URL (Infura, Alchemy, etc.)
4. Add your wallet private key (never share this)
5. Adjust trade amounts and stop loss settings

## Pages Explained

| Page | What It Does |
|------|-------------|
| **Dashboard** | See your balance, profit/loss, total trades, win rate. Start and stop the bot. |
| **Tokens** | Browse trending and new tokens. Search by name or symbol. Click a token to buy it. |
| **Portfolio** | See your current holdings, average buy price, and current value. Sell tokens here. |
| **History** | View all your past trades with timestamps and profit/loss per trade. |
| **Setup** | Configure trading mode, API keys, trade size, and stop loss percentage. |
| **Help** | Explanation of all features and settings. |

## Key Settings

| Setting | What It Does | Default |
|---------|-------------|---------|
| `PAPER_TRADING_MODE` | Use fake money instead of real money | `true` |
| `INITIAL_BALANCE` | Starting balance for paper trading | `10000` |
| `TRADE_AMOUNT` | How much to spend per trade | `100` |
| `STOP_LOSS` | Auto-sell if price drops this percentage | `5` |
| `MAX_SLIPPAGE` | Maximum price difference allowed during trade | `0.5` |
| `ETH_NODE_URL` | Your Ethereum RPC endpoint | empty |
| `PRIVATE_KEY` | Your wallet private key (live only) | empty |

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/health` | GET | Check if the server is running |
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
| `/api/paper/balance` | GET | Get paper trading balance |
| `/api/paper/reset` | POST | Reset paper trading to starting balance |
| `/api/paper/trade` | POST | Execute a paper trade |
| `/api/metrics` | GET | Get trading statistics |

## Development

```bash
# Run tests
go test ./internal/...

# Run the bot
go run ./cmd/cryptojackal

# Run the web dashboard
cd web && npm run dev

# Build for production
make build
cd web && npm run build
```

## License

MIT

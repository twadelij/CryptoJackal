# CryptoJackal Development Guide

> Een stap-voor-stap guide om CryptoJackal van prototype naar werkend product te brengen.
> Bedoeld voor een (junior) developer die het project niet kent.

---

## Inhoudsopgave

1. [Project Overzicht](#1-project-overzicht)
2. [Architectuur](#2-architectuur)
3. [Development Setup](#3-development-setup)
4. [Huidige Status](#4-huidige-status-wat-werkt-wat-niet)
5. [Fase 1: Fundament op orde](#fase-1-fundament-op-orde-week-1-2)
6. [Fase 2: Echte GUI](#fase-2-echte-gui-week-3-5)
7. [Fase 3: Persistence & Auth](#fase-3-persistence--auth-week-6-7)
8. [Fase 4: Real-time & Notifications](#fase-4-real-time--notifications-week-8-9)
9. [Fase 5: Live Trading](#fase-5-live-trading-week-10-12)
10. [Fase 6: Hardening & Deploy](#fase-6-hardening--deploy-week-13-14)
11. [API Reference](#api-reference)
12. [Coding Conventions](#coding-conventions)
13. [Troubleshooting](#troubleshooting)

---

## 1. Project Overzicht

CryptoJackal is een cryptocurrency trading bot geschreven in Go. Hij ontdekt tokens via CoinGecko en DexScreener, kan paper trades uitvoeren (simulatie), en heeft de basis voor live Ethereum trading.

**Doel:** Een bot die je via een web-GUI configureert, start, monitort, en die zowel paper als live kan traden.

**Tech stack:**
- **Backend:** Go 1.22, Gin (HTTP framework), go-ethereum (blockchain)
- **Frontend:** Momenteel een single embedded HTML file (geen React, ondanks wat README zegt)
- **Data sources:** CoinGecko API, DexScreener API
- **Infra:** Docker, docker-compose, Redis (nog niet in gebruik)

---

## 2. Architectuur

```
┌─────────────────────────────────────────────────────┐
│                    Web Browser                       │
│              (Dashboard / Setup GUI)                 │
└──────────────────────┬──────────────────────────────┘
                       │ HTTP / (toekomstig: WebSocket)
                       ▼
┌─────────────────────────────────────────────────────┐
│                   Gin HTTP Server                    │
│                 internal/api/server.go               │
│                                                      │
│  Routes:                                             │
│  /api/health          → Health check                 │
│  /api/bot/*           → Bot start/stop/status        │
│  /api/trading/*       → Opportunities, execute       │
│  /api/discovery/*     → Trending, new tokens         │
│  /api/paper/*         → Paper trading                │
│  /api/metrics         → Trading metrics              │
│  /                    → Embedded HTML dashboard      │
└────┬──────────┬───────────┬──────────┬──────────────┘
     │          │           │          │
     ▼          ▼           ▼          ▼
┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐
│Trading │ │Discover│ │ Paper  │ │ Wallet │
│Engine  │ │Service │ │Service │ │        │
│        │ │        │ │        │ │        │
│engine  │ │coingeck│ │in-mem  │ │go-eth  │
│.go     │ │o.go    │ │portfol │ │client  │
│        │ │dexscre │ │io      │ │        │
│        │ │ener.go │ │        │ │        │
└────────┘ └────────┘ └────────┘ └────────┘
```

### Bestanden en wat ze doen

| Bestand | Wat het doet |
|---------|-------------|
| `cmd/cryptojackal/main.go` | Entry point. Laadt config, init services, start server. |
| `cmd/demo/main.go` | Standalone demo die discovery + paper trading laat zien (CLI). |
| `internal/api/server.go` | Gin router setup, alle routes, embedded HTML. |
| `internal/api/handlers/handlers.go` | Alle HTTP handlers (request → response). |
| `internal/api/middleware/middleware.go` | Logger, recovery, JWT auth (niet actief), rate limiting. |
| `internal/api/templates/index.html` | De hele frontend: dashboard + setup wizard. ~480 regels vanilla JS + Tailwind. |
| `internal/config/config.go` | Leest `.env` file en environment variables. |
| `internal/discovery/discovery.go` | Orchestreert token discovery, caching, opportunity finding. |
| `internal/discovery/coingecko.go` | CoinGecko API client (trending, market data, token lookup). |
| `internal/discovery/dexscreener.go` | DexScreener API client (new pairs, boosted tokens, search). |
| `internal/models/models.go` | Alle data types: Token, Trade, Portfolio, BotStatus, Metrics. |
| `internal/paper/paper.go` | Paper trading: buy/sell simulatie, portfolio management, metrics. |
| `internal/trading/engine.go` | Trading engine: scan loop, opportunity detection, auto-trade. |
| `internal/wallet/wallet.go` | Ethereum wallet: connect, balance, sign & send transactions. |

---

## 3. Development Setup

### Vereisten

- **Go 1.22+** — `go version`
- **Docker** (optioneel) — `docker --version`
- **Git**

### Eerste keer opzetten

```bash
# 1. Clone het project (als je dat nog niet hebt)
cd ~/Projects
git clone <repo-url> CryptoJackal
cd CryptoJackal

# 2. Maak een .env file
cp .env.example .env

# 3. Dependencies downloaden
make deps
# Of: go mod download && go mod tidy

# 4. Draai de demo (test of alles compileert en de API's werken)
make demo

# 5. Start de volledige bot
make run
# Open http://localhost:8080 in je browser
```

### Handig tijdens development

```bash
# Code formatteren
make fmt

# Build binaries (naar bin/)
make build

# Docker draaien
docker compose up -d
docker compose logs -f cryptojackal

# Alles stoppen
docker compose down
```

### IDE tips
- Gebruik VSCode of GoLand met Go extension
- `gopls` voor autocompletion
- `golangci-lint` voor linting: `make lint`

---

## 4. Huidige Status: Wat werkt, wat niet

### ✅ Werkt

| Feature | Details |
|---------|---------|
| Backend compileert en start | `make run` werkt |
| API endpoints | Alle routes in server.go zijn actief |
| CoinGecko trending tokens | `GET /api/discovery/trending` |
| DexScreener new/boosted tokens | `GET /api/discovery/new` |
| Token analysis | `GET /api/discovery/analyze/:address` |
| Paper trading (buy/sell) | `POST /api/paper/trade` |
| Portfolio tracking | `GET /api/paper/balance` |
| Trading metrics | `GET /api/metrics` |
| Bot start/stop | `POST /api/bot/start` en `/stop` |
| Auto paper trading | Engine scant en trade automatisch bij hoge confidence |
| Dashboard HTML | Basis UI met token lijst, trade modal, history |
| Docker setup | Dockerfile + docker-compose met Redis |

### ❌ Werkt niet / ontbreekt

| Issue | Impact | Prioriteit |
|-------|--------|-----------|
| **Setup wizard is nep** | Slaat alleen op in browser localStorage, stuurt niets naar backend | HOOG |
| **Geen persistence** | Alle trades, portfolio, config weg bij restart | HOOG |
| **Geen tests** | Geen enkele test in het project | HOOG |
| **Geen authenticatie** | JWT middleware bestaat maar is niet gekoppeld aan routes | MEDIUM |
| **Redis ongebruikt** | Staat in docker-compose maar code gebruikt het niet | MEDIUM |
| **Geen real-time updates** | Frontend pollt elke 30s, geen WebSocket | MEDIUM |
| **Geen notificaties** | Telegram/Discord config fields bestaan, geen implementatie | LAAG |
| **Live trading is stub** | `ExecuteTrade` in engine.go retourneert `nil, nil` voor live | LAAG (eerst paper perfectioneren) |
| **Frontend is 1 groot HTML bestand** | Moeilijk te onderhouden, geen componenten | MEDIUM |
| **Geen error handling in frontend** | API failures worden stil geslikt | MEDIUM |

---

## Fase 1: Fundament op orde (Week 1-2)

> **Doel:** Tests, code quality, en de basis werkend krijgen zodat je veilig kunt doorontwikkelen.

### Stap 1.1: Eerste tests schrijven

Maak een `_test.go` file naast elk package. Begin met de makkelijkste: paper trading.

**Maak:** `internal/paper/paper_test.go`

```go
package paper

import (
    "context"
    "testing"

    "github.com/twadelij/cryptojackal/internal/models"
    "go.uber.org/zap"
)

func newTestService() *Service {
    logger, _ := zap.NewDevelopment()
    return NewService(10.0, logger) // 10 EUR startbalans
}

func TestNewService(t *testing.T) {
    svc := newTestService()
    portfolio := svc.GetPortfolio()

    if portfolio.Balance != 10.0 {
        t.Errorf("expected balance 10.0, got %f", portfolio.Balance)
    }
    if portfolio.Currency != "EUR" {
        t.Errorf("expected currency EUR, got %s", portfolio.Currency)
    }
}

func TestBuyTrade(t *testing.T) {
    svc := newTestService()
    ctx := context.Background()

    token := models.Token{
        Address: "0xTEST",
        Symbol:  "TEST",
        Name:    "Test Token",
        Price:   0.001, // 0.001 EUR per token
    }

    // Koop 1000 tokens = 1 EUR
    trade, err := svc.ExecuteTrade(ctx, token, models.TradeTypeBuy, 1000)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if trade.Status != models.TradeStatusExecuted {
        t.Errorf("expected status executed, got %s", trade.Status)
    }

    portfolio := svc.GetPortfolio()
    if portfolio.Balance != 9.0 {
        t.Errorf("expected balance 9.0, got %f", portfolio.Balance)
    }
}

func TestBuyInsufficientBalance(t *testing.T) {
    svc := newTestService()
    ctx := context.Background()

    token := models.Token{
        Address: "0xTEST",
        Symbol:  "TEST",
        Price:   1.0, // 1 EUR per token
    }

    // Probeer 100 tokens te kopen = 100 EUR (maar we hebben maar 10)
    _, err := svc.ExecuteTrade(ctx, token, models.TradeTypeBuy, 100)
    if err == nil {
        t.Error("expected error for insufficient balance")
    }
}

func TestSellTrade(t *testing.T) {
    svc := newTestService()
    ctx := context.Background()

    token := models.Token{
        Address: "0xTEST",
        Symbol:  "TEST",
        Price:   0.001,
    }

    // Eerst kopen
    svc.ExecuteTrade(ctx, token, models.TradeTypeBuy, 1000)

    // Dan verkopen met hogere prijs
    token.Price = 0.002
    trade, err := svc.ExecuteTrade(ctx, token, models.TradeTypeSell, 1000)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if trade.ProfitLoss <= 0 {
        t.Errorf("expected positive P&L, got %f", trade.ProfitLoss)
    }
}

func TestReset(t *testing.T) {
    svc := newTestService()
    ctx := context.Background()

    token := models.Token{Address: "0xTEST", Symbol: "TEST", Price: 0.001}
    svc.ExecuteTrade(ctx, token, models.TradeTypeBuy, 1000)

    svc.Reset()

    portfolio := svc.GetPortfolio()
    if portfolio.Balance != 10.0 {
        t.Errorf("expected balance reset to 10.0, got %f", portfolio.Balance)
    }
    if len(portfolio.TokenBalances) != 0 {
        t.Errorf("expected empty token balances after reset")
    }
}

func TestMetrics(t *testing.T) {
    svc := newTestService()
    ctx := context.Background()

    token := models.Token{Address: "0xTEST", Symbol: "TEST", Price: 0.001}
    svc.ExecuteTrade(ctx, token, models.TradeTypeBuy, 1000)

    metrics := svc.GetMetrics()
    if metrics.TotalTrades != 1 {
        t.Errorf("expected 1 trade, got %d", metrics.TotalTrades)
    }
}
```

**Run:** `go test -v ./internal/paper/`

**Daarna tests toevoegen voor:**
- `internal/config/` — test dat defaults kloppen
- `internal/models/` — test NewTrade, NewOpportunity
- `internal/api/handlers/` — HTTP handler tests met httptest

### Stap 1.2: Config API endpoint toevoegen

Het probleem: de setup wizard in de browser kan de backend niet configureren.

**Wat te doen:**
1. Voeg een `POST /api/config` endpoint toe aan `handlers.go`
2. Dit endpoint ontvangt de wizard-settings en schrijft ze naar een config bestand (of update runtime config)
3. Voeg een `GET /api/config` endpoint toe die de huidige config retourneert (zonder private key!)

```go
// In handlers.go - toe te voegen

type ConfigUpdateRequest struct {
    PaperTradingMode bool    `json:"paper_trading_mode"`
    InitialBalance   float64 `json:"initial_balance"`
    EthNodeURL       string  `json:"eth_node_url"`
    TradeAmount      float64 `json:"trade_amount"`
    MaxSlippage      float64 `json:"max_slippage"`
    StopLoss         float64 `json:"stop_loss"`
}

func (h *Handler) GetConfig(c *gin.Context) {
    // Return config WITHOUT sensitive fields
    c.JSON(http.StatusOK, Response{
        Success: true,
        Data: gin.H{
            "paper_trading_mode": true, // TODO: haal uit echte config
            "initial_balance":    10.0,
            // Voeg meer velden toe, NOOIT private_key retourneren
        },
    })
}

func (h *Handler) UpdateConfig(c *gin.Context) {
    var req ConfigUpdateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, Response{Success: false, Error: err.Error()})
        return
    }
    // TODO: validatie + opslaan
    c.JSON(http.StatusOK, Response{Success: true, Data: "Config updated"})
}
```

**En registreer de routes in server.go:**
```go
api.GET("/config", handler.GetConfig)
api.POST("/config", handler.UpdateConfig)
```

**En update de JavaScript in index.html** zodat `saveConfig()` daadwerkelijk een POST doet naar `/api/config`.

### Stap 1.3: Error handling verbeteren

In de frontend worden veel errors stilletjes geslikt. Voeg overal duidelijke foutmeldingen toe:

```javascript
// Slechte pattern (huidige code):
} catch (e) { console.error(e); }

// Goede pattern:
} catch (e) {
    console.error('Failed to load balance:', e);
    showNotification('Kon balans niet laden. Check of de server draait.', 'error');
}
```

Loop alle `catch (e)` blokken in `index.html` af en voeg gebruikersvriendelijke meldingen toe.

---

## Fase 2: Echte GUI (Week 3-5)

> **Doel:** De single-file HTML vervangen door een echte React app, zodat de GUI onderhoudbaar en uitbreidbaar wordt.

### Optie A: React frontend (aanbevolen)

Maak een aparte `web/` directory met een React app:

```bash
cd CryptoJackal
npx create-react-app web --template typescript
# Of met Vite (sneller):
npm create vite@latest web -- --template react-ts
cd web
npm install
```

**Aanbevolen libraries:**
- `@tanstack/react-query` — voor data fetching + caching
- `tailwindcss` — styling (al gebruikt in huidige HTML)
- `lucide-react` — icons
- `recharts` — grafieken voor P&L, portfolio
- `react-router-dom` — pagina navigatie
- `react-hot-toast` — notificaties

### Optie B: Huidige HTML verbeteren (sneller, minder onderhoudbaar)

Als je geen React wilt, kun je de huidige `index.html` verbeteren door:
- Alpine.js toe te voegen voor reactivity
- De code op te splitsen in meerdere `<script>` tags
- Betere error states toe te voegen

### Pagina's voor de React app

```
/                → Dashboard (huidige stats + bot controls)
/setup           → Setup wizard (stap 1-2-3)
/tokens          → Token discovery (trending + new)
/token/:address  → Token detail pagina met analyse
/trade           → Trade uitvoeren (paper of live)
/portfolio       → Portfolio overzicht met alle holdings
/history         → Trade history met filters
/settings        → Bot instellingen wijzigen
```

### Stap 2.1: Dashboard pagina

De dashboard pagina moet tonen:
- **Bovenaan:** Status badge (online/offline), mode (paper/live)
- **Stats cards:** Balance, P&L, Total Trades, Win Rate
- **Bot controls:** Start/Stop knoppen
- **Mini charts:** P&L over tijd (als je history hebt)

Data ophalen met react-query:

```typescript
// hooks/useApi.ts
import { useQuery, useMutation } from '@tanstack/react-query';

const API_BASE = '/api';

export function useBotStatus() {
    return useQuery({
        queryKey: ['bot-status'],
        queryFn: () => fetch(`${API_BASE}/bot/status`).then(r => r.json()),
        refetchInterval: 5000, // Poll elke 5 seconden
    });
}

export function usePaperBalance() {
    return useQuery({
        queryKey: ['paper-balance'],
        queryFn: () => fetch(`${API_BASE}/paper/balance`).then(r => r.json()),
        refetchInterval: 10000,
    });
}

export function useStartBot() {
    return useMutation({
        mutationFn: () => fetch(`${API_BASE}/bot/start`, { method: 'POST' }),
    });
}
```

### Stap 2.2: Setup Wizard

De wizard moet echt werken:

1. **Stap 1 - Mode:** Paper of Live selecteren
2. **Stap 2 - API Keys:** Ethereum node URL invullen. Bij live: private key.
3. **Stap 3 - Trading params:** Initial balance, trade size, stop loss.
4. **Stap 4 - Bevestiging:** Overzicht tonen, "Start" knop.

Bij "Start" → `POST /api/config` met alle settings → redirect naar dashboard.

### Stap 2.3: Token Discovery pagina

- Tabel met trending/new tokens
- Klik op een token → detail pagina met:
  - Prijs, volume, liquiditeit, 24h change
  - Security score (al berekend door backend)
  - "Paper Trade" knop
- Zoekbalk om token by address op te zoeken (`/api/discovery/analyze/:address`)

### Stap 2.4: Portfolio pagina

- Overzicht van alle token holdings
- Per token: amount, avg buy price, current price, P&L
- Totale portfolio waarde
- "Sell" knop per token

### Stap 2.5: Trade History pagina

- Lijst van alle trades
- Filters: type (buy/sell), token, datum
- Sorteerbaar op datum, amount, P&L

### Stap 2.6: Backend aanpassen voor frontend serving

Als je een aparte React app hebt, moet de backend die kunnen serven:

**Optie 1 (development):** React dev server op :3000, backend op :8080, CORS is al geconfigureerd.

**Optie 2 (production):** Build de React app (`npm run build`) en embed de output in de Go binary:

```go
// In server.go — vervang de huidige embed
//go:embed all:dist
var frontendDist embed.FS

// Serve de React build
router.NoRoute(func(c *gin.Context) {
    // Probeer eerst het gevraagde bestand te serven
    // Fallback naar index.html (voor client-side routing)
})
```

---

## Fase 3: Persistence & Auth (Week 6-7)

> **Doel:** Data bewaren na restart. API beveiligen.

### Stap 3.1: SQLite voor persistence

SQLite is de makkelijkste optie — geen extra server nodig.

```bash
go get github.com/mattn/go-sqlite3
# Of voor een pure-Go driver (geen CGO):
go get modernc.org/sqlite
```

**Maak:** `internal/storage/storage.go`

Sla op:
- **Trades:** Elke buy/sell met alle details
- **Portfolio snapshots:** Periodiek de portfolio state opslaan
- **Config:** Runtime configuratie
- **Token cache:** Ontdekte tokens cachen

**Schema voorbeeld:**

```sql
CREATE TABLE IF NOT EXISTS trades (
    id TEXT PRIMARY KEY,
    token_address TEXT NOT NULL,
    token_symbol TEXT NOT NULL,
    type TEXT NOT NULL,
    amount_in REAL NOT NULL,
    amount_out REAL,
    price REAL NOT NULL,
    profit_loss REAL,
    status TEXT NOT NULL,
    is_paper_trade BOOLEAN,
    executed_at DATETIME
);

CREATE TABLE IF NOT EXISTS portfolio (
    id TEXT PRIMARY KEY,
    balance REAL NOT NULL,
    total_value REAL NOT NULL,
    updated_at DATETIME
);

CREATE TABLE IF NOT EXISTS token_balances (
    portfolio_id TEXT,
    token_address TEXT,
    balance REAL,
    avg_price REAL,
    FOREIGN KEY (portfolio_id) REFERENCES portfolio(id)
);

CREATE TABLE IF NOT EXISTS config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at DATETIME
);
```

### Stap 3.2: Paper service aanpassen

De `paper.Service` moet trades schrijven naar de database in plaats van alleen in-memory.

```go
// Huidige code (in-memory):
s.trades = append(s.trades, *trade)

// Nieuwe code (met storage):
s.trades = append(s.trades, *trade)
if s.storage != nil {
    s.storage.SaveTrade(trade)
}
```

Bij startup: laad portfolio en trades uit de database.

### Stap 3.3: Redis voor caching (optioneel)

Redis staat al in docker-compose. Gebruik het voor:
- Token discovery cache (i.p.v. de huidige in-memory cache in discovery.go)
- Rate limit counters
- Session storage

```bash
go get github.com/redis/go-redis/v9
```

### Stap 3.4: JWT Authenticatie activeren

De JWT middleware staat al klaar in `middleware.go`. Om hem te activeren:

1. **Login endpoint toevoegen:**
```go
// POST /api/auth/login
type LoginRequest struct {
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required"`
}
```

2. **Token genereren:**
```go
token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
    "sub": username,
    "exp": time.Now().Add(24 * time.Hour).Unix(),
})
tokenString, _ := token.SignedString([]byte(cfg.JWTSecret))
```

3. **Middleware koppelen aan routes:**
```go
// In server.go — bescherm alle API routes behalve health en login
api.Use(middleware.JWTAuth(cfg.JWTSecret))
```

4. **Frontend aanpassen:** Token opslaan na login, meesturen als `Authorization: Bearer <token>` header.

**Tip:** Voor single-user setup (jij bent de enige gebruiker) kun je simpelweg een wachtwoord in de .env zetten. Geen user database nodig.

---

## Fase 4: Real-time & Notifications (Week 8-9)

> **Doel:** Live updates in de browser, meldingen bij trades.

### Stap 4.1: WebSocket toevoegen

Gin heeft geen ingebouwde WebSocket support. Gebruik `gorilla/websocket` (al in go.mod als indirect dependency).

```go
// internal/api/websocket.go
import "github.com/gorilla/websocket"

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool { return true },
}

type WSMessage struct {
    Type string      `json:"type"` // "trade", "balance", "opportunity", "status"
    Data interface{} `json:"data"`
}
```

**Events om te broadcasten:**
- `trade_executed` — als een trade is uitgevoerd
- `balance_updated` — als portfolio verandert
- `opportunity_found` — als de scanner een kans vindt
- `bot_status` — als bot start/stopt
- `price_update` — periodiek token prijzen

**Route:**
```go
router.GET("/ws", func(c *gin.Context) { handleWebSocket(c) })
```

**Frontend (React):**
```typescript
useEffect(() => {
    const ws = new WebSocket('ws://localhost:8080/ws');
    ws.onmessage = (event) => {
        const msg = JSON.parse(event.data);
        switch (msg.type) {
            case 'trade_executed':
                // Refresh trade history
                queryClient.invalidateQueries(['trade-history']);
                break;
            case 'balance_updated':
                // Refresh balance
                queryClient.invalidateQueries(['paper-balance']);
                break;
        }
    };
    return () => ws.close();
}, []);
```

### Stap 4.2: Telegram notificaties

**Maak:** `internal/notifications/telegram.go`

```go
package notifications

import (
    "fmt"
    "net/http"
    "net/url"
)

type TelegramNotifier struct {
    botToken string
    chatID   string
    client   *http.Client
}

func (t *TelegramNotifier) Send(message string) error {
    apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)
    _, err := t.client.PostForm(apiURL, url.Values{
        "chat_id": {t.chatID},
        "text":    {message},
        "parse_mode": {"HTML"},
    })
    return err
}
```

**Waar notificaties sturen:**
- Trade uitgevoerd (paper of live)
- Bot gestart/gestopt
- Significante P&L verandering
- Error (API failure, connection lost)

**Setup voor de gebruiker:**
1. Maak een Telegram bot via @BotFather
2. Krijg je `TELEGRAM_BOT_TOKEN`
3. Start een chat met de bot, stuur `/start`
4. Vind je `TELEGRAM_CHAT_ID` via `https://api.telegram.org/bot<TOKEN>/getUpdates`
5. Vul beide in via de .env of setup wizard

### Stap 4.3: Discord webhook (optioneel)

Simpeler dan Telegram — gewoon een POST naar de webhook URL:

```go
func (d *DiscordNotifier) Send(message string) error {
    body := fmt.Sprintf(`{"content": "%s"}`, message)
    _, err := http.Post(d.webhookURL, "application/json", strings.NewReader(body))
    return err
}
```

---

## Fase 5: Live Trading (Week 10-12)

> **Doel:** Echte trades op Ethereum uitvoeren. **VOORZICHTIG!**

### ⚠️ Waarschuwingen

- **Begin ALTIJD met een testnet** (Sepolia, Goerli)
- **Gebruik NOOIT je main wallet** — maak een aparte wallet met klein bedrag
- **Test uitgebreid** in paper mode voordat je live gaat
- **Set harde limieten**: max trade size, max dagelijks verlies, max open posities

### Stap 5.1: Uniswap integratie

De wallet kan al ETH transfers doen. Voor token trading heb je een DEX nodig.

```bash
go get github.com/ethereum/go-ethereum
# De Uniswap V2/V3 Router ABI's heb je nodig als Go bindings
```

**Maak:** `internal/trading/uniswap.go`

De Uniswap V2 Router (simpeler dan V3 voor beginners):
- `swapExactETHForTokens` — ETH → Token (buy)
- `swapExactTokensForETH` — Token → ETH (sell)

**Stappen:**
1. Token approval (allowance) voor de Uniswap router
2. Build de swap transaction
3. Estimate gas
4. Sign met wallet
5. Send transaction
6. Wait for receipt
7. Update portfolio

### Stap 5.2: Gas estimation & management

```go
// Haal gas prijs op
gasPrice, _ := wallet.GetGasPrice(ctx)

// Check tegen max
maxGasWei := new(big.Int).Mul(
    big.NewInt(int64(cfg.MaxGasPrice)),
    big.NewInt(1e9), // Gwei to Wei
)
if gasPrice.Cmp(maxGasWei) > 0 {
    return fmt.Errorf("gas price too high: %s > %s", gasPrice, maxGasWei)
}
```

### Stap 5.3: Slippage protection

```go
// Bereken minimum output met slippage tolerance
minOutput = expectedOutput * (1 - maxSlippage/100)
```

### Stap 5.4: Safety features

**Implementeer voordat je live gaat:**
- [ ] Max trade size limiet
- [ ] Max dagelijks verlies (kill switch)
- [ ] Max open posities
- [ ] Cooldown tussen trades
- [ ] Price sanity check (weiger trades bij extreme prijzen)
- [ ] Balance check voor elke trade
- [ ] Transaction confirmation wachten
- [ ] Retry logic met backoff

### Stap 5.5: Engine aanpassen

```go
// In engine.go — huidige stub vervangen:
func (e *Engine) ExecuteTrade(ctx context.Context, opp models.TradingOpportunity, amount float64) (*models.Trade, error) {
    if e.config.PaperTradingMode {
        return e.paper.ExecuteTrade(ctx, opp.Token, models.TradeTypeBuy, amount)
    }

    // Live trading
    if e.wallet == nil || !e.wallet.IsConfigured() {
        return nil, fmt.Errorf("wallet not configured for live trading")
    }

    // 1. Safety checks
    // 2. Build swap transaction
    // 3. Sign & send
    // 4. Wait for confirmation
    // 5. Record trade
    // 6. Send notification

    return trade, nil
}
```

---

## Fase 6: Hardening & Deploy (Week 13-14)

> **Doel:** Production-ready maken.

### Stap 6.1: Logging verbeteren

De huidige zap logger is goed. Voeg toe:
- Log rotation (met `lumberjack`)
- Aparte log files voor trades
- Structured logging met trade IDs voor debugging

### Stap 6.2: Monitoring

- `/api/health` uitbreiden met dependency checks (Redis, Ethereum node)
- `/api/metrics` compatible maken met Prometheus
- Uptime tracking
- Error rate tracking

### Stap 6.3: Rate limiting activeren

De rate limiter middleware staat klaar. Activeer hem:

```go
// In server.go
api.Use(middleware.RateLimit(10)) // 10 requests per seconde
```

### Stap 6.4: Docker optimalisatie

De Dockerfile is al multi-stage. Verbeteringen:
- `.dockerignore` toevoegen (skip `.git`, `node_modules`, etc.)
- Health check verbeteren
- Environment-specifieke compose files (dev vs prod)

### Stap 6.5: Deployment opties

**Optie 1: Docker op eigen server**
```bash
docker compose -f docker-compose.prod.yml up -d
```

**Optie 2: VPS (Hetzner, DigitalOcean)**
- Klein VPS is genoeg (2GB RAM)
- Caddy of nginx als reverse proxy
- Let's Encrypt voor HTTPS
- Systemd service als alternatief voor Docker

**Optie 3: Lokaal draaien**
- Gewoon `make run` op je eigen machine
- Gebruik ngrok of Tailscale voor remote access

---

## API Reference

Volledige lijst van alle endpoints:

### Health & Status

| Method | Endpoint | Body | Response |
|--------|----------|------|----------|
| GET | `/api/health` | - | `{status, version}` |
| GET | `/api/bot/status` | - | `{is_running, mode, total_trades, ...}` |
| POST | `/api/bot/start` | - | `"Bot started"` |
| POST | `/api/bot/stop` | - | `"Bot stopped"` |

### Discovery

| Method | Endpoint | Body | Response |
|--------|----------|------|----------|
| GET | `/api/discovery/trending` | - | `[Token, ...]` |
| GET | `/api/discovery/new?chain=ethereum` | - | `[Token, ...]` |
| GET | `/api/discovery/analyze/:address` | - | `Token` |

### Trading

| Method | Endpoint | Body | Response |
|--------|----------|------|----------|
| GET | `/api/trading/opportunities` | - | `[TradingOpportunity, ...]` |
| POST | `/api/trading/execute` | `{opportunity_id, amount}` | `Trade` |
| GET | `/api/trading/history` | - | `[Trade, ...]` |

### Paper Trading

| Method | Endpoint | Body | Response |
|--------|----------|------|----------|
| GET | `/api/paper/balance` | - | `Portfolio` |
| POST | `/api/paper/trade` | `{token_address, token_symbol, token_name, price, amount, type}` | `Trade` |
| POST | `/api/paper/reset` | - | `"Portfolio reset"` |
| GET | `/api/paper/history` | - | `[Trade, ...]` |

### Metrics

| Method | Endpoint | Body | Response |
|--------|----------|------|----------|
| GET | `/api/metrics` | - | `Metrics` |

### Config (nog te bouwen)

| Method | Endpoint | Body | Response |
|--------|----------|------|----------|
| GET | `/api/config` | - | Config (zonder secrets) |
| POST | `/api/config` | `ConfigUpdateRequest` | `"Config updated"` |

---

## Coding Conventions

### Go

- Alle code in `internal/` — niet bedoeld als library
- Elk package heeft een duidelijke verantwoordelijkheid
- Error handling: altijd `if err != nil` checken, nooit negeren
- Logging: gebruik `zap.Logger`, niet `fmt.Println`
- Context: propageer `context.Context` door alle functies
- Concurrency: gebruik `sync.RWMutex` voor shared state

### Frontend

- Tailwind CSS voor styling
- Consistente kleurpaletten (dark theme met pink/red accenten)
- Alle API calls via een centrale `api.ts` helper
- Error states tonen aan gebruiker, nooit stil falen

### Git

- Feature branches: `feature/setup-wizard`, `feature/persistence`
- Commit messages: `feat: add config API endpoint`, `fix: paper trade balance calc`
- PR per feature, niet alles in 1 commit

---

## Troubleshooting

### "make run" faalt met import errors

```bash
go mod tidy
go mod download
```

### CoinGecko API geeft 429 (rate limit)

De gratis CoinGecko API heeft een limiet van ~10-30 calls/minuut. Oplossingen:
- Verhoog `SCAN_INTERVAL_SECONDS` in .env
- Voeg een (gratis) API key toe: `COINGECKO_API_KEY`
- De cache in `discovery.go` voorkomt al herhaalde calls (TTL: 5 min)

### DexScreener retourneert lege data

DexScreener's "boosted tokens" endpoint kan leeg zijn als er geen recent geboostede tokens zijn. Dit is normaal. De tokens lijst in de UI zal dan leeg zijn.

### Docker build faalt

Check of je Go version in Dockerfile matcht met go.mod:
- `go.mod` zegt `go 1.22`
- `Dockerfile` zegt `golang:1.22-alpine`

### Private key format

De private key moet ZONDER `0x` prefix in de .env:
```
# Fout:
PRIVATE_KEY=0xabcdef1234...

# Goed:
PRIVATE_KEY=abcdef1234...
```

### Paper trading toont verkeerde valuta

Het project is halverwege gemigreerd van ETH naar EUR als base currency. De `ETHBalance` in Portfolio is deprecated maar wordt nog voor backward compatibility bijgehouden. Gebruik `Balance` en `Currency` fields.

---

## Checklist: Eerste test in the real world

Voordat je CryptoJackal "echt" gaat gebruiken, moet dit allemaal werken:

- [ ] `make run` start zonder errors
- [ ] Dashboard opent op http://localhost:8080
- [ ] Setup wizard configureert de backend (niet alleen localStorage)
- [ ] Paper trading werkt: koop een token, zie het in portfolio, verkoop het
- [ ] Data overleeft een restart (persistence)
- [ ] Je kunt trending tokens zien en er op klikken
- [ ] Trade history toont al je trades
- [ ] Bot auto-scan vindt opportunities en toont ze
- [ ] Notificaties werken (minimaal Telegram)
- [ ] (Optioneel) Live trading werkt op testnet
- [ ] (Optioneel) HTTPS en auth staan aan

---

*Laatste update: mei 2025*
*Vragen? Open een issue of vraag het aan de senior developer (Ted).*

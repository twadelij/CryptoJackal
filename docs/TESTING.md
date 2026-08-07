# CryptoJackal Testing Guide

## Hoe te testen (voor jouw uurtje per dag)

### Start de app
```bash
cd /home/twadelij/Projects/CryptoJackal
go run ./cmd/cryptojackal
```
Of met Docker:
```bash
docker compose up --build
```

### 1. Paper Trading Demo (de eerste 5 minuten)

**Doel:** Zorg dat een nieuwe gebruiker direct kan spelen zonder setup.

**Test stappen:**
1. Open http://localhost:8080
2. Log in met password `admin` (of wat je in .env hebt gezet)
3. Check `/api/paper/balance` - je zou 10.0 EUR moeten zien
4. Ga naar `/api/discovery/trending` - pak een token address
5. Doe een paper trade:
   ```bash
   curl -X POST http://localhost:8080/api/paper/trade \
     -H "Authorization: Bearer <token>" \
     -H "Content-Type: application/json" \
     -d '{"token_address":"0x1234...","token_symbol":"DOJI","price":0.000004,"amount":1000,"type":"buy"}'
   ```
6. Check `/api/paper/balance` - balance moet lager zijn, tokens hoger
7. Check `/api/paper/history` - je trade moet erin staan

**Wat moet werken:**
- [ ] Login geeft JWT token
- [ ] Paper balance is 10.0 EUR default
- [ ] Buy trade verlaagt EUR balance, verhoogt token count
- [ ] Sell trade verhoogt EUR balance, verlaagt token count
- [ ] Trade history toont trade met timestamp
- [ ] Metrics endpoint toont win/loss stats

### 2. Crash Recovery Test

**Doel:** App crasht, herstart, portfolio moet intact zijn.

**Test stappen:**
1. Start app, doe 2-3 paper trades
2. Kill de app (Ctrl+C of `kill -9`)
3. Herstart app
4. Check `/api/paper/balance` - portfolio moet exact hetzelfde zijn
5. Check `/api/paper/history` - alle trades moeten er nog zijn

**Wat moet werken:**
- [ ] Trades blijven na restart
- [ ] Portfolio balance blijft na restart
- [ ] Geen data corruption in SQLite

### 3. Input Validation Test (security check)

**Doel:** API weigert rotzooi.

**Test stappen:**
1. POST naar `/api/paper/trade` met `amount: -100` -> moet 400 geven
2. POST naar `/api/paper/trade` met `type: "hack"` -> moet 400 geven
3. GET `/api/discovery/new?chain=evil_chain` -> moet 400 geven
4. GET `/api/discovery/analyze/0xNOTANADDRESS` -> moet 400 geven
5. POST `/api/trading/execute` zonder `opportunity_id` -> moet 400 geven

**Wat moet werken:**
- [ ] Negatieve amounts worden geweigerd
- [ ] Invalid trade types worden geweigerd
- [ ] Invalid chain parameter wordt geweigerd
- [ ] Invalid Ethereum address wordt geweigerd
- [ ] Missing required fields worden geweigerd

### 4. Config Persistence Test

**Doel:** Config wijzigingen blijven bestaan.

**Test stappen:**
1. POST naar `/api/config` met `{"initial_balance": 50.0}`
2. Herstart app
3. GET `/api/config` - `initial_balance` moet 50.0 zijn
4. Reset naar 10.0 via POST

### 5. Frontend Check

**Doel:** Dashboard laad correct.

**Test stappen:**
1. Open http://localhost:8080
2. Log in
3. Dashboard toont portfolio value
4. "Trending Tokens" tab laadt tokens
5. "Paper Trading" tab laadt trades
6. Mobile view (Chrome DevTools responsive) werkt zonder scroll-issues

### 6. Strategy Engine Test

**Doel:** Strategieen genereren signals met juiste confidence.

**Test stappen:**
1. Start app in paper mode
2. Start bot (`POST /api/bot/start`)
3. Wacht een scan cycle (3 min op free tier)
4. Check `/api/trading/opportunities` - moet signals bevatten met strategy naam
5. Check `/api/strategies` - moet 3 strategieen tonen: momentum, dip_buy, volume_spike

**Wat moet werken:**
- [ ] Opportunities hebben een strategy field
- [ ] Confidence scores zijn tussen 0.0 en 1.0
- [ ] Best signal wordt auto-executed bij confidence > 0.55

### 7. Position Monitor Test

**Doel:** TP/SL werkt op open positions.

**Test stappen:**
1. Start bot, wacht tot er een auto-trade is
2. Check `/api/positions` - moet de open positie tonen
3. Wacht tot TP of SL threshold bereikt wordt (of simuleer met hoge TP/SL)
4. Position moet automatisch gesloten worden
5. Check `/api/paper/history` - moet een sell trade tonen

**Wat moet werken:**
- [ ] Open positions tonen entry price en strategy
- [ ] TP/SL triggers sluiten de positie automatisch
- [ ] Trailing stop werkt (volgt highest price)
- [ ] `POST /api/positions/:id/close` sluit handmatig

### 8. ML Model Test

**Doel:** ML predictor traint op trade history.

**Test stappen:**
1. Check `/api/ml/status` - toont trained=false, samples=0
2. Doe 20+ trades (paper mode, laat bot draaien)
3. Check `/api/ml/status` - trained=true na 20 completed trades
4. Accuracy moet > 0.5 zijn (beter dan random)

### 9. Multi-Source Failover Test

**Doel:** Als een API source eruit ligt, gaat de bot door met de volgende.

**Test stappen:**
1. Check `/api/datasources/status` - toont alle 3 providers
2. Check `/api/health/external` - toont health per provider
3. Als DexScreener down is, moet GeckoTerminal tokens leveren
4. Als beide down zijn, moet CoinGecko fallback geven

**Wat moet werken:**
- [ ] ProviderManager probeert providers in volgorde
- [ ] Rate limited providers worden overgeslagen (cooldown)
- [ ] Demo tokens fallback als alle sources down zijn

## Regression Test (voor na elke grote change)

Voer dit script uit na elke sessie:

```bash
cd /home/twadelij/Projects/CryptoJackal

echo "=== Build ==="
go build ./cmd/cryptojackal || exit 1

echo "=== Tests ==="
go test ./internal/... -v || exit 1

echo "=== Security ==="
$(go env GOPATH)/bin/gosec ./... || exit 1

echo "=== Vet ==="
go vet ./... || exit 1

echo "=== All OK ==="
```

## Bug Report Template

Als je iets vindt:
```
Wat: [kort beschrijving]
Hoe: [stappen om te reproduceren]
Verwacht: [wat je verwachtte]
Kreeg: [wat je kreeg]
Screenshot: [optioneel]
```

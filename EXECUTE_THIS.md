# EXECUTE THIS - CryptoJackal testen op s0irap0001g (.252)

Draai op s0irap0001g (10.4.82.252) als admtadj@cbsp.nl.
De app staat in /home/twadelij/testapp/CryptoJackal.

## Stap 1: Pull de laatste code

```bash
cd /home/twadelij/testapp/CryptoJackal
git pull origin master
```

## Stap 2: Build en test

```bash
# Zorg dat Go up to date is
go version

# Build alle packages
go build ./...

# Run alle tests
go test ./internal/... -v

# Vet check
go vet ./...
```

## Stap 3: Start de app

```bash
# Zorg dat .env bestaat (copy from .env.example als nodig)
cp .env.example .env

# Start in paper mode
go run ./cmd/cryptojackal
```

De app start op http://localhost:8080.

## Stap 4: Verify de nieuwe features

In een tweede terminal op .252:

```bash
# Login en haal token
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}' | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['token'])")

echo "Token: $TOKEN"

# Check health
curl -s http://localhost:8080/api/health | python3 -m json.tool

# Check datasource status (nieuw!)
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/datasources/status | python3 -m json.tool

# Check strategies (nieuw!)
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/strategies | python3 -m json.tool

# Check ML status (nieuw!)
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/ml/status | python3 -m json.tool

# Check positions (nieuw! - zal leeg zijn zonder open trades)
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/positions | python3 -m json.tool

# Check external API health
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/health/external | python3 -m json.tool

# Check config (api_tier moet zichtbaar zijn)
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/config | python3 -m json.tool
```

## Stap 5: Start bot en watch auto-trading

```bash
# Start de bot
curl -s -X POST -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/bot/start | python3 -m json.tool

# Wacht 3 minuten (1 scan cycle op free tier) en check dan:
sleep 180

# Check opportunities (moet signals bevatten met strategy naam)
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/trading/opportunities | python3 -m json.tool

# Check positions (als er een auto-trade was)
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/positions | python3 -m json.tool

# Check bot status
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/bot/status | python3 -m json.tool

# Stop de bot
curl -s -X POST -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/bot/stop | python3 -m json.tool
```

## Verwachte output

- `datasources/status`: 3 providers (dexscreener, geckoterminal, coingecko) met available=true
- `strategies`: ["momentum", "dip_buy", "volume_spike"]
- `ml/status`: trained=false, samples=0, completed=0 (nog geen trades)
- `positions`: [] (leeg bij start)
- `config`: api_tier=free
- Na 1 scan cycle: opportunities met strategy field en confidence score

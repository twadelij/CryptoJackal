# CryptoJackal Server Deploy Guide

## Snelle start (op je home server)

```bash
# 1. SSH naar je server
ssh twadelij@192.168.2.252

# 2. Maak testapp dir
mkdir -p /home/twadelij/testapp
cd /home/twadelij/testapp

# 3. Clone repo
git clone https://github.com/twadelij/CryptoJackal.git
cd CryptoJackal

# 4. Kopieer env en pas aan
cp .env.example .env
# Edit .env: wijzig ADMIN_PASSWORD en JWT_SECRET

# 5. Start met Docker
make docker-up

# 6. Check health
curl http://localhost:8080/api/health
```

## Productie setup (met HTTPS via Caddy)

### 1. Docker + Compose

```bash
sudo apt update
sudo apt install -y docker.io docker-compose-plugin
sudo usermod -aG docker $USER
newgrp docker
```

### 2. Caddy reverse proxy (voor HTTPS)

```bash
sudo apt install -y caddy
sudo cp Caddyfile /etc/caddy/Caddyfile
sudo systemctl restart caddy
```

### 3. Data dirs aanmaken

```bash
mkdir -p /home/twadelij/.cryptojackal/logs
mkdir -p /home/twadelij/backups/cryptojackal
```

### 4. Systemd service (auto-start)

```bash
sudo cp cryptojackal.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable cryptojackal
sudo systemctl start cryptojackal
```

### 5. Backup (dagelijks via cron)

```bash
chmod +x scripts/backup.sh
crontab -e
# Voeg toe:
0 2 * * * /home/twadelij/testapp/CryptoJackal/scripts/backup.sh >> /home/twadelij/.cryptojackal/logs/backup.log 2>&1
```

### 6. Log rotation

```bash
sudo cp scripts/logrotate.conf /etc/logrotate.d/cryptojackal
```

## Veiligheidschecklist voor live trading

**NOOIT live trading doen zonder deze checks:**

- [ ] Testnet wallet gemaakt (Sepolia)
- [ ] Testnet wallet heeft < 0.1 ETH
- [ ] Safety rails geconfigureerd in `.env`:
  - `MAX_DAILY_LOSS_PCT=5.0` (max 5% verlies per dag)
  - `MAX_TRADE_SIZE_PCT=1.0` (max 1% per trade)
  - `TRADE_COOLDOWN_MINUTES=5` (min 5 min tussen trades)
  - `MAX_OPEN_POSITIONS=3` (max 3 posities open)
  - `KILL_SWITCH_ENABLED=true`
- [ ] Paper trading draait 7 dagen stabiel
- [ ] Kill switch getest: bot stopt bij > 5% loss
- [ ] SQLite backups werken (check `/home/twadelij/backups/`)
- [ ] Admin password is NIET "admin" (wijzig in `.env`)
- [ ] JWT secret is lang en random (wijzig in `.env`)

## Monitoring

### Health check
```bash
curl http://localhost:8080/api/health
```

### Bot status
```bash
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/bot/status
```

### Logs bekijken
```bash
# Docker logs
docker logs -f cryptojackal

# Systemd logs
sudo journalctl -u cryptojackal -f
```

## Updates

```bash
cd /home/twadelij/testapp/CryptoJackal
git pull origin master
make docker-up
```

## Troubleshooting

| Probleem | Oplossing |
|----------|-----------|
| Port 8080 in use | `sudo lsof -i :8080` en kill proces, of wijzig SERVER_PORT |
| SQLite locked | WAL mode is actief, wacht even of restart |
| Frontend laadt niet | Check of `web/dist/` bestaat, run `make build` |
| JWT expired | Log opnieuw in, token is 24h geldig |
| Kill switch triggered | Restart bot via API of restart service |

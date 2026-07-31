# Server Setup: 192.168.2.252

## Requirements
- Docker + docker-compose-plugin
- make (optioneel, alternatief: `docker compose up -d --build`)

## Wat dit doet

Op jouw home server (192.168.2.252) wordt een `testapp` dir aangemaakt in `/home/twadelij` met:
- De laatste CryptoJackal code (gecloned van GitHub)
- Docker en docker-compose geinstalleerd (indien nodig)
- De app gebouwd en gestart
- Een systemd service voor auto-start

## Stappen (run op 192.168.2.252 als twadelij)

```bash
# 1. Maak testapp dir
mkdir -p /home/twadelij/testapp
cd /home/twadelij/testapp

# 2. Clone de repo
git clone https://github.com/twadelij/CryptoJackal.git
cd CryptoJackal

# 3. Check of Docker er is
if ! command -v docker &> /dev/null; then
    echo "Docker niet gevonden. Installatie nodig:"
    echo "sudo apt update && sudo apt install -y docker.io docker-compose"
    echo "sudo usermod -aG docker twadelij && newgrp docker"
    exit 1
fi

# 4. Kopieer env file
cp .env.example .env

# 5. Bouw en start (met make, of zonder)
make docker-up
# ALTERNATIEF als make niet geinstalleerd is:
# docker compose up -d --build

# 6. Check of het draait
echo "Wacht 10 seconden..."
sleep 10
curl -s http://localhost:8080/api/health | jq .

# 7. Test login (password = admin)
curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}' | jq .
```

## Systemd service (optioneel, voor auto-start)

```bash
sudo cp cryptojackal.service /etc/systemd/system/
sudo sed -i 's|/home/twadelij/Projects/CryptoJackal|/home/twadelij/testapp/CryptoJackal|g' /etc/systemd/system/cryptojackal.service
sudo systemctl daemon-reload
sudo systemctl enable cryptojackal
sudo systemctl start cryptojackal
```

## Test plan (wat je moet checken)

1. **Health check:** `curl http://localhost:8080/api/health`
2. **Login:** `curl -X POST http://localhost:8080/api/auth/login -H "Content-Type: application/json" -d '{"username":"admin","password":"admin"}'`
3. **Paper balance:** `curl -H "Authorization: Bearer <token>" http://localhost:8080/api/paper/balance`
4. **Trending tokens:** `curl -H "Authorization: Bearer <token>" http://localhost:8080/api/discovery/trending`
5. **Paper trade:** `curl -X POST -H "Authorization: Bearer <token>" -H "Content-Type: application/json" http://localhost:8080/api/paper/trade -d '{"token_address":"0xDOJI1234567890abcdef","token_symbol":"DOJI","price":0.000004,"amount":1000,"type":"buy"}'`
6. **History:** `curl -H "Authorization: Bearer <token>" http://localhost:8080/api/paper/history`
7. **Export CSV:** `curl -H "Authorization: Bearer <token>" "http://localhost:8080/api/paper/export?format=csv"`
8. **Crash test:** `kill -9 <pid>` dan herstarten, check of trades intact zijn

## Frontend check

Open `http://192.168.2.252:8080` in je browser (vanaf je laptop).

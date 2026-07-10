## Huidige Sprint: Fase 1 - Security & Audit (Week 1-2)

### Vandaag gedaan
- [x] Sprint-Status.md aangemaakt
- [x] gosec security audit: 13 findings gevonden
- [x] Alle 13 gosec findings gefixt:
  - G115 integer overflow (config.go) -> safeUint64 helper toegevoegd
  - G404 weak random (discovery.go) -> math/rand vervangen door crypto/rand
  - G703 path traversal (main.go) -> os.UserHomeDir() + error handling
  - G301 directory permissions (main.go) -> 0755 -> 0750
  - G202 SQL injection (storage.go) -> parameterized LIMIT query
  - G104 errors unhandled -> alle errors nu afgevangen
- [x] Input validatie toegevoegd aan API endpoints:
  - chain parameter whitelist (ethereum, bsc, polygon, arbitrum, base)
  - Ethereum address format validatie (0x + 40 chars)
  - Trade amount > 0, price > 0, type = buy|sell
- [x] GitHub Actions CI workflow aangemaakt (go vet, gosec, tests, secret scan)
- [x] CONTRIBUTING.md met security guidelines aangemaakt
- [x] .env.example gecontroleerd: geen echte secrets

### Morgen doen
- [ ] Commit + push security fixes
- [ ] Fase 2 starten: Paper Trading Perfectie

### Blokkers
- Geen

### Volgende sprint preview
- Fase 2 - Paper Trading Perfectie (Week 3-4)

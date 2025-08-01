# Configuration Management

CryptoJackal gebruikt een uitgebreid configuration management systeem dat ondersteuning biedt voor JSON bestanden, environment variables en default waarden.

## Configuratie Hiërarchie

De configuratie wordt geladen in de volgende volgorde (laatste heeft voorrang):

1. **Default waarden** (hardcoded in de applicatie)
2. **JSON configuratiebestand** (`config.json` of via `CONFIG_PATH`)
3. **Environment variables**
4. **Command-line arguments** (toekomstige feature)

## Configuratiebestand

### Voorbeeld config.json

```json
{
  "node_url": "https://eth-mainnet.alchemyapi.io/v2/your-api-key",
  "chain_id": 1,
  "private_key": "your-private-key-here",
  "wallet_address": "your-wallet-address-here",
  "scan_interval": 1000,
  "gas_limit": 300000,
  "gas_price_gwei": 20,
  "max_gas_price_gwei": 100,
  "slippage_tolerance": 2.0,
  "min_liquidity": 10.0,
  "max_price_impact": 5.0,
  "target_tokens": [
    "0xA0b86a33E6441b8c4C8C1C1B8c4C8C1C1B8c4C8C1"
  ],
  "max_trade_size_eth": 0.1,
  "min_profit_threshold": 0.5,
  "max_daily_trades": 10,
  "stop_loss_percentage": 10.0,
  "take_profit_percentage": 20.0,
  "uniswap_v2_router": "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D",
  "uniswap_v3_router": "0xE592427A0AEce92De3Edee1F18E0157C05861564",
  "weth_address": "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
  "enable_telegram_alerts": false,
  "telegram_bot_token": null,
  "telegram_chat_id": null,
  "enable_mev_protection": false,
  "flashbots_relay_url": null,
  "max_priority_fee_gwei": 2,
  "log_level": "info",
  "log_file_path": null
}
```

## Environment Variables

Alle configuratie parameters kunnen ook via environment variables worden ingesteld:

### Network Configuration
- `NODE_URL`: Ethereum node URL
- `CHAIN_ID`: Chain ID (1 voor mainnet, 5 voor Goerli, etc.)

### Wallet Configuration
- `PRIVATE_KEY`: Private key voor wallet
- `WALLET_ADDRESS`: Wallet address

### Trading Parameters
- `SCAN_INTERVAL`: Interval tussen scans in milliseconden
- `GAS_LIMIT`: Gas limit voor transacties
- `GAS_PRICE_GWEI`: Gas price in Gwei
- `MAX_GAS_PRICE_GWEI`: Maximum gas price in Gwei
- `SLIPPAGE_TOLERANCE`: Slippage tolerance percentage
- `MIN_LIQUIDITY`: Minimum liquidity in ETH
- `MAX_PRICE_IMPACT`: Maximum price impact percentage
- `TARGET_TOKENS`: Comma-separated lijst van token addresses

### Risk Management
- `MAX_TRADE_SIZE_ETH`: Maximum trade size in ETH
- `MIN_PROFIT_THRESHOLD`: Minimum profit threshold percentage
- `MAX_DAILY_TRADES`: Maximum aantal trades per dag
- `STOP_LOSS_PERCENTAGE`: Stop loss percentage
- `TAKE_PROFIT_PERCENTAGE`: Take profit percentage

### DEX Configuration
- `UNISWAP_V2_ROUTER`: Uniswap V2 router address
- `UNISWAP_V3_ROUTER`: Uniswap V3 router address
- `WETH_ADDRESS`: WETH token address

### Monitoring and Alerts
- `ENABLE_TELEGRAM_ALERTS`: Enable/disable Telegram alerts
- `TELEGRAM_BOT_TOKEN`: Telegram bot token
- `TELEGRAM_CHAT_ID`: Telegram chat ID

### Performance Settings
- `ENABLE_MEV_PROTECTION`: Enable/disable MEV protection
- `FLASHBOTS_RELAY_URL`: Flashbots relay URL
- `MAX_PRIORITY_FEE_GWEI`: Maximum priority fee in Gwei

### Logging Configuration
- `LOG_LEVEL`: Log level (trace, debug, info, warn, error)
- `LOG_FILE_PATH`: Path naar log bestand

## Gebruik

### Configuratie laden

```rust
use cryptojackal::core::Config;

// Laad configuratie (probeert eerst config.json, dan environment variables)
let config = Config::load()?;

// Of laad alleen van environment variables
let config = Config::load_from_env()?;

// Of laad van specifiek bestand
let config = Config::load_from_file("my_config.json")?;
```

### Configuratie opslaan

```rust
// Sla configuratie op naar JSON bestand
config.save_to_file("config.json")?;

// Maak default configuratie aan
Config::create_default_config("config.json")?;
```

### Configuratie valideren

```rust
// Valideer configuratie
config.validate()?;
```

## Validatie Regels

De configuratie wordt automatisch gevalideerd bij het laden:

- **Verplichte velden**: `NODE_URL`, `PRIVATE_KEY`, `WALLET_ADDRESS`
- **Numerieke ranges**: Percentages tussen 0-100, gas prices > 0
- **Consistentie**: `MAX_GAS_PRICE_GWEI` >= `GAS_PRICE_GWEI`
- **Conditionele validatie**: Telegram configuratie vereist bij enabled alerts
- **Log levels**: Moet een van de geldige waarden zijn

## Best Practices

### Security
- **Nooit private keys in config bestanden opslaan** in productie
- Gebruik environment variables voor gevoelige data
- Zorg dat config bestanden niet in version control komen

### Development
- Gebruik `config.example.json` als template
- Test configuratie validatie in CI/CD
- Documenteer nieuwe configuratie parameters

### Production
- Gebruik secrets management voor private keys
- Monitor configuratie wijzigingen
- Backup configuratie bestanden

## Troubleshooting

### Veelvoorkomende fouten

1. **"Missing environment variable"**: Zorg dat alle verplichte environment variables zijn ingesteld
2. **"Invalid value"**: Controleer dat numerieke waarden binnen geldige ranges vallen
3. **"Configuration validation failed"**: Controleer de validatie regels

### Debug tips

```rust
// Print configuratie voor debugging
println!("{:?}", config);

// Valideer specifieke secties
config.validate()?;
```

## Toekomstige Features

- Command-line argument support
- Hot-reload van configuratie
- Configuratie templates per netwerk
- Configuratie encryptie
- Remote configuratie via API 
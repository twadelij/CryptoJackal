# Liquidity Pair Detection System

CryptoJackal gebruikt een geavanceerd liquidity pair detection systeem dat real-time monitoring van Uniswap V2/V3 pairs biedt voor trading opportunities.

## Overzicht

Het liquidity pair detection systeem bestaat uit de volgende componenten:

- **Pair Scanning**: Automatische detectie van nieuwe en bestaande pairs
- **Opportunity Analysis**: Geavanceerde analyse van trading opportunities
- **Risk Assessment**: Risico-evaluatie en scoring
- **Performance Monitoring**: Metrics en monitoring van scan performance
- **Caching**: Efficiënte caching van pair data

## Core Components

### LiquidityPair

```rust
pub struct LiquidityPair {
    pub address: Address,
    pub token0: Address,
    pub token1: Address,
    pub token0_symbol: String,
    pub token1_symbol: String,
    pub reserve0: u128,
    pub reserve1: u128,
    pub total_supply: u128,
    pub fee: u32, // 0.3% = 3000, 0.05% = 500, etc.
    pub version: UniswapVersion,
    pub created_at: u64,
    pub liquidity_usd: f64,
    pub volume_24h: f64,
    pub price_change_24h: f64,
}
```

### Opportunity

```rust
pub struct Opportunity {
    pub token: TokenInfo,
    pub pair: LiquidityPair,
    pub price: f64,
    pub liquidity: f64,
    pub volatility: f64,
    pub price_impact: f64,
    pub volume_24h: f64,
    pub market_cap: f64,
    pub confidence_score: f64,
    pub risk_level: RiskLevel,
    pub detected_at: Instant,
    pub opportunity_type: OpportunityType,
}
```

## Opportunity Types

### NewToken
- **Beschrijving**: Nieuw gelijste tokens met hoge groeipotentieel
- **Detectie**: Pairs jonger dan 1 uur
- **Risico**: Hoog (nieuwe tokens zijn volatiel)
- **Potentieel**: Zeer hoog (early entry mogelijk)

### LiquidityAddition
- **Beschrijving**: Grote liquiditeit toevoegingen aan bestaande pairs
- **Detectie**: Significant hogere reserves dan gemiddeld
- **Risico**: Medium (afhankelijk van token fundamentals)
- **Potentieel**: Medium tot hoog

### PriceSpike
- **Beschrijving**: Significant prijsstijgingen in korte tijd
- **Detectie**: Prijsverandering > 20% in 24 uur
- **Risico**: Hoog (kan pump & dump zijn)
- **Potentieel**: Hoog (momentum trading)

### Arbitrage
- **Beschrijving**: Prijsverschillen tussen verschillende DEXs
- **Detectie**: Prijsverschil > 1% tussen platforms
- **Risico**: Laag (arbitrage is relatief veilig)
- **Potentieel**: Laag tot medium (afhankelijk van volume)

### Momentum
- **Beschrijving**: Sterke opwaartse momentum met volume
- **Detectie**: Consistente prijsstijgingen met hoog volume
- **Risico**: Medium (momentum kan omkeren)
- **Potentieel**: Medium tot hoog

## Risk Assessment

### RiskLevel Enum

```rust
pub enum RiskLevel {
    Low,      // Laag risico, veilige trades
    Medium,   // Gemiddeld risico, gebalanceerde trades
    High,     // Hoog risico, voorzichtig handelen
    Extreme,  // Extreem risico, vermijden of zeer kleine posities
}
```

### Risk Scoring Algorithm

Het risico wordt berekend op basis van:

1. **Volatility Score** (0-3 punten)
   - > 80% volatility: 3 punten
   - > 50% volatility: 2 punten
   - > 20% volatility: 1 punt
   - < 20% volatility: 0 punten

2. **Price Impact Score** (0-3 punten)
   - > 10% impact: 3 punten
   - > 5% impact: 2 punten
   - > 2% impact: 1 punt
   - < 2% impact: 0 punten

3. **Liquidity Score** (0-3 punten)
   - < $10K liquidity: 3 punten
   - < $50K liquidity: 2 punten
   - < $100K liquidity: 1 punt
   - > $100K liquidity: 0 punten

**Risk Level Mapping**:
- 0-2 punten: Low Risk
- 3-4 punten: Medium Risk
- 5-6 punten: High Risk
- 7-9 punten: Extreme Risk

## Scanner Configuration

### ScannerConfig

```rust
pub struct ScannerConfig {
    pub scan_interval_ms: u64,        // Interval tussen scans
    pub max_pairs_per_scan: usize,    // Maximum pairs per scan
    pub min_liquidity_usd: f64,       // Minimum liquidity in USD
    pub max_price_impact: f64,        // Maximum price impact percentage
    pub min_volume_24h: f64,          // Minimum 24h volume
    pub enable_v2: bool,              // Enable Uniswap V2 scanning
    pub enable_v3: bool,              // Enable Uniswap V3 scanning
    pub gas_price_threshold: u64,     // Maximum gas price in Gwei
    pub max_slippage: f64,            // Maximum slippage percentage
}
```

### Default Configuration

```rust
ScannerConfig {
    scan_interval_ms: 1000,           // 1 seconde
    max_pairs_per_scan: 1000,         // 1000 pairs per scan
    min_liquidity_usd: 10000.0,       // $10K minimum liquidity
    max_price_impact: 5.0,            // 5% maximum price impact
    min_volume_24h: 1000.0,           // $1K minimum 24h volume
    enable_v2: true,                  // Enable V2 scanning
    enable_v3: true,                  // Enable V3 scanning
    gas_price_threshold: 50,          // 50 Gwei max gas
    max_slippage: 2.0,                // 2% maximum slippage
}
```

## Performance Monitoring

### ScanMetrics

```rust
pub struct ScanMetrics {
    pub total_scans: u64,             // Totaal aantal scans
    pub pairs_scanned: u64,           // Totaal aantal pairs gescand
    pub opportunities_found: u64,     // Totaal aantal opportunities gevonden
    pub average_scan_time_ms: u64,    // Gemiddelde scan tijd
    pub last_scan_duration_ms: u64,   // Laatste scan duur
    pub cache_hit_rate: f64,          // Cache hit rate percentage
    pub error_count: u64,             // Aantal errors
}
```

## Gebruik

### Market Scanner Initialisatie

```rust
use cryptojackal::core::Market;

// Initialize market scanner
let provider = Provider::<Http>::try_from(&config.node_url)?;
let market = Market::new(&provider, &config).await?;

// Scan for opportunities
let opportunities = market.scan_opportunities().await?;

// Get scan metrics
let metrics = market.get_scan_metrics().await;
println!("Scan metrics: {:?}", metrics);

// Get opportunity history
let history = market.get_opportunity_history().await;
println!("Found {} opportunities", history.len());
```

### Opportunity Analysis

```rust
for opportunity in opportunities {
    println!("Token: {}", opportunity.token.symbol);
    println!("Price: ${:.6}", opportunity.price);
    println!("Liquidity: ${:.2}", opportunity.liquidity);
    println!("Risk Level: {:?}", opportunity.risk_level);
    println!("Confidence Score: {:.2}", opportunity.confidence_score);
    println!("Opportunity Type: {:?}", opportunity.opportunity_type);
    println!("---");
}
```

### Risk-Based Filtering

```rust
// Filter opportunities by risk level
let low_risk_opportunities: Vec<_> = opportunities
    .iter()
    .filter(|opp| opp.risk_level == RiskLevel::Low)
    .collect();

let high_potential_opportunities: Vec<_> = opportunities
    .iter()
    .filter(|opp| opp.confidence_score > 0.8)
    .collect();
```

## Scanning Process

### 1. Pair Discovery
- Scan Uniswap V2 factory voor nieuwe pairs
- Scan Uniswap V3 factory voor nieuwe pools
- Cache pair data voor performance

### 2. Data Collection
- Fetch reserves en liquidity data
- Calculate price en volume metrics
- Gather token information

### 3. Opportunity Analysis
- Calculate volatility en price impact
- Determine opportunity type
- Assess risk level
- Calculate confidence score

### 4. Filtering & Ranking
- Apply minimum criteria filters
- Sort by confidence score
- Limit to top opportunities
- Store in history

## Best Practices

### Performance Optimization

1. **Caching**: Cache pair data om RPC calls te minimaliseren
2. **Batch Processing**: Process multiple pairs in batches
3. **Rate Limiting**: Respect RPC rate limits
4. **Parallel Scanning**: Scan V2 en V3 parallel

### Risk Management

1. **Diversification**: Spread trades across multiple opportunities
2. **Position Sizing**: Base position size on risk level
3. **Stop Losses**: Always use stop losses voor high-risk trades
4. **Monitoring**: Continu monitor open positions

### Configuration Tuning

1. **Liquidity Thresholds**: Adjust based on trading strategy
2. **Scan Intervals**: Balance tussen responsiveness en performance
3. **Risk Tolerance**: Configure risk levels based on strategy
4. **Gas Optimization**: Monitor gas prices voor cost efficiency

## Troubleshooting

### Veelvoorkomende Problemen

1. **High Error Count**: Check RPC endpoint stability
2. **Low Cache Hit Rate**: Increase cache size of optimize cache strategy
3. **Slow Scan Times**: Reduce scan interval of optimize batch size
4. **No Opportunities**: Lower minimum criteria of check market conditions

### Debug Tips

```rust
// Enable debug logging
tracing::set_level(tracing::Level::DEBUG);

// Check scan metrics
let metrics = market.get_scan_metrics().await;
println!("Average scan time: {}ms", metrics.average_scan_time_ms);
println!("Cache hit rate: {:.2}%", metrics.cache_hit_rate);

// Monitor specific pairs
let history = market.get_opportunity_history().await;
for opp in history.iter().take(5) {
    println!("Recent opportunity: {:?}", opp);
}
```

## Toekomstige Features

- **Multi-DEX Support**: SushiSwap, PancakeSwap, etc.
- **Machine Learning**: ML-based opportunity prediction
- **Social Sentiment**: Twitter/Reddit sentiment analysis
- **Whale Tracking**: Large wallet movement monitoring
- **Cross-Chain**: Multi-chain opportunity detection
- **Real-time Alerts**: Push notifications voor opportunities 
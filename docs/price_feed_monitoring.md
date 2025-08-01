# Price Feed Monitoring System

CryptoJackal gebruikt een geavanceerd price feed monitoring systeem dat real-time prijsdata verzamelt van meerdere bronnen, aggregeert en analyseert voor trading opportunities.

## Overzicht

Het price feed monitoring systeem biedt:

- **Multi-source price aggregation** van CoinGecko, Uniswap V2/V3, DexScreener
- **Real-time price monitoring** met configurable update intervals
- **Outlier detection** en price validation
- **Price change alerts** voor significante bewegingen
- **Performance metrics** en monitoring
- **Historical price data** storage en retrieval

## Core Components

### PriceData

```rust
pub struct PriceData {
    pub token_address: String,
    pub symbol: String,
    pub price_usd: f64,
    pub volume_24h: f64,
    pub market_cap: Option<f64>,
    pub price_change_24h: f64,
    pub source: PriceSource,
    pub timestamp: DateTime<Utc>,
    pub confidence: f64, // 0.0 to 1.0
}
```

### AggregatedPrice

```rust
pub struct AggregatedPrice {
    pub token_address: String,
    pub symbol: String,
    pub price_usd: f64,
    pub volume_24h: f64,
    pub market_cap: Option<f64>,
    pub price_change_24h: f64,
    pub sources_count: usize,
    pub confidence: f64,
    pub volatility: f64,
    pub outlier_detected: bool,
    pub aggregated_at: DateTime<Utc>,
}
```

### PriceAlert

```rust
pub struct PriceAlert {
    pub token_address: String,
    pub symbol: String,
    pub alert_type: AlertType,
    pub old_price: f64,
    pub new_price: f64,
    pub change_percentage: f64,
    pub threshold: f64,
    pub timestamp: DateTime<Utc>,
}
```

## Price Sources

### CoinGecko
- **Beschrijving**: Gerenommeerde cryptocurrency data provider
- **Confidence**: 0.9 (hoogste betrouwbaarheid)
- **Data**: Price, volume, market cap, 24h change
- **Rate Limits**: Respecteert API limits

### Uniswap V2
- **Beschrijving**: Directe DEX data van Uniswap V2 pairs
- **Confidence**: 0.8 (directe blockchain data)
- **Data**: Price, volume, reserves
- **Voordelen**: Real-time, geen API limits

### Uniswap V3
- **Beschrijving**: Directe DEX data van Uniswap V3 pools
- **Confidence**: 0.85 (verbeterde V3 data)
- **Data**: Price, volume, concentrated liquidity
- **Voordelen**: Meer accurate pricing, multiple fee tiers

### DexScreener
- **Beschrijving**: Multi-DEX aggregator
- **Confidence**: 0.75 (aggregated data)
- **Data**: Cross-DEX price comparison
- **Voordelen**: Breed DEX coverage

## Alert Types

### PriceSpike
- **Trigger**: Significant prijsstijging (>5% default)
- **Use Case**: Early detection van pump events
- **Action**: Immediate trading opportunity evaluation

### PriceDrop
- **Trigger**: Significant prijsdaling (>5% default)
- **Use Case**: Risk management en stop-loss triggers
- **Action**: Position evaluation en exit strategies

### VolumeSpike
- **Trigger**: Unusual volume increase (>200% default)
- **Use Case**: Detectie van whale activity
- **Action**: Market sentiment analysis

### VolatilityAlert
- **Trigger**: High volatility (>50% default)
- **Use Case**: Risk assessment
- **Action**: Position sizing adjustment

### OutlierDetected
- **Trigger**: Price outlier in aggregation
- **Use Case**: Data quality assurance
- **Action**: Source validation en filtering

## Configuration

### PriceFeedConfig

```rust
pub struct PriceFeedConfig {
    pub update_interval_ms: u64,        // 5000ms default
    pub aggregation_timeout_ms: u64,    // 2000ms default
    pub outlier_threshold: f64,         // 3.0 std devs default
    pub alert_thresholds: AlertThresholds,
    pub enabled_sources: Vec<PriceSource>,
    pub max_retry_attempts: u32,        // 3 default
    pub retry_delay_ms: u64,            // 1000ms default
}
```

### Environment Variables

```bash
# Price feed configuration
PRICE_FEED_UPDATE_INTERVAL_MS=5000
PRICE_FEED_OUTLIER_THRESHOLD=3.0
PRICE_FEED_ALERT_THRESHOLD_PERCENT=5.0
PRICE_FEED_ENABLED_SOURCES=coingecko,uniswap_v2,uniswap_v3
```

## Gebruik

### Price Feed Monitor Initialisatie

```rust
use cryptojackal::core::price_feed::{PriceFeedMonitor, PriceFeedConfig};

// Create configuration
let config = PriceFeedConfig::default();

// Initialize monitor
let monitor = PriceFeedMonitor::new(config);

// Start monitoring
let target_tokens = vec![
    "0xA0b86a33E6441b8c4C8C8C8C8C8C8C8C8C8C8C8".to_string(),
    "0xB0b86a33E6441b8c4C8C8C8C8C8C8C8C8C8C8C8".to_string(),
];

monitor.start_monitoring(target_tokens).await?;
```

### Price Data Retrieval

```rust
// Get current price
if let Some(price) = monitor.get_current_price("0x...").await {
    println!("Current price: ${:.6}", price.price_usd);
    println!("Volume 24h: ${:.2}", price.volume_24h);
    println!("Confidence: {:.2}", price.confidence);
}

// Get price history
let history = monitor.get_price_history("0x...").await;
for price_data in history.iter().take(5) {
    println!("{}: ${:.6} from {:?}", 
             price_data.timestamp, price_data.price_usd, price_data.source);
}

// Get recent alerts
let alerts = monitor.get_recent_alerts(10).await;
for alert in alerts {
    println!("🚨 Alert: {} - {:?} - {:.2}% change", 
             alert.symbol, alert.alert_type, alert.change_percentage);
}
```

### Bot Integration

```rust
use cryptojackal::core::Bot;

// Initialize bot with price feed monitoring
let bot = Bot::new().await?;

// Get price feed metrics
let metrics = bot.get_price_feed_metrics().await;
println!("Price feed metrics: {:?}", metrics);

// Get token price
if let Some(price) = bot.get_token_price("0x...").await {
    println!("Token price: ${:.6}", price.price_usd);
}

// Get recent alerts
let alerts = bot.get_recent_price_alerts(5).await;
for alert in alerts {
    println!("Alert: {} - {:.2}% change", alert.symbol, alert.change_percentage);
}
```

## Price Aggregation Algorithm

### Weighted Average Calculation

```rust
// Calculate weighted average based on confidence
let total_weight: f64 = price_data_vec.iter().map(|p| p.confidence).sum();
let weighted_price: f64 = price_data_vec.iter()
    .map(|p| p.price_usd * p.confidence)
    .sum::<f64>() / total_weight;
```

### Outlier Detection

```rust
// Z-score method for outlier detection
let mean = prices.iter().sum::<f64>() / prices.len() as f64;
let variance = prices.iter()
    .map(|p| (p - mean).powi(2))
    .sum::<f64>() / (prices.len() - 1) as f64;
let std_dev = variance.sqrt();

// Check if any price is more than threshold standard deviations
prices.iter().any(|&price| {
    let z_score = (price - mean).abs() / std_dev;
    z_score > outlier_threshold
})
```

### Volatility Calculation

```rust
// Coefficient of variation
let mean = prices.iter().sum::<f64>() / prices.len() as f64;
let variance = prices.iter()
    .map(|p| (p - mean).powi(2))
    .sum::<f64>() / (prices.len() - 1) as f64;
let volatility = variance.sqrt() / mean;
```

## Performance Monitoring

### PriceFeedMetrics

```rust
pub struct PriceFeedMetrics {
    pub total_updates: u64,
    pub successful_updates: u64,
    pub failed_updates: u64,
    pub alerts_generated: u64,
    pub average_update_time_ms: f64,
    pub cache_hit_rate: f64,
    pub last_update_time: Option<DateTime<Utc>>,
}
```

### Monitoring Dashboard

```rust
// Get comprehensive metrics
let metrics = monitor.get_metrics().await;
println!("📊 Price Feed Metrics:");
println!("  Total Updates: {}", metrics.total_updates);
println!("  Success Rate: {:.1}%", 
         (metrics.successful_updates as f64 / metrics.total_updates as f64) * 100.0);
println!("  Average Update Time: {:.1}ms", metrics.average_update_time_ms);
println!("  Alerts Generated: {}", metrics.alerts_generated);
println!("  Last Update: {:?}", metrics.last_update_time);
```

## Error Handling

### Retry Logic

```rust
// Automatic retry with exponential backoff
let mut attempts = 0;
while attempts < max_retry_attempts {
    match fetch_price_from_source(token_address, source).await {
        Ok(price_data) => return Ok(price_data),
        Err(e) => {
            attempts += 1;
            warn!("Attempt {} failed: {}", attempts, e);
            if attempts < max_retry_attempts {
                sleep(Duration::from_millis(retry_delay_ms)).await;
            }
        }
    }
}
```

### Graceful Degradation

```rust
// Continue with available sources if some fail
let mut price_data_vec = Vec::new();
for source in &enabled_sources {
    if let Ok(price_data) = fetch_price_from_source(token_address, *source).await {
        price_data_vec.push(price_data);
    }
}

if price_data_vec.is_empty() {
    return Err(anyhow::anyhow!("No price data available"));
}
```

## Best Practices

### Performance Optimization

1. **Caching**: Cache aggregated prices om redundant calculations te voorkomen
2. **Batch Processing**: Process multiple tokens in batches
3. **Rate Limiting**: Respect API rate limits van externe bronnen
4. **Connection Pooling**: Hergebruik HTTP connections

### Data Quality

1. **Outlier Detection**: Filter onbetrouwbare price data
2. **Confidence Scoring**: Weight sources based on reliability
3. **Validation**: Cross-reference prices across sources
4. **Historical Analysis**: Use historical data voor trend analysis

### Monitoring

1. **Metrics Tracking**: Monitor success rates en performance
2. **Alert Management**: Configure appropriate alert thresholds
3. **Error Logging**: Log failed requests voor debugging
4. **Health Checks**: Regular health checks van price sources

## Troubleshooting

### Veelvoorkomende Problemen

1. **High Failure Rate**: Check API rate limits en network connectivity
2. **Outlier Alerts**: Verify source reliability en data quality
3. **Slow Updates**: Optimize update intervals en batch processing
4. **Missing Data**: Check source availability en fallback mechanisms

### Debug Tips

```rust
// Enable debug logging
tracing::set_level(tracing::Level::DEBUG);

// Check individual source performance
for source in &enabled_sources {
    match fetch_price_from_source(token_address, *source).await {
        Ok(price_data) => debug!("✅ {}: ${:.6}", source, price_data.price_usd),
        Err(e) => error!("❌ {}: {}", source, e),
    }
}

// Monitor aggregation process
let aggregated = aggregate_prices(&price_data_vec).await?;
debug!("Aggregated price: ${:.6} from {} sources", 
       aggregated.price_usd, aggregated.sources_count);
```

## Toekomstige Features

- **Machine Learning**: ML-based price prediction
- **Social Sentiment**: Twitter/Reddit sentiment integration
- **Cross-Chain**: Multi-chain price monitoring
- **Real-time Streaming**: WebSocket-based real-time updates
- **Advanced Analytics**: Technical indicators en pattern recognition
- **Custom Sources**: User-defined price sources 
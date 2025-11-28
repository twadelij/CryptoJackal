# CryptoJackal Development Guide

## 🚀 Overview

CryptoJackal is a high-performance cryptocurrency trading bot built in Rust with a modern web interface. This guide covers the complete development setup, architecture, and contribution guidelines.

## 📋 Table of Contents

- [Quick Start](#quick-start)
- [Architecture](#architecture)
- [Development Setup](#development-setup)
- [API Documentation](#api-documentation)
- [Testing](#testing)
- [Security](#security)
- [Monitoring](#monitoring)
- [Deployment](#deployment)
- [Contributing](#contributing)

## 🚀 Quick Start

### Prerequisites

- Rust 1.70+
- Node.js 16+
- Docker & Docker Compose
- PostgreSQL (for production)
- Redis (for production)

### Local Development

1. **Clone and setup**:
```bash
git clone https://github.com/twadelij/CryptoJackal.git
cd CryptoJackal
cp .env.example .env
```

2. **Start backend**:
```bash
cargo run
```

3. **Start frontend**:
```bash
cd web
npm install
npm run dev
```

4. **Access the application**:
- Frontend: http://localhost:3000
- API: http://localhost:8080
- Health Check: http://localhost:8081/health

### Docker Development

```bash
# Development environment
docker-compose -f docker-compose.dev.yml up

# Production environment
docker-compose up
```

## 🏗️ Architecture

### Core Components

- **Core Engine**: Trading logic and bot management
- **API Layer**: REST API with Axum framework
- **Web Frontend**: React/TypeScript interface
- **Token Discovery**: Automated opportunity detection
- **Paper Trading**: Risk-free simulation environment
- **Security Framework**: Authentication and encryption
- **Monitoring**: Metrics and health checks
- **Testing**: Comprehensive test suite

### Module Structure

```
src/
├── core/           # Core trading engine
├── api/            # REST API layer
├── discovery/      # Token discovery service
├── paper_trading/  # Paper trading simulation
├── security/       # Security framework
├── monitoring/     # Monitoring and metrics
├── testing/        # Testing framework
├── trading/        # Trading logic
└── wallet/         # Wallet integration
```

## 🛠️ Development Setup

### Backend Development

1. **Install dependencies**:
```bash
cargo build
```

2. **Run tests**:
```bash
cargo test
cargo test --features integration-tests
```

3. **Run with logging**:
```bash
RUST_LOG=debug cargo run
```

### Frontend Development

1. **Install dependencies**:
```bash
cd web
npm install
```

2. **Development server**:
```bash
npm run dev
```

3. **Build for production**:
```bash
npm run build
```

### Environment Configuration

Key environment variables:

```bash
# Core Configuration
NODE_URL=https://mainnet.infura.io/v3/YOUR_PROJECT_ID
CHAIN_ID=1
SCAN_INTERVAL=1000
TRADE_AMOUNT=1000000000000000000

# API Configuration
API_HOST=0.0.0.0
API_PORT=8080
CORS_ORIGINS=http://localhost:3000

# Security
JWT_SECRET=your-secure-secret-here
SESSION_TIMEOUT=3600

# Paper Trading
PAPER_TRADING_MODE=true
PAPER_TRADING_BALANCE=10.0

# Token Discovery
DEXSCREENER_API_URL=https://api.dexscreener.com/latest/dex
COINGECKO_API_KEY=your-coingecko-api-key

# Monitoring
METRICS_ENABLED=true
METRICS_PORT=9090
```

## 📚 API Documentation

### Authentication

Most endpoints require JWT authentication:

```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
     http://localhost:8080/api/bot/status
```

### Core Endpoints

#### Health Check
```bash
GET /health
```

#### Bot Control
```bash
GET  /api/bot/status
POST /api/bot/start
POST /api/bot/stop
```

#### Trading
```bash
GET  /api/trading/opportunities
POST /api/trading/execute
GET  /api/trading/history
```

#### Paper Trading
```bash
GET  /api/paper-trading/balance
POST /api/paper-trading/execute
POST /api/paper-trading/reset
```

#### Token Discovery
```bash
GET  /api/discovery/trending?time_window=24h
GET  /api/discovery/new?limit=10
GET  /api/discovery/analyze/0x...
```

### Response Format

All API responses follow this format:

```json
{
  "success": true,
  "data": { ... },
  "error": null,
  "timestamp": "2023-12-01T12:00:00Z"
}
```

## 🧪 Testing

### Test Categories

1. **Unit Tests**: Individual component tests
2. **Integration Tests**: Component interaction tests
3. **Performance Tests**: Response time and resource usage
4. **Security Tests**: Security validation and penetration
5. **Backtesting**: Trading strategy validation

### Running Tests

```bash
# All tests
cargo test

# Specific test category
cargo test --test integration_tests
cargo test --test performance_tests
cargo test --test security_tests

# With coverage
cargo tarpaulin --out Html
```

### Test Configuration

Tests use mock data and isolated environments:

```rust
// Example test setup
#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_paper_trading() {
        let config = Config::default();
        let service = PaperTradingService::new(config);
        
        let balance = service.get_portfolio_balance().await.unwrap();
        assert!(balance.eth_balance > 0.0);
    }
}
```

## 🔒 Security

### Security Features

- **JWT Authentication**: Secure token-based authentication
- **Input Validation**: Comprehensive input sanitization
- **Rate Limiting**: API endpoint protection
- **Encryption**: Sensitive data encryption
- **Audit Logging**: Complete security event tracking
- **Paper Trading**: Risk-free testing environment

### Security Best Practices

1. **Never commit secrets**: Use environment variables
2. **Validate all inputs**: Use the validation framework
3. **Use paper trading**: Test strategies with paper trading first
4. **Monitor security events**: Check audit logs regularly
5. **Keep dependencies updated**: Regular security updates

### Security Audit

Run security audit:

```bash
cargo run --bin security-audit
```

## 📊 Monitoring

### Metrics Collection

- **System Metrics**: CPU, memory, disk usage
- **Application Metrics**: Response times, error rates
- **Business Metrics**: Trading volume, success rates
- **Security Metrics**: Authentication attempts, rate limits

### Health Checks

Health check endpoints:

```bash
# Overall health
GET /health

# Detailed health
GET /health/detailed

# Component health
GET /health/database
GET /health/redis
GET /health/api
```

### Alerting

Configurable alerts for:

- High error rates
- Performance degradation
- Security events
- Resource exhaustion

## 🚀 Deployment

### Production Deployment

1. **Environment Setup**:
```bash
# Production environment variables
export NODE_URL=https://mainnet.infura.io/v3/PROD_KEY
export JWT_SECRET=$(openssl rand -base64 32)
export PAPER_TRADING_MODE=false
```

2. **Docker Deployment**:
```bash
docker-compose -f docker-compose.yml up -d
```

3. **Database Migration**:
```bash
# Run database migrations
cargo run --bin migrate
```

### Monitoring Setup

1. **Prometheus**: Metrics collection
2. **Grafana**: Visualization dashboard
3. **Alertmanager**: Alert routing

### Scaling Considerations

- **API Gateway**: Load balancing and rate limiting
- **Database**: Connection pooling and read replicas
- **Redis**: Clustering for high availability
- **Monitoring**: Distributed tracing

## 🤝 Contributing

### Development Workflow

1. **Fork** the repository
2. **Create** a feature branch
3. **Make** your changes
4. **Add** tests
5. **Run** the test suite
6. **Submit** a pull request

### Code Style

- Use `rustfmt` for formatting
- Use `clippy` for linting
- Write comprehensive tests
- Document public APIs
- Follow security best practices

### Pull Request Process

1. **Tests must pass**: All tests must pass
2. **Code coverage**: Maintain >80% coverage
3. **Documentation**: Update relevant documentation
4. **Security**: Ensure no security vulnerabilities
5. **Performance**: No performance regressions

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/twadelij/CryptoJackal/issues)
- **Discussions**: [GitHub Discussions](https://github.com/twadelij/CryptoJackal/discussions)
- **Documentation**: [Wiki](https://github.com/twadelij/CryptoJackal/wiki)

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Ethereum community for the amazing ecosystem
- Rust community for excellent tooling
- Open source contributors and maintainers

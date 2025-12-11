# 🚀 CryptoJackal Guided Setup

CryptoJackal now includes both **GUI Setup Wizard** and **Command-Line Setup Script** to guide you from test environment to production deployment.

## 📋 Setup Options

### Option 1: 🖥️ GUI Setup Wizard (Recommended)

A modern, interactive web-based setup wizard that guides you through every step.

**Access the GUI Setup Wizard:**
1. Start the application: `./setup.sh` or `docker-compose -f docker-compose.dev.yml up`
2. Open your browser to: `http://localhost:3000/setup`
3. Follow the step-by-step wizard

**GUI Wizard Features:**
- ✅ **Interactive Forms** - User-friendly configuration
- ✅ **Real-time Validation** - Instant feedback on inputs
- ✅ **Progress Tracking** - Visual progress indicators
- ✅ **Security Guidance** - Built-in security best practices
- ✅ **Environment Detection** - Automatic environment optimization
- ✅ **One-Click Deploy** - Automated deployment

### Option 2: 💻 Command-Line Setup Script

A comprehensive bash script for terminal-based setup.

**Run the CLI Setup:**
```bash
# Make executable
chmod +x setup.sh

# Run setup wizard
./setup.sh
```

**CLI Script Features:**
- ✅ **Interactive Prompts** - Step-by-step guidance
- ✅ **Prerequisites Check** - Automatic dependency verification
- ✅ **Environment Selection** - Test/Development/Production modes
- ✅ **Auto-Configuration** - Smart defaults and suggestions
- ✅ **Security Generation** - Automatic JWT secret generation
- ✅ **Docker Deployment** - Automated container deployment

## 🎯 Setup Journey

### Phase 1: Environment Selection
Choose your setup environment:
- **🧪 Test Environment** - Safe testing with paper trading
- **🎓 Development** - For developers and testing
- **🚀 Production** - Live trading with real funds

### Phase 2: Node Configuration
Configure your Ethereum node connection:
- **Infura** - Free tier available
- **Alchemy** - 300M compute units/month free
- **QuickNode** - Free tier available
- **Custom** - Your own node URL

### Phase 3: Trading Parameters
Set up your trading strategy:
- **Trade Amount** - Amount to trade per opportunity
- **Scan Interval** - Market scanning frequency
- **Gas Settings** - Gas limit and optimization
- **Risk Management** - Slippage and liquidity thresholds

### Phase 4: API Keys (Optional)
Enhance features with API keys:
- **CoinGecko** - Enhanced token data
- **DexScreener** - Real-time discovery
- **Telegram** - Trading notifications
- **Discord** - Community alerts

### Phase 5: Security Configuration
Configure security settings:
- **JWT Secret** - Automatic secure generation
- **CORS Origins** - Web interface security
- **Session Management** - Timeout and attempt limits

### Phase 6: Deployment
Build and deploy your setup:
- **Docker Compose** - Automated container deployment
- **Service Health** - Real-time status monitoring
- **Production Readiness** - Comprehensive validation

## 🛠️ Quick Start

### Method 1: GUI Setup (Recommended)
```bash
# 1. Clone repository
git clone https://github.com/twadelij/CryptoJackal.git
cd CryptoJackal

# 2. Run setup script
./setup.sh

# 3. Open browser
# Navigate to http://localhost:3000/setup
```

### Method 2: CLI Setup Only
```bash
# 1. Clone repository
git clone https://github.com/twadelij/CryptoJackal.git
cd CryptoJackal

# 2. Run setup script
./setup.sh

# 3. Follow prompts
# Answer questions and configure your setup
```

### Method 3: Manual Docker Setup
```bash
# 1. Copy environment template
cp .env.example .env

# 2. Edit configuration
nano .env

# 3. Start services
docker-compose up -d
```

## 📊 Environment Types

### 🧪 Test Environment
**Perfect for:**
- First-time users
- Strategy testing
- Learning the platform
- Safe experimentation

**Features:**
- ✅ Paper trading enabled
- ✅ Mock data for testing
- ✅ Debug mode enabled
- ✅ Hot reload active
- ✅ No real money risk

### 🎓 Development Environment
**Perfect for:**
- Developers
- Feature testing
- Integration testing
- Custom modifications

**Features:**
- ✅ Paper trading enabled
- ✅ Debug mode enabled
- ✅ Hot reload active
- ✅ Profiling enabled
- ✅ Development tools

### 🚀 Production Environment
**Perfect for:**
- Live trading
- Real money deployment
- Automated trading
- Professional use

**Features:**
- ✅ Real trading enabled
- ✅ Security hardened
- ✅ Performance optimized
- ✅ Monitoring enabled
- ✅ Production alerts

## 🔧 Prerequisites

### Required Dependencies
- **Docker** - Container management
- **Docker Compose** - Multi-container orchestration
- **Node.js** - Frontend build tool
- **npm** - Package manager

### Optional Dependencies
- **Rust/Cargo** - Backend compilation (for manual builds)
- **OpenSSL** - Security key generation

### Quick Install (Ubuntu/Debian)
```bash
# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Install Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Install Node.js
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt-get install -y nodejs
```

## 📱 Access Points

After setup completion, access your CryptoJackal instance:

### Web Interface
- **Dashboard**: `http://localhost:3000`
- **Trading**: `http://localhost:3000/trading`
- **Paper Trading**: `http://localhost:3000/paper-trading`
- **Discovery**: `http://localhost:3000/discovery`
- **Settings**: `http://localhost:3000/settings`

### API Endpoints
- **Health Check**: `http://localhost:8080/health`
- **API Base**: `http://localhost:8080/api`
- **Documentation**: `http://localhost:8080/docs`

### Monitoring
- **Metrics**: `http://localhost:9090/metrics`
- **Health Status**: `http://localhost:8081/health`

## 🔒 Security Best Practices

### Test Environment
- ✅ Use paper trading only
- ✅ Keep API keys optional
- ✅ Enable debug logging
- ✅ Use localhost only

### Production Environment
- ✅ Generate strong JWT secrets
- ✅ Configure proper CORS origins
- ✅ Use HTTPS endpoints
- ✅ Disable debug mode
- ✅ Enable monitoring
- ✅ Use secure API keys

### General Security
- ✅ Never commit `.env` files
- ✅ Use unique passwords
- ✅ Keep dependencies updated
- ✅ Monitor logs regularly
- ✅ Use MetaMask for signing

## 🚨 Troubleshooting

### Common Issues

#### Setup Script Fails
```bash
# Check dependencies
docker --version
docker-compose --version
node --version
npm --version

# Fix permissions
chmod +x setup.sh
```

#### Services Won't Start
```bash
# Check Docker status
sudo systemctl status docker

# Check logs
docker-compose logs

# Restart services
docker-compose down
docker-compose up -d
```

#### Can't Access Web Interface
```bash
# Check port availability
netstat -tlnp | grep :3000

# Check container status
docker-compose ps

# Restart frontend
docker-compose restart web
```

#### API Not Responding
```bash
# Check backend logs
docker-compose logs api

# Test API directly
curl http://localhost:8080/health

# Restart backend
docker-compose restart api
```

### Getting Help

1. **Check Logs**: `docker-compose logs`
2. **Health Check**: `curl http://localhost:8080/health`
3. **Documentation**: Read `DEVELOPMENT_GUIDE.md`
4. **Issues**: [GitHub Issues](https://github.com/twadelij/CryptoJackal/issues)
5. **Community**: [GitHub Discussions](https://github.com/twadelij/CryptoJackal/discussions)

## 📚 Next Steps

### After Setup Completion

1. **🔗 Connect Wallet** - Set up MetaMask
2. **📊 Explore Dashboard** - Monitor system status
3. **🧪 Test Paper Trading** - Try strategies risk-free
4. **🔍 Discover Tokens** - Find new opportunities
5. **📈 Monitor Performance** - Track results
6. **📚 Read Documentation** - Learn advanced features

### Production Migration

When ready for production:

1. **Backup Configuration** - Save test settings
2. **Run Setup Again** - Choose production environment
3. **Configure Security** - Set up production security
4. **Test Thoroughly** - Verify all features
5. **Deploy** - Go live with real trading
6. **Monitor** - Watch system performance

## 🎉 Success!

Congratulations! You now have a fully configured CryptoJackal instance running. Whether you're testing strategies with paper trading or running a production trading bot, you're all set.

**Happy Trading! 🚀**

---

*Need help? Check our [documentation](DEVELOPMENT_GUIDE.md) or [open an issue](https://github.com/twadelij/CryptoJackal/issues).*

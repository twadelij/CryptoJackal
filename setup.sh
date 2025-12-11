#!/bin/bash

# =============================================================================
# CryptoJackal Interactive Setup Wizard
# =============================================================================
# This script guides you through complete setup from test to production
# =============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration variables
ENVIRONMENT=""
NODE_URL=""
CHAIN_ID=""
TRADE_AMOUNT=""
PAPER_TRADING_MODE=""
API_KEYS_CONFIGURED=false

# =============================================================================
# Helper Functions
# =============================================================================

print_header() {
    echo -e "${CYAN}"
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║                    CryptoJackal Setup Wizard                    ║"
    echo "║                 🚀 Test → Production Setup                  ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

print_step() {
    echo -e "\n${BLUE}📍 Step $1: $2${NC}"
}

print_success() {
    echo -e "\n${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "\n${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "\n${RED}❌ $1${NC}"
}

print_info() {
    echo -e "${PURPLE}ℹ️  $1${NC}"
}

check_command() {
    if command -v "$1" >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

prompt_user() {
    local prompt="$1"
    local default="$2"
    local var_name="$3"
    
    if [ -n "$default" ]; then
        echo -e "${CYAN}$prompt [$default]: ${NC}"
    else
        echo -e "${CYAN}$prompt: ${NC}"
    fi
    
    read -r response
    if [ -z "$response" ] && [ -n "$default" ]; then
        response="$default"
    fi
    
    eval "$var_name='$response'"
}

prompt_password() {
    local prompt="$1"
    local var_name="$2"
    
    echo -e "${CYAN}$prompt: ${NC}"
    read -s response
    echo
    eval "$var_name='$response'"
}

validate_ethereum_address() {
    local address="$1"
    if [[ $address =~ ^0x[a-fA-F0-9]{40}$ ]]; then
        return 0
    else
        return 1
    fi
}

validate_url() {
    local url="$1"
    if [[ $url =~ ^https?:// ]]; then
        return 0
    else
        return 1
    fi
}

# =============================================================================
# Prerequisites Check
# =============================================================================

check_prerequisites() {
    print_step "1" "Checking Prerequisites"
    
    local missing_deps=()
    
    # Check for required commands
    if ! check_command "docker"; then
        missing_deps+=("docker")
    fi
    
    if ! check_command "docker-compose"; then
        missing_deps+=("docker-compose")
    fi
    
    if ! check_command "node"; then
        missing_deps+=("node")
    fi
    
    if ! check_command "npm"; then
        missing_deps+=("npm")
    fi
    
    if [ ${#missing_deps[@]} -gt 0 ]; then
        print_error "Missing required dependencies: ${missing_deps[*]}"
        print_info "Please install the missing dependencies and run this script again."
        print_info "Visit: https://github.com/twadelij/CryptoJackal#prerequisites"
        exit 1
    fi
    
    print_success "All prerequisites satisfied!"
}

# =============================================================================
# Environment Selection
# =============================================================================

select_environment() {
    print_step "2" "Select Environment"
    
    echo -e "${CYAN}Choose your setup environment:${NC}"
    echo "1) 🧪 Test Environment (Recommended for first-time users)"
    echo "2) 🚀 Production Environment (For experienced users)"
    echo "3) 🎓 Development Environment (For developers)"
    
    while true; do
        prompt_user "Enter your choice (1-3)" "1" "choice"
        case $choice in
            1)
                ENVIRONMENT="test"
                PAPER_TRADING_MODE="true"
                print_success "Selected: Test Environment"
                break
                ;;
            2)
                ENVIRONMENT="production"
                PAPER_TRADING_MODE="false"
                print_success "Selected: Production Environment"
                break
                ;;
            3)
                ENVIRONMENT="development"
                PAPER_TRADING_MODE="true"
                print_success "Selected: Development Environment"
                break
                ;;
            *)
                print_warning "Please enter 1, 2, or 3"
                ;;
        esac
    done
}

# =============================================================================
# Node Configuration
# =============================================================================

configure_node() {
    print_step "3" "Configure Ethereum Node"
    
    echo -e "${CYAN}Choose your Ethereum node provider:${NC}"
    echo "1) 🌐 Infura (Free tier available)"
    echo "2) 🔮 Alchemy (Free tier with 300M compute units/month)"
    echo "3) ⚡ QuickNode (Free tier available)"
    echo "4) 🔧 Custom Node URL"
    
    while true; do
        prompt_user "Enter your choice (1-4)" "1" "node_choice"
        case $node_choice in
            1)
                NODE_URL="https://mainnet.infura.io/v3/YOUR_PROJECT_ID"
                print_info "Please create an Infura project and replace YOUR_PROJECT_ID"
                break
                ;;
            2)
                NODE_URL="https://eth-mainnet.g.alchemy.com/v2/YOUR_API_KEY"
                print_info "Please create an Alchemy account and replace YOUR_API_KEY"
                break
                ;;
            3)
                NODE_URL="https://YOUR_ENDPOINT.quiknode.pro/YOUR_KEY"
                print_info "Please create a QuickNode account and replace YOUR_ENDPOINT and YOUR_KEY"
                break
                ;;
            4)
                while true; do
                    prompt_user "Enter your custom Ethereum node URL" "" "NODE_URL"
                    if validate_url "$NODE_URL"; then
                        break
                    else
                        print_warning "Please enter a valid URL starting with http:// or https://"
                    fi
                done
                break
                ;;
            *)
                print_warning "Please enter 1, 2, 3, or 4"
                ;;
        esac
    done
    
    # Chain ID
    prompt_user "Enter Chain ID (1 for Ethereum Mainnet)" "1" "CHAIN_ID"
    
    print_success "Node configuration completed!"
}

# =============================================================================
# Trading Configuration
# =============================================================================

configure_trading() {
    print_step "4" "Configure Trading Parameters"
    
    # Trade Amount
    echo -e "${CYAN}Trade Amount Configuration:${NC}"
    print_info "This is the amount of ETH to trade per opportunity (in wei)"
    print_info "Common amounts:"
    print_info "  • 0.1 ETH  = 100000000000000000000 wei"
    print_info "  • 0.5 ETH  = 500000000000000000000 wei"
    print_info "  • 1.0 ETH  = 1000000000000000000000 wei"
    
    prompt_user "Enter trade amount (in wei)" "100000000000000000000" "TRADE_AMOUNT"
    
    # Scan Interval
    prompt_user "Enter scan interval (milliseconds)" "1000" "SCAN_INTERVAL"
    
    # Gas Limit
    prompt_user "Enter gas limit" "300000" "GAS_LIMIT"
    
    # Slippage Tolerance
    prompt_user "Enter slippage tolerance (as decimal, e.g., 0.005 for 0.5%)" "0.005" "SLIPPAGE_TOLERANCE"
    
    # Minimum Liquidity
    prompt_user "Enter minimum liquidity (in ETH)" "10.0" "MIN_LIQUIDITY"
    
    print_success "Trading configuration completed!"
}

# =============================================================================
# API Keys Configuration
# =============================================================================

configure_api_keys() {
    print_step "5" "Configure API Keys (Optional but Recommended)"
    
    echo -e "${CYAN}API Keys enhance token discovery and features:${NC}"
    echo "1) 🎯 CoinGecko API Key (Free tier available)"
    echo "2) 📊 DexScreener API Key (Free tier available)"
    echo "3) 🔔 Telegram Bot Token (For notifications)"
    echo "4) 📱 Discord Webhook URL (For notifications)"
    echo "5) ⏭️  Skip API keys (Can configure later)"
    
    while true; do
        prompt_user "Enter your choice (1-5)" "5" "api_choice"
        case $api_choice in
            1)
                prompt_user "Enter CoinGecko API Key" "" "COINGECKO_API_KEY"
                API_KEYS_CONFIGURED=true
                break
                ;;
            2)
                prompt_user "Enter DexScreener API Key" "" "DEXSCREENER_API_KEY"
                API_KEYS_CONFIGURED=true
                break
                ;;
            3)
                prompt_user "Enter Telegram Bot Token" "" "TELEGRAM_BOT_TOKEN"
                API_KEYS_CONFIGURED=true
                break
                ;;
            4)
                prompt_user "Enter Discord Webhook URL" "" "DISCORD_WEBHOOK_URL"
                API_KEYS_CONFIGURED=true
                break
                ;;
            5)
                print_info "Skipping API key configuration (can be configured later)"
                break
                ;;
            *)
                print_warning "Please enter 1, 2, 3, 4, or 5"
                ;;
        esac
    done
    
    if [ "$API_KEYS_CONFIGURED" = true ]; then
        print_success "API keys configured!"
    else
        print_warning "API keys not configured - some features may be limited"
    fi
}

# =============================================================================
# Security Configuration
# =============================================================================

configure_security() {
    print_step "6" "Configure Security"
    
    # Generate secure JWT secret
    print_info "Generating secure JWT secret..."
    JWT_SECRET=$(openssl rand -base64 32)
    print_success "JWT secret generated securely!"
    
    # Session timeout
    prompt_user "Enter session timeout (seconds)" "3600" "SESSION_TIMEOUT"
    
    # Max login attempts
    prompt_user "Enter max login attempts" "5" "MAX_LOGIN_ATTEMPTS"
    
    # CORS origins
    if [ "$ENVIRONMENT" = "development" ]; then
        CORS_ORIGINS="http://localhost:3000,http://localhost:5173"
    else
        prompt_user "Enter CORS origins (comma-separated)" "https://yourdomain.com" "CORS_ORIGINS"
    fi
    
    print_success "Security configuration completed!"
}

# =============================================================================
# Environment File Creation
# =============================================================================

create_env_file() {
    print_step "7" "Create Environment Configuration"
    
    local env_file=".env"
    
    if [ -f "$env_file" ]; then
        print_warning "Environment file already exists!"
        prompt_user "Backup existing file and create new one? (y/n)" "y" "backup_choice"
        if [[ $backup_choice =~ ^[Yy]$ ]]; then
            cp "$env_file" "$env_file.backup.$(date +%Y%m%d_%H%M%S)"
            print_success "Existing file backed up"
        else
            print_info "Keeping existing environment file"
            return 0
        fi
    fi
    
    # Create environment file
    cat > "$env_file" << EOF
# =============================================================================
# CryptoJackal Environment Configuration
# =============================================================================
# Environment: $ENVIRONMENT
# Generated: $(date)
# =============================================================================

# --------------------------------------------------------------
# Node Configuration
# --------------------------------------------------------------
NODE_URL=$NODE_URL
CHAIN_ID=$CHAIN_ID
NETWORK_NAME=ethereum

# --------------------------------------------------------------
# Trading Parameters
# --------------------------------------------------------------
SCAN_INTERVAL=$SCAN_INTERVAL
GAS_LIMIT=$GAS_LIMIT
SLIPPAGE_TOLERANCE=$SLIPPAGE_TOLERANCE
MIN_LIQUIDITY=$MIN_LIQUIDITY
MAX_PRICE_IMPACT=0.02
TRADE_AMOUNT=$TRADE_AMOUNT

# --------------------------------------------------------------
# Paper Trading Configuration
# --------------------------------------------------------------
PAPER_TRADING_MODE=$PAPER_TRADING_MODE
PAPER_TRADING_BALANCE=10.0
PAPER_TRADING_DATA_SOURCE=historical

# --------------------------------------------------------------
# API Configuration
# --------------------------------------------------------------
API_HOST=0.0.0.0
API_PORT=8080
API_BASE_URL=http://localhost:8080
CORS_ORIGINS=$CORS_ORIGINS

# --------------------------------------------------------------
# Security Configuration
# --------------------------------------------------------------
JWT_SECRET=$JWT_SECRET
SESSION_TIMEOUT=$SESSION_TIMEOUT
MAX_LOGIN_ATTEMPTS=$MAX_LOGIN_ATTEMPTS

# --------------------------------------------------------------
# Logging Configuration
# --------------------------------------------------------------
LOG_LEVEL=info
LOG_FORMAT=json
LOG_FILE_ENABLED=true
LOG_FILE_PATH=/var/log/cryptojackal/app.log

# --------------------------------------------------------------
# Monitoring Configuration
# --------------------------------------------------------------
METRICS_ENABLED=true
METRICS_PORT=9090
HEALTH_CHECK_ENABLED=true
HEALTH_CHECK_PORT=8081

# --------------------------------------------------------------
# Token Discovery Configuration
# --------------------------------------------------------------
DEXSCREENER_API_URL=https://api.dexscreener.com/latest/dex
COINGECKO_API_URL=https://api.coingecko.com/api/v3
DISCOVERY_SCAN_INTERVAL=30000
MAX_NEW_TOKENS_PER_SCAN=10
TOKEN_SECURITY_CHECK_ENABLED=true

# --------------------------------------------------------------
# API Keys (Optional)
# --------------------------------------------------------------
EOF

    # Add API keys if configured
    if [ -n "$COINGECKO_API_KEY" ]; then
        echo "COINGECKO_API_KEY=$COINGECKO_API_KEY" >> "$env_file"
    fi
    
    if [ -n "$DEXSCREENER_API_KEY" ]; then
        echo "DEXSCREENER_API_KEY=$DEXSCREENER_API_KEY" >> "$env_file"
    fi
    
    if [ -n "$TELEGRAM_BOT_TOKEN" ]; then
        echo "TELEGRAM_BOT_TOKEN=$TELEGRAM_BOT_TOKEN" >> "$env_file"
    fi
    
    if [ -n "$DISCORD_WEBHOOK_URL" ]; then
        echo "DISCORD_WEBHOOK_URL=$DISCORD_WEBHOOK_URL" >> "$env_file"
    fi
    
    # Add environment-specific settings
    if [ "$ENVIRONMENT" = "test" ]; then
        cat >> "$env_file" << EOF

# --------------------------------------------------------------
# Test Environment Settings
# --------------------------------------------------------------
ENVIRONMENT=test
DEBUG_MODE=true
HOT_RELOAD=true
ENABLE_PROFILING=false
EOF
    elif [ "$ENVIRONMENT" = "production" ]; then
        cat >> "$env_file" << EOF

# --------------------------------------------------------------
# Production Environment Settings
# --------------------------------------------------------------
ENVIRONMENT=production
DEBUG_MODE=false
HOT_RELOAD=false
ENABLE_PROFILING=true
EOF
    else
        cat >> "$env_file" << EOF

# --------------------------------------------------------------
# Development Environment Settings
# --------------------------------------------------------------
ENVIRONMENT=development
DEBUG_MODE=true
HOT_RELOAD=true
ENABLE_PROFILING=true
EOF
    fi
    
    print_success "Environment file created: $env_file"
}

# =============================================================================
# Build and Deploy
# =============================================================================

build_and_deploy() {
    print_step "8" "Build and Deploy"
    
    echo -e "${CYAN}Building CryptoJackal...${NC}"
    
    # Build backend
    if check_command "cargo"; then
        print_info "Building Rust backend..."
        cargo build --release
        print_success "Backend built successfully!"
    else
        print_warning "Cargo not found - skipping backend build"
    fi
    
    # Build frontend
    if [ -d "web" ]; then
        print_info "Building frontend..."
        cd web
        npm install
        npm run build
        cd ..
        print_success "Frontend built successfully!"
    else
        print_warning "Frontend directory not found - skipping frontend build"
    fi
    
    # Start services based on environment
    print_info "Starting services..."
    
    if [ "$ENVIRONMENT" = "test" ]; then
        print_info "Starting test environment with Docker Compose..."
        docker-compose -f docker-compose.dev.yml up -d
        print_success "Test environment started!"
        print_info "Access at: http://localhost:3000"
        print_info "API at: http://localhost:8080"
        
    elif [ "$ENVIRONMENT" = "production" ]; then
        print_info "Starting production environment with Docker Compose..."
        docker-compose up -d
        print_success "Production environment started!"
        print_info "Access at: http://localhost:3000"
        print_info "API at: http://localhost:8080"
        
    else
        print_info "Starting development environment..."
        print_info "Backend: cargo run"
        print_info "Frontend: cd web && npm run dev"
    fi
}

# =============================================================================
# Testing
# =============================================================================

run_tests() {
    print_step "9" "Run Tests (Optional)"
    
    prompt_user "Run comprehensive tests? (y/n)" "y" "run_tests"
    
    if [[ $run_tests =~ ^[Yy]$ ]]; then
        print_info "Running tests..."
        
        if check_command "cargo"; then
            print_info "Running Rust tests..."
            cargo test
            print_success "Backend tests completed!"
        fi
        
        if [ -d "web" ]; then
            print_info "Running frontend tests..."
            cd web
            npm test 2>/dev/null || print_warning "Frontend tests not configured"
            cd ..
        fi
        
        print_success "All tests completed!"
    else
        print_info "Skipping tests"
    fi
}

# =============================================================================
# Production Readiness Check
# =============================================================================

production_readiness_check() {
    print_step "10" "Production Readiness Check"
    
    if [ "$ENVIRONMENT" = "production" ]; then
        print_warning "PRODUCTION ENVIRONMENT DETECTED"
        print_info "Performing production readiness checks..."
        
        local checks_passed=0
        local total_checks=5
        
        # Check 1: Environment variables
        if [ -f ".env" ] && grep -q "NODE_URL=" .env; then
            print_success "✓ Environment file configured"
            ((checks_passed++))
        else
            print_error "✗ Environment file not properly configured"
        fi
        
        # Check 2: JWT secret
        if grep -q "JWT_SECRET=" .env && ! grep -q "default-secret" .env; then
            print_success "✓ JWT secret is secure"
            ((checks_passed++))
        else
            print_error "✗ JWT secret not configured or using default"
        fi
        
        # Check 3: API keys
        if grep -q "API_KEY" .env; then
            print_success "✓ API keys configured"
            ((checks_passed++))
        else
            print_warning "⚠ API keys not configured (optional)"
            ((checks_passed++))
        fi
        
        # Check 4: Paper trading disabled
        if grep -q "PAPER_TRADING_MODE=false" .env; then
            print_success "✓ Paper trading disabled for production"
            ((checks_passed++))
        else
            print_error "✗ Paper trading still enabled - not production ready"
        fi
        
        # Check 5: Debug mode disabled
        if grep -q "DEBUG_MODE=false" .env; then
            print_success "✓ Debug mode disabled"
            ((checks_passed++))
        else
            print_error "✗ Debug mode still enabled - not production ready"
        fi
        
        # Final assessment
        local readiness_score=$((checks_passed * 100 / total_checks))
        
        echo -e "\n${CYAN}Production Readiness Score: $readiness_score%${NC}"
        
        if [ $readiness_score -ge 80 ]; then
            print_success "✅ READY FOR PRODUCTION!"
            print_info "Your CryptoJackal instance is production-ready"
        else
            print_warning "⚠️  NOT READY FOR PRODUCTION"
            print_info "Please address the failed checks above"
        fi
    else
        print_success "✅ Test/Development environment configured"
        print_info "Ready for testing and development"
    fi
}

# =============================================================================
# Main Setup Flow
# =============================================================================

main() {
    print_header
    
    check_prerequisites
    select_environment
    configure_node
    configure_trading
    configure_api_keys
    configure_security
    create_env_file
    build_and_deploy
    run_tests
    production_readiness_check
    
    print_success "🎉 CryptoJackal setup completed successfully!"
    
    echo -e "\n${CYAN}Next Steps:${NC}"
    echo "1. 🌐 Access the web interface at http://localhost:3000"
    echo "2. 🔑 Connect your MetaMask wallet"
    echo "3. 📊 Monitor the dashboard"
    echo "4. 🧪 Test with paper trading first"
    echo "5. 📚 Read the documentation at docs/"
    
    if [ "$ENVIRONMENT" = "test" ]; then
        echo "6. 🚀 When ready, run setup again and choose production"
    fi
    
    echo -e "\n${GREEN}Happy Trading! 🚀${NC}"
}

# Run the setup
main "$@"

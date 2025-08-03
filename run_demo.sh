#!/bin/bash

echo "🎭 CryptoJackal Transaction Signing Demo"
echo "========================================"
echo ""

# Set up demo environment variables
export RUST_LOG=info
export RUST_BACKTRACE=1

# Create a .env file for demo if it doesn't exist
if [ ! -f .env ]; then
    echo "📝 Creating demo .env file..."
    cat > .env << EOF
# Demo Configuration
NODE_URL=https://mainnet.infura.io/v3/demo
PRIVATE_KEY=0x0000000000000000000000000000000000000000000000000000000000000001
SCAN_INTERVAL=1000
GAS_LIMIT=200000
SLIPPAGE_TOLERANCE=500
MIN_LIQUIDITY=1000000
MAX_PRICE_IMPACT=0.05
TRADE_AMOUNT=1000000000000000000
TARGET_TOKENS=0xA0b86a33E6441b8C4C8C8C8C8C8C8C8C8C8C8C8
EOF
    echo "✅ Demo .env file created"
fi

echo "🚀 Running CryptoJackal Demo..."
echo ""

# Run the demo
cargo run --bin demo 2>&1 | tee demo_output.log

echo ""
echo "📊 Demo completed! Check demo_output.log for full output."
echo "🎯 Key features demonstrated:"
echo "   ✅ Transaction Signing Workflow integration"
echo "   ✅ Gas strategy management"
echo "   ✅ Transaction lifecycle simulation"
echo "   ✅ Market opportunity processing"
echo "   ✅ Performance metrics" 
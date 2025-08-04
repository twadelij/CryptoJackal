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
SCAN_INTERVAL=1000
GAS_LIMIT=200000
SLIPPAGE_TOLERANCE=0.005
MIN_LIQUIDITY=10.0
MAX_PRICE_IMPACT=0.02
TRADE_AMOUNT=100000000000000000
TARGET_TOKENS=0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984,0x6B175474E89094C44Da98b954EedeAC495271d0F
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
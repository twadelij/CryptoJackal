package main

import (
	"context"
	"fmt"
	"time"

	"github.com/twadelij/cryptojackal/internal/config"
	"github.com/twadelij/cryptojackal/internal/discovery"
	"github.com/twadelij/cryptojackal/internal/models"
	"github.com/twadelij/cryptojackal/internal/paper"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	fmt.Println("🐺 CryptoJackal Demo")
	fmt.Println("====================")
	fmt.Println()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	ctx := context.Background()

	// Demo discovery service
	fmt.Println("📡 Testing Token Discovery...")
	discSvc := discovery.NewService(cfg.CoinGeckoAPIKey, logger)
	
	trending, err := discSvc.GetTrendingTokens(ctx)
	if err != nil {
		fmt.Printf("   ❌ Failed to get trending: %v\n", err)
	} else {
		fmt.Printf("   ✅ Found %d trending tokens\n", len(trending))
		for i, t := range trending {
			if i >= 3 {
				break
			}
			fmt.Printf("      - %s (%s)\n", t.Name, t.Symbol)
		}
	}
	fmt.Println()

	// Demo paper trading
	fmt.Println("📝 Testing Paper Trading...")
	paperSvc := paper.NewService(10.0, logger)
	
	portfolio := paperSvc.GetPortfolio()
	fmt.Printf("   Initial balance: %.4f ETH\n", portfolio.ETHBalance)

	// Simulate a trade
	testToken := models.Token{
		Address: "0x1234567890abcdef1234567890abcdef12345678",
		Symbol:  "TEST",
		Name:    "Test Token",
		Price:   0.001,
	}

	trade, err := paperSvc.ExecuteTrade(ctx, testToken, models.TradeTypeBuy, 1000)
	if err != nil {
		fmt.Printf("   ❌ Trade failed: %v\n", err)
	} else {
		fmt.Printf("   ✅ Bought %s tokens for %.4f ETH\n", trade.TokenSymbol, trade.AmountIn*trade.Price)
	}

	portfolio = paperSvc.GetPortfolio()
	fmt.Printf("   Balance after trade: %.4f ETH\n", portfolio.ETHBalance)
	fmt.Printf("   Token holdings: %d different tokens\n", len(portfolio.TokenBalances))
	fmt.Println()

	// Demo metrics
	fmt.Println("📊 Trading Metrics...")
	metrics := paperSvc.GetMetrics()
	fmt.Printf("   Total trades: %d\n", metrics.TotalTrades)
	fmt.Printf("   Total volume: %.4f\n", metrics.TotalVolume)
	fmt.Println()

	// Demo opportunity finding
	fmt.Println("🔍 Scanning for Opportunities...")
	opportunities, err := discSvc.FindOpportunities(ctx, "ethereum", 10000)
	if err != nil {
		fmt.Printf("   ❌ Failed: %v\n", err)
	} else {
		fmt.Printf("   ✅ Found %d opportunities\n", len(opportunities))
		for i, opp := range opportunities {
			if i >= 3 {
				break
			}
			fmt.Printf("      - %s: %.2f%% expected profit (confidence: %.0f%%)\n", 
				opp.Token.Symbol, opp.ExpectedProfit, opp.ConfidenceScore*100)
		}
	}
	fmt.Println()

	fmt.Println("✅ Demo complete!")
	fmt.Printf("   Timestamp: %s\n", time.Now().Format(time.RFC3339))
	fmt.Println()
	fmt.Println("🚀 Run 'cryptojackal' to start the full trading bot")
}

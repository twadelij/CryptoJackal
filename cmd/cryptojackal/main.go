package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/twadelij/cryptojackal/internal/api"
	"github.com/twadelij/cryptojackal/internal/config"
	"github.com/twadelij/cryptojackal/internal/discovery"
	"github.com/twadelij/cryptojackal/internal/paper"
	"github.com/twadelij/cryptojackal/internal/trading"
	"github.com/twadelij/cryptojackal/internal/wallet"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// Initialize logger
	logConfig := zap.NewProductionConfig()
	logConfig.EncoderConfig.TimeKey = "timestamp"
	logConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	
	logger, err := logConfig.Build()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	logger.Info("🐺 CryptoJackal starting...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	logger.Info("configuration loaded",
		zap.String("environment", cfg.Environment),
		zap.Bool("paper_mode", cfg.PaperTradingMode),
	)

	// Initialize wallet (optional for paper trading)
	var w *wallet.Wallet
	if cfg.NodeURL != "" {
		w, err = wallet.New(cfg.NodeURL, cfg.PrivateKey, cfg.ChainID, logger)
		if err != nil {
			logger.Warn("wallet initialization failed, continuing in paper mode only", zap.Error(err))
		} else {
			logger.Info("wallet initialized", zap.String("address", w.Address().Hex()))
		}
	}

	// Initialize services
	discoverySvc := discovery.NewService(cfg.CoinGeckoAPIKey, logger)
	paperSvc := paper.NewService(cfg.InitialBalance, logger)
	engine := trading.NewEngine(cfg, w, discoverySvc, paperSvc, logger)

	// Initialize API server
	server := api.NewServer(cfg, engine, discoverySvc, paperSvc, logger)

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("shutdown signal received")
		engine.Stop()
		server.Shutdown(context.Background())
	}()

	// Start the server
	logger.Info("🚀 CryptoJackal is ready!")
	if err := server.Start(); err != nil && err.Error() != "http: Server closed" {
		logger.Fatal("server error", zap.Error(err))
	}

	logger.Info("👋 CryptoJackal shutdown complete")
}

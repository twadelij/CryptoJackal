package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/twadelij/cryptojackal/internal/api"
	"github.com/twadelij/cryptojackal/internal/config"
	"github.com/twadelij/cryptojackal/internal/discovery"
	"github.com/twadelij/cryptojackal/internal/paper"
	"github.com/twadelij/cryptojackal/internal/storage"
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

	// Initialize SQLite storage
	dataDir := filepath.Join(os.Getenv("HOME"), ".cryptojackal")
	os.MkdirAll(dataDir, 0755)
	store, err := storage.New(filepath.Join(dataDir, "cryptojackal.db"))
	if err != nil {
		logger.Fatal("failed to initialize storage", zap.Error(err))
	}
	defer store.Close()
	logger.Info("storage initialized", zap.String("path", filepath.Join(dataDir, "cryptojackal.db")))

	// Override config with stored values if they exist
	if err := cfg.LoadFromStorage(store); err != nil {
		logger.Warn("failed to load config from storage", zap.Error(err))
	}

	logger.Info("configuration loaded",
		zap.String("environment", cfg.Environment),
		zap.Bool("paper_mode", cfg.PaperTradingMode),
	)

	// Initialize wallet (optional for paper trading)
	var w *wallet.Wallet
	if !cfg.PaperTradingMode && cfg.NodeURL != "" {
		w, err = wallet.New(context.Background(), cfg.NodeURL, cfg.PrivateKey, cfg.ChainID, logger)
		if err != nil {
			logger.Warn("wallet initialization failed, continuing without live trading", zap.Error(err))
		} else {
			logger.Info("wallet initialized", zap.String("address", w.Address().Hex()))
		}
	} else {
		logger.Info("paper trading mode enabled; skipping wallet initialization")
	}

	// Initialize services
	discoverySvc := discovery.NewService(cfg.CoinGeckoAPIKey, logger)
	paperSvc := paper.NewServiceWithStorage(cfg.InitialBalance, logger, store)
	engine := trading.NewEngine(cfg, w, discoverySvc, paperSvc, logger)

	// Initialize API server
	server := api.NewServer(cfg, engine, discoverySvc, paperSvc, store, logger)

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("shutdown signal received")

		// Persist config before shutdown
		if err := cfg.SaveToStorage(store); err != nil {
			logger.Warn("failed to save config to storage", zap.Error(err))
		}

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

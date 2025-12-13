package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	ServerPort string
	ServerHost string

	// Ethereum
	NodeURL     string
	ChainID     int64
	PrivateKey  string
	WalletAddress string

	// Trading
	TradeAmount       float64
	MaxSlippage       float64
	MinLiquidity      float64
	MaxPriceImpact    float64
	ScanInterval      time.Duration
	GasLimit          uint64
	MaxGasPrice       uint64

	// Paper Trading
	PaperTradingMode  bool
	InitialBalance    float64

	// API Keys
	CoinGeckoAPIKey   string
	DexScreenerAPIKey string

	// Notifications
	TelegramBotToken  string
	TelegramChatID    string
	DiscordWebhookURL string

	// Security
	JWTSecret         string
	CORSOrigins       []string

	// Redis
	RedisURL          string

	// Environment
	Environment       string
}

func Load() (*Config, error) {
	// Load .env file if it exists
	godotenv.Load()

	cfg := &Config{
		// Server defaults
		ServerPort: getEnv("SERVER_PORT", "8080"),
		ServerHost: getEnv("SERVER_HOST", "0.0.0.0"),

		// Ethereum defaults
		NodeURL:    getEnv("ETH_NODE_URL", ""),
		ChainID:    getEnvInt64("CHAIN_ID", 1),
		PrivateKey: getEnv("PRIVATE_KEY", ""),

		// Trading defaults
		TradeAmount:    getEnvFloat("TRADE_AMOUNT", 0.1),
		MaxSlippage:    getEnvFloat("MAX_SLIPPAGE", 0.5),
		MinLiquidity:   getEnvFloat("MIN_LIQUIDITY", 10000),
		MaxPriceImpact: getEnvFloat("MAX_PRICE_IMPACT", 3.0),
		ScanInterval:   time.Duration(getEnvInt("SCAN_INTERVAL_SECONDS", 30)) * time.Second,
		GasLimit:       uint64(getEnvInt("GAS_LIMIT", 300000)),
		MaxGasPrice:    uint64(getEnvInt("MAX_GAS_PRICE_GWEI", 100)),

		// Paper trading defaults
		PaperTradingMode: getEnvBool("PAPER_TRADING_MODE", true),
		InitialBalance:   getEnvFloat("INITIAL_BALANCE", 10.0),

		// API Keys
		CoinGeckoAPIKey:   getEnv("COINGECKO_API_KEY", ""),
		DexScreenerAPIKey: getEnv("DEXSCREENER_API_KEY", ""),

		// Notifications
		TelegramBotToken:  getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:    getEnv("TELEGRAM_CHAT_ID", ""),
		DiscordWebhookURL: getEnv("DISCORD_WEBHOOK_URL", ""),

		// Security
		JWTSecret:   getEnv("JWT_SECRET", "change-me-in-production"),
		CORSOrigins: []string{getEnv("CORS_ORIGINS", "*")},

		// Redis
		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379"),

		// Environment
		Environment: getEnv("ENVIRONMENT", "development"),
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

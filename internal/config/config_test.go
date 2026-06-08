package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ServerPort != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.ServerPort)
	}
	if cfg.ServerHost != "0.0.0.0" {
		t.Errorf("expected default host 0.0.0.0, got %s", cfg.ServerHost)
	}
	if cfg.ChainID != 1 {
		t.Errorf("expected default chain ID 1, got %d", cfg.ChainID)
	}
	if cfg.TradeAmount != 0.1 {
		t.Errorf("expected default trade amount 0.1, got %f", cfg.TradeAmount)
	}
	if cfg.MaxSlippage != 0.5 {
		t.Errorf("expected default max slippage 0.5, got %f", cfg.MaxSlippage)
	}
	if cfg.MinLiquidity != 10000 {
		t.Errorf("expected default min liquidity 10000, got %f", cfg.MinLiquidity)
	}
	if cfg.MaxPriceImpact != 3.0 {
		t.Errorf("expected default max price impact 3.0, got %f", cfg.MaxPriceImpact)
	}
	if cfg.ScanInterval != 30*time.Second {
		t.Errorf("expected default scan interval 30s, got %v", cfg.ScanInterval)
	}
	if cfg.GasLimit != 300000 {
		t.Errorf("expected default gas limit 300000, got %d", cfg.GasLimit)
	}
	if cfg.MaxGasPrice != 100 {
		t.Errorf("expected default max gas price 100, got %d", cfg.MaxGasPrice)
	}
	if !cfg.PaperTradingMode {
		t.Errorf("expected default paper trading mode true")
	}
	if cfg.InitialBalance != 10.0 {
		t.Errorf("expected default initial balance 10.0, got %f", cfg.InitialBalance)
	}
	if cfg.JWTSecret != "change-me-in-production" {
		t.Errorf("expected default JWT secret, got %s", cfg.JWTSecret)
	}
	if cfg.Environment != "development" {
		t.Errorf("expected default environment development, got %s", cfg.Environment)
	}
}

func TestLoadFromEnv(t *testing.T) {
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("PAPER_TRADING_MODE", "false")
	os.Setenv("INITIAL_BALANCE", "5000")
	defer func() {
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("PAPER_TRADING_MODE")
		os.Unsetenv("INITIAL_BALANCE")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ServerPort != "9090" {
		t.Errorf("expected port 9090 from env, got %s", cfg.ServerPort)
	}
	if cfg.PaperTradingMode {
		t.Errorf("expected paper trading mode false from env")
	}
	if cfg.InitialBalance != 5000.0 {
		t.Errorf("expected initial balance 5000.0 from env, got %f", cfg.InitialBalance)
	}
}

func TestGetEnv(t *testing.T) {
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	val := getEnv("TEST_VAR", "default")
	if val != "test_value" {
		t.Errorf("expected test_value, got %s", val)
	}

	val = getEnv("NONEXISTENT_VAR", "default")
	if val != "default" {
		t.Errorf("expected default, got %s", val)
	}
}

func TestGetEnvInt(t *testing.T) {
	os.Setenv("TEST_INT", "42")
	defer os.Unsetenv("TEST_INT")

	val := getEnvInt("TEST_INT", 0)
	if val != 42 {
		t.Errorf("expected 42, got %d", val)
	}

	val = getEnvInt("NONEXISTENT_INT", 10)
	if val != 10 {
		t.Errorf("expected default 10, got %d", val)
	}

	val = getEnvInt("TEST_INVALID_INT", 10)
	if val != 10 {
		t.Errorf("expected default 10 for invalid, got %d", val)
	}
}

func TestGetEnvFloat(t *testing.T) {
	os.Setenv("TEST_FLOAT", "3.14")
	defer os.Unsetenv("TEST_FLOAT")

	val := getEnvFloat("TEST_FLOAT", 0.0)
	if val != 3.14 {
		t.Errorf("expected 3.14, got %f", val)
	}

	val = getEnvFloat("NONEXISTENT_FLOAT", 1.0)
	if val != 1.0 {
		t.Errorf("expected default 1.0, got %f", val)
	}
}

func TestGetEnvBool(t *testing.T) {
	os.Setenv("TEST_BOOL", "true")
	defer os.Unsetenv("TEST_BOOL")

	val := getEnvBool("TEST_BOOL", false)
	if !val {
		t.Errorf("expected true, got false")
	}

	val = getEnvBool("NONEXISTENT_BOOL", true)
	if !val {
		t.Errorf("expected default true, got false")
	}
}

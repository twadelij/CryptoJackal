package storage

import (
	"os"
	"testing"
	"time"

	"github.com/twadelij/cryptojackal/internal/models"
)

func newTestStorage(t *testing.T) *Storage {
	t.Helper()
	dbPath := "/tmp/cj_test_" + t.Name() + ".db"
	_ = os.Remove(dbPath) // Clean up previous test

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	return s
}

func TestSaveAndGetTrade(t *testing.T) {
	s := newTestStorage(t)
	defer s.Close()
	defer os.Remove(s.DBPath())

	trade := &models.Trade{
		ID:           "trade-1",
		TokenAddress: "0xTEST",
		TokenSymbol:  "TEST",
		Type:         models.TradeTypeBuy,
		AmountIn:     1000,
		Price:        0.001,
		Status:       models.TradeStatusExecuted,
		IsPaperTrade: true,
		ExecutedAt:   time.Now(),
	}

	if err := s.SaveTrade(trade); err != nil {
		t.Fatalf("failed to save trade: %v", err)
	}

	trades, err := s.GetTrades(10)
	if err != nil {
		t.Fatalf("failed to get trades: %v", err)
	}

	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}

	if trades[0].ID != trade.ID {
		t.Errorf("expected trade ID %s, got %s", trade.ID, trades[0].ID)
	}
	if trades[0].TokenSymbol != "TEST" {
		t.Errorf("expected symbol TEST, got %s", trades[0].TokenSymbol)
	}
}

func TestSaveAndLoadPortfolio(t *testing.T) {
	s := newTestStorage(t)
	defer s.Close()
	defer os.Remove(s.DBPath())

	portfolio := &models.Portfolio{
		ID:            "port-1",
		Balance:       9000,
		Currency:      "EUR",
		TotalValue:    9500,
		ProfitLoss:    500,
		UpdatedAt:     time.Now(),
		TokenBalances: map[string]models.TokenBalance{
			"0xTEST": {
				Token:    models.Token{Address: "0xTEST", Symbol: "TEST", Name: "Test Token", Price: 0.001},
				Balance:  1000,
				Value:    1.0,
				AvgPrice: 0.001,
			},
		},
	}

	if err := s.SavePortfolio(portfolio, 10000); err != nil {
		t.Fatalf("failed to save portfolio: %v", err)
	}

	loaded, initialBalance, err := s.LoadPortfolio("port-1")
	if err != nil {
		t.Fatalf("failed to load portfolio: %v", err)
	}

	if loaded == nil {
		t.Fatal("expected portfolio, got nil")
	}

	if loaded.Balance != 9000 {
		t.Errorf("expected balance 9000, got %f", loaded.Balance)
	}
	if initialBalance != 10000 {
		t.Errorf("expected initial balance 10000, got %f", initialBalance)
	}
	if len(loaded.TokenBalances) != 1 {
		t.Errorf("expected 1 token balance, got %d", len(loaded.TokenBalances))
	}
}

func TestConfig(t *testing.T) {
	s := newTestStorage(t)
	defer s.Close()
	defer os.Remove(s.DBPath())

	if err := s.SetConfig("paper_mode", "true"); err != nil {
		t.Fatalf("failed to set config: %v", err)
	}

	value, err := s.GetConfig("paper_mode")
	if err != nil {
		t.Fatalf("failed to get config: %v", err)
	}

	if value != "true" {
		t.Errorf("expected 'true', got '%s'", value)
	}

	all, err := s.GetAllConfigs()
	if err != nil {
		t.Fatalf("failed to get all configs: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("expected 1 config, got %d", len(all))
	}
}

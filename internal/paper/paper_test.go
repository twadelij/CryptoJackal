package paper

import (
	"context"
	"testing"

	"github.com/twadelij/cryptojackal/internal/models"
	"go.uber.org/zap"
)

func newTestService() *Service {
	logger, _ := zap.NewDevelopment()
	return NewService(10.0, logger)
}

func TestNewService(t *testing.T) {
	svc := newTestService()
	portfolio := svc.GetPortfolio()

	if portfolio.Balance != 10.0 {
		t.Errorf("expected balance 10.0, got %f", portfolio.Balance)
	}
	if portfolio.Currency != "EUR" {
		t.Errorf("expected currency EUR, got %s", portfolio.Currency)
	}
	if portfolio.TotalValue != 10.0 {
		t.Errorf("expected total value 10.0, got %f", portfolio.TotalValue)
	}
	if len(portfolio.TokenBalances) != 0 {
		t.Errorf("expected empty token balances, got %d", len(portfolio.TokenBalances))
	}
}

func TestBuyTrade(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	token := models.Token{
		Address: "0xTEST",
		Symbol:  "TEST",
		Name:    "Test Token",
		Price:   0.001,
	}

	trade, err := svc.ExecuteTrade(ctx, token, models.TradeTypeBuy, 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if trade.Status != models.TradeStatusExecuted {
		t.Errorf("expected status executed, got %s", trade.Status)
	}

	portfolio := svc.GetPortfolio()
	if portfolio.Balance != 9.0 {
		t.Errorf("expected balance 9.0, got %f", portfolio.Balance)
	}
	if len(portfolio.TokenBalances) != 1 {
		t.Errorf("expected 1 token balance, got %d", len(portfolio.TokenBalances))
	}
	if portfolio.TokenBalances[token.Address].Balance != 1000 {
		t.Errorf("expected token balance 1000, got %f", portfolio.TokenBalances[token.Address].Balance)
	}
}

func TestBuyInsufficientBalance(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	token := models.Token{
		Address: "0xTEST",
		Symbol:  "TEST",
		Price:   1.0,
	}

	_, err := svc.ExecuteTrade(ctx, token, models.TradeTypeBuy, 100)
	if err == nil {
		t.Error("expected error for insufficient balance")
	}

	portfolio := svc.GetPortfolio()
	if portfolio.Balance != 10.0 {
		t.Errorf("expected balance unchanged at 10.0, got %f", portfolio.Balance)
	}
}

func TestSellTrade(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	token := models.Token{
		Address: "0xTEST",
		Symbol:  "TEST",
		Price:   0.001,
	}

	svc.ExecuteTrade(ctx, token, models.TradeTypeBuy, 1000)

	token.Price = 0.002
	trade, err := svc.ExecuteTrade(ctx, token, models.TradeTypeSell, 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if trade.ProfitLoss <= 0 {
		t.Errorf("expected positive P&L, got %f", trade.ProfitLoss)
	}

	portfolio := svc.GetPortfolio()
	if portfolio.Balance != 11.0 {
		t.Errorf("expected balance 11.0 (10 + 1 profit), got %f", portfolio.Balance)
	}
	if len(portfolio.TokenBalances) != 0 {
		t.Errorf("expected empty token balances after full sell, got %d", len(portfolio.TokenBalances))
	}
}

func TestSellInsufficientTokens(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	token := models.Token{
		Address: "0xTEST",
		Symbol:  "TEST",
		Price:   0.001,
	}

	_, err := svc.ExecuteTrade(ctx, token, models.TradeTypeSell, 1000)
	if err == nil {
		t.Error("expected error for insufficient token balance")
	}
}

func TestPartialSell(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	token := models.Token{
		Address: "0xTEST",
		Symbol:  "TEST",
		Price:   0.001,
	}

	svc.ExecuteTrade(ctx, token, models.TradeTypeBuy, 1000)

	trade, err := svc.ExecuteTrade(ctx, token, models.TradeTypeSell, 500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if trade.Status != models.TradeStatusExecuted {
		t.Errorf("expected status executed, got %s", trade.Status)
	}

	portfolio := svc.GetPortfolio()
	if portfolio.TokenBalances[token.Address].Balance != 500 {
		t.Errorf("expected remaining balance 500, got %f", portfolio.TokenBalances[token.Address].Balance)
	}
}

func TestReset(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	token := models.Token{Address: "0xTEST", Symbol: "TEST", Price: 0.001}
	svc.ExecuteTrade(ctx, token, models.TradeTypeBuy, 1000)

	svc.Reset()

	portfolio := svc.GetPortfolio()
	if portfolio.Balance != 10.0 {
		t.Errorf("expected balance reset to 10.0, got %f", portfolio.Balance)
	}
	if len(portfolio.TokenBalances) != 0 {
		t.Errorf("expected empty token balances after reset")
	}
	if portfolio.ProfitLoss != 0 {
		t.Errorf("expected profit/loss reset to 0, got %f", portfolio.ProfitLoss)
	}
}

func TestMetrics(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	token := models.Token{Address: "0xTEST", Symbol: "TEST", Price: 0.001}
	svc.ExecuteTrade(ctx, token, models.TradeTypeBuy, 1000)

	metrics := svc.GetMetrics()
	if metrics.TotalTrades != 1 {
		t.Errorf("expected 1 trade, got %d", metrics.TotalTrades)
	}
	if metrics.TotalVolume != 1.0 {
		t.Errorf("expected total volume 1.0, got %f", metrics.TotalVolume)
	}
}

func TestTradeHistory(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	token := models.Token{Address: "0xTEST", Symbol: "TEST", Price: 0.001}
	svc.ExecuteTrade(ctx, token, models.TradeTypeBuy, 1000)
	svc.ExecuteTrade(ctx, token, models.TradeTypeBuy, 500)

	history := svc.GetTradeHistory(50)
	if len(history) != 2 {
		t.Errorf("expected 2 trades in history, got %d", len(history))
	}

	// Most recent first
	if history[0].AmountIn != 500 {
		t.Errorf("expected most recent trade amount 500, got %f", history[0].AmountIn)
	}
}

func TestTradeHistoryLimit(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	token := models.Token{Address: "0xTEST", Symbol: "TEST", Price: 0.001}
	svc.ExecuteTrade(ctx, token, models.TradeTypeBuy, 1000)
	svc.ExecuteTrade(ctx, token, models.TradeTypeBuy, 500)
	svc.ExecuteTrade(ctx, token, models.TradeTypeBuy, 250)

	history := svc.GetTradeHistory(2)
	if len(history) != 2 {
		t.Errorf("expected 2 trades with limit, got %d", len(history))
	}
}

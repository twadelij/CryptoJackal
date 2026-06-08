package models

import (
	"testing"
	"time"
)

func TestNewTrade(t *testing.T) {
	trade := NewTrade("0xABC", "ABC", TradeTypeBuy, 1000, 0.001, true)

	if trade.ID == "" {
		t.Error("expected trade ID to be generated")
	}
	if trade.TokenAddress != "0xABC" {
		t.Errorf("expected token address 0xABC, got %s", trade.TokenAddress)
	}
	if trade.TokenSymbol != "ABC" {
		t.Errorf("expected token symbol ABC, got %s", trade.TokenSymbol)
	}
	if trade.Type != TradeTypeBuy {
		t.Errorf("expected trade type buy, got %s", trade.Type)
	}
	if trade.AmountIn != 1000 {
		t.Errorf("expected amount in 1000, got %f", trade.AmountIn)
	}
	if trade.Price != 0.001 {
		t.Errorf("expected price 0.001, got %f", trade.Price)
	}
	if trade.Status != TradeStatusPending {
		t.Errorf("expected status pending, got %s", trade.Status)
	}
	if !trade.IsPaperTrade {
		t.Error("expected is_paper_trade to be true")
	}
	if trade.ExecutedAt.IsZero() {
		t.Error("expected executed_at to be set")
	}
}

func TestNewTradeSell(t *testing.T) {
	trade := NewTrade("0xDEF", "DEF", TradeTypeSell, 500, 0.002, false)

	if trade.Type != TradeTypeSell {
		t.Errorf("expected trade type sell, got %s", trade.Type)
	}
	if trade.IsPaperTrade {
		t.Error("expected is_paper_trade to be false")
	}
}

func TestNewOpportunity(t *testing.T) {
	token := Token{
		Address: "0xTOKEN",
		Symbol:  "TOKEN",
		Name:    "Test Token",
		Price:   0.001,
	}

	opp := NewOpportunity(token, 15.0, 0.5, 0.8, "momentum")

	if opp.ID == "" {
		t.Error("expected opportunity ID to be generated")
	}
	if opp.Token.Symbol != "TOKEN" {
		t.Errorf("expected token symbol TOKEN, got %s", opp.Token.Symbol)
	}
	if opp.ExpectedProfit != 15.0 {
		t.Errorf("expected expected profit 15.0, got %f", opp.ExpectedProfit)
	}
	if opp.PriceImpact != 0.5 {
		t.Errorf("expected price impact 0.5, got %f", opp.PriceImpact)
	}
	if opp.ConfidenceScore != 0.8 {
		t.Errorf("expected confidence score 0.8, got %f", opp.ConfidenceScore)
	}
	if opp.Strategy != "momentum" {
		t.Errorf("expected strategy momentum, got %s", opp.Strategy)
	}
	if opp.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}
	if opp.ExpiresAt.IsZero() {
		t.Error("expected expires_at to be set")
	}
	if !opp.ExpiresAt.After(opp.CreatedAt) {
		t.Error("expected expires_at to be after created_at")
	}
}

func TestTokenFields(t *testing.T) {
	token := Token{
		Address:        "0x123",
		Symbol:         "TEST",
		Name:           "Test Token",
		Decimals:       18,
		Price:          0.001,
		PriceChange24h: 5.5,
		MarketCap:      1000000,
		Volume24h:      500000,
		Liquidity:      200000,
		SecurityScore:  0.75,
		DiscoveredAt:   time.Now(),
		Tags:           []string{"trending"},
	}

	if token.Symbol != "TEST" {
		t.Errorf("expected symbol TEST, got %s", token.Symbol)
	}
	if token.Decimals != 18 {
		t.Errorf("expected decimals 18, got %d", token.Decimals)
	}
	if len(token.Tags) != 1 || token.Tags[0] != "trending" {
		t.Errorf("expected tags [trending], got %v", token.Tags)
	}
}

func TestPortfolioFields(t *testing.T) {
	portfolio := Portfolio{
		ID:            "test-id",
		Balance:       10.0,
		Currency:      "EUR",
		ETHBalance:    10.0,
		TokenBalances: make(map[string]TokenBalance),
		TotalValue:    10.0,
		ProfitLoss:    0.0,
		UpdatedAt:     time.Now(),
	}

	if portfolio.Currency != "EUR" {
		t.Errorf("expected currency EUR, got %s", portfolio.Currency)
	}
	if len(portfolio.TokenBalances) != 0 {
		t.Errorf("expected empty token balances, got %d", len(portfolio.TokenBalances))
	}
}

func TestTokenBalance(t *testing.T) {
	tb := TokenBalance{
		Token: Token{
			Address: "0xTEST",
			Symbol:  "TEST",
		},
		Balance:  1000,
		Value:    1.0,
		AvgPrice: 0.001,
	}

	if tb.Balance != 1000 {
		t.Errorf("expected balance 1000, got %f", tb.Balance)
	}
	if tb.Value != 1.0 {
		t.Errorf("expected value 1.0, got %f", tb.Value)
	}
	if tb.AvgPrice != 0.001 {
		t.Errorf("expected avg price 0.001, got %f", tb.AvgPrice)
	}
}

func TestBotStatus(t *testing.T) {
	now := time.Now()
	status := BotStatus{
		IsRunning:           true,
		Mode:                "paper",
		StartedAt:           &now,
		TotalTrades:         5,
		ProfitableTrades:    3,
		TotalProfitLoss:     2.5,
		CurrentBalance:      12.5,
		ActiveOpportunities: 2,
	}

	if !status.IsRunning {
		t.Error("expected is_running true")
	}
	if status.Mode != "paper" {
		t.Errorf("expected mode paper, got %s", status.Mode)
	}
	if status.TotalTrades != 5 {
		t.Errorf("expected total trades 5, got %d", status.TotalTrades)
	}
}

func TestMetrics(t *testing.T) {
	metrics := Metrics{
		TotalTrades:      10,
		SuccessfulTrades: 7,
		FailedTrades:     2,
		TotalVolume:      100.0,
		TotalProfitLoss:  5.0,
		WinRate:          0.7,
	}

	if metrics.WinRate != 0.7 {
		t.Errorf("expected win rate 0.7, got %f", metrics.WinRate)
	}
}

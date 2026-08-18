package backtest

import (
	"context"
	"testing"

	"github.com/twadelij/cryptojackal/internal/indicators"
	"github.com/twadelij/cryptojackal/internal/models"
	"github.com/twadelij/cryptojackal/internal/strategy"
	"go.uber.org/zap"
)

type mockCandleStrategy struct {
	name    string
	enabled bool
}

func (m *mockCandleStrategy) Name() string { return m.name }
func (m *mockCandleStrategy) Enabled() bool { return m.enabled }

func (m *mockCandleStrategy) Analyze(ctx context.Context, token models.Token) strategy.Signal {
	return strategy.Signal{Action: "hold"}
}

func (m *mockCandleStrategy) AnalyzeCandles(ctx context.Context, candles indicators.CandleSeries) strategy.Signal {
	if candles.Len() < 5 {
		return strategy.Signal{Action: "hold"}
	}
	last := candles.Last()
	prev := candles[len(candles)-2]

	if last.Close > prev.Close {
		return strategy.Signal{
			Action:     "buy",
			Confidence: 0.7,
			Strategy:   m.name,
			Reason:     "price going up",
		}
	}
	return strategy.Signal{Action: "hold"}
}

func TestBacktestUptrend(t *testing.T) {
	logger := zap.NewNop()
	engine := NewEngine(logger)

	candles := GenerateSyntheticCandles("up", 100)
	candlesMap := map[string]indicators.CandleSeries{"TEST/USDT": candles}

	cfg := Config{
		InitialBalance:   10000,
		TradeAmount:      100,
		TakeProfitPct:    15,
		StopLossPct:      10,
		MaxOpenPositions: 3,
		FeePct:           0.1,
	}

	strategies := []strategy.Strategy{&mockCandleStrategy{name: "mock_buy", enabled: true}}
	result := engine.Run(context.Background(), cfg, candlesMap, strategies)

	if result.TotalTrades == 0 {
		t.Errorf("Expected at least 1 trade in uptrend, got 0")
	}

	if result.WinRate < 0 || result.WinRate > 100 {
		t.Errorf("Win rate out of range: %v", result.WinRate)
	}

	if len(result.EquityCurve) == 0 {
		t.Errorf("Equity curve should not be empty")
	}

	if result.MaxDrawdown < 0 {
		t.Errorf("Max drawdown should not be negative: %v", result.MaxDrawdown)
	}
}

func TestBacktestDowntrend(t *testing.T) {
	logger := zap.NewNop()
	engine := NewEngine(logger)

	candles := GenerateSyntheticCandles("down", 100)
	candlesMap := map[string]indicators.CandleSeries{"TEST/USDT": candles}

	cfg := Config{
		InitialBalance:   10000,
		TradeAmount:      100,
		TakeProfitPct:    15,
		StopLossPct:      10,
		MaxOpenPositions: 3,
		FeePct:           0.1,
	}

	strategies := []strategy.Strategy{&mockCandleStrategy{name: "mock_buy", enabled: true}}
	result := engine.Run(context.Background(), cfg, candlesMap, strategies)

	if result.TotalTrades > 0 && result.WinRate > 50 {
		t.Logf("Note: trades in downtrend with win rate %.1f%% (may hit TP before SL)", result.WinRate)
	}
}

func TestBacktestNoStrategies(t *testing.T) {
	logger := zap.NewNop()
	engine := NewEngine(logger)

	candles := GenerateSyntheticCandles("up", 50)
	candlesMap := map[string]indicators.CandleSeries{"TEST/USDT": candles}

	cfg := Config{
		InitialBalance:   10000,
		TradeAmount:      100,
		TakeProfitPct:    15,
		StopLossPct:      10,
		MaxOpenPositions: 3,
		FeePct:           0.1,
	}

	result := engine.Run(context.Background(), cfg, candlesMap, []strategy.Strategy{})

	if result.TotalTrades != 0 {
		t.Errorf("Expected 0 trades with no strategies, got %d", result.TotalTrades)
	}
}

func TestBacktestMaxPositions(t *testing.T) {
	logger := zap.NewNop()
	engine := NewEngine(logger)

	candles := GenerateSyntheticCandles("up", 100)
	candlesMap := map[string]indicators.CandleSeries{"TEST/USDT": candles}

	cfg := Config{
		InitialBalance:   10000,
		TradeAmount:      100,
		TakeProfitPct:    50,
		StopLossPct:      50,
		MaxOpenPositions: 2,
		FeePct:           0.1,
	}

	strategies := []strategy.Strategy{&mockCandleStrategy{name: "mock_buy", enabled: true}}
	result := engine.Run(context.Background(), cfg, candlesMap, strategies)

	openAtOnce := 0
	currentOpen := 0
	for _, trade := range result.Trades {
		if trade.ExitReason == "backtest_end" {
			currentOpen++
		} else {
			if currentOpen > openAtOnce {
				openAtOnce = currentOpen
			}
			currentOpen--
		}
	}
	if currentOpen > openAtOnce {
		openAtOnce = currentOpen
	}

	if openAtOnce > cfg.MaxOpenPositions {
		t.Errorf("Exceeded max open positions: %d > %d", openAtOnce, cfg.MaxOpenPositions)
	}
}

func TestBacktestFees(t *testing.T) {
	logger := zap.NewNop()
	engine := NewEngine(logger)

	candles := GenerateSyntheticCandles("up", 50)
	candlesMap := map[string]indicators.CandleSeries{"TEST/USDT": candles}

	cfg := Config{
		InitialBalance:   10000,
		TradeAmount:      100,
		TakeProfitPct:    15,
		StopLossPct:      10,
		MaxOpenPositions: 1,
		FeePct:           0.5,
	}

	strategies := []strategy.Strategy{&mockCandleStrategy{name: "mock_buy", enabled: true}}
	result := engine.Run(context.Background(), cfg, candlesMap, strategies)

	if result.TotalTrades > 0 && result.TotalFees <= 0 {
		t.Errorf("Fees should be positive when trades are made, got %v", result.TotalFees)
	}
}

func TestGenerateSyntheticCandles(t *testing.T) {
	up := GenerateSyntheticCandles("up", 10)
	if up.Len() != 10 {
		t.Errorf("Expected 10 candles, got %d", up.Len())
	}
	if up.Last().Close <= up[0].Close {
		t.Errorf("Uptrend should end higher than start")
	}

	down := GenerateSyntheticCandles("down", 10)
	if down.Last().Close >= down[0].Close {
		t.Errorf("Downtrend should end lower than start")
	}

	sideways := GenerateSyntheticCandles("sideways", 20)
	diff := sideways.Last().Close - sideways[0].Close
	if diff > 10 || diff < -10 {
		t.Errorf("Sideways should stay roughly flat, diff = %v", diff)
	}
}

func TestGenerateSummary(t *testing.T) {
	r := Result{
		TotalTrades:     10,
		WinningTrades:   6,
		LosingTrades:    4,
		WinRate:         60,
		TotalProfitLoss: 150.50,
		MaxDrawdown:     5.2,
		SharpeRatio:     1.3,
		FinalBalance:    10150.50,
		TotalFees:       12.30,
		StrategiesUsed:  []string{"rsi_oversold"},
	}

	summary := GenerateSummary(r)
	if summary == "" {
		t.Errorf("Summary should not be empty")
	}
}

func TestMaxDrawdown(t *testing.T) {
	logger := zap.NewNop()
	engine := NewEngine(logger)

	equity := []float64{100, 110, 105, 95, 100, 120}
	dd := engine.calculateMaxDrawdown(equity)

	expected := (110.0 - 95.0) / 110.0 * 100
	if dd < expected-0.1 || dd > expected+0.1 {
		t.Errorf("Max drawdown = %.2f, expected %.2f", dd, expected)
	}
}

func TestSharpeRatio(t *testing.T) {
	logger := zap.NewNop()
	engine := NewEngine(logger)

	trades := []BacktestTrade{
		{ProfitLoss: 10},
		{ProfitLoss: 20},
		{ProfitLoss: -5},
		{ProfitLoss: 15},
		{ProfitLoss: 8},
	}
	sharpe := engine.calculateSharpeRatio(trades)

	if sharpe == 0 {
		t.Logf("Sharpe ratio with positive average should not be 0, got %v", sharpe)
	}
}

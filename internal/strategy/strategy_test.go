package strategy

import (
	"context"
	"testing"

	"github.com/twadelij/cryptojackal/internal/models"
)

func TestMomentumStrategy_BuySignal(t *testing.T) {
	s := NewMomentumStrategy(15, 100000, 50000)
	token := models.Token{
		Symbol:         "TEST",
		PriceChange24h: 25.5,
		Volume24h:      200000,
		Liquidity:      100000,
	}

	sig := s.Analyze(context.Background(), token)
	if sig.Action != "buy" {
		t.Errorf("expected buy, got %s", sig.Action)
	}
	if sig.Confidence < 0.5 {
		t.Errorf("expected confidence >= 0.5, got %.2f", sig.Confidence)
	}
	if sig.Strategy != "momentum" {
		t.Errorf("expected strategy momentum, got %s", sig.Strategy)
	}
}

func TestMomentumStrategy_HoldOnLowChange(t *testing.T) {
	s := NewMomentumStrategy(15, 100000, 50000)
	token := models.Token{
		Symbol:         "TEST",
		PriceChange24h: 5.0,
		Volume24h:      200000,
		Liquidity:      100000,
	}

	sig := s.Analyze(context.Background(), token)
	if sig.Action != "hold" {
		t.Errorf("expected hold, got %s", sig.Action)
	}
}

func TestDipBuyStrategy_BuySignal(t *testing.T) {
	s := NewDipBuyStrategy(-15, 50000, 50000)
	token := models.Token{
		Symbol:         "TEST",
		PriceChange24h: -20.0,
		Volume24h:      100000,
		Liquidity:      100000,
	}

	sig := s.Analyze(context.Background(), token)
	if sig.Action != "buy" {
		t.Errorf("expected buy, got %s", sig.Action)
	}
	if sig.Confidence <= 0.4 {
		t.Errorf("expected confidence > 0.4, got %.2f", sig.Confidence)
	}
}

func TestDipBuyStrategy_HoldOnSmallDip(t *testing.T) {
	s := NewDipBuyStrategy(-15, 50000, 50000)
	token := models.Token{
		Symbol:         "TEST",
		PriceChange24h: -5.0,
		Volume24h:      100000,
		Liquidity:      100000,
	}

	sig := s.Analyze(context.Background(), token)
	if sig.Action != "hold" {
		t.Errorf("expected hold, got %s", sig.Action)
	}
}

func TestVolumeSpikeStrategy_BuySignal(t *testing.T) {
	s := NewVolumeSpikeStrategy(3.0, 50000, 50000)
	token := models.Token{
		Symbol:         "TEST",
		Volume24h:      500000,
		Liquidity:      100000,
		PriceChange24h: 10.0,
	}

	sig := s.Analyze(context.Background(), token)
	if sig.Action != "buy" {
		t.Errorf("expected buy, got %s", sig.Action)
	}
}

func TestVolumeSpikeStrategy_HoldOnLowRatio(t *testing.T) {
	s := NewVolumeSpikeStrategy(3.0, 50000, 50000)
	token := models.Token{
		Symbol:    "TEST",
		Volume24h: 100000,
		Liquidity: 100000,
	}

	sig := s.Analyze(context.Background(), token)
	if sig.Action != "hold" {
		t.Errorf("expected hold, got %s", sig.Action)
	}
}

func TestEngine_AnalyzeTokens(t *testing.T) {
	e := NewEngine(nil)
	e.AddStrategy(NewMomentumStrategy(15, 100000, 50000))
	e.AddStrategy(NewDipBuyStrategy(-15, 50000, 50000))
	e.AddStrategy(NewVolumeSpikeStrategy(3.0, 50000, 50000))

	tokens := []models.Token{
		{Symbol: "HOT", PriceChange24h: 30, Volume24h: 200000, Liquidity: 100000},
		{Symbol: "DIP", PriceChange24h: -20, Volume24h: 100000, Liquidity: 100000},
		{Symbol: "FLAT", PriceChange24h: 1, Volume24h: 50000, Liquidity: 50000},
	}

	signals := e.AnalyzeTokens(context.Background(), tokens)
	if len(signals) < 2 {
		t.Errorf("expected at least 2 signals, got %d", len(signals))
	}

	best := GetBestSignal(signals)
	if best == nil {
		t.Fatal("expected best signal, got nil")
	}
	if best.Action != "buy" {
		t.Errorf("expected best signal to be buy, got %s", best.Action)
	}
}

func TestEngine_ListStrategies(t *testing.T) {
	e := NewEngine(nil)
	e.AddStrategy(NewMomentumStrategy(15, 100000, 50000))
	e.AddStrategy(NewDipBuyStrategy(-15, 50000, 50000))

	names := e.ListStrategies()
	if len(names) != 2 {
		t.Errorf("expected 2 strategies, got %d", len(names))
	}
}

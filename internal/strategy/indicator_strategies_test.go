package strategy

import (
	"context"
	"testing"
	"time"

	"github.com/twadelij/cryptojackal/internal/indicators"
)

func makeTestCandles(closes []float64) indicators.CandleSeries {
	candles := make(indicators.CandleSeries, len(closes))
	for i, c := range closes {
		candles[i] = indicators.Candle{
			Open:   c,
			High:   c * 1.02,
			Low:    c * 0.98,
			Close:  c,
			Volume: 1000,
			Time:   time.Unix(int64(i*3600), 0),
		}
	}
	return candles
}

func TestRSIOversoldStrategy(t *testing.T) {
	s := NewRSIOversoldStrategy()
	ctx := context.Background()

	closes := make([]float64, 30)
	for i := range closes {
		if i < 20 {
			closes[i] = 100 - float64(i)*2
		} else {
			closes[i] = 60 + float64(i-20)*3
		}
	}
	candles := makeTestCandles(closes)

	sig := s.AnalyzeCandles(ctx, candles)
	if sig.Action != "hold" && sig.Action != "buy" && sig.Action != "sell" {
		t.Errorf("Invalid signal action: %s", sig.Action)
	}
	if sig.Strategy != "rsi_oversold" {
		t.Errorf("Strategy name = %s, expected rsi_oversold", sig.Strategy)
	}

	shortCandles := makeTestCandles([]float64{100, 101, 102})
	sig = s.AnalyzeCandles(ctx, shortCandles)
	if sig.Action != "hold" {
		t.Errorf("Should hold with insufficient data, got %s", sig.Action)
	}
}

func TestRSIOversoldBuySignal(t *testing.T) {
	s := NewRSIOversoldStrategy()
	s.OversoldLevel = 50
	ctx := context.Background()

	closes := make([]float64, 20)
	for i := range closes {
		closes[i] = 100 - float64(i)*3
	}
	candles := makeTestCandles(closes)

	sig := s.AnalyzeCandles(ctx, candles)
	if sig.Action != "buy" {
		t.Errorf("Expected buy signal with strong downtrend and high oversold level, got %s", sig.Action)
	}
	if sig.Confidence <= 0 {
		t.Errorf("Buy confidence should be positive, got %v", sig.Confidence)
	}
}

func TestMACDCrossoverStrategy(t *testing.T) {
	s := NewMACDCrossoverStrategy()
	ctx := context.Background()

	candles := makeTestCandles([]float64{
		100, 102, 104, 106, 108, 110, 112, 114, 116, 118,
		120, 122, 124, 126, 128, 130, 132, 134, 136, 138,
		140, 142, 144, 146, 148, 150, 152, 154, 156, 158,
		160, 162, 164, 166, 168, 170, 172, 174, 176, 178,
	})

	sig := s.AnalyzeCandles(ctx, candles)
	if sig.Strategy != "macd_crossover" {
		t.Errorf("Strategy name = %s, expected macd_crossover", sig.Strategy)
	}

	shortCandles := makeTestCandles([]float64{100, 101, 102})
	sig = s.AnalyzeCandles(ctx, shortCandles)
	if sig.Action != "hold" {
		t.Errorf("Should hold with insufficient data, got %s", sig.Action)
	}
}

func TestMACDBullishCrossover(t *testing.T) {
	s := NewMACDCrossoverStrategy()
	ctx := context.Background()

	closes := make([]float64, 50)
	for i := range closes {
		if i < 30 {
			closes[i] = 100 - float64(i)*0.5
		} else {
			closes[i] = 85 + float64(i-30)*2
		}
	}
	candles := makeTestCandles(closes)

	sig := s.AnalyzeCandles(ctx, candles)
	if sig.Action == "buy" {
		if sig.Confidence <= 0 {
			t.Errorf("Buy confidence should be positive, got %v", sig.Confidence)
		}
	}
}

func TestBollingerBounceStrategy(t *testing.T) {
	s := NewBollingerBounceStrategy()
	ctx := context.Background()

	closes := make([]float64, 30)
	for i := range closes {
		closes[i] = 100 + float64(i%7-3)*2
	}
	candles := makeTestCandles(closes)

	sig := s.AnalyzeCandles(ctx, candles)
	if sig.Strategy != "bollinger_bounce" {
		t.Errorf("Strategy name = %s, expected bollinger_bounce", sig.Strategy)
	}

	shortCandles := makeTestCandles([]float64{100, 101, 102})
	sig = s.AnalyzeCandles(ctx, shortCandles)
	if sig.Action != "hold" {
		t.Errorf("Should hold with insufficient data, got %s", sig.Action)
	}
}

func TestBollingerBounceBuySignal(t *testing.T) {
	s := NewBollingerBounceStrategy()
	ctx := context.Background()

	closes := []float64{
		100, 100, 100, 100, 100, 100, 100, 100, 100, 100,
		100, 100, 100, 100, 100, 100, 100, 100, 100, 100,
		100, 100, 100, 100, 100, 100, 100, 100, 100, 80,
	}
	candles := makeTestCandles(closes)

	sig := s.AnalyzeCandles(ctx, candles)
	if sig.Action != "buy" {
		t.Errorf("Expected buy signal when price drops to lower band, got %s", sig.Action)
	}
}

func TestBollingerBounceSellSignal(t *testing.T) {
	s := NewBollingerBounceStrategy()
	ctx := context.Background()

	closes := []float64{
		100, 100, 100, 100, 100, 100, 100, 100, 100, 100,
		100, 100, 100, 100, 100, 100, 100, 100, 100, 100,
		100, 100, 100, 100, 100, 100, 100, 100, 100, 120,
	}
	candles := makeTestCandles(closes)

	sig := s.AnalyzeCandles(ctx, candles)
	if sig.Action != "sell" {
		t.Errorf("Expected sell signal when price rises to upper band, got %s", sig.Action)
	}
}

func TestIndicatorStrategiesImplementCandleStrategy(t *testing.T) {
	var _ CandleStrategy = (*RSIOversoldStrategy)(nil)
	var _ CandleStrategy = (*MACDCrossoverStrategy)(nil)
	var _ CandleStrategy = (*BollingerBounceStrategy)(nil)
}

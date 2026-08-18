package strategy

import (
	"context"
	"fmt"

	"github.com/twadelij/cryptojackal/internal/indicators"
	"github.com/twadelij/cryptojackal/internal/models"
)

// MACDCrossoverStrategy buys on bullish crossover (MACD line crosses above signal line)
// and sells on bearish crossover (MACD line crosses below signal line)
type MACDCrossoverStrategy struct {
	FastPeriod   int
	SlowPeriod   int
	SignalPeriod int
	enabled      bool
}

// NewMACDCrossoverStrategy creates a new MACD crossover strategy with defaults
func NewMACDCrossoverStrategy() *MACDCrossoverStrategy {
	return &MACDCrossoverStrategy{
		FastPeriod:   12,
		SlowPeriod:   26,
		SignalPeriod: 9,
		enabled:      true,
	}
}

func (s *MACDCrossoverStrategy) Name() string { return "macd_crossover" }
func (s *MACDCrossoverStrategy) Enabled() bool { return s.enabled }

func (s *MACDCrossoverStrategy) Analyze(ctx context.Context, token models.Token) Signal {
	return Signal{
		Token:      token,
		Action:     "hold",
		Confidence: 0,
		Strategy:   s.Name(),
		Reason:     "MACD strategy requires candle data, use AnalyzeCandles for backtesting",
	}
}

func (s *MACDCrossoverStrategy) AnalyzeCandles(ctx context.Context, candles indicators.CandleSeries) Signal {
	minCandles := s.SlowPeriod + s.SignalPeriod
	if candles.Len() < minCandles {
		return Signal{Action: "hold", Strategy: s.Name()}
	}

	result := indicators.MACD(candles, s.FastPeriod, s.SlowPeriod, s.SignalPeriod)

	n := candles.Len()
	macdNow := result.MACDLine[n-1]
	macdPrev := result.MACDLine[n-2]
	signalNow := result.SignalLine[n-1]
	signalPrev := result.SignalLine[n-2]

	if macdNow == 0 || signalNow == 0 {
		return Signal{Action: "hold", Strategy: s.Name()}
	}

	bullishCross := macdPrev <= signalPrev && macdNow > signalNow
	bearishCross := macdPrev >= signalPrev && macdNow < signalNow

	if bullishCross {
		histDiff := macdNow - signalNow
		confidence := 0.6
		if histDiff > 0 {
			confidence += 0.2
		}
		if confidence > 1 {
			confidence = 1
		}
		return Signal{
			Action:     "buy",
			Confidence: confidence,
			Strategy:   s.Name(),
			Reason:     fmt.Sprintf("MACD bullish crossover: MACD %.4f crossed above signal %.4f", macdNow, signalNow),
		}
	}

	if bearishCross {
		histDiff := signalNow - macdNow
		confidence := 0.6
		if histDiff > 0 {
			confidence += 0.2
		}
		if confidence > 1 {
			confidence = 1
		}
		return Signal{
			Action:     "sell",
			Confidence: confidence,
			Strategy:   s.Name(),
			Reason:     fmt.Sprintf("MACD bearish crossover: MACD %.4f crossed below signal %.4f", macdNow, signalNow),
		}
	}

	return Signal{Action: "hold", Strategy: s.Name()}
}

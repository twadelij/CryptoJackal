package strategy

import (
	"context"
	"fmt"

	"github.com/twadelij/cryptojackal/internal/indicators"
	"github.com/twadelij/cryptojackal/internal/models"
)

// RSIOversoldStrategy buys when RSI is below oversoldLevel and sells when RSI is above overboughtLevel
type RSIOversoldStrategy struct {
	Period          int
	OversoldLevel   float64
	OverboughtLevel float64
	enabled         bool
}

// NewRSIOversoldStrategy creates a new RSI oversold strategy with defaults
func NewRSIOversoldStrategy() *RSIOversoldStrategy {
	return &RSIOversoldStrategy{
		Period:          14,
		OversoldLevel:   30,
		OverboughtLevel: 70,
		enabled:         true,
	}
}

func (s *RSIOversoldStrategy) Name() string  { return "rsi_oversold" }
func (s *RSIOversoldStrategy) Enabled() bool { return s.enabled }

func (s *RSIOversoldStrategy) Analyze(ctx context.Context, token models.Token) Signal {
	return Signal{
		Token:      token,
		Action:     "hold",
		Confidence: 0,
		Strategy:   s.Name(),
		Reason:     "RSI strategy requires candle data, use AnalyzeCandles for backtesting",
	}
}

func (s *RSIOversoldStrategy) AnalyzeCandles(ctx context.Context, candles indicators.CandleSeries) Signal {
	if candles.Len() < s.Period+1 {
		return Signal{Action: "hold", Strategy: s.Name()}
	}

	rsi := indicators.RSI(candles, s.Period)
	lastRSI := rsi[candles.Len()-1]

	if candles.Len() <= s.Period {
		return Signal{Action: "hold", Strategy: s.Name()}
	}

	if lastRSI < s.OversoldLevel {
		confidence := (s.OversoldLevel - lastRSI) / s.OversoldLevel
		if confidence > 1 {
			confidence = 1
		}
		return Signal{
			Action:     "buy",
			Confidence: 0.5 + confidence*0.5,
			Strategy:   s.Name(),
			Reason:     fmt.Sprintf("RSI %.1f is oversold (below %.0f)", lastRSI, s.OversoldLevel),
		}
	}

	if lastRSI > s.OverboughtLevel {
		confidence := (lastRSI - s.OverboughtLevel) / (100 - s.OverboughtLevel)
		if confidence > 1 {
			confidence = 1
		}
		return Signal{
			Action:     "sell",
			Confidence: 0.5 + confidence*0.5,
			Strategy:   s.Name(),
			Reason:     fmt.Sprintf("RSI %.1f is overbought (above %.0f)", lastRSI, s.OverboughtLevel),
		}
	}

	return Signal{Action: "hold", Strategy: s.Name()}
}

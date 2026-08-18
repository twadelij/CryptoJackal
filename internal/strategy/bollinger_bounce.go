package strategy

import (
	"context"
	"fmt"

	"github.com/twadelij/cryptojackal/internal/indicators"
	"github.com/twadelij/cryptojackal/internal/models"
)

// BollingerBounceStrategy buys when price touches the lower band (mean reversion)
// and sells when price touches the upper band
type BollingerBounceStrategy struct {
	Period  int
	StdDev  float64
	enabled bool
}

// NewBollingerBounceStrategy creates a new Bollinger Bounce strategy with defaults
func NewBollingerBounceStrategy() *BollingerBounceStrategy {
	return &BollingerBounceStrategy{
		Period:  20,
		StdDev:  2.0,
		enabled: true,
	}
}

func (s *BollingerBounceStrategy) Name() string { return "bollinger_bounce" }
func (s *BollingerBounceStrategy) Enabled() bool { return s.enabled }

func (s *BollingerBounceStrategy) Analyze(ctx context.Context, token models.Token) Signal {
	return Signal{
		Token:      token,
		Action:     "hold",
		Confidence: 0,
		Strategy:   s.Name(),
		Reason:     "Bollinger strategy requires candle data, use AnalyzeCandles for backtesting",
	}
}

func (s *BollingerBounceStrategy) AnalyzeCandles(ctx context.Context, candles indicators.CandleSeries) Signal {
	if candles.Len() < s.Period {
		return Signal{Action: "hold", Strategy: s.Name()}
	}

	upper, _, lower := indicators.BollingerBands(candles, s.Period, s.StdDev)

	n := candles.Len()
	lastCandle := candles[n-1]
	upperNow := upper[n-1]
	lowerNow := lower[n-1]

	if upperNow == 0 || lowerNow == 0 {
		return Signal{Action: "hold", Strategy: s.Name()}
	}

	bandWidth := upperNow - lowerNow
	if bandWidth == 0 {
		return Signal{Action: "hold", Strategy: s.Name()}
	}

	if lastCandle.Close <= lowerNow {
		deviation := (lowerNow - lastCandle.Close) / bandWidth
		confidence := 0.6 + deviation*0.3
		if confidence > 1 {
			confidence = 1
		}
		return Signal{
			Action:     "buy",
			Confidence: confidence,
			Strategy:   s.Name(),
			Reason:     fmt.Sprintf("Price %.4f touched lower Bollinger band %.4f (oversold)", lastCandle.Close, lowerNow),
		}
	}

	if lastCandle.Close >= upperNow {
		deviation := (lastCandle.Close - upperNow) / bandWidth
		confidence := 0.6 + deviation*0.3
		if confidence > 1 {
			confidence = 1
		}
		return Signal{
			Action:     "sell",
			Confidence: confidence,
			Strategy:   s.Name(),
			Reason:     fmt.Sprintf("Price %.4f touched upper Bollinger band %.4f (overbought)", lastCandle.Close, upperNow),
		}
	}

	return Signal{Action: "hold", Strategy: s.Name()}
}

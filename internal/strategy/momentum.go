package strategy

import (
	"context"
	"fmt"

	"github.com/twadelij/cryptojackal/internal/models"
)

// MomentumStrategy detects momentum breakouts: price change >X% with rising volume
type MomentumStrategy struct {
	MinPriceChange float64
	MinVolume      float64
	MinLiquidity   float64
	EnabledFlag    bool
}

// NewMomentumStrategy creates a new momentum breakout strategy
func NewMomentumStrategy(minPriceChange, minVolume, minLiquidity float64) *MomentumStrategy {
	return &MomentumStrategy{
		MinPriceChange: minPriceChange,
		MinVolume:      minVolume,
		MinLiquidity:   minLiquidity,
		EnabledFlag:    true,
	}
}

func (m *MomentumStrategy) Name() string { return "momentum" }
func (m *MomentumStrategy) Enabled() bool { return m.EnabledFlag }

func (m *MomentumStrategy) Analyze(_ context.Context, token models.Token) Signal {
	if token.PriceChange24h < m.MinPriceChange {
		return Signal{Token: token, Action: "hold", Strategy: m.Name()}
	}
	if token.Volume24h < m.MinVolume {
		return Signal{Token: token, Action: "hold", Strategy: m.Name()}
	}
	if token.Liquidity < m.MinLiquidity {
		return Signal{Token: token, Action: "hold", Strategy: m.Name()}
	}

	confidence := 0.5
	if token.PriceChange24h > m.MinPriceChange*2 {
		confidence += 0.15
	}
	if token.Volume24h > m.MinVolume*2 {
		confidence += 0.1
	}
	if token.Liquidity > m.MinLiquidity*5 {
		confidence += 0.05
	}
	if confidence > 1.0 {
		confidence = 1.0
	}

	return Signal{
		Token:      token,
		Action:     "buy",
		Confidence: confidence,
		Strategy:   m.Name(),
		Reason:     fmt.Sprintf("Momentum breakout: %s +%.1f%% in 24h, volume $%.0f, liquidity $%.0f",
			token.Symbol, token.PriceChange24h, token.Volume24h, token.Liquidity),
	}
}

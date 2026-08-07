package strategy

import (
	"context"
	"fmt"

	"github.com/twadelij/cryptojackal/internal/models"
)

// DipBuyStrategy detects oversold tokens: sharp price drop with healthy fundamentals
type DipBuyStrategy struct {
	MinDipPct    float64 // negative number, e.g. -15 for 15% drop
	MinLiquidity float64
	MinVolume    float64
	EnabledFlag  bool
}

// NewDipBuyStrategy creates a new dip buying strategy
func NewDipBuyStrategy(minDipPct, minLiquidity, minVolume float64) *DipBuyStrategy {
	return &DipBuyStrategy{
		MinDipPct:    minDipPct,
		MinLiquidity: minLiquidity,
		MinVolume:    minVolume,
		EnabledFlag:  true,
	}
}

func (d *DipBuyStrategy) Name() string { return "dip_buy" }
func (d *DipBuyStrategy) Enabled() bool { return d.EnabledFlag }

func (d *DipBuyStrategy) Analyze(_ context.Context, token models.Token) Signal {
	if token.PriceChange24h > d.MinDipPct {
		return Signal{Token: token, Action: "hold", Strategy: d.Name()}
	}
	if token.Liquidity < d.MinLiquidity {
		return Signal{Token: token, Action: "hold", Strategy: d.Name()}
	}
	if token.Volume24h < d.MinVolume {
		return Signal{Token: token, Action: "hold", Strategy: d.Name()}
	}

	confidence := 0.45
	dipDepth := -token.PriceChange24h
	if dipDepth > -d.MinDipPct*2 {
		confidence += 0.15
	}
	if token.Liquidity > d.MinLiquidity*3 {
		confidence += 0.1
	}
	if token.Volume24h > d.MinVolume*2 {
		confidence += 0.1
	}
	if confidence > 1.0 {
		confidence = 1.0
	}

	return Signal{
		Token:      token,
		Action:     "buy",
		Confidence: confidence,
		Strategy:   d.Name(),
		Reason:     fmt.Sprintf("Dip buy: %s %.1f%% drop with $%.0f liquidity, $%.0f volume",
			token.Symbol, token.PriceChange24h, token.Liquidity, token.Volume24h),
	}
}

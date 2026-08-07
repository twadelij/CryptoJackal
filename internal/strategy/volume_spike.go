package strategy

import (
	"context"
	"fmt"

	"github.com/twadelij/cryptojackal/internal/models"
)

// VolumeSpikeStrategy detects unusual volume compared to typical levels
type VolumeSpikeStrategy struct {
	VolumeMultiplier float64 // e.g. 3.0 for 3x average
	MinAbsVolume     float64
	MinLiquidity     float64
	EnabledFlag      bool
}

// NewVolumeSpikeStrategy creates a new volume spike detection strategy
func NewVolumeSpikeStrategy(volumeMultiplier, minAbsVolume, minLiquidity float64) *VolumeSpikeStrategy {
	return &VolumeSpikeStrategy{
		VolumeMultiplier: volumeMultiplier,
		MinAbsVolume:     minAbsVolume,
		MinLiquidity:     minLiquidity,
		EnabledFlag:      true,
	}
}

func (v *VolumeSpikeStrategy) Name() string { return "volume_spike" }
func (v *VolumeSpikeStrategy) Enabled() bool { return v.EnabledFlag }

func (v *VolumeSpikeStrategy) Analyze(_ context.Context, token models.Token) Signal {
	if token.Volume24h < v.MinAbsVolume {
		return Signal{Token: token, Action: "hold", Strategy: v.Name()}
	}
	if token.Liquidity < v.MinLiquidity {
		return Signal{Token: token, Action: "hold", Strategy: v.Name()}
	}

	// Volume-to-liquidity ratio as a proxy for "unusual" volume
	// High volume relative to liquidity means lots of trading activity
	volToLiqRatio := token.Volume24h / token.Liquidity
	if volToLiqRatio < v.VolumeMultiplier {
		return Signal{Token: token, Action: "hold", Strategy: v.Name()}
	}

	confidence := 0.5
	if volToLiqRatio > v.VolumeMultiplier*2 {
		confidence += 0.15
	}
	if token.PriceChange24h > 0 {
		confidence += 0.1
	}
	if token.Liquidity > v.MinLiquidity*3 {
		confidence += 0.05
	}
	if confidence > 1.0 {
		confidence = 1.0
	}

	return Signal{
		Token:      token,
		Action:     "buy",
		Confidence: confidence,
		Strategy:   v.Name(),
		Reason:     fmt.Sprintf("Volume spike: %s vol/liq ratio %.1fx (threshold %.1fx), volume $%.0f",
			token.Symbol, volToLiqRatio, v.VolumeMultiplier, token.Volume24h),
	}
}

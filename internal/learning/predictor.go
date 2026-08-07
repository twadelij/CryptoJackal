package learning

import (
	"math"
	"sync"

	"github.com/twadelij/cryptojackal/internal/journal"
	"go.uber.org/zap"
)

// Predictor implements a simple logistic regression model for trade prediction
type Predictor struct {
	mu          sync.RWMutex
	logger      *zap.Logger
	weights     []float64
	bias        float64
	trained     bool
	sampleCount int
	minSamples  int
}

// NewPredictor creates a new ML predictor
func NewPredictor(logger *zap.Logger, minSamples int) *Predictor {
	return &Predictor{
		logger:     logger,
		weights:    make([]float64, 9), // 9 features
		minSamples: minSamples,
	}
}

// featureVector converts TradeFeatures to a normalized float slice
func featureVector(f journal.TradeFeatures) []float64 {
	return []float64{
		normalize(f.PriceChange24h, -50, 100),
		normalize(f.Volume24h, 0, 1000000),
		normalize(f.Liquidity, 0, 1000000),
		normalize(f.MarketCap, 0, 1000000000),
		f.SecurityScore, // already 0-1
		float64(f.HourOfDay) / 23.0,
		strategyToFloat(f.StrategyType),
		normalize(f.DipDepth, -50, 0),
		normalize(f.VolumeRatio, 0, 20),
	}
}

func normalize(val, min, max float64) float64 {
	if max == min {
		return 0
	}
	v := (val - min) / (max - min)
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func strategyToFloat(s string) float64 {
	switch s {
	case "momentum":
		return 0.33
	case "dip_buy":
		return 0.66
	case "volume_spike":
		return 1.0
	default:
		return 0
	}
}

// sigmoid computes the logistic function
func sigmoid(z float64) float64 {
	return 1.0 / (1.0 + math.Exp(-z))
}

// Predict returns probability of success (0.0-1.0) for given features
func (p *Predictor) Predict(features journal.TradeFeatures) float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.trained || p.sampleCount < p.minSamples {
		// Fallback: use simple heuristic based on security score and volume
		score := 0.4 + features.SecurityScore*0.2
		if features.Volume24h > 100000 {
			score += 0.1
		}
		if features.Liquidity > 50000 {
			score += 0.1
		}
		if score > 1.0 {
			score = 1.0
		}
		return score
	}

	x := featureVector(features)
	z := p.bias
	for i := range x {
		z += p.weights[i] * x[i]
	}
	return sigmoid(z)
}

// Train trains the logistic regression model on completed trade data
func (p *Predictor) Train(samples []journal.TrainingSample) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(samples) < p.minSamples {
		p.logger.Info("not enough samples for training",
			zap.Int("samples", len(samples)),
			zap.Int("min_required", p.minSamples))
		return
	}

	// Gradient descent
	learningRate := 0.01
	epochs := 500

	weights := make([]float64, 9)
	bias := 0.0

	for epoch := 0; epoch < epochs; epoch++ {
		gradW := make([]float64, 9)
		gradB := 0.0

		for _, s := range samples {
			x := featureVector(s.Features)
			z := bias
			for i := range x {
				z += weights[i] * x[i]
			}
			pred := sigmoid(z)
			error := pred - s.Label

			for i := range x {
				gradW[i] += error * x[i]
			}
			gradB += error
		}

		n := float64(len(samples))
		for i := range weights {
			weights[i] -= learningRate * gradW[i] / n
		}
		bias -= learningRate * gradB / n
	}

	p.weights = weights
	p.bias = bias
	p.trained = true
	p.sampleCount = len(samples)

	// Calculate accuracy
	correct := 0
	for _, s := range samples {
		x := featureVector(s.Features)
		z := bias
		for i := range x {
			z += weights[i] * x[i]
		}
		pred := sigmoid(z)
		predicted := 0.0
		if pred >= 0.5 {
			predicted = 1.0
		}
		if predicted == s.Label {
			correct++
		}
	}
	accuracy := float64(correct) / float64(len(samples))

	p.logger.Info("ML model trained",
		zap.Int("samples", len(samples)),
		zap.Float64("accuracy", accuracy),
		zap.Int("epochs", epochs))
}

// IsTrained returns whether the model has been trained
func (p *Predictor) IsTrained() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.trained
}

// GetSampleCount returns the number of samples the model was trained on
func (p *Predictor) GetSampleCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.sampleCount
}

// GetAccuracy returns a simple accuracy metric on the training data
func (p *Predictor) GetAccuracy(samples []journal.TrainingSample) float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.trained || len(samples) == 0 {
		return 0
	}

	correct := 0
	for _, s := range samples {
		x := featureVector(s.Features)
		z := p.bias
		for i := range x {
			z += p.weights[i] * x[i]
		}
		pred := sigmoid(z)
		predicted := 0.0
		if pred >= 0.5 {
			predicted = 1.0
		}
		if predicted == s.Label {
			correct++
		}
	}
	return float64(correct) / float64(len(samples))
}

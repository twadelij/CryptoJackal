package strategy

import (
	"context"
	"sync"

	"github.com/twadelij/cryptojackal/internal/models"
	"go.uber.org/zap"
)

// Signal represents a trading signal from a strategy
type Signal struct {
	Token       models.Token
	Action      string  // "buy", "sell", "hold"
	Confidence  float64 // 0.0-1.0
	Strategy    string  // strategy name
	Reason      string  // human-readable explanation
}

// Strategy interface for all trading strategies
type Strategy interface {
	Analyze(ctx context.Context, token models.Token) Signal
	Name() string
	Enabled() bool
}

// Engine runs multiple strategies and combines their signals
type Engine struct {
	strategies []Strategy
	logger     *zap.Logger
	mu         sync.RWMutex
}

// NewEngine creates a new strategy engine
func NewEngine(logger *zap.Logger) *Engine {
	return &Engine{
		logger: logger,
	}
}

// AddStrategy adds a strategy to the engine
func (e *Engine) AddStrategy(s Strategy) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.strategies = append(e.strategies, s)
}

// AnalyzeToken runs all enabled strategies on a token and returns combined signals
func (e *Engine) AnalyzeToken(ctx context.Context, token models.Token) []Signal {
	e.mu.RLock()
	defer e.mu.RUnlock()

	signals := make([]Signal, 0, len(e.strategies))
	for _, s := range e.strategies {
		if !s.Enabled() {
			continue
		}
		sig := s.Analyze(ctx, token)
		if sig.Action != "hold" && sig.Confidence > 0 {
			signals = append(signals, sig)
		}
	}
	return signals
}

// AnalyzeTokens runs strategies on multiple tokens and returns all buy signals
func (e *Engine) AnalyzeTokens(ctx context.Context, tokens []models.Token) []Signal {
	allSignals := make([]Signal, 0)
	for _, token := range tokens {
		sigs := e.AnalyzeToken(ctx, token)
		allSignals = append(allSignals, sigs...)
	}
	return allSignals
}

// GetBestSignal returns the highest confidence buy signal from a list
func GetBestSignal(signals []Signal) *Signal {
	if len(signals) == 0 {
		return nil
	}
	best := signals[0]
	for _, s := range signals[1:] {
		if s.Confidence > best.Confidence {
			best = s
		}
	}
	return &best
}

// ListStrategies returns names of all registered strategies
func (e *Engine) ListStrategies() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	names := make([]string, 0, len(e.strategies))
	for _, s := range e.strategies {
		names = append(names, s.Name())
	}
	return names
}

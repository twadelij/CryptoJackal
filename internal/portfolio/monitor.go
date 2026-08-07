package portfolio

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/twadelij/cryptojackal/internal/discovery"
	"github.com/twadelij/cryptojackal/internal/models"
	"go.uber.org/zap"
)

// Position represents an open trading position
type Position struct {
	Token         models.Token
	EntryPrice    float64
	Amount        float64
	BuyTime       time.Time
	StopLossPct   float64
	TakeProfitPct float64
	Strategy      string
	MLConfidence  float64
}

// Action represents a sell action for a position
type Action struct {
	Position *Position
	Reason   string
	Type     string // "take_profit", "stop_loss", "trailing_stop"
}

// Monitor tracks open positions and checks them against TP/SL thresholds
type Monitor struct {
	positions      map[string]*Position
	mu             sync.RWMutex
	logger         *zap.Logger
	discovery      *discovery.Service
	takeProfitPct  float64
	stopLossPct    float64
	trailingStop   bool
	highestPrice   map[string]float64 // token address -> highest price seen
}

// NewMonitor creates a new position monitor
func NewMonitor(logger *zap.Logger, discovery *discovery.Service, takeProfitPct, stopLossPct float64, trailingStop bool) *Monitor {
	return &Monitor{
		positions:     make(map[string]*Position),
		highestPrice:  make(map[string]float64),
		logger:        logger,
		discovery:     discovery,
		takeProfitPct: takeProfitPct,
		stopLossPct:   stopLossPct,
		trailingStop:  trailingStop,
	}
}

// AddPosition opens a new position for tracking
func (m *Monitor) AddPosition(token models.Token, entryPrice, amount float64, strategy string, mlConfidence float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	addr := token.Address
	m.positions[addr] = &Position{
		Token:         token,
		EntryPrice:    entryPrice,
		Amount:        amount,
		BuyTime:       time.Now(),
		StopLossPct:   m.stopLossPct,
		TakeProfitPct: m.takeProfitPct,
		Strategy:      strategy,
		MLConfidence:  mlConfidence,
	}
	m.highestPrice[addr] = entryPrice

	m.logger.Info("position opened",
		zap.String("token", token.Symbol),
		zap.Float64("entry_price", entryPrice),
		zap.Float64("amount", amount),
		zap.String("strategy", strategy),
		zap.Float64("tp_pct", m.takeProfitPct),
		zap.Float64("sl_pct", m.stopLossPct))
}

// RemovePosition removes a position from tracking
func (m *Monitor) RemovePosition(address string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.positions, address)
	delete(m.highestPrice, address)
}

// GetPositions returns all open positions
func (m *Monitor) GetPositions() []Position {
	m.mu.RLock()
	defer m.mu.RUnlock()

	positions := make([]Position, 0, len(m.positions))
	for _, p := range m.positions {
		positions = append(positions, *p)
	}
	return positions
}

// GetPosition returns a specific position by token address
func (m *Monitor) GetPosition(address string) *Position {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if p, ok := m.positions[address]; ok {
		return p
	}
	return nil
}

// CheckPositions checks all open positions against TP/SL thresholds
func (m *Monitor) CheckPositions(ctx context.Context) []Action {
	m.mu.Lock()
	defer m.mu.Unlock()

	actions := make([]Action, 0)

	for addr, pos := range m.positions {
		currentPrice, err := m.discovery.GetProviderManager().GetTokenPrice(ctx, "eth", addr)
		if err != nil {
			m.logger.Debug("failed to get current price for position",
				zap.String("token", pos.Token.Symbol),
				zap.Error(err))
			continue
		}

		// Update highest price for trailing stop
		if m.trailingStop {
			if currentPrice > m.highestPrice[addr] {
				m.highestPrice[addr] = currentPrice
			}
		}

		pnlPct := ((currentPrice - pos.EntryPrice) / pos.EntryPrice) * 100

		// Take profit check
		if pnlPct >= pos.TakeProfitPct {
			actions = append(actions, Action{
				Position: pos,
				Reason:   fmt.Sprintf("Take profit: %s +%.1f%% (threshold %.0f%%)", pos.Token.Symbol, pnlPct, pos.TakeProfitPct),
				Type:     "take_profit",
			})
			continue
		}

		// Stop loss check (regular)
		if pnlPct <= -pos.StopLossPct {
			actions = append(actions, Action{
				Position: pos,
				Reason:   fmt.Sprintf("Stop loss: %s %.1f%% (threshold -%.0f%%)", pos.Token.Symbol, pnlPct, pos.StopLossPct),
				Type:     "stop_loss",
			})
			continue
		}

		// Trailing stop check
		if m.trailingStop {
			highest := m.highestPrice[addr]
			if highest > pos.EntryPrice {
				drawdownFromHigh := ((currentPrice - highest) / highest) * 100
				trailingThreshold := -pos.StopLossPct
				if drawdownFromHigh <= trailingThreshold {
					actions = append(actions, Action{
						Position: pos,
						Reason:   fmt.Sprintf("Trailing stop: %s dropped %.1f%% from high $%.8f", pos.Token.Symbol, drawdownFromHigh, highest),
						Type:     "trailing_stop",
					})
				}
			}
		}
	}

	return actions
}

// PositionCount returns the number of open positions
func (m *Monitor) PositionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.positions)
}

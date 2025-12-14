package paper

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/twadelij/cryptojackal/internal/models"
	"go.uber.org/zap"
)

// Service manages paper trading simulation
type Service struct {
	mu             sync.RWMutex
	portfolio      *models.Portfolio
	trades         []models.Trade
	initialBalance float64
	logger         *zap.Logger
}

// NewService creates a new paper trading service
func NewService(initialBalance float64, logger *zap.Logger) *Service {
	return &Service{
		portfolio: &models.Portfolio{
			ID:            uuid.New().String(),
			Balance:       initialBalance,
			Currency:      "EUR",
			ETHBalance:    initialBalance, // Keep for backward compat
			TokenBalances: make(map[string]models.TokenBalance),
			TotalValue:    initialBalance,
			UpdatedAt:     time.Now(),
		},
		trades:         make([]models.Trade, 0),
		initialBalance: initialBalance,
		logger:         logger,
	}
}

// GetPortfolio returns the current portfolio
func (s *Service) GetPortfolio() *models.Portfolio {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Calculate total value
	total := s.portfolio.Balance
	for _, balance := range s.portfolio.TokenBalances {
		total += balance.Value
	}
	s.portfolio.TotalValue = total
	s.portfolio.ETHBalance = s.portfolio.Balance // Keep in sync
	s.portfolio.ProfitLoss = total - s.initialBalance
	if s.initialBalance > 0 {
		s.portfolio.ProfitLossPct = (s.portfolio.ProfitLoss / s.initialBalance) * 100
	}

	return s.portfolio
}

// ExecuteTrade executes a paper trade
func (s *Service) ExecuteTrade(ctx context.Context, token models.Token, tradeType models.TradeType, amount float64) (*models.Trade, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	trade := models.NewTrade(token.Address, token.Symbol, tradeType, amount, token.Price, true)

	switch tradeType {
	case models.TradeTypeBuy:
		cost := amount * token.Price
		if cost > s.portfolio.Balance {
			trade.Status = models.TradeStatusFailed
			return trade, fmt.Errorf("insufficient balance: need %.2f EUR, have %.2f", cost, s.portfolio.Balance)
		}

		s.portfolio.Balance -= cost

		// Update token balance
		existing, ok := s.portfolio.TokenBalances[token.Address]
		if ok {
			newBalance := existing.Balance + amount
			newAvgPrice := (existing.AvgPrice*existing.Balance + token.Price*amount) / newBalance
			s.portfolio.TokenBalances[token.Address] = models.TokenBalance{
				Token:    token,
				Balance:  newBalance,
				Value:    newBalance * token.Price,
				AvgPrice: newAvgPrice,
			}
		} else {
			s.portfolio.TokenBalances[token.Address] = models.TokenBalance{
				Token:    token,
				Balance:  amount,
				Value:    amount * token.Price,
				AvgPrice: token.Price,
			}
		}

		trade.AmountOut = amount
		trade.Status = models.TradeStatusExecuted

	case models.TradeTypeSell:
		existing, ok := s.portfolio.TokenBalances[token.Address]
		if !ok || existing.Balance < amount {
			trade.Status = models.TradeStatusFailed
			return trade, fmt.Errorf("insufficient token balance")
		}

		proceeds := amount * token.Price
		s.portfolio.Balance += proceeds

		newBalance := existing.Balance - amount
		if newBalance < 0.0001 {
			delete(s.portfolio.TokenBalances, token.Address)
		} else {
			s.portfolio.TokenBalances[token.Address] = models.TokenBalance{
				Token:    token,
				Balance:  newBalance,
				Value:    newBalance * token.Price,
				AvgPrice: existing.AvgPrice,
			}
		}

		// Calculate profit/loss
		trade.ProfitLoss = (token.Price - existing.AvgPrice) * amount
		trade.AmountOut = proceeds
		trade.Status = models.TradeStatusExecuted
	}

	trade.ExecutedAt = time.Now()
	s.trades = append(s.trades, *trade)
	s.portfolio.UpdatedAt = time.Now()

	s.logger.Info("paper trade executed",
		zap.String("type", string(tradeType)),
		zap.String("token", token.Symbol),
		zap.Float64("amount", amount),
		zap.Float64("price", token.Price),
	)

	return trade, nil
}

// GetTrades returns all paper trades
func (s *Service) GetTrades() []models.Trade {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.trades
}

// GetTradeHistory returns trades with optional filtering
func (s *Service) GetTradeHistory(limit int) []models.Trade {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.trades) {
		limit = len(s.trades)
	}

	// Return most recent trades first
	result := make([]models.Trade, limit)
	for i := 0; i < limit; i++ {
		result[i] = s.trades[len(s.trades)-1-i]
	}
	return result
}

// GetMetrics returns trading metrics
func (s *Service) GetMetrics() models.Metrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metrics := models.Metrics{
		TotalTrades: len(s.trades),
	}

	var totalProfit float64
	for _, trade := range s.trades {
		if trade.Status == models.TradeStatusExecuted {
			if trade.Type == models.TradeTypeSell {
				metrics.TotalVolume += trade.AmountOut
				if trade.ProfitLoss > 0 {
					metrics.SuccessfulTrades++
				}
				totalProfit += trade.ProfitLoss
			} else {
				metrics.TotalVolume += trade.AmountIn * trade.Price
			}
		} else if trade.Status == models.TradeStatusFailed {
			metrics.FailedTrades++
		}
	}

	metrics.TotalProfitLoss = totalProfit
	if metrics.TotalTrades > 0 {
		metrics.WinRate = float64(metrics.SuccessfulTrades) / float64(metrics.TotalTrades)
		metrics.AverageProfitPerTrade = totalProfit / float64(metrics.TotalTrades)
	}

	return metrics
}

// Reset resets the paper trading portfolio
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.portfolio = &models.Portfolio{
		ID:            uuid.New().String(),
		Balance:       s.initialBalance,
		Currency:      "EUR",
		ETHBalance:    s.initialBalance,
		TokenBalances: make(map[string]models.TokenBalance),
		TotalValue:    s.initialBalance,
		UpdatedAt:     time.Now(),
	}
	s.trades = make([]models.Trade, 0)

	s.logger.Info("paper trading portfolio reset", zap.Float64("balance", s.initialBalance))
}

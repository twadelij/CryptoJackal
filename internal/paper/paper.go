package paper

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/twadelij/cryptojackal/internal/models"
	"github.com/twadelij/cryptojackal/internal/storage"
	"go.uber.org/zap"
)

// Service manages paper trading simulation
type Service struct {
	mu             sync.RWMutex
	portfolio      *models.Portfolio
	trades         []models.Trade
	initialBalance float64
	logger         *zap.Logger
	storage        *storage.Storage
	discovery      DiscoveryProvider
}

// DiscoveryProvider provides current token prices for portfolio valuation
type DiscoveryProvider interface {
	GetTrendingTokens(ctx context.Context) ([]models.Token, error)
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

// NewServiceWithStorage creates a new paper trading service with SQLite persistence
func NewServiceWithStorage(initialBalance float64, logger *zap.Logger, store *storage.Storage) *Service {
	s := &Service{
		initialBalance: initialBalance,
		logger:         logger,
		storage:        store,
	}

	// Try to load existing portfolio from storage
	if store != nil {
		portfolios, err := s.loadAllPortfolios()
		if err != nil {
			logger.Warn("failed to load portfolios from storage", zap.Error(err))
		}
		if len(portfolios) > 0 {
			s.portfolio = portfolios[0]
			logger.Info("loaded portfolio from storage",
				zap.String("id", s.portfolio.ID),
				zap.Float64("balance", s.portfolio.Balance))
		}

		// Load trades
		trades, err := store.GetTrades(0)
		if err != nil {
			logger.Warn("failed to load trades from storage", zap.Error(err))
		} else {
			s.trades = trades
			logger.Info("loaded trades from storage", zap.Int("count", len(trades)))
		}
	}

	if s.portfolio == nil {
		s.portfolio = &models.Portfolio{
			ID:            uuid.New().String(),
			Balance:       initialBalance,
			Currency:      "EUR",
			ETHBalance:    initialBalance,
			TokenBalances: make(map[string]models.TokenBalance),
			TotalValue:    initialBalance,
			UpdatedAt:     time.Now(),
		}
		s.trades = make([]models.Trade, 0)
	}

	// Demo mode: populate with fictitious trades if empty
	if len(s.trades) == 0 {
		s.populateDemoTrades()
	}

	return s
}

// populateDemoTrades creates sample trades for new users to see the app in action
func (s *Service) populateDemoTrades() {
	s.mu.Lock()
	defer s.mu.Unlock()

	demoTokens := []models.Token{
		{Address: "0xDOJI1234567890abcdef", Symbol: "DOJI", Name: "DOJI/Degen", Price: 0.000004},
		{Address: "0xFLOKI1234567890abcdef", Symbol: "FLOKI", Name: "FLOKI/USD", Price: 0.000042},
		{Address: "0xPEPE1234567890abcdef", Symbol: "PEPE", Name: "PEPE/ETH", Price: 0.000001},
	}

	now := time.Now()
	// Buy DOJI at lower price (will show profit)
	buy1 := models.NewTrade(demoTokens[0].Address, demoTokens[0].Symbol, models.TradeTypeBuy, 1000000, 0.000003, true)
	buy1.Status = models.TradeStatusExecuted
	buy1.AmountOut = 1000000
	buy1.ExecutedAt = now.Add(-2 * time.Hour)
	s.trades = append(s.trades, *buy1)
	s.portfolio.TokenBalances[demoTokens[0].Address] = models.TokenBalance{
		Token:    demoTokens[0],
		Balance:  1000000,
		Value:    1000000 * demoTokens[0].Price,
		AvgPrice: 0.000003,
	}
	s.portfolio.Balance -= 1000000 * 0.000003

	// Buy FLOKI (will show loss)
	buy2 := models.NewTrade(demoTokens[1].Address, demoTokens[1].Symbol, models.TradeTypeBuy, 5000, 0.000050, true)
	buy2.Status = models.TradeStatusExecuted
	buy2.AmountOut = 5000
	buy2.ExecutedAt = now.Add(-1 * time.Hour)
	s.trades = append(s.trades, *buy2)
	s.portfolio.TokenBalances[demoTokens[1].Address] = models.TokenBalance{
		Token:    demoTokens[1],
		Balance:  5000,
		Value:    5000 * demoTokens[1].Price,
		AvgPrice: 0.000050,
	}
	s.portfolio.Balance -= 5000 * 0.000050

	// Sell PEPE (small profit)
	buy3 := models.NewTrade(demoTokens[2].Address, demoTokens[2].Symbol, models.TradeTypeBuy, 2000000, 0.0000008, true)
	buy3.Status = models.TradeStatusExecuted
	buy3.AmountOut = 2000000
	buy3.ExecutedAt = now.Add(-3 * time.Hour)
	s.trades = append(s.trades, *buy3)

	sell3 := models.NewTrade(demoTokens[2].Address, demoTokens[2].Symbol, models.TradeTypeSell, 2000000, 0.000001, true)
	sell3.Status = models.TradeStatusExecuted
	sell3.AmountOut = 2000000 * 0.000001
	sell3.ProfitLoss = 2000000 * (0.000001 - 0.0000008)
	sell3.ExecutedAt = now.Add(-30 * time.Minute)
	s.trades = append(s.trades, *sell3)
	s.portfolio.Balance += sell3.AmountOut

	// Recalculate total value
	total := s.portfolio.Balance
	for _, bal := range s.portfolio.TokenBalances {
		total += bal.Value
	}
	s.portfolio.TotalValue = total
	s.portfolio.ProfitLoss = total - s.initialBalance
	if s.initialBalance > 0 {
		s.portfolio.ProfitLossPct = (s.portfolio.ProfitLoss / s.initialBalance) * 100
	}

	// Persist demo trades
	if s.storage != nil {
		for i := range s.trades {
			if err := s.storage.SaveTrade(&s.trades[i]); err != nil {
				s.logger.Warn("failed to save demo trade", zap.Error(err))
			}
		}
		if err := s.storage.SavePortfolio(s.portfolio, s.initialBalance); err != nil {
			s.logger.Warn("failed to save demo portfolio", zap.Error(err))
		}
	}

	s.logger.Info("demo trades populated", zap.Int("count", len(s.trades)))
}

// loadAllPortfolios loads all portfolios from storage (helper for startup)
func (s *Service) loadAllPortfolios() ([]*models.Portfolio, error) {
	if s.storage == nil {
		return nil, nil
	}
	// Since we store only one portfolio, try loading with a known ID.
	// If no portfolio exists yet, return nil.
	if s.portfolio != nil {
		portfolio, _, err := s.storage.LoadPortfolio(s.portfolio.ID)
		if err != nil {
			return nil, err
		}
		if portfolio != nil {
			return []*models.Portfolio{portfolio}, nil
		}
	}
	return nil, nil
}

// SetDiscoveryService injects the discovery service for real-time price lookups
func (s *Service) SetDiscoveryService(d DiscoveryProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.discovery = d
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

// GetPortfolioRealTime returns portfolio with current market prices
func (s *Service) GetPortfolioRealTime(ctx context.Context) *models.Portfolio {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.discovery != nil {
		if tokens, err := s.discovery.GetTrendingTokens(ctx); err == nil {
			priceMap := make(map[string]float64)
			for _, t := range tokens {
				priceMap[t.Address] = t.Price
			}
			for addr, bal := range s.portfolio.TokenBalances {
				if price, ok := priceMap[addr]; ok {
					bal.Value = bal.Balance * price
					bal.Token.Price = price
					s.portfolio.TokenBalances[addr] = bal
				}
			}
		}
	}

	total := s.portfolio.Balance
	for _, balance := range s.portfolio.TokenBalances {
		total += balance.Value
	}
	s.portfolio.TotalValue = total
	s.portfolio.ETHBalance = s.portfolio.Balance
	s.portfolio.ProfitLoss = total - s.initialBalance
	if s.initialBalance > 0 {
		s.portfolio.ProfitLossPct = (s.portfolio.ProfitLoss / s.initialBalance) * 100
	}
	s.portfolio.UpdatedAt = time.Now()

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

	// Persist to storage
	if s.storage != nil {
		if err := s.storage.SaveTrade(trade); err != nil {
			s.logger.Warn("failed to save trade to storage", zap.Error(err))
		}
		if err := s.storage.SavePortfolio(s.portfolio, s.initialBalance); err != nil {
			s.logger.Warn("failed to save portfolio to storage", zap.Error(err))
		}
	}

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

// GetFilteredTrades returns trades filtered by type, status, with pagination
func (s *Service) GetFilteredTrades(tradeType string, status string, limit int, offset int) ([]models.Trade, error) {
	if s.storage != nil {
		return s.storage.GetTradesFiltered(tradeType, status, limit, offset)
	}
	// Fallback: filter in memory
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []models.Trade
	for _, t := range s.trades {
		if tradeType != "" && string(t.Type) != tradeType {
			continue
		}
		if status != "" && string(t.Status) != status {
			continue
		}
		filtered = append(filtered, t)
	}

	// Reverse (newest first) and paginate
	total := len(filtered)
	if offset >= total {
		return []models.Trade{}, nil
	}
	end := offset + limit
	if end > total || limit == 0 {
		end = total
	}
	result := make([]models.Trade, 0, end-offset)
	for i := total - 1 - offset; i >= total-end && i >= 0; i-- {
		result = append(result, filtered[i])
	}
	return result, nil
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

	// Persist reset state
	if s.storage != nil {
		if err := s.storage.SavePortfolio(s.portfolio, s.initialBalance); err != nil {
			s.logger.Warn("failed to save reset portfolio to storage", zap.Error(err))
		}
	}

	s.logger.Info("paper trading portfolio reset", zap.Float64("balance", s.initialBalance))
}

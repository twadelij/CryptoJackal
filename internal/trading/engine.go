package trading

import (
	"context"
	"sync"
	"time"

	"github.com/twadelij/cryptojackal/internal/config"
	"github.com/twadelij/cryptojackal/internal/discovery"
	"github.com/twadelij/cryptojackal/internal/models"
	"github.com/twadelij/cryptojackal/internal/paper"
	"github.com/twadelij/cryptojackal/internal/wallet"
	"go.uber.org/zap"
)

// Engine is the main trading engine
type Engine struct {
	config       *config.Config
	wallet       *wallet.Wallet
	discovery    *discovery.Service
	paper        *paper.Service
	logger       *zap.Logger

	mu           sync.RWMutex
	isRunning    bool
	startedAt    *time.Time
	stopChan     chan struct{}
	
	// Stats
	totalTrades      int
	profitableTrades int
	totalProfitLoss  float64
	opportunities    []models.TradingOpportunity
}

// NewEngine creates a new trading engine
func NewEngine(cfg *config.Config, w *wallet.Wallet, disc *discovery.Service, paperSvc *paper.Service, logger *zap.Logger) *Engine {
	return &Engine{
		config:    cfg,
		wallet:    w,
		discovery: disc,
		paper:     paperSvc,
		logger:    logger,
		stopChan:  make(chan struct{}),
	}
}

// Start starts the trading engine
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.isRunning {
		e.mu.Unlock()
		return nil
	}
	e.isRunning = true
	now := time.Now()
	e.startedAt = &now
	e.stopChan = make(chan struct{})
	e.mu.Unlock()

	e.logger.Info("trading engine started",
		zap.Bool("paper_mode", e.config.PaperTradingMode),
		zap.Duration("scan_interval", e.config.ScanInterval),
	)

	go e.runLoop(ctx)
	return nil
}

// Stop stops the trading engine
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.isRunning {
		return
	}

	close(e.stopChan)
	e.isRunning = false
	e.logger.Info("trading engine stopped")
}

// IsRunning returns whether the engine is running
func (e *Engine) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.isRunning
}

// GetStatus returns the current bot status
func (e *Engine) GetStatus() models.BotStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	mode := "paper"
	if !e.config.PaperTradingMode {
		mode = "live"
	}

	var balance float64
	if e.config.PaperTradingMode {
		balance = e.paper.GetPortfolio().ETHBalance
	} else if e.wallet != nil && e.wallet.IsConfigured() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		balance, _ = e.wallet.GetBalanceETH(ctx)
	}

	return models.BotStatus{
		IsRunning:           e.isRunning,
		Mode:                mode,
		StartedAt:           e.startedAt,
		TotalTrades:         e.totalTrades,
		ProfitableTrades:    e.profitableTrades,
		TotalProfitLoss:     e.totalProfitLoss,
		CurrentBalance:      balance,
		ActiveOpportunities: len(e.opportunities),
	}
}

// GetOpportunities returns current trading opportunities
func (e *Engine) GetOpportunities() []models.TradingOpportunity {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.opportunities
}

// ExecuteTrade manually executes a trade
func (e *Engine) ExecuteTrade(ctx context.Context, opportunity models.TradingOpportunity, amount float64) (*models.Trade, error) {
	if e.config.PaperTradingMode {
		return e.paper.ExecuteTrade(ctx, opportunity.Token, models.TradeTypeBuy, amount)
	}
	// TODO: Implement live trading
	return nil, nil
}

func (e *Engine) runLoop(ctx context.Context) {
	ticker := time.NewTicker(e.config.ScanInterval)
	defer ticker.Stop()

	// Initial scan
	e.scan(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopChan:
			return
		case <-ticker.C:
			e.scan(ctx)
		}
	}
}

func (e *Engine) scan(ctx context.Context) {
	e.logger.Debug("scanning for opportunities")

	opportunities, err := e.discovery.FindOpportunities(ctx, "ethereum", e.config.MinLiquidity)
	if err != nil {
		e.logger.Error("failed to find opportunities", zap.Error(err))
		return
	}

	e.mu.Lock()
	e.opportunities = opportunities
	e.mu.Unlock()

	if len(opportunities) > 0 {
		e.logger.Info("found opportunities", zap.Int("count", len(opportunities)))
		
		// Auto-execute in paper mode if enabled
		if e.config.PaperTradingMode && len(opportunities) > 0 {
			// Execute top opportunity
			opp := opportunities[0]
			if opp.ConfidenceScore > 0.6 {
				trade, err := e.paper.ExecuteTrade(ctx, opp.Token, models.TradeTypeBuy, e.config.TradeAmount)
				if err != nil {
					e.logger.Error("paper trade failed", zap.Error(err))
				} else {
					e.mu.Lock()
					e.totalTrades++
					e.mu.Unlock()
					e.logger.Info("auto paper trade executed",
						zap.String("token", trade.TokenSymbol),
						zap.Float64("amount", trade.AmountIn),
					)
				}
			}
		}
	}
}

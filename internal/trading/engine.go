package trading

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/twadelij/cryptojackal/internal/config"
	"github.com/twadelij/cryptojackal/internal/discovery"
	"github.com/twadelij/cryptojackal/internal/journal"
	"github.com/twadelij/cryptojackal/internal/learning"
	"github.com/twadelij/cryptojackal/internal/models"
	"github.com/twadelij/cryptojackal/internal/paper"
	"github.com/twadelij/cryptojackal/internal/portfolio"
	"github.com/twadelij/cryptojackal/internal/strategy"
	"github.com/twadelij/cryptojackal/internal/wallet"
	"go.uber.org/zap"
)

// Engine is the main trading engine
type Engine struct {
	config    *config.Config
	wallet    *wallet.Wallet
	discovery *discovery.Service
	paper     *paper.Service
	logger    *zap.Logger

	strategyEngine  *strategy.Engine
	positionMonitor *portfolio.Monitor
	journal         *journal.Journal
	predictor       *learning.Predictor

	mu        sync.RWMutex
	isRunning bool
	startedAt *time.Time
	stopChan  chan struct{}

	// Stats
	totalTrades      int
	profitableTrades int
	totalProfitLoss  float64
	opportunities    []models.TradingOpportunity

	// Safety rails
	lastTradeTime       time.Time
	dailyStartValue     float64
	dailyLoss           float64
	killSwitchTriggered bool
}

// NewEngine creates a new trading engine
func NewEngine(cfg *config.Config, w *wallet.Wallet, disc *discovery.Service, paperSvc *paper.Service, logger *zap.Logger) *Engine {
	// Initialize strategy engine with default parameters
	se := strategy.NewEngine(logger)
	se.AddStrategy(strategy.NewMomentumStrategy(15, 100000, 50000))
	se.AddStrategy(strategy.NewDipBuyStrategy(-15, 50000, 50000))
	se.AddStrategy(strategy.NewVolumeSpikeStrategy(3.0, 50000, 50000))

	// Initialize position monitor with TP/SL
	pm := portfolio.NewMonitor(logger, disc, 15, 10, true)

	// Initialize journal and ML predictor
	j := journal.New()
	pred := learning.NewPredictor(logger, 20)

	return &Engine{
		config:          cfg,
		wallet:          w,
		discovery:       disc,
		paper:           paperSvc,
		logger:          logger,
		strategyEngine:  se,
		positionMonitor: pm,
		journal:         j,
		predictor:       pred,
		stopChan:        make(chan struct{}),
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
	// Reset daily tracking on start
	if e.config.PaperTradingMode {
		e.dailyStartValue = e.paper.GetPortfolio().TotalValue
	} else if e.wallet != nil && e.wallet.IsConfigured() {
		bal, _ := e.wallet.GetBalanceETH(ctx)
		e.dailyStartValue = bal
	}
	e.dailyLoss = 0
	e.killSwitchTriggered = false
	e.mu.Unlock()

	e.logger.Info("trading engine started",
		zap.Bool("paper_mode", e.config.PaperTradingMode),
		zap.Duration("scan_interval", e.config.ScanInterval),
		zap.Float64("max_daily_loss_pct", e.config.MaxDailyLossPct),
		zap.Float64("max_trade_size_pct", e.config.MaxTradeSizePct),
		zap.Duration("trade_cooldown", e.config.TradeCooldown),
		zap.Int("max_open_positions", e.config.MaxOpenPositions),
	)

	go e.runLoop(ctx)
	go e.monitorLoop(ctx)
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
		KillSwitchTriggered: e.killSwitchTriggered,
		DailyLoss:           e.dailyLoss,
	}
}

// GetOpportunities returns current trading opportunities
func (e *Engine) GetOpportunities() []models.TradingOpportunity {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.opportunities
}

// ExecuteTrade manually executes a trade with safety rail checks
func (e *Engine) ExecuteTrade(ctx context.Context, opportunity models.TradingOpportunity, amount float64) (*models.Trade, error) {
	if err := e.checkSafetyRails(amount); err != nil {
		return nil, err
	}

	if e.config.PaperTradingMode {
		trade, err := e.paper.ExecuteTrade(ctx, opportunity.Token, models.TradeTypeBuy, amount)
		if err == nil && trade != nil {
			e.recordTradeResult(trade)
		}
		return trade, err
	}
	// TODO: Implement live trading with safety rails
	return nil, fmt.Errorf("live trading not yet implemented")
}

// checkSafetyRails validates trade against safety limits
func (e *Engine) checkSafetyRails(amount float64) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.killSwitchTriggered {
		return fmt.Errorf("kill switch triggered: bot stopped due to excessive daily loss")
	}

	// Cooldown check
	if !e.lastTradeTime.IsZero() && time.Since(e.lastTradeTime) < e.config.TradeCooldown {
		return fmt.Errorf("trade cooldown active: wait %v", e.config.TradeCooldown-time.Since(e.lastTradeTime))
	}

	// Max trade size check
	var portfolioValue float64
	if e.config.PaperTradingMode {
		portfolioValue = e.paper.GetPortfolio().TotalValue
	} else if e.wallet != nil && e.wallet.IsConfigured() {
		portfolioValue, _ = e.wallet.GetBalanceETH(context.Background())
	}
	if portfolioValue > 0 {
		maxTradeSize := portfolioValue * (e.config.MaxTradeSizePct / 100)
		if amount > maxTradeSize {
			return fmt.Errorf("trade amount %.6f exceeds max trade size %.6f (%.1f%% of portfolio)", amount, maxTradeSize, e.config.MaxTradeSizePct)
		}
	}

	// Max open positions check (paper mode only for now)
	if e.config.PaperTradingMode {
		openPositions := 0
		for _, bal := range e.paper.GetPortfolio().TokenBalances {
			if bal.Balance > 0 {
				openPositions++
			}
		}
		if openPositions >= e.config.MaxOpenPositions {
			return fmt.Errorf("max open positions reached: %d", e.config.MaxOpenPositions)
		}
	}

	return nil
}

// recordTradeResult updates safety rail state after a trade
func (e *Engine) recordTradeResult(trade *models.Trade) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.lastTradeTime = time.Now()
	e.totalTrades++

	if trade.Status == models.TradeStatusExecuted {
		if trade.ProfitLoss != 0 {
			e.totalProfitLoss += trade.ProfitLoss
			if trade.ProfitLoss > 0 {
				e.profitableTrades++
			}
		}
		// Track daily loss for kill switch
		if trade.ProfitLoss < 0 {
			e.dailyLoss += -trade.ProfitLoss
			if e.dailyStartValue > 0 {
				lossPct := (e.dailyLoss / e.dailyStartValue) * 100
				if lossPct >= e.config.MaxDailyLossPct && e.config.KillSwitchEnabled {
					e.killSwitchTriggered = true
					e.logger.Warn("KILL SWITCH TRIGGERED",
						zap.Float64("daily_loss_pct", lossPct),
						zap.Float64("max_allowed_pct", e.config.MaxDailyLossPct),
					)
				}
			}
		}
	}
}

// GetStrategyEngine returns the strategy engine
func (e *Engine) GetStrategyEngine() *strategy.Engine {
	return e.strategyEngine
}

// GetPositionMonitor returns the position monitor
func (e *Engine) GetPositionMonitor() *portfolio.Monitor {
	return e.positionMonitor
}

// GetJournal returns the trade journal
func (e *Engine) GetJournal() *journal.Journal {
	return e.journal
}

// GetPredictor returns the ML predictor
func (e *Engine) GetPredictor() *learning.Predictor {
	return e.predictor
}

func (e *Engine) runLoop(ctx context.Context) {
	// Recover from panics so the bot doesn't crash
	defer func() {
		if r := recover(); r != nil {
			e.logger.Error("trading engine panic recovered", zap.Any("panic", r))
			e.mu.Lock()
			e.isRunning = false
			e.mu.Unlock()
		}
	}()

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

	// Get tokens from discovery service (via ProviderManager with failover)
	tokens, err := e.discovery.GetTopGainers(ctx, "ethereum", e.config.MinLiquidity)
	if err != nil {
		e.logger.Warn("failed to get tokens from discovery", zap.Error(err))
		tokens = []models.Token{}
	}

	// Also get new tokens and merge
	newTokens, err := e.discovery.GetNewTokens(ctx, "ethereum")
	if err == nil {
		tokens = mergeTokens(tokens, newTokens)
	}

	// Run strategies on all tokens
	signals := e.strategyEngine.AnalyzeTokens(ctx, tokens)

	// Convert signals to opportunities for backward compatibility
	opportunities := make([]models.TradingOpportunity, 0, len(signals))
	for _, sig := range signals {
		opp := models.NewOpportunity(
			sig.Token,
			sig.Token.PriceChange24h*0.1,
			0.01,
			sig.Confidence,
			sig.Strategy,
		)
		opportunities = append(opportunities, *opp)
	}

	e.mu.Lock()
	e.opportunities = opportunities
	e.mu.Unlock()

	if len(opportunities) > 0 {
		e.logger.Info("found opportunities", zap.Int("count", len(opportunities)))

		// Auto-execute in paper mode: pick best signal
		if e.config.PaperTradingMode {
			best := strategy.GetBestSignal(signals)
			if best != nil && best.Confidence > 0.55 {
				if err := e.checkSafetyRails(e.config.TradeAmount); err != nil {
					e.logger.Warn("safety rail blocked auto-trade", zap.Error(err))
				} else {
					trade, err := e.paper.ExecuteTrade(ctx, best.Token, models.TradeTypeBuy, e.config.TradeAmount)
					if err != nil {
						e.logger.Error("paper trade failed", zap.Error(err))
					} else {
						e.recordTradeResult(trade)
						// Track position for TP/SL monitoring
						e.positionMonitor.AddPosition(best.Token, best.Token.Price, e.config.TradeAmount, best.Strategy, best.Confidence)
						// Record in journal for ML training
						e.journal.RecordBuy(trade.ID, best.Strategy, journal.TradeFeatures{
							PriceChange24h: best.Token.PriceChange24h,
							Volume24h:      best.Token.Volume24h,
							Liquidity:      best.Token.Liquidity,
							MarketCap:      best.Token.MarketCap,
							SecurityScore:  best.Token.SecurityScore,
							HourOfDay:      time.Now().Hour(),
							StrategyType:   best.Strategy,
						})
						e.logger.Info("auto paper trade executed",
							zap.String("token", trade.TokenSymbol),
							zap.Float64("amount", trade.AmountIn),
							zap.String("strategy", best.Strategy),
							zap.Float64("confidence", best.Confidence),
							zap.String("reason", best.Reason),
						)
					}
				}
			}
		}
	}
}

// monitorLoop runs periodically to check open positions for TP/SL
func (e *Engine) monitorLoop(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			e.logger.Error("position monitor panic recovered", zap.Any("panic", r))
		}
	}()

	monitorInterval := 30 * time.Second
	ticker := time.NewTicker(monitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopChan:
			return
		case <-ticker.C:
			e.checkPositions(ctx)
		}
	}
}

// checkPositions checks all open positions and executes sells for TP/SL
func (e *Engine) checkPositions(ctx context.Context) {
	if e.positionMonitor.PositionCount() == 0 {
		return
	}

	actions := e.positionMonitor.CheckPositions(ctx)
	for _, action := range actions {
		if !e.config.PaperTradingMode {
			continue
		}

		trade, err := e.paper.ExecuteTrade(ctx, action.Position.Token, models.TradeTypeSell, action.Position.Amount)
		if err != nil {
			e.logger.Error("failed to execute sell", zap.Error(err), zap.String("token", action.Position.Token.Symbol))
			continue
		}

		e.recordTradeResult(trade)
		e.positionMonitor.RemovePosition(action.Position.Token.Address)

		// Record sell in journal for ML training
		profitPct := 0.0
		if action.Position.EntryPrice > 0 {
			profitPct = ((trade.Price - action.Position.EntryPrice) / action.Position.EntryPrice) * 100
		}
		e.journal.RecordSell(trade.ID, profitPct, time.Since(action.Position.BuyTime))

		// Retrain ML model if we have enough samples
		if e.journal.GetCompletedCount() >= 20 && e.journal.GetCompletedCount()%10 == 0 {
			samples := e.journal.GetTrainingData()
			e.predictor.Train(samples)
		}

		e.logger.Info("position closed",
			zap.String("token", action.Position.Token.Symbol),
			zap.String("reason", action.Reason),
			zap.String("type", action.Type),
			zap.Float64("profit_loss", trade.ProfitLoss),
		)
	}
}

// mergeTokens combines two token lists, deduplicating by address
func mergeTokens(a, b []models.Token) []models.Token {
	seen := make(map[string]bool)
	result := make([]models.Token, 0, len(a)+len(b))
	for _, t := range a {
		if !seen[t.Address] {
			seen[t.Address] = true
			result = append(result, t)
		}
	}
	for _, t := range b {
		if !seen[t.Address] {
			seen[t.Address] = true
			result = append(result, t)
		}
	}
	return result
}

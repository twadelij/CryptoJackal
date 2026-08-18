package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/twadelij/cryptojackal/internal/backtest"
	"github.com/twadelij/cryptojackal/internal/config"
	"github.com/twadelij/cryptojackal/internal/discovery"
	"github.com/twadelij/cryptojackal/internal/indicators"
	"github.com/twadelij/cryptojackal/internal/models"
	"github.com/twadelij/cryptojackal/internal/paper"
	"github.com/twadelij/cryptojackal/internal/storage"
	"github.com/twadelij/cryptojackal/internal/strategy"
	"github.com/twadelij/cryptojackal/internal/trading"
	"go.uber.org/zap"
)

// Handler contains all HTTP handlers
type Handler struct {
	config    *config.Config
	engine    *trading.Engine
	discovery *discovery.Service
	paper     *paper.Service
	storage   *storage.Storage
	logger    *zap.Logger
}

// NewHandler creates a new handler
func NewHandler(cfg *config.Config, engine *trading.Engine, disc *discovery.Service, paperSvc *paper.Service, store *storage.Storage, logger *zap.Logger) *Handler {
	return &Handler{
		config:    cfg,
		engine:    engine,
		discovery: disc,
		paper:     paperSvc,
		storage:   store,
		logger:    logger,
	}
}

// Response is a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Health returns health status
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: gin.H{
			"status":  "healthy",
			"version": "1.0.0",
		},
	})
}

// GetStatus returns bot status
func (h *Handler) GetStatus(c *gin.Context) {
	status := h.engine.GetStatus()
	c.JSON(http.StatusOK, Response{Success: true, Data: status})
}

// StartBot starts the trading bot
func (h *Handler) StartBot(c *gin.Context) {
	if err := h.engine.Start(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, Response{Success: true, Data: "Bot started"})
}

// StopBot stops the trading bot
func (h *Handler) StopBot(c *gin.Context) {
	h.engine.Stop()
	c.JSON(http.StatusOK, Response{Success: true, Data: "Bot stopped"})
}

// GetOpportunities returns current trading opportunities
func (h *Handler) GetOpportunities(c *gin.Context) {
	opportunities := h.engine.GetOpportunities()
	c.JSON(http.StatusOK, Response{Success: true, Data: opportunities})
}

// ExecuteTradeRequest is the request body for executing a trade
type ExecuteTradeRequest struct {
	OpportunityID string  `json:"opportunity_id" binding:"required"`
	Amount        float64 `json:"amount" binding:"required,gt=0"`
}

// ExecuteTrade executes a trade
func (h *Handler) ExecuteTrade(c *gin.Context) {
	var req ExecuteTradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Success: false, Error: err.Error()})
		return
	}

	// Find the opportunity
	opportunities := h.engine.GetOpportunities()
	var opp *models.TradingOpportunity
	for _, o := range opportunities {
		if o.ID == req.OpportunityID {
			opp = &o
			break
		}
	}

	if opp == nil {
		c.JSON(http.StatusNotFound, Response{Success: false, Error: "opportunity not found"})
		return
	}

	trade, err := h.engine.ExecuteTrade(c.Request.Context(), *opp, req.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, Response{Success: true, Data: trade})
}

// GetTradingHistory returns trading history with optional filtering
func (h *Handler) GetTradingHistory(c *gin.Context) {
	tradeType := c.Query("type")
	status := c.Query("status")
	limit := 50
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 && l <= 1000 {
		limit = l
	}
	offset := 0
	if o, err := strconv.Atoi(c.Query("offset")); err == nil && o >= 0 {
		offset = o
	}

	trades, err := h.paper.GetFilteredTrades(tradeType, status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, Response{Success: true, Data: trades})
}

// ExportPaperTrades exports paper trades as JSON or CSV
func (h *Handler) ExportPaperTrades(c *gin.Context) {
	format := c.Query("format")
	if format != "csv" {
		format = "json"
	}

	trades, err := h.paper.GetFilteredTrades("", "", 0, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	if format == "csv" {
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename=trades.csv")
		c.String(http.StatusOK, "ID,Symbol,Type,Amount,Price,ProfitLoss,Status,ExecutedAt\n")
		for _, t := range trades {
			c.String(http.StatusOK, "%s,%s,%s,%.6f,%.6f,%.6f,%s,%s\n",
				t.ID, t.TokenSymbol, t.Type, t.AmountIn, t.Price, t.ProfitLoss, t.Status, t.ExecutedAt.Format(time.RFC3339))
		}
		return
	}

	c.JSON(http.StatusOK, Response{Success: true, Data: trades})
}

// GetTrendingTokens returns trending tokens
func (h *Handler) GetTrendingTokens(c *gin.Context) {
	tokens, err := h.discovery.GetTrendingTokens(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, Response{Success: true, Data: tokens})
}

// GetNewTokens returns newly discovered tokens
func (h *Handler) GetNewTokens(c *gin.Context) {
	chain := c.DefaultQuery("chain", "ethereum")
	// Validate chain parameter to prevent injection
	validChains := map[string]bool{"ethereum": true, "bsc": true, "polygon": true, "arbitrum": true, "base": true}
	if !validChains[chain] {
		c.JSON(http.StatusBadRequest, Response{Success: false, Error: "invalid chain parameter"})
		return
	}
	tokens, err := h.discovery.GetNewTokens(c.Request.Context(), chain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, Response{Success: true, Data: tokens})
}

// AnalyzeToken analyzes a specific token
func (h *Handler) AnalyzeToken(c *gin.Context) {
	address := c.Param("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, Response{Success: false, Error: "address required"})
		return
	}
	// Basic Ethereum address validation (0x + 40 hex chars)
	if len(address) != 42 || address[:2] != "0x" {
		c.JSON(http.StatusBadRequest, Response{Success: false, Error: "invalid Ethereum address format"})
		return
	}

	token, err := h.discovery.AnalyzeToken(c.Request.Context(), address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}
	if token == nil {
		c.JSON(http.StatusNotFound, Response{Success: false, Error: "token not found"})
		return
	}

	c.JSON(http.StatusOK, Response{Success: true, Data: token})
}

// GetPaperBalance returns paper trading balance with real-time prices
func (h *Handler) GetPaperBalance(c *gin.Context) {
	portfolio := h.paper.GetPortfolioRealTime(c.Request.Context())
	c.JSON(http.StatusOK, Response{Success: true, Data: portfolio})
}

// ResetPaperBalance resets paper trading balance
func (h *Handler) ResetPaperBalance(c *gin.Context) {
	h.paper.Reset()
	c.JSON(http.StatusOK, Response{Success: true, Data: "Portfolio reset"})
}

// PaperTradeRequest is the request body for paper trading
type PaperTradeRequest struct {
	TokenAddress string  `json:"token_address" binding:"required"`
	TokenSymbol  string  `json:"token_symbol" binding:"required"`
	TokenName    string  `json:"token_name"`
	Price        float64 `json:"price" binding:"required,gt=0"`
	Amount       float64 `json:"amount" binding:"required,gt=0"`
	Type         string  `json:"type" binding:"required,oneof=buy sell"` // "buy" or "sell"
}

// ExecutePaperTrade executes a paper trade
func (h *Handler) ExecutePaperTrade(c *gin.Context) {
	var req PaperTradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Success: false, Error: err.Error()})
		return
	}

	token := models.Token{
		Address: req.TokenAddress,
		Symbol:  req.TokenSymbol,
		Name:    req.TokenName,
		Price:   req.Price,
	}

	var tradeType models.TradeType
	if req.Type == "buy" {
		tradeType = models.TradeTypeBuy
	} else {
		tradeType = models.TradeTypeSell
	}

	trade, err := h.paper.ExecuteTrade(c.Request.Context(), token, tradeType, req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, Response{Success: true, Data: trade})
}

// GetMetrics returns trading metrics
func (h *Handler) GetMetrics(c *gin.Context) {
	metrics := h.paper.GetMetrics()
	c.JSON(http.StatusOK, Response{Success: true, Data: metrics})
}

// GetExternalHealth returns the health of external APIs (CoinGecko, DexScreener, GeckoTerminal)
func (h *Handler) GetExternalHealth(c *gin.Context) {
	status := h.discovery.Health(c.Request.Context())
	c.JSON(http.StatusOK, Response{Success: true, Data: status})
}

// GetPositions returns all open positions with live P&L
func (h *Handler) GetPositions(c *gin.Context) {
	pm := h.engine.GetPositionMonitor()
	positions := pm.GetPositions()
	c.JSON(http.StatusOK, Response{Success: true, Data: positions})
}

// ClosePosition manually closes a position
func (h *Handler) ClosePosition(c *gin.Context) {
	address := c.Param("id")
	pm := h.engine.GetPositionMonitor()
	pos := pm.GetPosition(address)
	if pos == nil {
		c.JSON(http.StatusNotFound, Response{Success: false, Error: "position not found"})
		return
	}

	trade, err := h.paper.ExecuteTrade(c.Request.Context(), pos.Token, models.TradeTypeSell, pos.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	pm.RemovePosition(address)
	c.JSON(http.StatusOK, Response{Success: true, Data: trade})
}

// GetStrategies returns strategy statistics
func (h *Handler) GetStrategies(c *gin.Context) {
	se := h.engine.GetStrategyEngine()
	names := se.ListStrategies()
	c.JSON(http.StatusOK, Response{Success: true, Data: gin.H{"strategies": names}})
}

// GetMLStatus returns ML model status
func (h *Handler) GetMLStatus(c *gin.Context) {
	pred := h.engine.GetPredictor()
	j := h.engine.GetJournal()
	samples := j.GetTrainingData()
	accuracy := 0.0
	if pred.IsTrained() && len(samples) > 0 {
		accuracy = pred.GetAccuracy(samples)
	}
	c.JSON(http.StatusOK, Response{Success: true, Data: gin.H{
		"trained":     pred.IsTrained(),
		"samples":     pred.GetSampleCount(),
		"completed":   j.GetCompletedCount(),
		"accuracy":    accuracy,
		"min_samples": 20,
	}})
}

// GetDatasourceStatus returns the status of all data providers
func (h *Handler) GetDatasourceStatus(c *gin.Context) {
	pm := h.discovery.GetProviderManager()
	statuses := pm.GetProviderStatus()
	c.JSON(http.StatusOK, Response{Success: true, Data: statuses})
}

// ConfigUpdateRequest is the request body for updating configuration
type ConfigUpdateRequest struct {
	PaperTradingMode bool    `json:"paper_trading_mode"`
	InitialBalance   float64 `json:"initial_balance"`
	EthNodeURL       string  `json:"eth_node_url"`
	TradeAmount      float64 `json:"trade_amount"`
	MaxSlippage      float64 `json:"max_slippage"`
	StopLoss         float64 `json:"stop_loss"`
}

// GetConfig returns the current configuration without sensitive fields
func (h *Handler) GetConfig(c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: gin.H{
			"paper_trading_mode": h.config.PaperTradingMode,
			"initial_balance":    h.config.InitialBalance,
			"trade_amount":       h.config.TradeAmount,
			"max_slippage":       h.config.MaxSlippage,
			"min_liquidity":      h.config.MinLiquidity,
			"max_price_impact":   h.config.MaxPriceImpact,
			"scan_interval_sec":  int(h.config.ScanInterval.Seconds()),
			"gas_limit":          h.config.GasLimit,
			"max_gas_price":      h.config.MaxGasPrice,
			"environment":        h.config.Environment,
			"api_tier":           h.config.APITier,
		},
	})
}

// UpdateConfig updates the runtime configuration
func (h *Handler) UpdateConfig(c *gin.Context) {
	var req ConfigUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Success: false, Error: err.Error()})
		return
	}

	// Validate
	if req.InitialBalance < 0 {
		c.JSON(http.StatusBadRequest, Response{Success: false, Error: "initial balance must be non-negative"})
		return
	}
	if req.TradeAmount <= 0 {
		c.JSON(http.StatusBadRequest, Response{Success: false, Error: "trade amount must be positive"})
		return
	}
	if req.MaxSlippage < 0 || req.MaxSlippage > 100 {
		c.JSON(http.StatusBadRequest, Response{Success: false, Error: "max slippage must be between 0 and 100"})
		return
	}

	// Update config in-place
	h.config.PaperTradingMode = req.PaperTradingMode
	h.config.InitialBalance = req.InitialBalance
	h.config.TradeAmount = req.TradeAmount
	h.config.MaxSlippage = req.MaxSlippage

	if req.EthNodeURL != "" {
		h.config.NodeURL = req.EthNodeURL
	}

	// Persist to storage
	if h.storage != nil {
		if err := h.config.SaveToStorage(h.storage); err != nil {
			h.logger.Warn("failed to persist config", zap.Error(err))
		}
	}

	h.logger.Info("configuration updated",
		zap.Bool("paper_mode", req.PaperTradingMode),
		zap.Float64("initial_balance", req.InitialBalance),
	)

	c.JSON(http.StatusOK, Response{Success: true, Data: "Config updated"})
}

// LoginRequest is the request body for login
// swagger:model LoginRequest
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login handles user authentication and returns a JWT token
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Success: false, Error: err.Error()})
		return
	}

	// For single-user setup, only check password (username is ignored)
	if req.Password != h.config.AdminPassword {
		c.JSON(http.StatusUnauthorized, Response{Success: false, Error: "invalid credentials"})
		return
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "admin",
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	})

	tokenString, err := token.SignedString([]byte(h.config.JWTSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Success: false, Error: "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: gin.H{
			"token": tokenString,
			"type":  "Bearer",
		},
	})
}

// BacktestRequest is the request body for running a backtest
type BacktestRequest struct {
	Pair             string   `json:"pair"`
	Exchange         string   `json:"exchange"`
	Interval         string   `json:"interval"`
	Limit            int      `json:"limit"`
	InitialBalance   float64  `json:"initial_balance"`
	TradeAmount      float64  `json:"trade_amount"`
	TakeProfitPct    float64  `json:"take_profit_pct"`
	StopLossPct      float64  `json:"stop_loss_pct"`
	MaxOpenPositions int      `json:"max_open_positions"`
	FeePct           float64  `json:"fee_pct"`
	Strategies       []string `json:"strategies"`
}

// RunBacktest downloads candle data and runs a backtest
func (h *Handler) RunBacktest(c *gin.Context) {
	var req BacktestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Success: false, Error: err.Error()})
		return
	}

	if req.Pair == "" {
		req.Pair = "BTCUSDT"
	}
	if req.Exchange == "" {
		req.Exchange = "binance"
	}
	if req.Interval == "" {
		req.Interval = "1h"
	}
	if req.Limit <= 0 || req.Limit > 1000 {
		req.Limit = 500
	}
	if req.InitialBalance <= 0 {
		req.InitialBalance = 10000
	}
	if req.TradeAmount <= 0 {
		req.TradeAmount = 100
	}
	if req.TakeProfitPct <= 0 {
		req.TakeProfitPct = 15
	}
	if req.StopLossPct <= 0 {
		req.StopLossPct = 10
	}
	if req.MaxOpenPositions <= 0 {
		req.MaxOpenPositions = 3
	}
	if req.FeePct <= 0 {
		req.FeePct = 0.1
	}

	downloader := backtest.NewDataDownloader()

	var candles indicators.CandleSeries
	var err error

	switch req.Exchange {
	case "binance":
		candles, err = downloader.DownloadFromBinance(req.Pair, req.Interval, req.Limit)
	case "kraken":
		var intervalInt int
		switch req.Interval {
		case "1h":
			intervalInt = 60
		case "4h":
			intervalInt = 240
		case "1d":
			intervalInt = 1440
		default:
			intervalInt = 60
		}
		candles, err = downloader.DownloadFromKraken(req.Pair, intervalInt)
	default:
		candles, err = downloader.DownloadFromBinance(req.Pair, req.Interval, req.Limit)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Success: false, Error: "failed to download candle data: " + err.Error()})
		return
	}

	if candles.Len() < 30 {
		c.JSON(http.StatusBadRequest, Response{Success: false, Error: "insufficient candle data for backtesting (need at least 30 candles)"})
		return
	}

	cfg := backtest.Config{
		InitialBalance:   req.InitialBalance,
		TradeAmount:      req.TradeAmount,
		TakeProfitPct:    req.TakeProfitPct,
		StopLossPct:      req.StopLossPct,
		MaxOpenPositions: req.MaxOpenPositions,
		FeePct:           req.FeePct,
	}

	allStrategies := []strategy.Strategy{
		strategy.NewRSIOversoldStrategy(),
		strategy.NewMACDCrossoverStrategy(),
		strategy.NewBollingerBounceStrategy(),
	}

	var selectedStrategies []strategy.Strategy
	if len(req.Strategies) == 0 {
		selectedStrategies = allStrategies
	} else {
		for _, name := range req.Strategies {
			for _, s := range allStrategies {
				if s.Name() == name {
					selectedStrategies = append(selectedStrategies, s)
					break
				}
			}
		}
	}

	if len(selectedStrategies) == 0 {
		selectedStrategies = allStrategies
	}

	btEngine := backtest.NewEngine(h.logger)
	candlesMap := map[string]indicators.CandleSeries{req.Pair: candles}
	result := btEngine.Run(c.Request.Context(), cfg, candlesMap, selectedStrategies)

	c.JSON(http.StatusOK, Response{Success: true, Data: result})
}

// GetBacktestHistory returns a list of previous backtest results (placeholder)
func (h *Handler) GetBacktestHistory(c *gin.Context) {
	c.JSON(http.StatusOK, Response{Success: true, Data: []interface{}{}})
}

// GetIndicators returns technical indicator values for a given token
func (h *Handler) GetIndicators(c *gin.Context) {
	pair := c.Param("pair")
	if pair == "" {
		pair = "BTCUSDT"
	}

	exchange := c.DefaultQuery("exchange", "binance")
	interval := c.DefaultQuery("interval", "1h")
	limitStr := c.DefaultQuery("limit", "100")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	downloader := backtest.NewDataDownloader()

	var candles indicators.CandleSeries
	var err error

	switch exchange {
	case "binance":
		candles, err = downloader.DownloadFromBinance(pair, interval, limit)
	case "kraken":
		candles, err = downloader.DownloadFromKraken(pair, 60)
	default:
		candles, err = downloader.DownloadFromBinance(pair, interval, limit)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Success: false, Error: "failed to download data: " + err.Error()})
		return
	}

	if candles.Len() < 30 {
		c.JSON(http.StatusBadRequest, Response{Success: false, Error: "insufficient data for indicators"})
		return
	}

	rsi := indicators.RSI(candles, 14)
	macd := indicators.MACD(candles, 12, 26, 9)
	sma20 := indicators.SMA(candles, 20)
	ema12 := indicators.EMA(candles, 12)
	upper, middle, lower := indicators.BollingerBands(candles, 20, 2.0)

	n := candles.Len()
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: gin.H{
			"pair":     pair,
			"interval": interval,
			"candles":  candles.Len(),
			"indicators": gin.H{
				"rsi":         rsi[n-1],
				"macd_line":   macd.MACDLine[n-1],
				"macd_signal": macd.SignalLine[n-1],
				"macd_hist":   macd.Histogram[n-1],
				"sma_20":      sma20[n-1],
				"ema_12":      ema12[n-1],
				"bb_upper":    upper[n-1],
				"bb_middle":   middle[n-1],
				"bb_lower":    lower[n-1],
			},
			"last_price": candles.Last().Close,
		},
	})
}

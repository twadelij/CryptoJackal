package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/twadelij/cryptojackal/internal/config"
	"github.com/twadelij/cryptojackal/internal/discovery"
	"github.com/twadelij/cryptojackal/internal/models"
	"github.com/twadelij/cryptojackal/internal/paper"
	"github.com/twadelij/cryptojackal/internal/storage"
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
	OpportunityID string  `json:"opportunity_id"`
	Amount        float64 `json:"amount"`
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

// GetTradingHistory returns trading history
func (h *Handler) GetTradingHistory(c *gin.Context) {
	trades := h.paper.GetTradeHistory(50)
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

// GetPaperBalance returns paper trading balance
func (h *Handler) GetPaperBalance(c *gin.Context) {
	portfolio := h.paper.GetPortfolio()
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
	Price        float64 `json:"price" binding:"required"`
	Amount       float64 `json:"amount" binding:"required"`
	Type         string  `json:"type" binding:"required"` // "buy" or "sell"
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

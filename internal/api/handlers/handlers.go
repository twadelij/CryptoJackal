package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/twadelij/cryptojackal/internal/discovery"
	"github.com/twadelij/cryptojackal/internal/models"
	"github.com/twadelij/cryptojackal/internal/paper"
	"github.com/twadelij/cryptojackal/internal/trading"
	"go.uber.org/zap"
)

// Handler contains all HTTP handlers
type Handler struct {
	engine    *trading.Engine
	discovery *discovery.Service
	paper     *paper.Service
	logger    *zap.Logger
}

// NewHandler creates a new handler
func NewHandler(engine *trading.Engine, disc *discovery.Service, paperSvc *paper.Service, logger *zap.Logger) *Handler {
	return &Handler{
		engine:    engine,
		discovery: disc,
		paper:     paperSvc,
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

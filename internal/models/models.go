package models

import (
	"time"

	"github.com/google/uuid"
)

// Token represents a discovered token
type Token struct {
	Address          string    `json:"address"`
	Symbol           string    `json:"symbol"`
	Name             string    `json:"name"`
	Decimals         int       `json:"decimals"`
	Price            float64   `json:"price"`
	PriceChange24h   float64   `json:"price_change_24h"`
	MarketCap        float64   `json:"market_cap"`
	Volume24h        float64   `json:"volume_24h"`
	Liquidity        float64   `json:"liquidity"`
	SecurityScore    float64   `json:"security_score"`
	DiscoveredAt     time.Time `json:"discovered_at"`
	Tags             []string  `json:"tags"`
}

// TradingOpportunity represents a potential trade
type TradingOpportunity struct {
	ID              string    `json:"id"`
	Token           Token     `json:"token"`
	ExpectedProfit  float64   `json:"expected_profit"`
	PriceImpact     float64   `json:"price_impact"`
	ConfidenceScore float64   `json:"confidence_score"`
	Strategy        string    `json:"strategy"`
	CreatedAt       time.Time `json:"created_at"`
	ExpiresAt       time.Time `json:"expires_at"`
}

// Trade represents an executed trade
type Trade struct {
	ID            string    `json:"id"`
	TokenAddress  string    `json:"token_address"`
	TokenSymbol   string    `json:"token_symbol"`
	Type          TradeType `json:"type"`
	AmountIn      float64   `json:"amount_in"`
	AmountOut     float64   `json:"amount_out"`
	Price         float64   `json:"price"`
	GasUsed       uint64    `json:"gas_used"`
	GasPrice      uint64    `json:"gas_price"`
	TxHash        string    `json:"tx_hash"`
	Status        TradeStatus `json:"status"`
	ProfitLoss    float64   `json:"profit_loss"`
	ExecutedAt    time.Time `json:"executed_at"`
	IsPaperTrade  bool      `json:"is_paper_trade"`
}

type TradeType string

const (
	TradeTypeBuy  TradeType = "buy"
	TradeTypeSell TradeType = "sell"
)

type TradeStatus string

const (
	TradeStatusPending   TradeStatus = "pending"
	TradeStatusExecuted  TradeStatus = "executed"
	TradeStatusFailed    TradeStatus = "failed"
	TradeStatusCancelled TradeStatus = "cancelled"
)

// Portfolio represents a trading portfolio
type Portfolio struct {
	ID           string            `json:"id"`
	ETHBalance   float64           `json:"eth_balance"`
	TokenBalances map[string]TokenBalance `json:"token_balances"`
	TotalValue   float64           `json:"total_value"`
	ProfitLoss   float64           `json:"profit_loss"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type TokenBalance struct {
	Token    Token   `json:"token"`
	Balance  float64 `json:"balance"`
	Value    float64 `json:"value"`
	AvgPrice float64 `json:"avg_price"`
}

// BotStatus represents the current bot status
type BotStatus struct {
	IsRunning       bool      `json:"is_running"`
	Mode            string    `json:"mode"` // "paper" or "live"
	StartedAt       *time.Time `json:"started_at,omitempty"`
	TotalTrades     int       `json:"total_trades"`
	ProfitableTrades int      `json:"profitable_trades"`
	TotalProfitLoss float64   `json:"total_profit_loss"`
	CurrentBalance  float64   `json:"current_balance"`
	ActiveOpportunities int   `json:"active_opportunities"`
}

// Metrics for monitoring
type Metrics struct {
	Uptime            time.Duration `json:"uptime"`
	TotalTrades       int           `json:"total_trades"`
	SuccessfulTrades  int           `json:"successful_trades"`
	FailedTrades      int           `json:"failed_trades"`
	TotalVolume       float64       `json:"total_volume"`
	TotalProfitLoss   float64       `json:"total_profit_loss"`
	WinRate           float64       `json:"win_rate"`
	AverageProfitPerTrade float64   `json:"average_profit_per_trade"`
	TokensDiscovered  int           `json:"tokens_discovered"`
	OpportunitiesFound int          `json:"opportunities_found"`
}

// NewTrade creates a new trade with a generated ID
func NewTrade(tokenAddress, tokenSymbol string, tradeType TradeType, amountIn, price float64, isPaper bool) *Trade {
	return &Trade{
		ID:           uuid.New().String(),
		TokenAddress: tokenAddress,
		TokenSymbol:  tokenSymbol,
		Type:         tradeType,
		AmountIn:     amountIn,
		Price:        price,
		Status:       TradeStatusPending,
		ExecutedAt:   time.Now(),
		IsPaperTrade: isPaper,
	}
}

// NewOpportunity creates a new trading opportunity
func NewOpportunity(token Token, expectedProfit, priceImpact, confidence float64, strategy string) *TradingOpportunity {
	return &TradingOpportunity{
		ID:              uuid.New().String(),
		Token:           token,
		ExpectedProfit:  expectedProfit,
		PriceImpact:     priceImpact,
		ConfidenceScore: confidence,
		Strategy:        strategy,
		CreatedAt:       time.Now(),
		ExpiresAt:       time.Now().Add(5 * time.Minute),
	}
}

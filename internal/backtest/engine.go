package backtest

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/twadelij/cryptojackal/internal/indicators"
	"github.com/twadelij/cryptojackal/internal/strategy"
	"go.uber.org/zap"
)

// Config holds parameters for a backtest run
type Config struct {
	InitialBalance   float64   `json:"initial_balance"`
	TradeAmount      float64   `json:"trade_amount"`
	TakeProfitPct    float64   `json:"take_profit_pct"`
	StopLossPct      float64   `json:"stop_loss_pct"`
	MaxOpenPositions int       `json:"max_open_positions"`
	FeePct           float64   `json:"fee_pct"`
	StartTime        time.Time `json:"start_time"`
	EndTime          time.Time `json:"end_time"`
}

// BacktestTrade represents a single trade in a backtest
type BacktestTrade struct {
	EntryIndex int       `json:"entry_index"`
	ExitIndex  int       `json:"exit_index"`
	EntryPrice float64   `json:"entry_price"`
	ExitPrice  float64   `json:"exit_price"`
	Amount     float64   `json:"amount"`
	EntryFee   float64   `json:"entry_fee"`
	ExitFee    float64   `json:"exit_fee"`
	ProfitLoss float64   `json:"profit_loss"`
	Strategy   string    `json:"strategy"`
	EntryTime  time.Time `json:"entry_time"`
	ExitTime   time.Time `json:"exit_time"`
	ExitReason string    `json:"exit_reason"`
}

// Result holds the outcome of a backtest
type Result struct {
	TotalTrades     int             `json:"total_trades"`
	WinningTrades   int             `json:"winning_trades"`
	LosingTrades    int             `json:"losing_trades"`
	TotalProfitLoss float64         `json:"total_profit_loss"`
	MaxDrawdown     float64         `json:"max_drawdown"`
	WinRate         float64         `json:"win_rate"`
	SharpeRatio     float64         `json:"sharpe_ratio"`
	FinalBalance    float64         `json:"final_balance"`
	TotalFees       float64         `json:"total_fees"`
	Trades          []BacktestTrade `json:"trades"`
	EquityCurve     []float64       `json:"equity_curve"`
	Config          Config          `json:"config"`
	StrategiesUsed  []string        `json:"strategies_used"`
	CreatedAt       time.Time       `json:"created_at"`
}

// openPosition tracks an open position during backtest
type openPosition struct {
	entryIndex int
	entryPrice float64
	amount     float64
	entryFee   float64
	strategy   string
	entryTime  time.Time
}

// Engine runs backtests against historical candle data
type Engine struct {
	logger *zap.Logger
}

// NewEngine creates a new backtest engine
func NewEngine(logger *zap.Logger) *Engine {
	return &Engine{logger: logger}
}

// Run executes a backtest with the given config, candle data, and strategies.
// candlesMap maps a pair name to its candle series.
// Only CandleStrategy implementations are used for backtesting.
func (e *Engine) Run(ctx context.Context, cfg Config, candlesMap map[string]indicators.CandleSeries, strategies []strategy.Strategy) Result {
	result := Result{
		Config:         cfg,
		CreatedAt:      time.Now(),
		StrategiesUsed: make([]string, 0),
	}

	candleStrategies := make([]strategy.CandleStrategy, 0)
	for _, s := range strategies {
		if cs, ok := s.(strategy.CandleStrategy); ok {
			candleStrategies = append(candleStrategies, cs)
			result.StrategiesUsed = append(result.StrategiesUsed, s.Name())
		}
	}

	if len(candleStrategies) == 0 {
		e.logger.Warn("No candle-based strategies available for backtesting")
		return result
	}

	balance := cfg.InitialBalance
	totalFees := 0.0
	var positions []openPosition
	var completedTrades []BacktestTrade
	equityCurve := make([]float64, 0)

	maxCandleLen := 0
	for _, cs := range candlesMap {
		if cs.Len() > maxCandleLen {
			maxCandleLen = cs.Len()
		}
	}

	for i := 0; i < maxCandleLen; i++ {
		select {
		case <-ctx.Done():
			result.Trades = completedTrades
			result.EquityCurve = equityCurve
			result.FinalBalance = balance
			result.TotalFees = totalFees
			e.computeMetrics(&result)
			return result
		default:
		}

		for _, candles := range candlesMap {
			if i >= candles.Len() {
				continue
			}

			updatedPositions := make([]openPosition, 0, len(positions))
			for _, pos := range positions {
				candle := candles[i]
				tpPrice := pos.entryPrice * (1 + cfg.TakeProfitPct/100)
				slPrice := pos.entryPrice * (1 - cfg.StopLossPct/100)

				exitReason := ""
				exitPrice := 0.0

				if candle.High >= tpPrice {
					exitPrice = tpPrice
					exitReason = "take_profit"
				} else if candle.Low <= slPrice {
					exitPrice = slPrice
					exitReason = "stop_loss"
				}

				if exitReason != "" {
					exitFee := exitPrice * pos.amount * cfg.FeePct / 100
					totalFees += exitFee
					pnl := (exitPrice-pos.entryPrice)*pos.amount - pos.entryFee - exitFee

					trade := BacktestTrade{
						EntryIndex: pos.entryIndex,
						ExitIndex:  i,
						EntryPrice: pos.entryPrice,
						ExitPrice:  exitPrice,
						Amount:     pos.amount,
						EntryFee:   pos.entryFee,
						ExitFee:    exitFee,
						ProfitLoss: pnl,
						Strategy:   pos.strategy,
						EntryTime:  pos.entryTime,
						ExitTime:   candle.Time,
						ExitReason: exitReason,
					}
					completedTrades = append(completedTrades, trade)
					balance += pnl
				} else {
					updatedPositions = append(updatedPositions, pos)
				}
			}
			positions = updatedPositions

			if len(positions) < cfg.MaxOpenPositions && i >= 30 {
				subCandles := candles[:i+1]
				for _, cs := range candleStrategies {
					sig := cs.AnalyzeCandles(ctx, subCandles)
					if sig.Action == "buy" && sig.Confidence > 0 {
						entryPrice := candles[i].Close
						amount := cfg.TradeAmount / entryPrice
						entryFee := entryPrice * amount * cfg.FeePct / 100
						totalFees += entryFee
						balance -= entryPrice * amount

						positions = append(positions, openPosition{
							entryIndex: i,
							entryPrice: entryPrice,
							amount:     amount,
							entryFee:   entryFee,
							strategy:   cs.Name(),
							entryTime:  candles[i].Time,
						})
						break
					}
				}
			}
		}

		equity := balance
		for _, pos := range positions {
			equity += pos.entryPrice * pos.amount
		}
		equityCurve = append(equityCurve, math.Round(equity*100)/100)
	}

	for _, pos := range positions {
		for _, candles := range candlesMap {
			if pos.entryIndex < candles.Len() {
				lastCandle := candles.Last()
				exitPrice := lastCandle.Close
				exitFee := exitPrice * pos.amount * cfg.FeePct / 100
				totalFees += exitFee
				pnl := (exitPrice-pos.entryPrice)*pos.amount - pos.entryFee - exitFee
				completedTrades = append(completedTrades, BacktestTrade{
					EntryIndex: pos.entryIndex,
					ExitIndex:  candles.Len() - 1,
					EntryPrice: pos.entryPrice,
					ExitPrice:  exitPrice,
					Amount:     pos.amount,
					EntryFee:   pos.entryFee,
					ExitFee:    exitFee,
					ProfitLoss: pnl,
					Strategy:   pos.strategy,
					EntryTime:  pos.entryTime,
					ExitTime:   lastCandle.Time,
					ExitReason: "backtest_end",
				})
				balance += pnl
				break
			}
		}
	}

	result.Trades = completedTrades
	result.EquityCurve = equityCurve
	result.FinalBalance = balance
	result.TotalFees = totalFees
	e.computeMetrics(&result)

	return result
}

func (e *Engine) computeMetrics(r *Result) {
	r.TotalTrades = len(r.Trades)
	r.WinningTrades = 0
	r.LosingTrades = 0
	r.TotalProfitLoss = 0

	for _, t := range r.Trades {
		if t.ProfitLoss > 0 {
			r.WinningTrades++
		} else {
			r.LosingTrades++
		}
		r.TotalProfitLoss += t.ProfitLoss
	}

	if r.TotalTrades > 0 {
		r.WinRate = float64(r.WinningTrades) / float64(r.TotalTrades) * 100
	}

	r.MaxDrawdown = e.calculateMaxDrawdown(r.EquityCurve)
	r.SharpeRatio = e.calculateSharpeRatio(r.Trades)
}

func (e *Engine) calculateMaxDrawdown(equity []float64) float64 {
	if len(equity) == 0 {
		return 0
	}

	peak := equity[0]
	maxDD := 0.0

	for _, v := range equity {
		if v > peak {
			peak = v
		}
		dd := (peak - v) / peak * 100
		if dd > maxDD {
			maxDD = dd
		}
	}

	return math.Round(maxDD*100) / 100
}

func (e *Engine) calculateSharpeRatio(trades []BacktestTrade) float64 {
	if len(trades) < 2 {
		return 0
	}

	returns := make([]float64, len(trades))
	sum := 0.0
	for i, t := range trades {
		ret := t.ProfitLoss
		returns[i] = ret
		sum += ret
	}

	mean := sum / float64(len(returns))

	variance := 0.0
	for _, r := range returns {
		diff := r - mean
		variance += diff * diff
	}
	variance /= float64(len(returns) - 1)
	stddev := math.Sqrt(variance)

	if stddev == 0 {
		return 0
	}

	sharpe := mean / stddev * math.Sqrt(252)
	return math.Round(sharpe*100) / 100
}

// GenerateSummary returns a human-readable summary of the backtest result
func GenerateSummary(r Result) string {
	return fmt.Sprintf(
		"Backtest Results:\n"+
			"  Trades: %d (W: %d, L: %d)\n"+
			"  Win Rate: %.1f%%\n"+
			"  Total P&L: $%.2f\n"+
			"  Max Drawdown: %.1f%%\n"+
			"  Sharpe Ratio: %.2f\n"+
			"  Final Balance: $%.2f\n"+
			"  Total Fees: $%.2f\n"+
			"  Strategies: %v\n",
		r.TotalTrades, r.WinningTrades, r.LosingTrades,
		r.WinRate, r.TotalProfitLoss, r.MaxDrawdown,
		r.SharpeRatio, r.FinalBalance, r.TotalFees,
		r.StrategiesUsed,
	)
}

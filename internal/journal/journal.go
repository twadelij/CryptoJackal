package journal

import (
	"sync"
	"time"
)

// Entry represents a completed trade with features and outcome for ML training
type Entry struct {
	TradeID      string
	Strategy     string
	Features     TradeFeatures
	Outcome      string  // "win", "loss", "neutral"
	ProfitPct    float64
	HoldDuration time.Duration
	BuyTime      time.Time
	SellTime     time.Time
}

// TradeFeatures captures token metrics at buy time for ML prediction
type TradeFeatures struct {
	PriceChange24h float64
	Volume24h      float64
	Liquidity      float64
	MarketCap      float64
	SecurityScore  float64
	HourOfDay      int
	StrategyType   string
	DipDepth       float64
	VolumeRatio    float64
}

// StrategyStats holds per-strategy performance metrics
type StrategyStats struct {
	Strategy  string
	WinRate   float64
	AvgProfit float64
	Count     int
	Wins      int
	Losses    int
}

// Journal stores trade entries for ML training and strategy analysis
type Journal struct {
	mu      sync.RWMutex
	entries []Entry
}

// New creates a new trade journal
func New() *Journal {
	return &Journal{
		entries: make([]Entry, 0),
	}
}

// RecordBuy records a buy with features at buy time
func (j *Journal) RecordBuy(tradeID, strategy string, features TradeFeatures) {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.entries = append(j.entries, Entry{
		TradeID:    tradeID,
		Strategy:   strategy,
		Features:   features,
		BuyTime:    time.Now(),
	})
}

// RecordSell updates a trade entry with the outcome
func (j *Journal) RecordSell(tradeID string, profitPct float64, holdDuration time.Duration) {
	j.mu.Lock()
	defer j.mu.Unlock()

	for i := range j.entries {
		if j.entries[i].TradeID == tradeID {
			j.entries[i].SellTime = time.Now()
			j.entries[i].ProfitPct = profitPct
			j.entries[i].HoldDuration = holdDuration
			j.entries[i].Outcome = classifyOutcome(profitPct)
			return
		}
	}
}

// GetTrainingData returns all completed trades as training samples
func (j *Journal) GetTrainingData() []TrainingSample {
	j.mu.RLock()
	defer j.mu.RUnlock()

	samples := make([]TrainingSample, 0, len(j.entries))
	for _, e := range j.entries {
		if e.Outcome == "" {
			continue // skip open trades
		}
		label := 0.0
		if e.Outcome == "win" {
			label = 1.0
		}
		samples = append(samples, TrainingSample{
			Features: e.Features,
			Label:    label,
		})
	}
	return samples
}

// GetStrategyStats returns per-strategy performance
func (j *Journal) GetStrategyStats() map[string]StrategyStats {
	j.mu.RLock()
	defer j.mu.RUnlock()

	stats := make(map[string]StrategyStats)
	for _, e := range j.entries {
		if e.Outcome == "" {
			continue
		}
		s := stats[e.Strategy]
		s.Strategy = e.Strategy
		s.Count++
		s.AvgProfit += e.ProfitPct
		if e.Outcome == "win" {
			s.Wins++
		} else if e.Outcome == "loss" {
			s.Losses++
		}
		stats[e.Strategy] = s
	}

	for k, v := range stats {
		if v.Count > 0 {
			v.WinRate = float64(v.Wins) / float64(v.Count)
			v.AvgProfit /= float64(v.Count)
		}
		stats[k] = v
	}
	return stats
}

// GetEntryCount returns total number of entries
func (j *Journal) GetEntryCount() int {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return len(j.entries)
}

// GetCompletedCount returns number of completed trades (with outcome)
func (j *Journal) GetCompletedCount() int {
	j.mu.RLock()
	defer j.mu.RUnlock()
	count := 0
	for _, e := range j.entries {
		if e.Outcome != "" {
			count++
		}
	}
	return count
}

// TrainingSample is a single training example for the ML model
type TrainingSample struct {
	Features TradeFeatures
	Label    float64 // 1.0 = win, 0.0 = loss
}

func classifyOutcome(profitPct float64) string {
	if profitPct > 2.0 {
		return "win"
	}
	if profitPct < -2.0 {
		return "loss"
	}
	return "neutral"
}

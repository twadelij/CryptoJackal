package learning

import (
	"testing"

	"github.com/twadelij/cryptojackal/internal/journal"
	"go.uber.org/zap"
)

func TestPredictor_UntrainedFallback(t *testing.T) {
	pred := NewPredictor(zap.NewNop(), 20)
	features := journal.TradeFeatures{
		SecurityScore: 0.8,
		Volume24h:     200000,
		Liquidity:     100000,
	}

	score := pred.Predict(features)
	if score <= 0 || score > 1.0 {
		t.Errorf("expected score in (0, 1], got %.2f", score)
	}
	if pred.IsTrained() {
		t.Error("expected predictor to be untrained")
	}
}

func TestPredictor_TrainAndPredict(t *testing.T) {
	pred := NewPredictor(zap.NewNop(), 5)

	samples := []journal.TrainingSample{
		{Features: journal.TradeFeatures{PriceChange24h: 30, Volume24h: 200000, Liquidity: 100000, SecurityScore: 0.8, StrategyType: "momentum"}, Label: 1.0},
		{Features: journal.TradeFeatures{PriceChange24h: 25, Volume24h: 150000, Liquidity: 80000, SecurityScore: 0.7, StrategyType: "momentum"}, Label: 1.0},
		{Features: journal.TradeFeatures{PriceChange24h: -20, Volume24h: 50000, Liquidity: 20000, SecurityScore: 0.3, StrategyType: "dip_buy"}, Label: 0.0},
		{Features: journal.TradeFeatures{PriceChange24h: -15, Volume24h: 30000, Liquidity: 10000, SecurityScore: 0.2, StrategyType: "dip_buy"}, Label: 0.0},
		{Features: journal.TradeFeatures{PriceChange24h: 40, Volume24h: 300000, Liquidity: 150000, SecurityScore: 0.9, StrategyType: "momentum"}, Label: 1.0},
		{Features: journal.TradeFeatures{PriceChange24h: 35, Volume24h: 250000, Liquidity: 120000, SecurityScore: 0.85, StrategyType: "momentum"}, Label: 1.0},
		{Features: journal.TradeFeatures{PriceChange24h: -25, Volume24h: 40000, Liquidity: 15000, SecurityScore: 0.25, StrategyType: "dip_buy"}, Label: 0.0},
		{Features: journal.TradeFeatures{PriceChange24h: 20, Volume24h: 180000, Liquidity: 90000, SecurityScore: 0.75, StrategyType: "momentum"}, Label: 1.0},
		{Features: journal.TradeFeatures{PriceChange24h: -10, Volume24h: 20000, Liquidity: 5000, SecurityScore: 0.15, StrategyType: "dip_buy"}, Label: 0.0},
		{Features: journal.TradeFeatures{PriceChange24h: -30, Volume24h: 10000, Liquidity: 8000, SecurityScore: 0.1, StrategyType: "dip_buy"}, Label: 0.0},
	}

	pred.Train(samples)

	if !pred.IsTrained() {
		t.Error("expected predictor to be trained after Train()")
	}

	// Predict on a "winning" pattern
	winScore := pred.Predict(journal.TradeFeatures{
		PriceChange24h: 35,
		Volume24h:      250000,
		Liquidity:      120000,
		SecurityScore:  0.85,
		StrategyType:   "momentum",
	})
	if winScore < 0.5 {
		t.Errorf("expected high score for winning pattern, got %.2f", winScore)
	}

	// Predict on a "losing" pattern
	loseScore := pred.Predict(journal.TradeFeatures{
		PriceChange24h: -18,
		Volume24h:      40000,
		Liquidity:      15000,
		SecurityScore:  0.25,
		StrategyType:   "dip_buy",
	})
	if loseScore > 0.5 {
		t.Errorf("expected low score for losing pattern, got %.2f", loseScore)
	}
}

func TestPredictor_NotEnoughSamples(t *testing.T) {
	pred := NewPredictor(zap.NewNop(), 100)
	samples := []journal.TrainingSample{
		{Features: journal.TradeFeatures{}, Label: 1.0},
	}

	pred.Train(samples)
	if pred.IsTrained() {
		t.Error("expected predictor to remain untrained with too few samples")
	}
}

func TestJournal_RecordAndRetrieve(t *testing.T) {
	j := journal.New()

	j.RecordBuy("trade-1", "momentum", journal.TradeFeatures{
		PriceChange24h: 25,
		Volume24h:      200000,
		Liquidity:      100000,
	})

	if j.GetEntryCount() != 1 {
		t.Errorf("expected 1 entry, got %d", j.GetEntryCount())
	}
	if j.GetCompletedCount() != 0 {
		t.Errorf("expected 0 completed, got %d", j.GetCompletedCount())
	}

	j.RecordSell("trade-1", 5.5, 360000000000)

	if j.GetCompletedCount() != 1 {
		t.Errorf("expected 1 completed, got %d", j.GetCompletedCount())
	}

	samples := j.GetTrainingData()
	if len(samples) != 1 {
		t.Fatalf("expected 1 training sample, got %d", len(samples))
	}
	if samples[0].Label != 1.0 {
		t.Errorf("expected label 1.0 for win, got %.1f", samples[0].Label)
	}
}

func TestJournal_StrategyStats(t *testing.T) {
	j := journal.New()

	j.RecordBuy("t1", "momentum", journal.TradeFeatures{})
	j.RecordBuy("t2", "momentum", journal.TradeFeatures{})
	j.RecordBuy("t3", "dip_buy", journal.TradeFeatures{})

	j.RecordSell("t1", 5.0, 1000000000)
	j.RecordSell("t2", -3.0, 1000000000)
	j.RecordSell("t3", -4.0, 1000000000)

	stats := j.GetStrategyStats()
	momentumStats, ok := stats["momentum"]
	if !ok {
		t.Fatal("expected momentum stats")
	}
	if momentumStats.Count != 2 {
		t.Errorf("expected 2 momentum trades, got %d", momentumStats.Count)
	}
	if momentumStats.Wins != 1 {
		t.Errorf("expected 1 win, got %d", momentumStats.Wins)
	}
	if momentumStats.WinRate != 0.5 {
		t.Errorf("expected 0.5 win rate, got %.2f", momentumStats.WinRate)
	}
}

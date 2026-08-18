package indicators

import (
	"math"
	"testing"
	"time"
)

func makeCandles(closes []float64) CandleSeries {
	candles := make(CandleSeries, len(closes))
	for i, c := range closes {
		candles[i] = Candle{
			Open:   c,
			High:   c * 1.01,
			Low:    c * 0.99,
			Close:  c,
			Volume: 1000,
			Time:   time.Unix(int64(i*3600), 0),
		}
	}
	return candles
}

func TestSMA(t *testing.T) {
	candles := makeCandles([]float64{10, 20, 30, 40, 50})
	sma := SMA(candles, 3)

	if sma[0] != 0 || sma[1] != 0 {
		t.Errorf("SMA should be 0 for insufficient data, got %v, %v", sma[0], sma[1])
	}

	if math.Abs(sma[2]-20) > 0.0001 {
		t.Errorf("SMA[2] = %v, expected 20", sma[2])
	}
	if math.Abs(sma[3]-30) > 0.0001 {
		t.Errorf("SMA[3] = %v, expected 30", sma[3])
	}
	if math.Abs(sma[4]-40) > 0.0001 {
		t.Errorf("SMA[4] = %v, expected 40", sma[4])
	}
}

func TestSMAValues(t *testing.T) {
	values := []float64{1, 2, 3, 4, 5}
	sma := SMAValues(values, 2)

	if math.Abs(sma[1]-1.5) > 0.0001 {
		t.Errorf("SMAValues[1] = %v, expected 1.5", sma[1])
	}
	if math.Abs(sma[4]-4.5) > 0.0001 {
		t.Errorf("SMAValues[4] = %v, expected 4.5", sma[4])
	}
}

func TestEMA(t *testing.T) {
	candles := makeCandles([]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12})
	ema := EMA(candles, 3)

	if ema[0] != 0 || ema[1] != 0 {
		t.Errorf("EMA should be 0 for insufficient data")
	}

	if math.Abs(ema[2]-2) > 0.0001 {
		t.Errorf("EMA[2] = %v, expected 2 (SMA seed)", ema[2])
	}

	mult := 2.0 / 4.0
	expected := 4*mult + ema[2]*(1-mult)
	if math.Abs(ema[3]-expected) > 0.0001 {
		t.Errorf("EMA[3] = %v, expected %v", ema[3], expected)
	}
}

func TestEMAValues(t *testing.T) {
	values := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	ema := EMAValues(values, 5)

	if ema[4] != 30 {
		t.Errorf("EMA seed should be SMA = 30, got %v", ema[4])
	}

	mult := 2.0 / 6.0
	expected := 60*mult + 30*(1-mult)
	if math.Abs(ema[5]-expected) > 0.0001 {
		t.Errorf("EMA[5] = %v, expected %v", ema[5], expected)
	}
}

func TestRSI(t *testing.T) {
	closes := make([]float64, 30)
	for i := range closes {
		if i < 15 {
			closes[i] = 100 + float64(i)
		} else {
			closes[i] = 114 - float64(i-15)
		}
	}
	candles := makeCandles(closes)
	rsi := RSI(candles, 14)

	if rsi[14] < 0 || rsi[14] > 100 {
		t.Errorf("RSI should be between 0 and 100, got %v", rsi[14])
	}
	if rsi[14] < 50 {
		t.Errorf("RSI after 14 consecutive gains should be high, got %v", rsi[14])
	}

	allUp := makeCandles([]float64{10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25})
	rsiUp := RSI(allUp, 14)
	if math.Abs(rsiUp[14]-100) > 0.0001 {
		t.Errorf("RSI of all gains should be 100, got %v", rsiUp[14])
	}

	allDown := makeCandles([]float64{25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10})
	rsiDown := RSI(allDown, 14)
	if math.Abs(rsiDown[14]-0) > 0.0001 {
		t.Errorf("RSI of all losses should be 0, got %v", rsiDown[14])
	}
}

func TestRSIShortData(t *testing.T) {
	candles := makeCandles([]float64{10, 20, 30})
	rsi := RSI(candles, 14)
	for _, v := range rsi {
		if v != 0 {
			t.Errorf("RSI with insufficient data should be 0, got %v", v)
		}
	}
}

func TestMACD(t *testing.T) {
	candles := makeCandles([]float64{
		100, 102, 104, 106, 108, 110, 112, 114, 116, 118,
		120, 122, 124, 126, 128, 130, 132, 134, 136, 138,
		140, 142, 144, 146, 148, 150, 152, 154, 156, 158,
		160, 162, 164, 166, 168, 170,
	})
	result := MACD(candles, 12, 26, 9)

	if len(result.MACDLine) != 36 {
		t.Errorf("MACD line length = %d, expected 36", len(result.MACDLine))
	}
	if len(result.SignalLine) != 36 {
		t.Errorf("Signal line length = %d, expected 36", len(result.SignalLine))
	}
	if len(result.Histogram) != 36 {
		t.Errorf("Histogram length = %d, expected 36", len(result.Histogram))
	}

	for i := 0; i < 25; i++ {
		if result.MACDLine[i] != 0 {
			t.Errorf("MACD line before slow period should be 0 at index %d, got %v", i, result.MACDLine[i])
		}
	}

	if result.MACDLine[25] <= 0 {
		t.Errorf("MACD line in uptrend should be positive, got %v", result.MACDLine[25])
	}

	lastIdx := 35
	if math.Abs(result.Histogram[lastIdx]-(result.MACDLine[lastIdx]-result.SignalLine[lastIdx])) > 0.0001 {
		t.Errorf("Histogram should equal MACD - Signal, got %v vs %v",
			result.Histogram[lastIdx], result.MACDLine[lastIdx]-result.SignalLine[lastIdx])
	}
}

func TestBollingerBands(t *testing.T) {
	candles := makeCandles([]float64{
		10, 12, 14, 16, 18, 20, 22, 24, 26, 28,
		30, 32, 34, 36, 38, 40, 42, 44, 46, 48, 50,
	})
	upper, middle, lower := BollingerBands(candles, 20, 2.0)

	if middle[19] != 29 {
		t.Errorf("Middle band (SMA) at index 19 = %v, expected 29", middle[19])
	}

	if upper[19] <= middle[19] {
		t.Errorf("Upper band should be above middle band: upper=%v, middle=%v", upper[19], middle[19])
	}

	if lower[19] >= middle[19] {
		t.Errorf("Lower band should be below middle band: lower=%v, middle=%v", lower[19], middle[19])
	}

	if upper[19] <= lower[19] {
		t.Errorf("Upper band should be above lower band: upper=%v, lower=%v", upper[19], lower[19])
	}

	if upper[0] != 0 || lower[0] != 0 || middle[0] != 0 {
		t.Errorf("Bands should be 0 for insufficient data")
	}
}

func TestBollingerBandsConstant(t *testing.T) {
	candles := makeCandles([]float64{
		50, 50, 50, 50, 50, 50, 50, 50, 50, 50,
		50, 50, 50, 50, 50, 50, 50, 50, 50, 50,
	})
	upper, middle, lower := BollingerBands(candles, 20, 2.0)

	if math.Abs(middle[19]-50) > 0.0001 {
		t.Errorf("Middle band for constant data = %v, expected 50", middle[19])
	}
	if math.Abs(upper[19]-50) > 0.0001 {
		t.Errorf("Upper band for constant data = %v, expected 50 (no variance)", upper[19])
	}
	if math.Abs(lower[19]-50) > 0.0001 {
		t.Errorf("Lower band for constant data = %v, expected 50 (no variance)", lower[19])
	}
}

func TestCandleSeriesHelpers(t *testing.T) {
	candles := makeCandles([]float64{10, 20, 30})
	closes := candles.Closes()
	if closes[0] != 10 || closes[1] != 20 || closes[2] != 30 {
		t.Errorf("Closes() = %v", closes)
	}

	last := candles.Last()
	if last.Close != 30 {
		t.Errorf("Last().Close = %v, expected 30", last.Close)
	}

	if candles.Len() != 3 {
		t.Errorf("Len() = %d, expected 3", candles.Len())
	}
}

func TestEmptyCandleSeries(t *testing.T) {
	var cs CandleSeries
	last := cs.Last()
	if last.Close != 0 {
		t.Errorf("Last() of empty series should be zero, got %v", last.Close)
	}
	if cs.Len() != 0 {
		t.Errorf("Len() of empty series should be 0, got %d", cs.Len())
	}
}

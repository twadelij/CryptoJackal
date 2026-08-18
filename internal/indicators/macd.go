package indicators

// MACDResult holds the three MACD output lines
type MACDResult struct {
	MACDLine   []float64 `json:"macd_line"`
	SignalLine []float64 `json:"signal_line"`
	Histogram  []float64 `json:"histogram"`
}

// MACD calculates the Moving Average Convergence Divergence.
// fast: short EMA period (default 12)
// slow: long EMA period (default 26)
// signal: signal line EMA period (default 9)
// MACD Line = EMA(fast) - EMA(slow)
// Signal Line = EMA(MACD Line, signal)
// Histogram = MACD Line - Signal Line
func MACD(candles CandleSeries, fast, slow, signal int) MACDResult {
	n := len(candles)
	closes := candles.Closes()

	emaFast := EMAValues(closes, fast)
	emaSlow := EMAValues(closes, slow)

	macdLine := make([]float64, n)
	for i := 0; i < n; i++ {
		if i < slow-1 {
			macdLine[i] = 0
			continue
		}
		macdLine[i] = emaFast[i] - emaSlow[i]
	}

	signalLine := emaOfSlice(macdLine, signal, slow-1)

	histogram := make([]float64, n)
	for i := 0; i < n; i++ {
		histogram[i] = macdLine[i] - signalLine[i]
	}

	return MACDResult{
		MACDLine:   macdLine,
		SignalLine: signalLine,
		Histogram:  histogram,
	}
}

// emaOfSlice calculates EMA starting from a given offset (where data becomes valid)
func emaOfSlice(values []float64, period, offset int) []float64 {
	n := len(values)
	result := make([]float64, n)

	if n <= offset+period {
		return result
	}

	mult := 2.0 / float64(period+1)

	var sum float64
	for i := offset; i < offset+period; i++ {
		sum += values[i]
	}
	result[offset+period-1] = sum / float64(period)

	for i := offset + period; i < n; i++ {
		result[i] = values[i]*mult + result[i-1]*(1-mult)
	}

	return result
}

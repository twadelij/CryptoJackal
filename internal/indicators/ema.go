package indicators

// EMA calculates the Exponential Moving Average for the given period.
// The multiplier is 2 / (period + 1).
// The first EMA value is the SMA of the first `period` values.
// Subsequent values use the EMA formula: EMA = close * mult + prevEMA * (1 - mult)
func EMA(candles CandleSeries, period int) []float64 {
	closes := candles.Closes()
	return EMAValues(closes, period)
}

// EMAValues calculates EMA on a plain float64 slice
func EMAValues(values []float64, period int) []float64 {
	n := len(values)
	result := make([]float64, n)

	if n == 0 {
		return result
	}

	mult := 2.0 / float64(period+1)

	for i := 0; i < n; i++ {
		if i < period-1 {
			result[i] = 0
			continue
		}
		if i == period-1 {
			sum := 0.0
			for j := 0; j < period; j++ {
				sum += values[j]
			}
			result[i] = sum / float64(period)
			continue
		}
		result[i] = values[i]*mult + result[i-1]*(1-mult)
	}

	return result
}

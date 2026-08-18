package indicators

// SMA calculates the Simple Moving Average for the given period.
// Returns a slice of the same length as the input, with NaN values for
// indices where there are not enough data points.
func SMA(candles CandleSeries, period int) []float64 {
	n := len(candles)
	result := make([]float64, n)

	for i := 0; i < n; i++ {
		if i < period-1 {
			result[i] = 0
			continue
		}
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += candles[j].Close
		}
		result[i] = sum / float64(period)
	}

	return result
}

// SMAValues calculates SMA on a plain float64 slice
func SMAValues(values []float64, period int) []float64 {
	n := len(values)
	result := make([]float64, n)

	for i := 0; i < n; i++ {
		if i < period-1 {
			result[i] = 0
			continue
		}
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += values[j]
		}
		result[i] = sum / float64(period)
	}

	return result
}

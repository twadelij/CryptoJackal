package indicators

import "math"

// BollingerBands calculates the Bollinger Bands.
// period: number of candles for the moving average (default 20)
// stdDev: number of standard deviations (default 2.0)
// Middle band = SMA(close, period)
// Upper band = Middle + stdDev * stddev
// Lower band = Middle - stdDev * stddev
func BollingerBands(candles CandleSeries, period int, stdDev float64) (upper, middle, lower []float64) {
	n := len(candles)
	upper = make([]float64, n)
	middle = make([]float64, n)
	lower = make([]float64, n)

	closes := candles.Closes()

	for i := 0; i < n; i++ {
		if i < period-1 {
			continue
		}

		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += closes[j]
		}
		mean := sum / float64(period)
		middle[i] = mean

		variance := 0.0
		for j := i - period + 1; j <= i; j++ {
			diff := closes[j] - mean
			variance += diff * diff
		}
		variance /= float64(period)
		std := math.Sqrt(variance)

		upper[i] = mean + stdDev*std
		lower[i] = mean - stdDev*std
	}

	return upper, middle, lower
}

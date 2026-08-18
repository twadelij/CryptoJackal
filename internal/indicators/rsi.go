package indicators

import "math"

// RSI calculates the Relative Strength Index using Wilder's smoothing method.
// Default period is 14.
// RSI = 100 - (100 / (1 + RS))
// RS = Average Gain / Average Loss
// First RS uses simple average, subsequent use Wilder's smoothing.
func RSI(candles CandleSeries, period int) []float64 {
	n := len(candles)
	result := make([]float64, n)

	if n <= period {
		return result
	}

	closes := candles.Closes()

	gains := make([]float64, n)
	losses := make([]float64, n)

	for i := 1; i < n; i++ {
		diff := closes[i] - closes[i-1]
		if diff > 0 {
			gains[i] = diff
		} else {
			losses[i] = math.Abs(diff)
		}
	}

	var avgGain, avgLoss float64

	for i := 1; i <= period; i++ {
		avgGain += gains[i]
		avgLoss += losses[i]
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	if avgLoss == 0 {
		result[period] = 100
	} else {
		rs := avgGain / avgLoss
		result[period] = 100 - (100 / (1 + rs))
	}

	for i := period + 1; i < n; i++ {
		avgGain = (avgGain*float64(period-1) + gains[i]) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + losses[i]) / float64(period)

		if avgLoss == 0 {
			result[i] = 100
		} else {
			rs := avgGain / avgLoss
			result[i] = 100 - (100 / (1 + rs))
		}
	}

	return result
}

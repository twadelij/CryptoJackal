package indicators

import "time"

// Candle represents a single OHLCV candle
type Candle struct {
	Open   float64   `json:"open"`
	High   float64   `json:"high"`
	Low    float64   `json:"low"`
	Close  float64   `json:"close"`
	Volume float64   `json:"volume"`
	Time   time.Time `json:"time"`
}

// CandleSeries is a slice of candles ordered by time ascending
type CandleSeries []Candle

// Closes returns the close prices as a slice
func (cs CandleSeries) Closes() []float64 {
	closes := make([]float64, len(cs))
	for i, c := range cs {
		closes[i] = c.Close
	}
	return closes
}

// Highs returns the high prices as a slice
func (cs CandleSeries) Highs() []float64 {
	highs := make([]float64, len(cs))
	for i, c := range cs {
		highs[i] = c.High
	}
	return highs
}

// Lows returns the low prices as a slice
func (cs CandleSeries) Lows() []float64 {
	lows := make([]float64, len(cs))
	for i, c := range cs {
		lows[i] = c.Low
	}
	return lows
}

// Volumes returns the volume values as a slice
func (cs CandleSeries) Volumes() []float64 {
	vols := make([]float64, len(cs))
	for i, c := range cs {
		vols[i] = c.Volume
	}
	return vols
}

// Last returns the last candle in the series, or a zero candle if empty
func (cs CandleSeries) Last() Candle {
	if len(cs) == 0 {
		return Candle{}
	}
	return cs[len(cs)-1]
}

// Len returns the number of candles
func (cs CandleSeries) Len() int {
	return len(cs)
}

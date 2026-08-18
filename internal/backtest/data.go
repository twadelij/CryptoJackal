package backtest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/twadelij/cryptojackal/internal/indicators"
)

// DataDownloader fetches historical OHLCV data from external APIs
type DataDownloader struct {
	client *http.Client
}

// NewDataDownloader creates a new data downloader
func NewDataDownloader() *DataDownloader {
	return &DataDownloader{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// DownloadFromDexScreener downloads candle data from DexScreener API
// DexScreener doesn't provide historical OHLCV directly, so this is a placeholder
// that returns an error. Use CEX APIs for historical data.
func (d *DataDownloader) DownloadFromDexScreener(tokenAddress string) (indicators.CandleSeries, error) {
	return nil, fmt.Errorf("DexScreener does not support historical OHLCV downloads")
}

// DownloadFromBinance downloads candle data from Binance public API
// pair: e.g. "BTCUSDT"
// interval: e.g. "1h", "4h", "1d"
// limit: max 1000 candles per request
func (d *DataDownloader) DownloadFromBinance(pair, interval string, limit int) (indicators.CandleSeries, error) {
	if limit <= 0 || limit > 1000 {
		limit = 500
	}

	url := fmt.Sprintf("https://api.binance.com/api/v3/klines?symbol=%s&interval=%s&limit=%d",
		pair, interval, limit)

	resp, err := d.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("binance API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("binance API returned %d: %s", resp.StatusCode, string(body))
	}

	var raw [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode binance response: %w", err)
	}

	candles := make(indicators.CandleSeries, len(raw))
	for i, k := range raw {
		openTime := int64(k[0].(float64)) / 1000
		open, _ := strconv.ParseFloat(k[1].(string), 64)
		high, _ := strconv.ParseFloat(k[2].(string), 64)
		low, _ := strconv.ParseFloat(k[3].(string), 64)
		close, _ := strconv.ParseFloat(k[4].(string), 64)
		volume, _ := strconv.ParseFloat(k[5].(string), 64)

		candles[i] = indicators.Candle{
			Open:   open,
			High:   high,
			Low:    low,
			Close:  close,
			Volume: volume,
			Time:   time.Unix(openTime, 0),
		}
	}

	return candles, nil
}

// DownloadFromKraken downloads candle data from Kraken public API
// pair: e.g. "XBTUSD"
// interval: minutes (60, 240, 1440)
func (d *DataDownloader) DownloadFromKraken(pair string, interval int) (indicators.CandleSeries, error) {
	url := fmt.Sprintf("https://api.kraken.com/0/public/OHLC?pair=%s&interval=%d", pair, interval)

	resp, err := d.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("kraken API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("kraken API returned %d: %s", resp.StatusCode, string(body))
	}

	var raw struct {
		Result map[string][][]float64 `json:"result"`
		Error  []string               `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode kraken response: %w", err)
	}

	if len(raw.Error) > 0 {
		return nil, fmt.Errorf("kraken API error: %v", raw.Error)
	}

	var pairData [][]float64
	for key, data := range raw.Result {
		if key == "last" {
			continue
		}
		pairData = data
		break
	}

	if pairData == nil {
		return nil, fmt.Errorf("no candle data returned from kraken for pair %s", pair)
	}

	candles := make(indicators.CandleSeries, len(pairData))
	for i, k := range pairData {
		candles[i] = indicators.Candle{
			Open:   k[1],
			High:   k[2],
			Low:    k[3],
			Close:  k[4],
			Volume: k[6],
			Time:   time.Unix(int64(k[0]), 0),
		}
	}

	return candles, nil
}

// GenerateSyntheticCandles creates synthetic candle data for testing
// trend: "up", "down", "sideways"
// count: number of candles to generate
func GenerateSyntheticCandles(trend string, count int) indicators.CandleSeries {
	candles := make(indicators.CandleSeries, count)
	price := 100.0

	for i := 0; i < count; i++ {
		var change float64
		switch trend {
		case "up":
			change = 1.0 + float64(i%5)*0.2
		case "down":
			change = -1.0 - float64(i%5)*0.2
		case "sideways":
			if i%10 < 5 {
				change = 0.5
			} else {
				change = -0.5
			}
		default:
			change = 0
		}

		open := price
		close := price + change
		high := max(open, close) + 0.5
		low := min(open, close) - 0.5

		candles[i] = indicators.Candle{
			Open:   open,
			High:   high,
			Low:    low,
			Close:  close,
			Volume: 1000 + float64(i%100)*10,
			Time:   time.Unix(int64(i*3600), 0),
		}
		price = close
	}

	return candles
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

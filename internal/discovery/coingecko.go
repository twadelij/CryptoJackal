package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/twadelij/cryptojackal/internal/models"
	"go.uber.org/zap"
)

const (
	coingeckoBaseURL = "https://api.coingecko.com/api/v3"
)

// CoinGeckoClient handles CoinGecko API interactions
type CoinGeckoClient struct {
	httpClient *http.Client
	apiKey     string
	logger     *zap.Logger
}

// NewCoinGeckoClient creates a new CoinGecko client
func NewCoinGeckoClient(apiKey string, logger *zap.Logger) *CoinGeckoClient {
	return &CoinGeckoClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     apiKey,
		logger:     logger,
	}
}

type coinGeckoTrending struct {
	Coins []struct {
		Item struct {
			ID            string  `json:"id"`
			Symbol        string  `json:"symbol"`
			Name          string  `json:"name"`
			MarketCapRank int     `json:"market_cap_rank"`
			PriceBTC      float64 `json:"price_btc"`
			Data          struct {
				Price                    float64 `json:"price"`
				PriceChangePercentage24h map[string]float64 `json:"price_change_percentage_24h"`
				MarketCap                string `json:"market_cap"`
				TotalVolume              string `json:"total_volume"`
			} `json:"data"`
		} `json:"item"`
	} `json:"coins"`
}

type coinGeckoMarketData struct {
	ID                       string  `json:"id"`
	Symbol                   string  `json:"symbol"`
	Name                     string  `json:"name"`
	CurrentPrice             float64 `json:"current_price"`
	MarketCap                float64 `json:"market_cap"`
	TotalVolume              float64 `json:"total_volume"`
	PriceChangePercentage24h float64 `json:"price_change_percentage_24h"`
}

// GetTrendingTokens fetches trending tokens from CoinGecko
func (c *CoinGeckoClient) GetTrendingTokens(ctx context.Context) ([]models.Token, error) {
	url := fmt.Sprintf("%s/search/trending", coingeckoBaseURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	if c.apiKey != "" {
		req.Header.Set("x-cg-demo-api-key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch trending: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CoinGecko API error: %d", resp.StatusCode)
	}

	var trending coinGeckoTrending
	if err := json.NewDecoder(resp.Body).Decode(&trending); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	tokens := make([]models.Token, 0, len(trending.Coins))
	for _, coin := range trending.Coins {
		tokens = append(tokens, models.Token{
			Symbol:       coin.Item.Symbol,
			Name:         coin.Item.Name,
			DiscoveredAt: time.Now(),
			Tags:         []string{"trending"},
		})
	}

	c.logger.Info("fetched trending tokens", zap.Int("count", len(tokens)))
	return tokens, nil
}

// GetMarketData fetches market data for top tokens
func (c *CoinGeckoClient) GetMarketData(ctx context.Context, limit int) ([]models.Token, error) {
	url := fmt.Sprintf("%s/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=%d&page=1&sparkline=false", 
		coingeckoBaseURL, limit)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	if c.apiKey != "" {
		req.Header.Set("x-cg-demo-api-key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch market data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CoinGecko API error: %d", resp.StatusCode)
	}

	var marketData []coinGeckoMarketData
	if err := json.NewDecoder(resp.Body).Decode(&marketData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	tokens := make([]models.Token, 0, len(marketData))
	for _, data := range marketData {
		tokens = append(tokens, models.Token{
			Symbol:         data.Symbol,
			Name:           data.Name,
			Price:          data.CurrentPrice,
			PriceChange24h: data.PriceChangePercentage24h,
			MarketCap:      data.MarketCap,
			Volume24h:      data.TotalVolume,
			DiscoveredAt:   time.Now(),
		})
	}

	c.logger.Info("fetched market data", zap.Int("count", len(tokens)))
	return tokens, nil
}

// GetTokenByContract fetches token info by contract address
func (c *CoinGeckoClient) GetTokenByContract(ctx context.Context, platform, contractAddress string) (*models.Token, error) {
	url := fmt.Sprintf("%s/coins/%s/contract/%s", coingeckoBaseURL, platform, contractAddress)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	if c.apiKey != "" {
		req.Header.Set("x-cg-demo-api-key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CoinGecko API error: %d", resp.StatusCode)
	}

	var data struct {
		Symbol     string `json:"symbol"`
		Name       string `json:"name"`
		MarketData struct {
			CurrentPrice             map[string]float64 `json:"current_price"`
			MarketCap                map[string]float64 `json:"market_cap"`
			TotalVolume              map[string]float64 `json:"total_volume"`
			PriceChangePercentage24h float64            `json:"price_change_percentage_24h"`
		} `json:"market_data"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &models.Token{
		Address:        contractAddress,
		Symbol:         data.Symbol,
		Name:           data.Name,
		Price:          data.MarketData.CurrentPrice["usd"],
		PriceChange24h: data.MarketData.PriceChangePercentage24h,
		MarketCap:      data.MarketData.MarketCap["usd"],
		Volume24h:      data.MarketData.TotalVolume["usd"],
		DiscoveredAt:   time.Now(),
	}, nil
}

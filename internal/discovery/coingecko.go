package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/twadelij/cryptojackal/internal/datasource"
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
	limiter    *datasource.RateLimiter
	cache      *datasource.ResponseCache
	available  bool
}

// NewCoinGeckoClient creates a new CoinGecko client
func NewCoinGeckoClient(apiKey string, logger *zap.Logger, limiter *datasource.RateLimiter, cache *datasource.ResponseCache) *CoinGeckoClient {
	return &CoinGeckoClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     apiKey,
		logger:     logger,
		limiter:    limiter,
		cache:      cache,
		available:  true,
	}
}

// Name returns the provider name
func (c *CoinGeckoClient) Name() string { return "coingecko" }

// IsAvailable returns true if the provider is not in cooldown and not over monthly cap
func (c *CoinGeckoClient) IsAvailable() bool {
	return c.available && !c.limiter.IsInCooldown() && !c.limiter.IsMonthlyCapReached()
}

// SetAvailable sets the availability flag
func (c *CoinGeckoClient) SetAvailable(available bool) {
	c.available = available
}

// doRequest performs an HTTP request with rate limiting and 429 detection
func (c *CoinGeckoClient) doRequest(ctx context.Context, url string) (*http.Response, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	if c.apiKey != "" {
		req.Header.Set("x-cg-demo-api-key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		resp.Body.Close()
		c.limiter.TriggerCooldown(5 * time.Minute)
		c.logger.Warn("CoinGecko rate limited, entering 5 min cooldown")
		return nil, fmt.Errorf("coingecko: rate limited (429)")
	}

	return resp, nil
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
				Price                    float64            `json:"price"`
				PriceChangePercentage24h map[string]float64 `json:"price_change_percentage_24h"`
				MarketCap                string             `json:"market_cap"`
				TotalVolume              string             `json:"total_volume"`
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
	cacheKey := "cg:trending"
	if cached, ok := c.cache.Get(cacheKey); ok {
		if tokens, ok := cached.([]models.Token); ok {
			return tokens, nil
		}
	}

	url := fmt.Sprintf("%s/search/trending", coingeckoBaseURL)
	resp, err := c.doRequest(ctx, url)
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
			Tags:         []string{"trending", "coingecko"},
		})
	}

	c.cache.Set(cacheKey, tokens, datasource.CacheTTLs.TrendingTokens)
	c.logger.Info("fetched trending tokens", zap.Int("count", len(tokens)))
	return tokens, nil
}

// GetMarketData fetches market data for top tokens
func (c *CoinGeckoClient) GetMarketData(ctx context.Context, limit int) ([]models.Token, error) {
	cacheKey := fmt.Sprintf("cg:market:%d", limit)
	if cached, ok := c.cache.Get(cacheKey); ok {
		if tokens, ok := cached.([]models.Token); ok {
			return tokens, nil
		}
	}

	url := fmt.Sprintf("%s/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=%d&page=1&sparkline=false",
		coingeckoBaseURL, limit)
	resp, err := c.doRequest(ctx, url)
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

	c.cache.Set(cacheKey, tokens, datasource.CacheTTLs.MarketCap)
	c.logger.Info("fetched market data", zap.Int("count", len(tokens)))
	return tokens, nil
}

// GetTokenByContract fetches token info by contract address
func (c *CoinGeckoClient) GetTokenByContract(ctx context.Context, platform, contractAddress string) (*models.Token, error) {
	cacheKey := fmt.Sprintf("cg:contract:%s:%s", platform, contractAddress)
	if cached, ok := c.cache.Get(cacheKey); ok {
		if token, ok := cached.(*models.Token); ok {
			return token, nil
		}
	}

	url := fmt.Sprintf("%s/coins/%s/contract/%s", coingeckoBaseURL, platform, contractAddress)
	resp, err := c.doRequest(ctx, url)
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

	token := &models.Token{
		Address:        contractAddress,
		Symbol:         data.Symbol,
		Name:           data.Name,
		Price:          data.MarketData.CurrentPrice["usd"],
		PriceChange24h: data.MarketData.PriceChangePercentage24h,
		MarketCap:      data.MarketData.MarketCap["usd"],
		Volume24h:      data.MarketData.TotalVolume["usd"],
		DiscoveredAt:   time.Now(),
	}

	c.cache.Set(cacheKey, token, datasource.CacheTTLs.MarketCap)
	return token, nil
}

// GetTrending implements the Provider interface
func (c *CoinGeckoClient) GetTrending(ctx context.Context) ([]models.Token, error) {
	return c.GetTrendingTokens(ctx)
}

// GetNewTokens implements the Provider interface - CoinGecko doesn't have a new tokens endpoint
func (c *CoinGeckoClient) GetNewTokens(ctx context.Context) ([]models.Token, error) {
	return c.GetTrendingTokens(ctx)
}

// GetTopGainers implements the Provider interface
func (c *CoinGeckoClient) GetTopGainers(ctx context.Context) ([]models.Token, error) {
	tokens, err := c.GetMarketData(ctx, 50)
	if err != nil {
		return nil, err
	}
	gainers := make([]models.Token, 0)
	for _, t := range tokens {
		if t.PriceChange24h > 0 {
			gainers = append(gainers, t)
		}
	}
	return gainers, nil
}

// GetTokenPrice implements the Provider interface
func (c *CoinGeckoClient) GetTokenPrice(ctx context.Context, network, address string) (float64, error) {
	token, err := c.GetTokenByContract(ctx, network, address)
	if err != nil {
		return 0, err
	}
	if token == nil {
		return 0, fmt.Errorf("token not found: %s/%s", network, address)
	}
	return token.Price, nil
}

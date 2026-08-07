package datasource

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/twadelij/cryptojackal/internal/models"
	"go.uber.org/zap"
)

const (
	geckoTerminalBaseURL = "https://api.geckoterminal.com/api/v2"
)

// GeckoTerminalClient implements the Provider interface using GeckoTerminal API.
type GeckoTerminalClient struct {
	httpClient *http.Client
	logger     *zap.Logger
	limiter    *RateLimiter
	cache      *ResponseCache
	available  bool
}

// NewGeckoTerminalClient creates a new GeckoTerminal client.
func NewGeckoTerminalClient(logger *zap.Logger, limiter *RateLimiter, cache *ResponseCache) *GeckoTerminalClient {
	return &GeckoTerminalClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     logger,
		limiter:    limiter,
		cache:      cache,
		available:  true,
	}
}

type gtTrendingResponse struct {
	Data []gtPool `json:"data"`
}

type gtPool struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Name            string `json:"name"`
		BaseTokenPrice  string `json:"base_token_price_usd"`
		QuoteTokenPrice string `json:"quote_token_price_usd"`
		Volume          struct {
			H24 float64 `json:"h24"`
		} `json:"volume_usd"`
		Reserve         struct {
			H24 float64 `json:"h24"`
		} `json:"reserve_in_usd"`
		PriceChangePercentage struct {
			H24 float64 `json:"h24"`
		} `json:"price_change_percentage"`
	} `json:"attributes"`
	Relationships struct {
		BaseToken struct {
			Data struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"data"`
		} `json:"base_token"`
		Dex struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		} `json:"dex"`
	} `json:"relationships"`
}

type gtTokenResponse struct {
	Data struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			Symbol         string `json:"symbol"`
			Name           string `json:"name"`
			PriceUSD       string `json:"price_usd"`
			Volume         struct {
				H24 float64 `json:"h24"`
			} `json:"volume_usd"`
			MarketCap      string `json:"market_cap_usd"`
			PriceChangePercentage struct {
				H24 float64 `json:"h24"`
			} `json:"price_change_percentage"`
		} `json:"attributes"`
	} `json:"data"`
}

// Name returns the provider name.
func (g *GeckoTerminalClient) Name() string { return "geckoterminal" }

// IsAvailable returns true if the provider is not in cooldown.
func (g *GeckoTerminalClient) IsAvailable() bool {
	return g.available && !g.limiter.IsInCooldown() && !g.limiter.IsMonthlyCapReached()
}

// SetAvailable sets the availability flag.
func (g *GeckoTerminalClient) SetAvailable(available bool) {
	g.available = available
}

// GetTrending fetches trending pools from GeckoTerminal.
func (g *GeckoTerminalClient) GetTrending(ctx context.Context) ([]models.Token, error) {
	cacheKey := "gt:trending"
	if cached, ok := g.cache.Get(cacheKey); ok {
		if tokens, ok := cached.([]models.Token); ok {
			return tokens, nil
		}
	}

	if err := g.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	url := fmt.Sprintf("%s/networks/trending_pools", geckoTerminalBaseURL)
	resp, err := g.doRequest(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := g.checkResponse(resp); err != nil {
		return nil, err
	}

	var data gtTrendingResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode trending response: %w", err)
	}

	tokens := make([]models.Token, 0, len(data.Data))
	for _, pool := range data.Data {
		token := g.poolToToken(pool)
		tokens = append(tokens, token)
	}

	g.cache.Set(cacheKey, tokens, CacheTTLs.TrendingTokens)
	g.logger.Info("fetched trending pools from GeckoTerminal", zap.Int("count", len(tokens)))
	return tokens, nil
}

// GetNewTokens fetches newly created pools from GeckoTerminal.
func (g *GeckoTerminalClient) GetNewTokens(ctx context.Context) ([]models.Token, error) {
	cacheKey := "gt:new_pools"
	if cached, ok := g.cache.Get(cacheKey); ok {
		if tokens, ok := cached.([]models.Token); ok {
			return tokens, nil
		}
	}

	if err := g.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	url := fmt.Sprintf("%s/networks/new_pools?include=base_token", geckoTerminalBaseURL)
	resp, err := g.doRequest(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := g.checkResponse(resp); err != nil {
		return nil, err
	}

	var data gtTrendingResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode new pools response: %w", err)
	}

	tokens := make([]models.Token, 0, len(data.Data))
	for _, pool := range data.Data {
		token := g.poolToToken(pool)
		tokens = append(tokens, token)
	}

	g.cache.Set(cacheKey, tokens, CacheTTLs.NewPools)
	g.logger.Info("fetched new pools from GeckoTerminal", zap.Int("count", len(tokens)))
	return tokens, nil
}

// GetTokenPrice fetches the current price for a token by address.
func (g *GeckoTerminalClient) GetTokenPrice(ctx context.Context, network, address string) (float64, error) {
	cacheKey := fmt.Sprintf("gt:price:%s:%s", network, address)
	if cached, ok := g.cache.Get(cacheKey); ok {
		if price, ok := cached.(float64); ok {
			return price, nil
		}
	}

	if err := g.limiter.Wait(ctx); err != nil {
		return 0, fmt.Errorf("rate limiter: %w", err)
	}

	url := fmt.Sprintf("%s/networks/%s/tokens/%s", geckoTerminalBaseURL, network, address)
	resp, err := g.doRequest(ctx, url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if err := g.checkResponse(resp); err != nil {
		return 0, err
	}

	var data gtTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, fmt.Errorf("failed to decode token response: %w", err)
	}

	price, _ := strconv.ParseFloat(data.Data.Attributes.PriceUSD, 64)
	g.cache.Set(cacheKey, price, CacheTTLs.TokenPrice)
	return price, nil
}

// GetTopGainers fetches trending pools and filters for positive price change.
func (g *GeckoTerminalClient) GetTopGainers(ctx context.Context) ([]models.Token, error) {
	cacheKey := "gt:top_gainers"
	if cached, ok := g.cache.Get(cacheKey); ok {
		if tokens, ok := cached.([]models.Token); ok {
			return tokens, nil
		}
	}

	tokens, err := g.GetTrending(ctx)
	if err != nil {
		return nil, err
	}

	gainers := make([]models.Token, 0)
	for _, t := range tokens {
		if t.PriceChange24h > 0 {
			gainers = append(gainers, t)
		}
	}

	g.cache.Set(cacheKey, gainers, CacheTTLs.TopGainers)
	return gainers, nil
}

func (g *GeckoTerminalClient) doRequest(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	return g.httpClient.Do(req)
}

func (g *GeckoTerminalClient) checkResponse(resp *http.Response) error {
	if resp.StatusCode == http.StatusTooManyRequests {
		g.limiter.TriggerCooldown(5 * time.Minute)
		g.logger.Warn("GeckoTerminal rate limited, entering 5 min cooldown")
		return fmt.Errorf("geckoterminal: rate limited (429)")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("geckoterminal API error: %d", resp.StatusCode)
	}
	return nil
}

func (g *GeckoTerminalClient) poolToToken(pool gtPool) models.Token {
	price, _ := strconv.ParseFloat(pool.Attributes.BaseTokenPrice, 64)
	tokenID := pool.Relationships.BaseToken.Data.ID
	parts := splitID(tokenID)
	address := tokenID
	symbol := ""
	name := pool.Attributes.Name
	if len(parts) >= 2 {
		address = parts[1]
		symbol = parts[0]
	}

	return models.Token{
		Address:        address,
		Symbol:         symbol,
		Name:           name,
		Price:          price,
		PriceChange24h: pool.Attributes.PriceChangePercentage.H24,
		Volume24h:      pool.Attributes.Volume.H24,
		Liquidity:      pool.Attributes.Reserve.H24,
		DiscoveredAt:   time.Now(),
		Tags:           []string{"geckoterminal", "trending"},
	}
}

// splitID splits a GeckoTerminal token ID like "eth_0x1234..." into ["eth", "0x1234..."]
func splitID(id string) []string {
	for i, c := range id {
		if c == '_' {
			return []string{id[:i], id[i+1:]}
		}
	}
	return []string{id}
}

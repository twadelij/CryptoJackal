package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/twadelij/cryptojackal/internal/datasource"
	"github.com/twadelij/cryptojackal/internal/models"
	"go.uber.org/zap"
)

const (
	dexScreenerBaseURL = "https://api.dexscreener.com"
)

// DexScreenerClient handles DexScreener API interactions
type DexScreenerClient struct {
	httpClient *http.Client
	logger     *zap.Logger
	limiter    *datasource.RateLimiter
	cache      *datasource.ResponseCache
	available  bool
}

// NewDexScreenerClient creates a new DexScreener client
func NewDexScreenerClient(logger *zap.Logger, limiter *datasource.RateLimiter, cache *datasource.ResponseCache) *DexScreenerClient {
	return &DexScreenerClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     logger,
		limiter:    limiter,
		cache:      cache,
		available:  true,
	}
}

// Name returns the provider name
func (d *DexScreenerClient) Name() string { return "dexscreener" }

// IsAvailable returns true if the provider is not in cooldown
func (d *DexScreenerClient) IsAvailable() bool {
	return d.available && !d.limiter.IsInCooldown()
}

// SetAvailable sets the availability flag
func (d *DexScreenerClient) SetAvailable(available bool) {
	d.available = available
}

// ErrRateLimited is returned when the API returns 429
type ErrRateLimited struct{ Source string }

func (e *ErrRateLimited) Error() string { return fmt.Sprintf("%s: rate limited (429)", e.Source) }

// doRequestWithBackoff performs an HTTP request with rate limiting and exponential backoff on 429
func (d *DexScreenerClient) doRequestWithBackoff(ctx context.Context, endpoint string) (*http.Response, error) {
	if err := d.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	backoff := 2 * time.Second
	maxAttempts := 3

	for attempt := 0; attempt < maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
		if err != nil {
			return nil, err
		}

		resp, err := d.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			if attempt < maxAttempts-1 {
				d.logger.Warn("DexScreener 429, backing off",
					zap.Int("attempt", attempt+1),
					zap.Duration("backoff", backoff))
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(backoff):
				}
				backoff *= 2
				continue
			}
			// Final attempt failed, trigger cooldown
			d.limiter.TriggerCooldown(5 * time.Minute)
			d.logger.Warn("DexScreener rate limited after retries, entering 5 min cooldown")
			return nil, &ErrRateLimited{Source: "dexscreener"}
		}

		return resp, nil
	}

	return nil, fmt.Errorf("dexscreener: max retries exceeded")
}

type dexScreenerResponse struct {
	Pairs []dexScreenerPair `json:"pairs"`
}

type dexScreenerPair struct {
	ChainID     string   `json:"chainId"`
	DexID       string   `json:"dexId"`
	PairAddress string   `json:"pairAddress"`
	BaseToken   dexToken `json:"baseToken"`
	QuoteToken  dexToken `json:"quoteToken"`
	PriceNative string   `json:"priceNative"`
	PriceUSD    string   `json:"priceUsd"`
	Liquidity   struct {
		USD float64 `json:"usd"`
	} `json:"liquidity"`
	Volume struct {
		H24 float64 `json:"h24"`
	} `json:"volume"`
	PriceChange struct {
		H24 float64 `json:"h24"`
	} `json:"priceChange"`
	TxCount struct {
		H24 map[string]int `json:"h24"`
	} `json:"txns"`
}

type dexToken struct {
	Address string `json:"address"`
	Name    string `json:"name"`
	Symbol  string `json:"symbol"`
}

// GetPairsByToken fetches pools/pairs for a given token address.
// Docs: GET /latest/dex/tokens/{tokenAddress}
func (d *DexScreenerClient) GetPairsByToken(ctx context.Context, address string) ([]dexScreenerPair, error) {
	cacheKey := fmt.Sprintf("ds:pairs:%s", address)
	if cached, ok := d.cache.Get(cacheKey); ok {
		if pairs, ok := cached.([]dexScreenerPair); ok {
			return pairs, nil
		}
	}

	endpoint := fmt.Sprintf("%s/latest/dex/tokens/%s", dexScreenerBaseURL, address)
	resp, err := d.doRequestWithBackoff(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch token pairs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DexScreener API error: %d", resp.StatusCode)
	}

	var data dexScreenerResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	d.cache.Set(cacheKey, data.Pairs, datasource.CacheTTLs.TokenPrice)
	return data.Pairs, nil
}

// SearchPairs searches for pairs matching a query.
// Docs: GET /latest/dex/search?q={query}
func (d *DexScreenerClient) SearchPairs(ctx context.Context, query string) ([]dexScreenerPair, error) {
	q := url.QueryEscape(strings.TrimSpace(query))
	endpoint := fmt.Sprintf("%s/latest/dex/search?q=%s", dexScreenerBaseURL, q)
	resp, err := d.doRequestWithBackoff(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to search pairs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DexScreener API error: %d", resp.StatusCode)
	}

	var data dexScreenerResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return data.Pairs, nil
}

// GetBoostedTokens fetches recently boosted tokens (proxy for new/hot tokens).
// Docs: GET /token-boosts/latest/v1
func (d *DexScreenerClient) GetBoostedTokens(ctx context.Context) ([]models.Token, error) {
	cacheKey := "ds:boosted"
	if cached, ok := d.cache.Get(cacheKey); ok {
		if tokens, ok := cached.([]models.Token); ok {
			return tokens, nil
		}
	}

	endpoint := fmt.Sprintf("%s/token-boosts/latest/v1", dexScreenerBaseURL)
	resp, err := d.doRequestWithBackoff(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch boosted tokens: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DexScreener API error: %d", resp.StatusCode)
	}

	var boosts []struct {
		URL          string `json:"url"`
		ChainID      string `json:"chainId"`
		TokenAddress string `json:"tokenAddress"`
		Amount       int    `json:"amount"`
		TotalAmount  int    `json:"totalAmount"`
		Icon         string `json:"icon"`
		Header       string `json:"header"`
		Description  string `json:"description"`
		Links        []struct {
			Type  string `json:"type"`
			Label string `json:"label"`
			URL   string `json:"url"`
		} `json:"links"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&boosts); err != nil {
		return nil, fmt.Errorf("failed to decode boosted tokens: %w", err)
	}

	// Limit pair lookups to first 10 boosted tokens to avoid N+1 API calls
	maxLookups := 10
	if len(boosts) > maxLookups {
		boosts = boosts[:maxLookups]
	}

	tokens := make([]models.Token, 0, len(boosts))
	for _, boost := range boosts {
		pairs, err := d.GetPairsByToken(ctx, boost.TokenAddress)
		if err != nil {
			d.logger.Warn("failed to get pairs for boosted token", zap.String("address", boost.TokenAddress), zap.Error(err))
			continue
		}

		if len(pairs) == 0 {
			continue
		}

		pair := pairs[0]
		var price float64
		if p, err := strconv.ParseFloat(pair.PriceUSD, 64); err == nil {
			price = p
		}

		tokens = append(tokens, models.Token{
			Address:        pair.BaseToken.Address,
			Symbol:         pair.BaseToken.Symbol,
			Name:           pair.BaseToken.Name,
			Price:          price,
			PriceChange24h: pair.PriceChange.H24,
			Volume24h:      pair.Volume.H24,
			Liquidity:      pair.Liquidity.USD,
			DiscoveredAt:   time.Now(),
			Tags:           []string{"dexscreener", "boosted", pair.DexID},
		})
	}

	d.cache.Set(cacheKey, tokens, datasource.CacheTTLs.TrendingTokens)
	d.logger.Info("fetched boosted tokens from DexScreener", zap.Int("count", len(tokens)))
	return tokens, nil
}

// GetNewTokens fetches newly created/boosted tokens (alias for GetBoostedTokens)
func (d *DexScreenerClient) GetNewPairs(ctx context.Context, chain string) ([]models.Token, error) {
	// DexScreener doesn't have a direct "new pairs by chain" endpoint
	// Use boosted tokens as a proxy for new/hot tokens
	return d.GetBoostedTokens(ctx)
}

// SearchToken searches for a token by address
func (d *DexScreenerClient) SearchToken(ctx context.Context, address string) ([]models.Token, error) {
	pairs, err := d.GetPairsByToken(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("failed to search token: %w", err)
	}

	tokens := make([]models.Token, 0, len(pairs))
	seen := make(map[string]bool)

	for _, pair := range pairs {
		if seen[pair.BaseToken.Address] {
			continue
		}
		seen[pair.BaseToken.Address] = true

		var price float64
		if p, err := strconv.ParseFloat(pair.PriceUSD, 64); err == nil {
			price = p
		}

		tokens = append(tokens, models.Token{
			Address:        pair.BaseToken.Address,
			Symbol:         pair.BaseToken.Symbol,
			Name:           pair.BaseToken.Name,
			Price:          price,
			PriceChange24h: pair.PriceChange.H24,
			Volume24h:      pair.Volume.H24,
			Liquidity:      pair.Liquidity.USD,
			DiscoveredAt:   time.Now(),
			Tags:           []string{"dexscreener"},
		})
	}

	return tokens, nil
}

// GetTopGainersFiltered fetches tokens with highest price gains, filtered by chain and liquidity
func (d *DexScreenerClient) GetTopGainersFiltered(ctx context.Context, chain string, minLiquidity float64) ([]models.Token, error) {
	tokens, err := d.GetBoostedTokens(ctx)
	if err != nil {
		return nil, err
	}

	gainers := make([]models.Token, 0)
	for _, token := range tokens {
		if token.Liquidity >= minLiquidity && token.PriceChange24h > 0 {
			gainers = append(gainers, token)
		}
	}

	sort.Slice(gainers, func(i, j int) bool {
		return gainers[i].PriceChange24h > gainers[j].PriceChange24h
	})

	if len(gainers) > 20 {
		gainers = gainers[:20]
	}

	return gainers, nil
}

// GetTrending implements the Provider interface - uses boosted tokens as trending
func (d *DexScreenerClient) GetTrending(ctx context.Context) ([]models.Token, error) {
	return d.GetBoostedTokens(ctx)
}

// GetNewTokens implements the Provider interface - uses boosted tokens as new tokens
func (d *DexScreenerClient) GetNewTokens(ctx context.Context) ([]models.Token, error) {
	return d.GetBoostedTokens(ctx)
}

// GetTopGainers implements the Provider interface version without extra params
func (d *DexScreenerClient) GetTopGainers(ctx context.Context) ([]models.Token, error) {
	return d.GetTopGainersFiltered(ctx, "", 0)
}

// GetTokenPrice implements the Provider interface
func (d *DexScreenerClient) GetTokenPrice(ctx context.Context, network, address string) (float64, error) {
	cacheKey := fmt.Sprintf("ds:price:%s", address)
	if cached, ok := d.cache.Get(cacheKey); ok {
		if price, ok := cached.(float64); ok {
			return price, nil
		}
	}

	pairs, err := d.GetPairsByToken(ctx, address)
	if err != nil {
		return 0, fmt.Errorf("failed to get token price: %w", err)
	}
	if len(pairs) == 0 {
		return 0, fmt.Errorf("no pairs found for token %s", address)
	}

	price, _ := strconv.ParseFloat(pairs[0].PriceUSD, 64)
	d.cache.Set(cacheKey, price, datasource.CacheTTLs.TokenPrice)
	return price, nil
}

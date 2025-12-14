package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

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
}

// NewDexScreenerClient creates a new DexScreener client
func NewDexScreenerClient(logger *zap.Logger) *DexScreenerClient {
	return &DexScreenerClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     logger,
	}
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
	endpoint := fmt.Sprintf("%s/latest/dex/tokens/%s", dexScreenerBaseURL, address)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := d.httpClient.Do(req)
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

	return data.Pairs, nil
}

// SearchPairs searches for pairs matching a query.
// Docs: GET /latest/dex/search?q={query}
func (d *DexScreenerClient) SearchPairs(ctx context.Context, query string) ([]dexScreenerPair, error) {
	q := url.QueryEscape(strings.TrimSpace(query))
	endpoint := fmt.Sprintf("%s/latest/dex/search?q=%s", dexScreenerBaseURL, q)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := d.httpClient.Do(req)
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
	endpoint := fmt.Sprintf("%s/token-boosts/latest/v1", dexScreenerBaseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := d.httpClient.Do(req)
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

	// Get detailed pair info for each boosted token
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

		// Use the first pair for token info
		pair := pairs[0]
		var price float64
		fmt.Sscanf(pair.PriceUSD, "%f", &price)

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
		fmt.Sscanf(pair.PriceUSD, "%f", &price)

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

// GetTopGainers fetches tokens with highest price gains
func (d *DexScreenerClient) GetTopGainers(ctx context.Context, chain string, minLiquidity float64) ([]models.Token, error) {
	tokens, err := d.GetBoostedTokens(ctx)
	if err != nil {
		return nil, err
	}

	// Filter by liquidity and sort by price change
	gainers := make([]models.Token, 0)
	for _, token := range tokens {
		if token.Liquidity >= minLiquidity && token.PriceChange24h > 0 {
			gainers = append(gainers, token)
		}
	}

	// Sort by price change (descending)
	sort.Slice(gainers, func(i, j int) bool {
		return gainers[i].PriceChange24h > gainers[j].PriceChange24h
	})

	if len(gainers) > 20 {
		gainers = gainers[:20]
	}

	return gainers, nil
}

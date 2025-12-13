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
	dexScreenerBaseURL = "https://api.dexscreener.com/latest"
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
	ChainID       string `json:"chainId"`
	DexID         string `json:"dexId"`
	PairAddress   string `json:"pairAddress"`
	BaseToken     dexToken `json:"baseToken"`
	QuoteToken    dexToken `json:"quoteToken"`
	PriceNative   string `json:"priceNative"`
	PriceUSD      string `json:"priceUsd"`
	Liquidity     struct {
		USD float64 `json:"usd"`
	} `json:"liquidity"`
	Volume struct {
		H24 float64 `json:"h24"`
	} `json:"volume"`
	PriceChange struct {
		H24 float64 `json:"h24"`
	} `json:"priceChange"`
	TxCount struct {
		H24 int `json:"h24"`
	} `json:"txns"`
}

type dexToken struct {
	Address string `json:"address"`
	Name    string `json:"name"`
	Symbol  string `json:"symbol"`
}

// GetNewPairs fetches newly created pairs
func (d *DexScreenerClient) GetNewPairs(ctx context.Context, chain string) ([]models.Token, error) {
	url := fmt.Sprintf("%s/dex/pairs/%s", dexScreenerBaseURL, chain)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pairs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DexScreener API error: %d", resp.StatusCode)
	}

	var data dexScreenerResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	tokens := make([]models.Token, 0, len(data.Pairs))
	for _, pair := range data.Pairs {
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
			Tags:           []string{"dexscreener", pair.DexID},
		})
	}

	d.logger.Info("fetched pairs from DexScreener", zap.String("chain", chain), zap.Int("count", len(tokens)))
	return tokens, nil
}

// SearchToken searches for a token by address
func (d *DexScreenerClient) SearchToken(ctx context.Context, address string) ([]models.Token, error) {
	url := fmt.Sprintf("%s/dex/tokens/%s", dexScreenerBaseURL, address)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DexScreener API error: %d", resp.StatusCode)
	}

	var data dexScreenerResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	tokens := make([]models.Token, 0, len(data.Pairs))
	seen := make(map[string]bool)
	
	for _, pair := range data.Pairs {
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
	tokens, err := d.GetNewPairs(ctx, chain)
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

	// Simple bubble sort by price change (descending)
	for i := 0; i < len(gainers)-1; i++ {
		for j := 0; j < len(gainers)-i-1; j++ {
			if gainers[j].PriceChange24h < gainers[j+1].PriceChange24h {
				gainers[j], gainers[j+1] = gainers[j+1], gainers[j]
			}
		}
	}

	if len(gainers) > 20 {
		gainers = gainers[:20]
	}

	return gainers, nil
}

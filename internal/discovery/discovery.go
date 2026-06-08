package discovery

import (
	"context"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/twadelij/cryptojackal/internal/models"
	"go.uber.org/zap"
)

// Service manages token discovery from multiple sources
type Service struct {
	coingecko    *CoinGeckoClient
	dexscreener  *DexScreenerClient
	logger       *zap.Logger
	
	// Cache
	mu           sync.RWMutex
	trendingCache []models.Token
	cacheTime    time.Time
	cacheTTL     time.Duration
	
	// Demo state for fallback tokens
	demoTokens   []models.Token
	demoSeed     int64
}

// HealthStatus represents the health of external APIs
type HealthStatus struct {
	CoinGecko   bool `json:"coingecko"`
	DexScreener bool `json:"dexscreener"`
}

// NewService creates a new discovery service
func NewService(coingeckoAPIKey string, logger *zap.Logger) *Service {
	s := &Service{
		coingecko:   NewCoinGeckoClient(coingeckoAPIKey, logger),
		dexscreener: NewDexScreenerClient(logger),
		logger:      logger,
		cacheTTL:    5 * time.Minute,
		demoSeed:    time.Now().Unix(),
	}
	s.demoTokens = s.initialDemoTokens()
	return s
}

// Health checks if external APIs are reachable
func (s *Service) Health(ctx context.Context) HealthStatus {
	status := HealthStatus{}

	// Check CoinGecko with a quick ping
	cgCtx, cgCancel := context.WithTimeout(ctx, 5*time.Second)
	defer cgCancel()
	_, cgErr := s.coingecko.GetTrendingTokens(cgCtx)
	status.CoinGecko = cgErr == nil

	// Check DexScreener with a quick ping
	dsCtx, dsCancel := context.WithTimeout(ctx, 5*time.Second)
	defer dsCancel()
	_, dsErr := s.dexscreener.GetBoostedTokens(dsCtx)
	status.DexScreener = dsErr == nil

	return status
}

// GetTrendingTokens returns trending tokens (cached)
func (s *Service) GetTrendingTokens(ctx context.Context) ([]models.Token, error) {
	s.mu.RLock()
	if time.Since(s.cacheTime) < s.cacheTTL && len(s.trendingCache) > 0 {
		tokens := s.trendingCache
		s.mu.RUnlock()
		return tokens, nil
	}
	s.mu.RUnlock()

	tokens, err := s.coingecko.GetTrendingTokens(ctx)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.trendingCache = tokens
	s.cacheTime = time.Now()
	s.mu.Unlock()

	return tokens, nil
}

// GetNewTokens discovers new tokens from DexScreener
func (s *Service) GetNewTokens(ctx context.Context, chain string) ([]models.Token, error) {
	return s.dexscreener.GetNewPairs(ctx, chain)
}

// GetTopGainers returns top gaining tokens
func (s *Service) GetTopGainers(ctx context.Context, chain string, minLiquidity float64) ([]models.Token, error) {
	return s.dexscreener.GetTopGainers(ctx, chain, minLiquidity)
}

// AnalyzeToken analyzes a specific token
func (s *Service) AnalyzeToken(ctx context.Context, address string) (*models.Token, error) {
	// Try DexScreener first
	tokens, err := s.dexscreener.SearchToken(ctx, address)
	if err == nil && len(tokens) > 0 {
		token := tokens[0]
		// Calculate a basic security score
		token.SecurityScore = s.calculateSecurityScore(&token)
		return &token, nil
	}

	// Fallback to CoinGecko
	token, err := s.coingecko.GetTokenByContract(ctx, "ethereum", address)
	if err != nil {
		return nil, err
	}
	if token != nil {
		token.SecurityScore = s.calculateSecurityScore(token)
	}
	return token, nil
}

// calculateSecurityScore calculates a basic security score for a token
func (s *Service) calculateSecurityScore(token *models.Token) float64 {
	score := 0.5 // Base score

	// Higher liquidity = higher score
	if token.Liquidity > 100000 {
		score += 0.2
	} else if token.Liquidity > 50000 {
		score += 0.1
	}

	// Higher volume = higher score
	if token.Volume24h > 100000 {
		score += 0.15
	} else if token.Volume24h > 50000 {
		score += 0.1
	}

	// Market cap presence is good
	if token.MarketCap > 0 {
		score += 0.1
	}

	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// FindOpportunities scans for trading opportunities
func (s *Service) FindOpportunities(ctx context.Context, chain string, minLiquidity float64) ([]models.TradingOpportunity, error) {
	tokens, err := s.GetTopGainers(ctx, chain, minLiquidity)
	if err != nil {
		s.logger.Warn("API failed, using fallback demo tokens", zap.Error(err))
		tokens = s.fallbackTokens()
	}

	opportunities := make([]models.TradingOpportunity, 0)
	for _, token := range tokens {
		// Simple opportunity detection based on momentum
		if token.PriceChange24h > 5 && token.Liquidity > minLiquidity {
			confidence := 0.5
			if token.PriceChange24h > 20 {
				confidence = 0.7
			}
			if token.Volume24h > 100000 {
				confidence += 0.1
			}

			opp := models.NewOpportunity(
				token,
				token.PriceChange24h * 0.1, // Expected 10% of current momentum
				0.01, // 1% price impact estimate
				confidence,
				"momentum",
			)
			opportunities = append(opportunities, *opp)
		}
	}

	s.logger.Info("found trading opportunities", zap.Int("count", len(opportunities)))
	return opportunities, nil
}

// fallbackTokens returns demo tokens when external APIs are unavailable
// Prices fluctuate slightly each call for a dynamic demo feel
func (s *Service) fallbackTokens() []models.Token {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Update prices with small random variation (-5% to +5%)
	s.demoSeed++
	r := rand.New(rand.NewSource(s.demoSeed))
	for i := range s.demoTokens {
		variation := 1.0 + (r.Float64()*0.1 - 0.05) // +/- 5%
		s.demoTokens[i].Price = math.Max(0.000001, s.demoTokens[i].Price*variation)
		s.demoTokens[i].PriceChange24h = s.demoTokens[i].PriceChange24h + (r.Float64()*4 - 2)
		s.demoTokens[i].Volume24h = math.Max(1000, s.demoTokens[i].Volume24h*(1.0+(r.Float64()*0.06-0.03)))
	}

	result := make([]models.Token, len(s.demoTokens))
	copy(result, s.demoTokens)
	return result
}

// initialDemoTokens creates the base set of demo tokens
func (s *Service) initialDemoTokens() []models.Token {
	return []models.Token{
		{
			Address:        "0xDOJI1234567890abcdef",
			Symbol:         "DOJI",
			Name:           "DOJI/Degen",
			Price:          0.000004,
			PriceChange24h: 25.5,
			Volume24h:      150000,
			Liquidity:      500000,
			Tags:           []string{"demo", "meme"},
		},
		{
			Address:        "0xBOUNTREE1234567890ab",
			Symbol:         "BOUNTREE",
			Name:           "Bountree",
			Price:          0.0000099,
			PriceChange24h: 15.2,
			Volume24h:      85000,
			Liquidity:      300000,
			Tags:           []string{"demo", "defi"},
		},
		{
			Address:        "0xCOMPANY1234567890ab",
			Symbol:         "Company",
			Name:           "The Company",
			Price:          0.000006,
			PriceChange24h: 32.1,
			Volume24h:      200000,
			Liquidity:      800000,
			Tags:           []string{"demo", "utility"},
		},
		{
			Address:        "0xDAB1234567890abcdef",
			Symbol:         "DAB",
			Name:           "Official DAB Coin",
			Price:          0.000004,
			PriceChange24h: 18.7,
			Volume24h:      120000,
			Liquidity:      450000,
			Tags:           []string{"demo", "meme"},
		},
	}
}

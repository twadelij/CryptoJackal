package discovery

import (
	"context"
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
}

// NewService creates a new discovery service
func NewService(coingeckoAPIKey string, logger *zap.Logger) *Service {
	return &Service{
		coingecko:   NewCoinGeckoClient(coingeckoAPIKey, logger),
		dexscreener: NewDexScreenerClient(logger),
		logger:      logger,
		cacheTTL:    5 * time.Minute,
	}
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
		return nil, err
	}

	opportunities := make([]models.TradingOpportunity, 0)
	for _, token := range tokens {
		// Simple opportunity detection based on momentum
		if token.PriceChange24h > 10 && token.Liquidity > minLiquidity {
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

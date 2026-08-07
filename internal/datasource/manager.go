package datasource

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/twadelij/cryptojackal/internal/models"
	"go.uber.org/zap"
)

// Provider is the interface that all data sources must implement.
type Provider interface {
	GetTrending(ctx context.Context) ([]models.Token, error)
	GetNewTokens(ctx context.Context) ([]models.Token, error)
	GetTopGainers(ctx context.Context) ([]models.Token, error)
	GetTokenPrice(ctx context.Context, network, address string) (float64, error)
	Name() string
	IsAvailable() bool
}

// ProviderManager manages multiple data sources with automatic failover.
// It tries providers in priority order and falls back to the next on error.
type ProviderManager struct {
	mu        sync.RWMutex
	providers []Provider
	limiter   *RateLimiter
	cache     *ResponseCache
	logger    *zap.Logger

	// stats
	callStats map[string]*providerStats
}

type providerStats struct {
	totalCalls    int
	successfulCalls int
	failedCalls   int
	lastError     string
	lastErrorAt   time.Time
}

// NewProviderManager creates a new ProviderManager with the given providers in priority order.
func NewProviderManager(logger *zap.Logger, cache *ResponseCache, providers ...Provider) *ProviderManager {
	stats := make(map[string]*providerStats)
	for _, p := range providers {
		stats[p.Name()] = &providerStats{}
	}
	return &ProviderManager{
		providers: providers,
		cache:     cache,
		logger:    logger,
		callStats: stats,
	}
}

// GetTrending fetches trending tokens from the first available provider.
func (pm *ProviderManager) GetTrending(ctx context.Context) ([]models.Token, error) {
	return pm.tryProviders(ctx, "GetTrending", func(p Provider) ([]models.Token, error) {
		return p.GetTrending(ctx)
	})
}

// GetNewTokens fetches new tokens from the first available provider.
func (pm *ProviderManager) GetNewTokens(ctx context.Context) ([]models.Token, error) {
	return pm.tryProviders(ctx, "GetNewTokens", func(p Provider) ([]models.Token, error) {
		return p.GetNewTokens(ctx)
	})
}

// GetTopGainers fetches top gainers from the first available provider.
func (pm *ProviderManager) GetTopGainers(ctx context.Context) ([]models.Token, error) {
	return pm.tryProviders(ctx, "GetTopGainers", func(p Provider) ([]models.Token, error) {
		return p.GetTopGainers(ctx)
	})
}

// GetTokenPrice fetches the current price for a token from the first available provider.
func (pm *ProviderManager) GetTokenPrice(ctx context.Context, network, address string) (float64, error) {
	cacheKey := fmt.Sprintf("pm:price:%s:%s", network, address)
	if cached, ok := pm.cache.Get(cacheKey); ok {
		if price, ok := cached.(float64); ok {
			return price, nil
		}
	}

	for _, p := range pm.providers {
		if !p.IsAvailable() {
			continue
		}
		pm.recordCall(p.Name())

		price, err := p.GetTokenPrice(ctx, network, address)
		if err != nil {
			pm.recordFailure(p.Name(), err.Error())
			pm.logger.Warn("provider failed for GetTokenPrice, trying next",
				zap.String("provider", p.Name()),
				zap.Error(err))
			continue
		}
		pm.recordSuccess(p.Name())
		pm.cache.Set(cacheKey, price, CacheTTLs.TokenPrice)
		return price, nil
	}

	return 0, fmt.Errorf("all providers failed for GetTokenPrice")
}

// tryProviders iterates through providers in order, returning the first successful result.
func (pm *ProviderManager) tryProviders(ctx context.Context, method string, fn func(Provider) ([]models.Token, error)) ([]models.Token, error) {
	var lastErr error

	for _, p := range pm.providers {
		if !p.IsAvailable() {
			continue
		}
		pm.recordCall(p.Name())

		tokens, err := fn(p)
		if err != nil {
			pm.recordFailure(p.Name(), err.Error())
			lastErr = err
			pm.logger.Warn("provider failed, trying next",
				zap.String("provider", p.Name()),
				zap.String("method", method),
				zap.Error(err))
			continue
		}
		pm.recordSuccess(p.Name())
		return tokens, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all providers failed for %s: %w", method, lastErr)
	}
	return []models.Token{}, nil
}

// GetProviderStatus returns the status of all providers.
func (pm *ProviderManager) GetProviderStatus() []ProviderStatus {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	statuses := make([]ProviderStatus, 0, len(pm.providers))
	for _, p := range pm.providers {
		stats := pm.callStats[p.Name()]
		statuses = append(statuses, ProviderStatus{
			Name:            p.Name(),
			Available:       p.IsAvailable(),
			TotalCalls:      stats.totalCalls,
			SuccessfulCalls: stats.successfulCalls,
			FailedCalls:     stats.failedCalls,
			LastError:       stats.lastError,
			LastErrorAt:     stats.lastErrorAt,
		})
	}
	return statuses
}

// ProviderStatus represents the current status of a data provider.
type ProviderStatus struct {
	Name            string    `json:"name"`
	Available       bool      `json:"available"`
	TotalCalls      int       `json:"total_calls"`
	SuccessfulCalls int       `json:"successful_calls"`
	FailedCalls     int       `json:"failed_calls"`
	LastError       string    `json:"last_error,omitempty"`
	LastErrorAt     time.Time `json:"last_error_at,omitempty"`
}

func (pm *ProviderManager) recordCall(name string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if stats, ok := pm.callStats[name]; ok {
		stats.totalCalls++
	}
}

func (pm *ProviderManager) recordSuccess(name string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if stats, ok := pm.callStats[name]; ok {
		stats.successfulCalls++
	}
}

func (pm *ProviderManager) recordFailure(name, errMsg string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if stats, ok := pm.callStats[name]; ok {
		stats.failedCalls++
		stats.lastError = errMsg
		stats.lastErrorAt = time.Now()
	}
}

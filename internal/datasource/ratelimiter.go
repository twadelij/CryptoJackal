package datasource

import (
	"context"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter per data source.
// Each source gets its own bucket with a configurable refill rate.
type RateLimiter struct {
	mu sync.Mutex

	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time

	// cooldown tracking
	inCooldown   bool
	cooldownUntil time.Time

	// monthly call tracking (for CoinGecko cap)
	monthlyCount   int
	monthlyResetAt time.Time
	monthlyCap     int // 0 = no cap

	// request count for logging
	hourlyCount   int
	hourlyResetAt time.Time
}

// NewRateLimiter creates a rate limiter with the given requests-per-minute limit.
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	rps := float64(requestsPerMinute) / 60.0
	return &RateLimiter{
		tokens:         float64(requestsPerMinute), // start full
		maxTokens:      float64(requestsPerMinute),
		refillRate:     rps,
		lastRefill:     time.Now(),
		monthlyResetAt: time.Now().AddDate(0, 1, 0),
		hourlyResetAt:  time.Now().Add(1 * time.Hour),
	}
}

// SetMonthlyCap sets a monthly call cap. 0 = no cap.
func (r *RateLimiter) SetMonthlyCap(cap int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.monthlyCap = cap
}

// Wait blocks until a token is available or ctx is cancelled.
func (r *RateLimiter) Wait(ctx context.Context) error {
	r.mu.Lock()
	r.refill()

	if r.inCooldown {
		if time.Now().Before(r.cooldownUntil) {
			wait := time.Until(r.cooldownUntil)
			r.mu.Unlock()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wait):
			}
			r.mu.Lock()
		} else {
			r.inCooldown = false
		}
	}

	if r.tokens >= 1.0 {
		r.tokens--
		r.trackCall()
		r.mu.Unlock()
		return nil
	}

	// Not enough tokens, wait for refill
	needed := 1.0 - r.tokens
	waitDuration := time.Duration(needed/r.refillRate*float64(time.Second))
	r.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitDuration):
	}

	// Retry after waiting
	r.mu.Lock()
	defer r.mu.Unlock()
	r.refill()
	if r.tokens >= 1.0 {
		r.tokens--
		r.trackCall()
		return nil
	}
	return nil // best effort
}

// TriggerCooldown puts this source in cooldown for the given duration.
func (r *RateLimiter) TriggerCooldown(duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.inCooldown = true
	r.cooldownUntil = time.Now().Add(duration)
}

// IsInCooldown returns true if the source is currently in cooldown.
func (r *RateLimiter) IsInCooldown() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.inCooldown && time.Now().After(r.cooldownUntil) {
		r.inCooldown = false
	}
	return r.inCooldown
}

// GetMonthlyCallCount returns the number of calls this month.
func (r *RateLimiter) GetMonthlyCallCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checkMonthlyReset()
	return r.monthlyCount
}

// GetHourlyCallCount returns the number of calls in the current hour.
func (r *RateLimiter) GetHourlyCallCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checkHourlyReset()
	return r.hourlyCount
}

// IsMonthlyCapReached returns true if the monthly cap has been reached.
func (r *RateLimiter) IsMonthlyCapReached() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checkMonthlyReset()
	return r.monthlyCap > 0 && r.monthlyCount >= r.monthlyCap
}

func (r *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(r.lastRefill)
	r.tokens += r.refillRate * elapsed.Seconds()
	if r.tokens > r.maxTokens {
		r.tokens = r.maxTokens
	}
	r.lastRefill = now
}

func (r *RateLimiter) trackCall() {
	r.checkMonthlyReset()
	r.checkHourlyReset()
	r.monthlyCount++
	r.hourlyCount++
}

func (r *RateLimiter) checkMonthlyReset() {
	if time.Now().After(r.monthlyResetAt) {
		r.monthlyCount = 0
		r.monthlyResetAt = time.Now().AddDate(0, 1, 0)
	}
}

func (r *RateLimiter) checkHourlyReset() {
	if time.Now().After(r.hourlyResetAt) {
		r.hourlyCount = 0
		r.hourlyResetAt = time.Now().Add(1 * time.Hour)
	}
}

// TierLimits defines rate limits per API tier.
type TierLimits struct {
	DexScreener    int
	GeckoTerminal  int
	CoinGecko      int
	CoinGeckoCap   int // monthly cap, 0 = none
	ScanIntervalSec int
}

// GetTierLimits returns rate limits for the given API tier.
func GetTierLimits(tier string) TierLimits {
	switch tier {
	case "basic":
		return TierLimits{
			DexScreener:     250,
			GeckoTerminal:   240,
			CoinGecko:       280,
			CoinGeckoCap:    100000,
			ScanIntervalSec: 60,
		}
	case "analyst":
		return TierLimits{
			DexScreener:     250,
			GeckoTerminal:   480,
			CoinGecko:       480,
			CoinGeckoCap:    500000,
			ScanIntervalSec: 30,
		}
	default: // "free"
		return TierLimits{
			DexScreener:     250,
			GeckoTerminal:   25,
			CoinGecko:       50,
			CoinGeckoCap:    10000,
			ScanIntervalSec: 180,
		}
	}
}

package provider

import (
	"context"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter for API calls.
type RateLimiter struct {
	mu             sync.Mutex
	tokens         int
	maxTokens      int
	refillInterval time.Duration
	lastRefill     time.Time
}

// NewRateLimiter creates a limiter that allows maxTokens calls per refillInterval.
func NewRateLimiter(maxTokens int, refillInterval time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:         maxTokens,
		maxTokens:      maxTokens,
		refillInterval: refillInterval,
		lastRefill:     time.Now(),
	}
}

// Wait blocks until a token is available or ctx is cancelled.
func (r *RateLimiter) Wait(ctx context.Context) error {
	for {
		r.mu.Lock()
		r.refill()
		if r.tokens > 0 {
			r.tokens--
			r.mu.Unlock()
			return nil
		}
		r.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(r.refillInterval):
		}
	}
}

func (r *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(r.lastRefill)
	newTokens := int(elapsed / r.refillInterval)
	if newTokens > 0 {
		r.tokens += newTokens
		if r.tokens > r.maxTokens {
			r.tokens = r.maxTokens
		}
		r.lastRefill = r.lastRefill.Add(time.Duration(newTokens) * r.refillInterval)
	}
}

package provider

import (
	"context"
	"testing"
	"time"
)

func TestRateLimiterAllowsBurst(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)
	ctx := context.Background()

	start := time.Now()
	if err := limiter.Wait(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := limiter.Wait(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if time.Since(start) > 10*time.Millisecond {
		t.Fatalf("burst waits should return immediately")
	}
}

func TestRateLimiterRefill(t *testing.T) {
	limiter := NewRateLimiter(1, 5*time.Millisecond)
	ctx := context.Background()

	if err := limiter.Wait(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := limiter.Wait(ctx); err != nil {
		t.Fatalf("expected token after refill, got %v", err)
	}
}

func TestRateLimiterHonorsContext(t *testing.T) {
	limiter := NewRateLimiter(1, time.Second)
	ctx := context.Background()
	_ = limiter.Wait(ctx)

	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
	defer cancel()

	start := time.Now()
	if err := limiter.Wait(timeoutCtx); err == nil {
		t.Fatal("expected context deadline error")
	}
	if time.Since(start) > 200*time.Millisecond {
		t.Fatalf("wait should stop after context cancellation")
	}
}

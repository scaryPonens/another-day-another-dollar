package cache

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestInitRedisWithCustomAddr(t *testing.T) {
	t.Setenv("REDIS_URL", "redis:9999")

	origNewClient := newRedisClient
	origPing := pingRedis
	t.Cleanup(func() {
		newRedisClient = origNewClient
		pingRedis = origPing
		Client = nil
	})

	var capturedAddr string
	newRedisClient = func(opts *redis.Options) *redis.Client {
		capturedAddr = opts.Addr
		return redis.NewClient(opts)
	}
	pingRedis = func(ctx context.Context, client *redis.Client) error {
		return nil
	}

	InitRedis(context.Background())
	if capturedAddr != "redis:9999" {
		t.Fatalf("expected custom addr, got %s", capturedAddr)
	}
}

func TestInitRedisDefaults(t *testing.T) {
	t.Setenv("REDIS_URL", "")

	origNewClient := newRedisClient
	origPing := pingRedis
	t.Cleanup(func() {
		newRedisClient = origNewClient
		pingRedis = origPing
		Client = nil
	})

	var capturedAddr string
	newRedisClient = func(opts *redis.Options) *redis.Client {
		capturedAddr = opts.Addr
		return redis.NewClient(opts)
	}
	pingRedis = func(ctx context.Context, client *redis.Client) error {
		return nil
	}

	InitRedis(context.Background())
	if capturedAddr != "localhost:6379" {
		t.Fatalf("expected default addr, got %s", capturedAddr)
	}
}

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
	origParse := parseRedisURL
	t.Cleanup(func() {
		newRedisClient = origNewClient
		pingRedis = origPing
		parseRedisURL = origParse
		Client = nil
	})

	var capturedAddr string
	newRedisClient = func(opts *redis.Options) *redis.Client {
		capturedAddr = opts.Addr
		return redis.NewClient(opts)
	}
	parseRedisURL = redis.ParseURL
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
	origParse := parseRedisURL
	t.Cleanup(func() {
		newRedisClient = origNewClient
		pingRedis = origPing
		parseRedisURL = origParse
		Client = nil
	})

	var capturedAddr string
	newRedisClient = func(opts *redis.Options) *redis.Client {
		capturedAddr = opts.Addr
		return redis.NewClient(opts)
	}
	parseRedisURL = redis.ParseURL
	pingRedis = func(ctx context.Context, client *redis.Client) error {
		return nil
	}

	InitRedis(context.Background())
	if capturedAddr != "localhost:6379" {
		t.Fatalf("expected default addr, got %s", capturedAddr)
	}
}

func TestInitRedisWithURL(t *testing.T) {
	t.Setenv("REDIS_URL", "rediss://default:secret@redis.example.com:6380/0")

	origNewClient := newRedisClient
	origPing := pingRedis
	origParse := parseRedisURL
	t.Cleanup(func() {
		newRedisClient = origNewClient
		pingRedis = origPing
		parseRedisURL = origParse
		Client = nil
	})

	parseCalled := false
	parseRedisURL = func(rawURL string) (*redis.Options, error) {
		parseCalled = true
		if rawURL != "rediss://default:secret@redis.example.com:6380/0" {
			t.Fatalf("unexpected redis url passed to parser: %s", rawURL)
		}
		return &redis.Options{
			Addr:     "parsed-host:6380",
			Username: "default",
			Password: "secret",
		}, nil
	}

	var capturedAddr string
	var capturedUser string
	var capturedPassword string
	newRedisClient = func(opts *redis.Options) *redis.Client {
		capturedAddr = opts.Addr
		capturedUser = opts.Username
		capturedPassword = opts.Password
		return redis.NewClient(opts)
	}
	pingRedis = func(ctx context.Context, client *redis.Client) error { return nil }

	InitRedis(context.Background())

	if !parseCalled {
		t.Fatal("expected parseRedisURL to be called")
	}
	if capturedAddr != "parsed-host:6380" || capturedUser != "default" || capturedPassword != "secret" {
		t.Fatalf("unexpected parsed options: addr=%s user=%s password=%s", capturedAddr, capturedUser, capturedPassword)
	}
}

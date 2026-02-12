package cache

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/redis/go-redis/v9"
)

var Client *redis.Client

var (
	newRedisClient = func(opts *redis.Options) *redis.Client {
		return redis.NewClient(opts)
	}
	pingRedis = func(ctx context.Context, client *redis.Client) error {
		return client.Ping(ctx).Err()
	}
	parseRedisURL = redis.ParseURL
)

func InitRedis(ctx context.Context) {
	addr := os.Getenv("REDIS_URL")
	if addr == "" {
		addr = "localhost:6379"
	}

	opts := &redis.Options{Addr: addr}
	if strings.HasPrefix(addr, "redis://") || strings.HasPrefix(addr, "rediss://") {
		parsed, err := parseRedisURL(addr)
		if err != nil {
			log.Fatalf("failed to parse REDIS_URL: %v", err)
		}
		opts = parsed
	}

	Client = newRedisClient(opts)
	if err := pingRedis(ctx, Client); err != nil {
		log.Fatalf("failed to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis")
}

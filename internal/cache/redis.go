package cache

import (
	"context"
	"log"
	"os"

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
)

func InitRedis(ctx context.Context) {
	addr := os.Getenv("REDIS_URL")
	if addr == "" {
		addr = "localhost:6379"
	}
	Client = newRedisClient(&redis.Options{
		Addr: addr,
	})
	if err := pingRedis(ctx, Client); err != nil {
		log.Fatalf("failed to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis")
}

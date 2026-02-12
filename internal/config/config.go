package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	TelegramBotToken  string
	DatabaseURL       string
	RedisURL          string
	CoinGeckoPollSecs int
}

func Load() *Config {
	cfg := &Config{
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		RedisURL:         os.Getenv("REDIS_URL"),
	}

	if cfg.TelegramBotToken == "" {
		log.Println("Warning: TELEGRAM_BOT_TOKEN not set")
	}
	if cfg.DatabaseURL == "" {
		log.Println("Warning: DATABASE_URL not set")
	}
	if cfg.RedisURL == "" {
		log.Println("Warning: REDIS_URL not set, defaulting to localhost:6379")
		cfg.RedisURL = "localhost:6379"
	}

	cfg.CoinGeckoPollSecs = 60
	if v := os.Getenv("COINGECKO_POLL_SECS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.CoinGeckoPollSecs = n
		}
	}

	return cfg
}

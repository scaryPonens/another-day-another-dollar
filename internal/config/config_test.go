package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REDIS_URL", "")
	t.Setenv("COINGECKO_POLL_SECS", "")

	cfg := Load()
	if cfg.RedisURL != "localhost:6379" {
		t.Fatalf("expected default redis url, got %s", cfg.RedisURL)
	}
	if cfg.CoinGeckoPollSecs != 60 {
		t.Fatalf("expected default poll secs 60, got %d", cfg.CoinGeckoPollSecs)
	}
}

func TestLoadWithEnv(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "token")
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("REDIS_URL", "redis:6379")
	t.Setenv("COINGECKO_POLL_SECS", "120")

	cfg := Load()
	if cfg.TelegramBotToken != "token" || cfg.DatabaseURL != "postgres://example" || cfg.RedisURL != "redis:6379" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if cfg.CoinGeckoPollSecs != 120 {
		t.Fatalf("expected poll secs 120, got %d", cfg.CoinGeckoPollSecs)
	}

	t.Setenv("COINGECKO_POLL_SECS", "bad")
	cfg = Load()
	if cfg.CoinGeckoPollSecs != 60 {
		t.Fatalf("invalid poll secs should fall back to default, got %d", cfg.CoinGeckoPollSecs)
	}
}

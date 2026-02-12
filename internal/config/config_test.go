package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REDIS_URL", "")
	t.Setenv("COINGECKO_POLL_SECS", "")
	t.Setenv("MCP_TRANSPORT", "")
	t.Setenv("MCP_HTTP_ENABLED", "")
	t.Setenv("MCP_HTTP_BIND", "")
	t.Setenv("MCP_HTTP_PORT", "")
	t.Setenv("MCP_AUTH_TOKEN", "")
	t.Setenv("MCP_REQUEST_TIMEOUT_SECS", "")
	t.Setenv("MCP_RATE_LIMIT_PER_MIN", "")
	t.Setenv("ML_ENABLED", "")
	t.Setenv("ML_INTERVAL", "")
	t.Setenv("ML_TARGET_HOURS", "")
	t.Setenv("ML_TRAIN_WINDOW_DAYS", "")
	t.Setenv("ML_INFER_POLL_SECS", "")
	t.Setenv("ML_RESOLVE_POLL_SECS", "")
	t.Setenv("ML_TRAIN_HOUR_UTC", "")
	t.Setenv("ML_LONG_THRESHOLD", "")
	t.Setenv("ML_SHORT_THRESHOLD", "")
	t.Setenv("ML_MIN_TRAIN_SAMPLES", "")

	cfg := Load()
	if cfg.RedisURL != "localhost:6379" {
		t.Fatalf("expected default redis url, got %s", cfg.RedisURL)
	}
	if cfg.CoinGeckoPollSecs != 60 {
		t.Fatalf("expected default poll secs 60, got %d", cfg.CoinGeckoPollSecs)
	}
	if cfg.MCPTransport != "stdio" {
		t.Fatalf("expected default MCP transport stdio, got %s", cfg.MCPTransport)
	}
	if cfg.MCPHTTPBind != "127.0.0.1" || cfg.MCPHTTPPort != 8090 {
		t.Fatalf("unexpected MCP http defaults: %s:%d", cfg.MCPHTTPBind, cfg.MCPHTTPPort)
	}
	if cfg.MCPRequestTimeoutSecs != 5 || cfg.MCPRateLimitPerMin != 60 {
		t.Fatalf("unexpected MCP defaults: timeout=%d rate=%d", cfg.MCPRequestTimeoutSecs, cfg.MCPRateLimitPerMin)
	}
	if cfg.MLEnabled || cfg.MLInterval != "1h" || cfg.MLTargetHours != 4 || cfg.MLTrainWindowDays != 90 {
		t.Fatalf("unexpected ML defaults: %+v", cfg)
	}
	if cfg.MLInferPollSecs != 900 || cfg.MLResolvePollSecs != 1800 || cfg.MLTrainHourUTC != 0 {
		t.Fatalf("unexpected ML poll defaults: %+v", cfg)
	}
	if cfg.MLLongThreshold != 0.55 || cfg.MLShortThreshold != 0.45 || cfg.MLMinTrainSamples != 1000 {
		t.Fatalf("unexpected ML threshold defaults: %+v", cfg)
	}
}

func TestLoadWithEnv(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "token")
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("REDIS_URL", "redis:6379")
	t.Setenv("COINGECKO_POLL_SECS", "120")
	t.Setenv("MCP_TRANSPORT", "http")
	t.Setenv("MCP_HTTP_ENABLED", "true")
	t.Setenv("MCP_HTTP_BIND", "0.0.0.0")
	t.Setenv("MCP_HTTP_PORT", "9191")
	t.Setenv("MCP_AUTH_TOKEN", "secret")
	t.Setenv("MCP_REQUEST_TIMEOUT_SECS", "9")
	t.Setenv("MCP_RATE_LIMIT_PER_MIN", "75")
	t.Setenv("ML_ENABLED", "true")
	t.Setenv("ML_INTERVAL", "1h")
	t.Setenv("ML_TARGET_HOURS", "6")
	t.Setenv("ML_TRAIN_WINDOW_DAYS", "30")
	t.Setenv("ML_INFER_POLL_SECS", "600")
	t.Setenv("ML_RESOLVE_POLL_SECS", "1200")
	t.Setenv("ML_TRAIN_HOUR_UTC", "3")
	t.Setenv("ML_LONG_THRESHOLD", "0.60")
	t.Setenv("ML_SHORT_THRESHOLD", "0.40")
	t.Setenv("ML_MIN_TRAIN_SAMPLES", "200")

	cfg := Load()
	if cfg.TelegramBotToken != "token" || cfg.DatabaseURL != "postgres://example" || cfg.RedisURL != "redis:6379" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if cfg.CoinGeckoPollSecs != 120 {
		t.Fatalf("expected poll secs 120, got %d", cfg.CoinGeckoPollSecs)
	}
	if cfg.MCPTransport != "http" || !cfg.MCPHTTPEnabled || cfg.MCPHTTPBind != "0.0.0.0" || cfg.MCPHTTPPort != 9191 || cfg.MCPAuthToken != "secret" {
		t.Fatalf("unexpected MCP config: %+v", cfg)
	}
	if cfg.MCPRequestTimeoutSecs != 9 || cfg.MCPRateLimitPerMin != 75 {
		t.Fatalf("unexpected MCP timeout/rate: %+v", cfg)
	}
	if !cfg.MLEnabled || cfg.MLInterval != "1h" || cfg.MLTargetHours != 6 || cfg.MLTrainWindowDays != 30 {
		t.Fatalf("unexpected ML env values: %+v", cfg)
	}
	if cfg.MLInferPollSecs != 600 || cfg.MLResolvePollSecs != 1200 || cfg.MLTrainHourUTC != 3 {
		t.Fatalf("unexpected ML poll env values: %+v", cfg)
	}
	if cfg.MLLongThreshold != 0.60 || cfg.MLShortThreshold != 0.40 || cfg.MLMinTrainSamples != 200 {
		t.Fatalf("unexpected ML threshold env values: %+v", cfg)
	}

	t.Setenv("COINGECKO_POLL_SECS", "bad")
	t.Setenv("MCP_HTTP_PORT", "bad")
	t.Setenv("MCP_REQUEST_TIMEOUT_SECS", "bad")
	t.Setenv("MCP_RATE_LIMIT_PER_MIN", "bad")
	t.Setenv("ML_TARGET_HOURS", "bad")
	t.Setenv("ML_TRAIN_WINDOW_DAYS", "bad")
	t.Setenv("ML_INFER_POLL_SECS", "bad")
	t.Setenv("ML_RESOLVE_POLL_SECS", "bad")
	t.Setenv("ML_TRAIN_HOUR_UTC", "99")
	t.Setenv("ML_LONG_THRESHOLD", "bad")
	t.Setenv("ML_SHORT_THRESHOLD", "bad")
	t.Setenv("ML_MIN_TRAIN_SAMPLES", "bad")
	cfg = Load()
	if cfg.CoinGeckoPollSecs != 60 {
		t.Fatalf("invalid poll secs should fall back to default, got %d", cfg.CoinGeckoPollSecs)
	}
	if cfg.MCPHTTPPort != 8090 || cfg.MCPRequestTimeoutSecs != 5 || cfg.MCPRateLimitPerMin != 60 {
		t.Fatalf("invalid MCP numeric values should fall back to defaults: %+v", cfg)
	}
	if cfg.MLTargetHours != 4 || cfg.MLTrainWindowDays != 90 || cfg.MLInferPollSecs != 900 || cfg.MLResolvePollSecs != 1800 {
		t.Fatalf("invalid ML numeric values should fall back to defaults: %+v", cfg)
	}
	if cfg.MLTrainHourUTC != 0 || cfg.MLLongThreshold != 0.55 || cfg.MLShortThreshold != 0.45 || cfg.MLMinTrainSamples != 1000 {
		t.Fatalf("invalid ML threshold values should fall back to defaults: %+v", cfg)
	}
}

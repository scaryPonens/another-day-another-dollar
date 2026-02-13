package config

import (
	"reflect"
	"testing"
)

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
	t.Setenv("ML_INTERVALS", "")
	t.Setenv("ML_TARGET_HOURS", "")
	t.Setenv("ML_TRAIN_WINDOW_DAYS", "")
	t.Setenv("ML_INFER_POLL_SECS", "")
	t.Setenv("ML_RESOLVE_POLL_SECS", "")
	t.Setenv("ML_TRAIN_HOUR_UTC", "")
	t.Setenv("ML_LONG_THRESHOLD", "")
	t.Setenv("ML_SHORT_THRESHOLD", "")
	t.Setenv("ML_MIN_TRAIN_SAMPLES", "")
	t.Setenv("ML_ENABLE_IFOREST", "")
	t.Setenv("ML_ANOMALY_THRESHOLD", "")
	t.Setenv("ML_ANOMALY_DAMP_MAX", "")
	t.Setenv("ML_IFOREST_TREES", "")
	t.Setenv("ML_IFOREST_SAMPLE_SIZE", "")
	t.Setenv("MARKET_INTEL_ENABLED", "")
	t.Setenv("MARKET_INTEL_INTERVALS", "")
	t.Setenv("MARKET_INTEL_POLL_SECS", "")
	t.Setenv("MARKET_INTEL_LONG_THRESHOLD", "")
	t.Setenv("MARKET_INTEL_SHORT_THRESHOLD", "")
	t.Setenv("MARKET_INTEL_LOOKBACK_HOURS_1H", "")
	t.Setenv("MARKET_INTEL_LOOKBACK_HOURS_4H", "")
	t.Setenv("MARKET_INTEL_NEWS_FEEDS", "")
	t.Setenv("MARKET_INTEL_REDDIT_SUBS", "")
	t.Setenv("MARKET_INTEL_REDDIT_POST_LIMIT", "")
	t.Setenv("MARKET_INTEL_SCORING_MODEL", "")
	t.Setenv("MARKET_INTEL_SCORING_BATCH_SIZE", "")
	t.Setenv("MARKET_INTEL_RETENTION_DAYS", "")
	t.Setenv("MARKET_INTEL_ENABLE_ONCHAIN", "")
	t.Setenv("MARKET_INTEL_ONCHAIN_SYMBOLS", "")
	t.Setenv("ONCHAIN_BTC_MEMPOOL_BASE_URL", "")
	t.Setenv("ONCHAIN_ETH_BLOCKSCOUT_BASE_URL", "")
	t.Setenv("ONCHAIN_ADA_KOIOS_BASE_URL", "")
	t.Setenv("ONCHAIN_XRP_API_BASE_URL", "")

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
	if !reflect.DeepEqual(cfg.MLIntervals, []string{"1h"}) {
		t.Fatalf("unexpected ML interval defaults: %+v", cfg.MLIntervals)
	}
	if cfg.MLInferPollSecs != 900 || cfg.MLResolvePollSecs != 1800 || cfg.MLTrainHourUTC != 0 {
		t.Fatalf("unexpected ML poll defaults: %+v", cfg)
	}
	if cfg.MLLongThreshold != 0.55 || cfg.MLShortThreshold != 0.45 || cfg.MLMinTrainSamples != 1000 {
		t.Fatalf("unexpected ML threshold defaults: %+v", cfg)
	}
	if !cfg.MLEnableIForest || cfg.MLAnomalyThresh != 0.62 || cfg.MLAnomalyDampMax != 0.65 {
		t.Fatalf("unexpected ML anomaly defaults: %+v", cfg)
	}
	if cfg.MLIForestTrees != 200 || cfg.MLIForestSample != 256 {
		t.Fatalf("unexpected ML iforest defaults: %+v", cfg)
	}
	if cfg.MarketIntelEnabled {
		t.Fatalf("expected market intel disabled by default")
	}
	if !reflect.DeepEqual(cfg.MarketIntelIntervals, []string{"1h", "4h"}) {
		t.Fatalf("unexpected market intel intervals default: %+v", cfg.MarketIntelIntervals)
	}
	if cfg.MarketIntelPollSecs != 900 || cfg.MarketIntelLongThreshold != 0.20 || cfg.MarketIntelShortThreshold != -0.20 {
		t.Fatalf("unexpected market intel threshold defaults: %+v", cfg)
	}
	if cfg.MarketIntelLookbackHours1H != 12 || cfg.MarketIntelLookbackHours4H != 24 {
		t.Fatalf("unexpected market intel lookback defaults: %+v", cfg)
	}
	if cfg.MarketIntelRedditPostLimit != 40 || cfg.MarketIntelScoringBatchSize != 24 || cfg.MarketIntelRetentionDays != 90 {
		t.Fatalf("unexpected market intel numeric defaults: %+v", cfg)
	}
	if !cfg.MarketIntelEnableOnChain || !reflect.DeepEqual(cfg.MarketIntelOnChainSymbols, []string{"BTC", "ETH", "ADA", "XRP"}) {
		t.Fatalf("unexpected market intel onchain defaults: %+v", cfg)
	}
	if cfg.OnChainBTCMempoolBaseURL == "" || cfg.OnChainETHBlockscoutBaseURL == "" || cfg.OnChainADAKoiosBaseURL == "" || cfg.OnChainXRPAPIBaseURL == "" {
		t.Fatalf("expected onchain base urls to have defaults: %+v", cfg)
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
	t.Setenv("ML_INTERVALS", "1h,4h,invalid,1h")
	t.Setenv("ML_TARGET_HOURS", "6")
	t.Setenv("ML_TRAIN_WINDOW_DAYS", "30")
	t.Setenv("ML_INFER_POLL_SECS", "600")
	t.Setenv("ML_RESOLVE_POLL_SECS", "1200")
	t.Setenv("ML_TRAIN_HOUR_UTC", "3")
	t.Setenv("ML_LONG_THRESHOLD", "0.60")
	t.Setenv("ML_SHORT_THRESHOLD", "0.40")
	t.Setenv("ML_MIN_TRAIN_SAMPLES", "200")
	t.Setenv("ML_ENABLE_IFOREST", "false")
	t.Setenv("ML_ANOMALY_THRESHOLD", "0.70")
	t.Setenv("ML_ANOMALY_DAMP_MAX", "0.50")
	t.Setenv("ML_IFOREST_TREES", "111")
	t.Setenv("ML_IFOREST_SAMPLE_SIZE", "333")
	t.Setenv("MARKET_INTEL_ENABLED", "true")
	t.Setenv("MARKET_INTEL_INTERVALS", "1h,4h,invalid,1h")
	t.Setenv("MARKET_INTEL_POLL_SECS", "600")
	t.Setenv("MARKET_INTEL_LONG_THRESHOLD", "0.25")
	t.Setenv("MARKET_INTEL_SHORT_THRESHOLD", "-0.35")
	t.Setenv("MARKET_INTEL_LOOKBACK_HOURS_1H", "10")
	t.Setenv("MARKET_INTEL_LOOKBACK_HOURS_4H", "20")
	t.Setenv("MARKET_INTEL_NEWS_FEEDS", "https://a.example/rss,https://b.example/rss")
	t.Setenv("MARKET_INTEL_REDDIT_SUBS", "CryptoCurrency,Bitcoin")
	t.Setenv("MARKET_INTEL_REDDIT_POST_LIMIT", "15")
	t.Setenv("MARKET_INTEL_SCORING_MODEL", "gpt-4o-mini")
	t.Setenv("MARKET_INTEL_SCORING_BATCH_SIZE", "12")
	t.Setenv("MARKET_INTEL_RETENTION_DAYS", "30")
	t.Setenv("MARKET_INTEL_ENABLE_ONCHAIN", "false")
	t.Setenv("MARKET_INTEL_ONCHAIN_SYMBOLS", "btc,eth,invalid")
	t.Setenv("ONCHAIN_BTC_MEMPOOL_BASE_URL", "https://mempool.custom")
	t.Setenv("ONCHAIN_ETH_BLOCKSCOUT_BASE_URL", "https://eth.custom")
	t.Setenv("ONCHAIN_ADA_KOIOS_BASE_URL", "https://koios.custom")
	t.Setenv("ONCHAIN_XRP_API_BASE_URL", "https://xrp.custom")

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
	if !reflect.DeepEqual(cfg.MLIntervals, []string{"1h", "4h"}) {
		t.Fatalf("unexpected ML interval list: %+v", cfg.MLIntervals)
	}
	if cfg.MLInferPollSecs != 600 || cfg.MLResolvePollSecs != 1200 || cfg.MLTrainHourUTC != 3 {
		t.Fatalf("unexpected ML poll env values: %+v", cfg)
	}
	if cfg.MLLongThreshold != 0.60 || cfg.MLShortThreshold != 0.40 || cfg.MLMinTrainSamples != 200 {
		t.Fatalf("unexpected ML threshold env values: %+v", cfg)
	}
	if cfg.MLEnableIForest || cfg.MLAnomalyThresh != 0.70 || cfg.MLAnomalyDampMax != 0.50 {
		t.Fatalf("unexpected ML anomaly env values: %+v", cfg)
	}
	if cfg.MLIForestTrees != 111 || cfg.MLIForestSample != 333 {
		t.Fatalf("unexpected ML iforest env values: %+v", cfg)
	}
	if !cfg.MarketIntelEnabled || cfg.MarketIntelPollSecs != 600 {
		t.Fatalf("unexpected market intel enabled/poll env values: %+v", cfg)
	}
	if !reflect.DeepEqual(cfg.MarketIntelIntervals, []string{"1h", "4h"}) {
		t.Fatalf("unexpected market intel intervals env values: %+v", cfg.MarketIntelIntervals)
	}
	if cfg.MarketIntelLongThreshold != 0.25 || cfg.MarketIntelShortThreshold != -0.35 {
		t.Fatalf("unexpected market intel threshold env values: %+v", cfg)
	}
	if cfg.MarketIntelLookbackHours1H != 10 || cfg.MarketIntelLookbackHours4H != 20 {
		t.Fatalf("unexpected market intel lookback env values: %+v", cfg)
	}
	if !reflect.DeepEqual(cfg.MarketIntelNewsFeeds, []string{"https://a.example/rss", "https://b.example/rss"}) {
		t.Fatalf("unexpected market intel news feeds: %+v", cfg.MarketIntelNewsFeeds)
	}
	if !reflect.DeepEqual(cfg.MarketIntelRedditSubs, []string{"CryptoCurrency", "Bitcoin"}) {
		t.Fatalf("unexpected market intel reddit subs: %+v", cfg.MarketIntelRedditSubs)
	}
	if cfg.MarketIntelRedditPostLimit != 15 || cfg.MarketIntelScoringBatchSize != 12 || cfg.MarketIntelRetentionDays != 30 {
		t.Fatalf("unexpected market intel numeric env values: %+v", cfg)
	}
	if cfg.MarketIntelEnableOnChain || !reflect.DeepEqual(cfg.MarketIntelOnChainSymbols, []string{"BTC", "ETH"}) {
		t.Fatalf("unexpected market intel onchain env values: %+v", cfg)
	}
	if cfg.OnChainBTCMempoolBaseURL != "https://mempool.custom" ||
		cfg.OnChainETHBlockscoutBaseURL != "https://eth.custom" ||
		cfg.OnChainADAKoiosBaseURL != "https://koios.custom" ||
		cfg.OnChainXRPAPIBaseURL != "https://xrp.custom" {
		t.Fatalf("unexpected onchain base url env values: %+v", cfg)
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
	t.Setenv("ML_INTERVALS", "bad,")
	t.Setenv("ML_ENABLE_IFOREST", "bad")
	t.Setenv("ML_ANOMALY_THRESHOLD", "bad")
	t.Setenv("ML_ANOMALY_DAMP_MAX", "bad")
	t.Setenv("ML_IFOREST_TREES", "bad")
	t.Setenv("ML_IFOREST_SAMPLE_SIZE", "bad")
	t.Setenv("MARKET_INTEL_INTERVALS", "bad")
	t.Setenv("MARKET_INTEL_POLL_SECS", "bad")
	t.Setenv("MARKET_INTEL_LONG_THRESHOLD", "bad")
	t.Setenv("MARKET_INTEL_SHORT_THRESHOLD", "bad")
	t.Setenv("MARKET_INTEL_LOOKBACK_HOURS_1H", "bad")
	t.Setenv("MARKET_INTEL_LOOKBACK_HOURS_4H", "bad")
	t.Setenv("MARKET_INTEL_REDDIT_POST_LIMIT", "bad")
	t.Setenv("MARKET_INTEL_SCORING_BATCH_SIZE", "bad")
	t.Setenv("MARKET_INTEL_RETENTION_DAYS", "bad")
	t.Setenv("MARKET_INTEL_ENABLE_ONCHAIN", "bad")
	t.Setenv("MARKET_INTEL_ONCHAIN_SYMBOLS", "notasymbol")
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
	if !reflect.DeepEqual(cfg.MLIntervals, []string{"1h"}) {
		t.Fatalf("invalid ML interval list should fall back to ML_INTERVAL: %+v", cfg.MLIntervals)
	}
	if !cfg.MLEnableIForest || cfg.MLAnomalyThresh != 0.62 || cfg.MLAnomalyDampMax != 0.65 || cfg.MLIForestTrees != 200 || cfg.MLIForestSample != 256 {
		t.Fatalf("invalid ML anomaly values should fall back to defaults: %+v", cfg)
	}
	if !reflect.DeepEqual(cfg.MarketIntelIntervals, []string{"1h", "4h"}) {
		t.Fatalf("invalid market intel intervals should fall back to defaults: %+v", cfg.MarketIntelIntervals)
	}
	if cfg.MarketIntelPollSecs != 900 || cfg.MarketIntelLongThreshold != 0.20 || cfg.MarketIntelShortThreshold != -0.20 {
		t.Fatalf("invalid market intel thresholds should fall back to defaults: %+v", cfg)
	}
	if cfg.MarketIntelLookbackHours1H != 12 || cfg.MarketIntelLookbackHours4H != 24 {
		t.Fatalf("invalid market intel lookbacks should fall back to defaults: %+v", cfg)
	}
	if cfg.MarketIntelRedditPostLimit != 40 || cfg.MarketIntelScoringBatchSize != 24 || cfg.MarketIntelRetentionDays != 90 {
		t.Fatalf("invalid market intel numeric values should fall back to defaults: %+v", cfg)
	}
	if !cfg.MarketIntelEnableOnChain || !reflect.DeepEqual(cfg.MarketIntelOnChainSymbols, []string{"BTC", "ETH", "ADA", "XRP"}) {
		t.Fatalf("invalid market intel onchain values should fall back to defaults: %+v", cfg)
	}
}

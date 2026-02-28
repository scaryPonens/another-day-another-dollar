package config

import (
	"bug-free-umbrella/internal/domain"
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	TelegramBotToken  string
	DatabaseURL       string
	RedisURL          string
	CoinGeckoPollSecs int

	MCPTransport          string
	MCPHTTPEnabled        bool
	MCPHTTPBind           string
	MCPHTTPPort           int
	MCPAuthToken          string
	MCPRequestTimeoutSecs int
	MCPRateLimitPerMin    int

	OpenAIAPIKey      string
	OpenAIModel       string
	AdvisorMaxHistory int

	MLEnabled         bool
	MLInterval        string
	MLIntervals       []string
	MLTargetHours     int
	MLTrainWindowDays int
	MLInferPollSecs   int
	MLResolvePollSecs int
	MLTrainHourUTC    int
	MLLongThreshold   float64
	MLShortThreshold  float64
	MLMinTrainSamples int

	MLEnableIForest  bool
	MLAnomalyThresh  float64
	MLAnomalyDampMax float64
	MLIForestTrees   int
	MLIForestSample  int

	MarketIntelEnabled          bool
	MarketIntelIntervals        []string
	MarketIntelPollSecs         int
	MarketIntelLongThreshold    float64
	MarketIntelShortThreshold   float64
	MarketIntelLookbackHours1H  int
	MarketIntelLookbackHours4H  int
	MarketIntelNewsFeeds        []string
	MarketIntelRedditSubs       []string
	MarketIntelRedditPostLimit  int
	MarketIntelScoringModel     string
	MarketIntelScoringBatchSize int
	MarketIntelRetentionDays    int
	MarketIntelEnableOnChain    bool
	MarketIntelOnChainSymbols   []string
	OnChainBTCMempoolBaseURL    string
	OnChainETHBlockscoutBaseURL string
	OnChainADAKoiosBaseURL      string
	OnChainXRPAPIBaseURL        string

	SSHEnabled     bool
	SSHPort        int
	SSHHostKeyPath string
	SSHIdleTimeout int

	RESTAPIKey         string
	CORSAllowedOrigins []string
}

func Load() *Config {
	cfg := &Config{
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		RedisURL:         os.Getenv("REDIS_URL"),
		MCPAuthToken:     os.Getenv("MCP_AUTH_TOKEN"),
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

	cfg.MCPTransport = strings.ToLower(strings.TrimSpace(os.Getenv("MCP_TRANSPORT")))
	if cfg.MCPTransport == "" {
		cfg.MCPTransport = "stdio"
	}
	if cfg.MCPTransport != "stdio" && cfg.MCPTransport != "http" {
		log.Printf("Warning: unsupported MCP_TRANSPORT=%q, defaulting to stdio", cfg.MCPTransport)
		cfg.MCPTransport = "stdio"
	}

	cfg.MCPHTTPEnabled = strings.EqualFold(strings.TrimSpace(os.Getenv("MCP_HTTP_ENABLED")), "true")

	cfg.MCPHTTPBind = strings.TrimSpace(os.Getenv("MCP_HTTP_BIND"))
	if cfg.MCPHTTPBind == "" {
		cfg.MCPHTTPBind = "127.0.0.1"
	}

	cfg.MCPHTTPPort = 8090
	if v := strings.TrimSpace(os.Getenv("MCP_HTTP_PORT")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MCPHTTPPort = n
		}
	}

	cfg.MCPRequestTimeoutSecs = 5
	if v := strings.TrimSpace(os.Getenv("MCP_REQUEST_TIMEOUT_SECS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MCPRequestTimeoutSecs = n
		}
	}

	cfg.MCPRateLimitPerMin = 60
	if v := strings.TrimSpace(os.Getenv("MCP_RATE_LIMIT_PER_MIN")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MCPRateLimitPerMin = n
		}
	}

	cfg.OpenAIAPIKey = os.Getenv("OPENAI_API_KEY")
	if cfg.OpenAIAPIKey == "" {
		log.Println("Warning: OPENAI_API_KEY not set, advisor will be disabled")
	}

	cfg.OpenAIModel = strings.TrimSpace(os.Getenv("OPENAI_MODEL"))
	if cfg.OpenAIModel == "" {
		cfg.OpenAIModel = "gpt-4o-mini"
	}

	cfg.AdvisorMaxHistory = 20
	if v := os.Getenv("ADVISOR_MAX_HISTORY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.AdvisorMaxHistory = n
		}
	}

	cfg.MLEnabled = strings.EqualFold(strings.TrimSpace(os.Getenv("ML_ENABLED")), "true")

	cfg.MLInterval = strings.TrimSpace(os.Getenv("ML_INTERVAL"))
	if cfg.MLInterval == "" {
		cfg.MLInterval = "1h"
	}
	cfg.MLIntervals = parseMLIntervals(strings.TrimSpace(os.Getenv("ML_INTERVALS")), cfg.MLInterval)

	cfg.MLTargetHours = 4
	if v := strings.TrimSpace(os.Getenv("ML_TARGET_HOURS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MLTargetHours = n
		}
	}

	cfg.MLTrainWindowDays = 90
	if v := strings.TrimSpace(os.Getenv("ML_TRAIN_WINDOW_DAYS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MLTrainWindowDays = n
		}
	}

	cfg.MLInferPollSecs = 900
	if v := strings.TrimSpace(os.Getenv("ML_INFER_POLL_SECS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MLInferPollSecs = n
		}
	}

	cfg.MLResolvePollSecs = 1800
	if v := strings.TrimSpace(os.Getenv("ML_RESOLVE_POLL_SECS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MLResolvePollSecs = n
		}
	}

	cfg.MLTrainHourUTC = 0
	if v := strings.TrimSpace(os.Getenv("ML_TRAIN_HOUR_UTC")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 && n <= 23 {
			cfg.MLTrainHourUTC = n
		}
	}

	cfg.MLLongThreshold = 0.55
	if v := strings.TrimSpace(os.Getenv("ML_LONG_THRESHOLD")); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil && n > 0 && n < 1 {
			cfg.MLLongThreshold = n
		}
	}

	cfg.MLShortThreshold = 0.45
	if v := strings.TrimSpace(os.Getenv("ML_SHORT_THRESHOLD")); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil && n > 0 && n < 1 {
			cfg.MLShortThreshold = n
		}
	}

	cfg.MLMinTrainSamples = 1000
	if v := strings.TrimSpace(os.Getenv("ML_MIN_TRAIN_SAMPLES")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MLMinTrainSamples = n
		}
	}

	cfg.MLEnableIForest = true
	if v := strings.TrimSpace(os.Getenv("ML_ENABLE_IFOREST")); v != "" {
		if strings.EqualFold(v, "true") {
			cfg.MLEnableIForest = true
		} else if strings.EqualFold(v, "false") {
			cfg.MLEnableIForest = false
		}
	}

	cfg.MLAnomalyThresh = 0.62
	if v := strings.TrimSpace(os.Getenv("ML_ANOMALY_THRESHOLD")); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil && n > 0 && n < 1 {
			cfg.MLAnomalyThresh = n
		}
	}

	cfg.MLAnomalyDampMax = 0.65
	if v := strings.TrimSpace(os.Getenv("ML_ANOMALY_DAMP_MAX")); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil && n >= 0 && n <= 1 {
			cfg.MLAnomalyDampMax = n
		}
	}

	cfg.MLIForestTrees = 200
	if v := strings.TrimSpace(os.Getenv("ML_IFOREST_TREES")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MLIForestTrees = n
		}
	}

	cfg.MLIForestSample = 256
	if v := strings.TrimSpace(os.Getenv("ML_IFOREST_SAMPLE_SIZE")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MLIForestSample = n
		}
	}

	cfg.MarketIntelEnabled = strings.EqualFold(strings.TrimSpace(os.Getenv("MARKET_INTEL_ENABLED")), "true")
	cfg.MarketIntelIntervals = parseIntervalList(strings.TrimSpace(os.Getenv("MARKET_INTEL_INTERVALS")), []string{"1h", "4h"})

	cfg.MarketIntelPollSecs = 900
	if v := strings.TrimSpace(os.Getenv("MARKET_INTEL_POLL_SECS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MarketIntelPollSecs = n
		}
	}

	cfg.MarketIntelLongThreshold = 0.20
	if v := strings.TrimSpace(os.Getenv("MARKET_INTEL_LONG_THRESHOLD")); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil && n > -1 && n < 1 {
			cfg.MarketIntelLongThreshold = n
		}
	}

	cfg.MarketIntelShortThreshold = -0.20
	if v := strings.TrimSpace(os.Getenv("MARKET_INTEL_SHORT_THRESHOLD")); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil && n > -1 && n < 1 {
			cfg.MarketIntelShortThreshold = n
		}
	}
	if cfg.MarketIntelShortThreshold > cfg.MarketIntelLongThreshold {
		cfg.MarketIntelShortThreshold = -0.20
		cfg.MarketIntelLongThreshold = 0.20
	}

	cfg.MarketIntelLookbackHours1H = 12
	if v := strings.TrimSpace(os.Getenv("MARKET_INTEL_LOOKBACK_HOURS_1H")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MarketIntelLookbackHours1H = n
		}
	}

	cfg.MarketIntelLookbackHours4H = 24
	if v := strings.TrimSpace(os.Getenv("MARKET_INTEL_LOOKBACK_HOURS_4H")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MarketIntelLookbackHours4H = n
		}
	}

	cfg.MarketIntelNewsFeeds = parseCSVWithDefault(
		os.Getenv("MARKET_INTEL_NEWS_FEEDS"),
		[]string{
			"https://www.coindesk.com/arc/outboundfeeds/rss/",
			"https://cointelegraph.com/rss",
		},
	)
	cfg.MarketIntelRedditSubs = parseCSVWithDefault(
		os.Getenv("MARKET_INTEL_REDDIT_SUBS"),
		[]string{"CryptoCurrency", "Bitcoin", "Ethereum", "Cardano", "Ripple"},
	)

	cfg.MarketIntelRedditPostLimit = 40
	if v := strings.TrimSpace(os.Getenv("MARKET_INTEL_REDDIT_POST_LIMIT")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MarketIntelRedditPostLimit = n
		}
	}

	cfg.MarketIntelScoringModel = strings.TrimSpace(os.Getenv("MARKET_INTEL_SCORING_MODEL"))
	if cfg.MarketIntelScoringModel == "" {
		cfg.MarketIntelScoringModel = cfg.OpenAIModel
	}
	if cfg.MarketIntelScoringModel == "" {
		cfg.MarketIntelScoringModel = "gpt-4o-mini"
	}

	cfg.MarketIntelScoringBatchSize = 24
	if v := strings.TrimSpace(os.Getenv("MARKET_INTEL_SCORING_BATCH_SIZE")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MarketIntelScoringBatchSize = n
		}
	}

	cfg.MarketIntelRetentionDays = 90
	if v := strings.TrimSpace(os.Getenv("MARKET_INTEL_RETENTION_DAYS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MarketIntelRetentionDays = n
		}
	}

	cfg.MarketIntelEnableOnChain = true
	if v := strings.TrimSpace(os.Getenv("MARKET_INTEL_ENABLE_ONCHAIN")); v != "" {
		if strings.EqualFold(v, "true") {
			cfg.MarketIntelEnableOnChain = true
		} else if strings.EqualFold(v, "false") {
			cfg.MarketIntelEnableOnChain = false
		}
	}

	cfg.MarketIntelOnChainSymbols = parseSymbolListWithDefault(
		os.Getenv("MARKET_INTEL_ONCHAIN_SYMBOLS"),
		[]string{"BTC", "ETH", "ADA", "XRP"},
	)

	cfg.OnChainBTCMempoolBaseURL = strings.TrimSpace(os.Getenv("ONCHAIN_BTC_MEMPOOL_BASE_URL"))
	if cfg.OnChainBTCMempoolBaseURL == "" {
		cfg.OnChainBTCMempoolBaseURL = "https://mempool.space"
	}
	cfg.OnChainETHBlockscoutBaseURL = strings.TrimSpace(os.Getenv("ONCHAIN_ETH_BLOCKSCOUT_BASE_URL"))
	if cfg.OnChainETHBlockscoutBaseURL == "" {
		cfg.OnChainETHBlockscoutBaseURL = "https://eth.blockscout.com"
	}
	cfg.OnChainADAKoiosBaseURL = strings.TrimSpace(os.Getenv("ONCHAIN_ADA_KOIOS_BASE_URL"))
	if cfg.OnChainADAKoiosBaseURL == "" {
		cfg.OnChainADAKoiosBaseURL = "https://api.koios.rest"
	}
	cfg.OnChainXRPAPIBaseURL = strings.TrimSpace(os.Getenv("ONCHAIN_XRP_API_BASE_URL"))
	if cfg.OnChainXRPAPIBaseURL == "" {
		cfg.OnChainXRPAPIBaseURL = "https://api.xrpscan.com"
	}

	cfg.SSHEnabled = strings.EqualFold(strings.TrimSpace(os.Getenv("SSH_ENABLED")), "true")

	cfg.SSHPort = 2222
	if v := strings.TrimSpace(os.Getenv("SSH_PORT")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.SSHPort = n
		}
	}

	cfg.SSHHostKeyPath = strings.TrimSpace(os.Getenv("SSH_HOST_KEY_PATH"))
	if cfg.SSHHostKeyPath == "" {
		cfg.SSHHostKeyPath = ".ssh/id_ed25519"
	}

	cfg.SSHIdleTimeout = 300
	if v := strings.TrimSpace(os.Getenv("SSH_IDLE_TIMEOUT_SECS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			cfg.SSHIdleTimeout = n
		}
	}

	cfg.RESTAPIKey = strings.TrimSpace(os.Getenv("REST_API_KEY"))
	if cfg.RESTAPIKey == "" {
		log.Println("Warning: REST_API_KEY not set, REST API will be unauthenticated")
	}

	raw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if raw == "" {
		cfg.CORSAllowedOrigins = []string{"*"}
	} else {
		for _, o := range strings.Split(raw, ",") {
			if s := strings.TrimSpace(o); s != "" {
				cfg.CORSAllowedOrigins = append(cfg.CORSAllowedOrigins, s)
			}
		}
	}

	return cfg
}

func parseMLIntervals(raw string, fallback string) []string {
	return parseIntervalList(raw, []string{fallback})
}

func parseIntervalList(raw string, fallback []string) []string {
	if len(fallback) == 0 {
		fallback = []string{"1h"}
	}

	supported := make(map[string]struct{}, len(domain.SupportedIntervals))
	for _, interval := range domain.SupportedIntervals {
		supported[interval] = struct{}{}
	}

	if strings.TrimSpace(raw) == "" {
		cleanFallback := make([]string, 0, len(fallback))
		for _, interval := range fallback {
			interval = strings.TrimSpace(interval)
			if interval == "" {
				continue
			}
			if _, ok := supported[interval]; !ok {
				continue
			}
			cleanFallback = append(cleanFallback, interval)
		}
		if len(cleanFallback) == 0 {
			return []string{"1h"}
		}
		return cleanFallback
	}

	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		interval := strings.TrimSpace(part)
		if interval == "" {
			continue
		}
		if _, ok := supported[interval]; !ok {
			continue
		}
		if _, ok := seen[interval]; ok {
			continue
		}
		seen[interval] = struct{}{}
		out = append(out, interval)
	}
	if len(out) == 0 {
		return parseIntervalList("", fallback)
	}
	return out
}

func parseCSVWithDefault(raw string, fallback []string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return append([]string(nil), fallback...)
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		val := strings.TrimSpace(part)
		if val == "" {
			continue
		}
		if _, ok := seen[val]; ok {
			continue
		}
		seen[val] = struct{}{}
		out = append(out, val)
	}
	if len(out) == 0 {
		return append([]string(nil), fallback...)
	}
	return out
}

func parseSymbolListWithDefault(raw string, fallback []string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return append([]string(nil), fallback...)
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		symbol := strings.ToUpper(strings.TrimSpace(part))
		if symbol == "" {
			continue
		}
		if _, ok := domain.CoinGeckoID[symbol]; !ok {
			continue
		}
		if _, ok := seen[symbol]; ok {
			continue
		}
		seen[symbol] = struct{}{}
		out = append(out, symbol)
	}
	if len(out) == 0 {
		return append([]string(nil), fallback...)
	}
	return out
}

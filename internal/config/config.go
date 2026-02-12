package config

import (
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
	MLTargetHours     int
	MLTrainWindowDays int
	MLInferPollSecs   int
	MLResolvePollSecs int
	MLTrainHourUTC    int
	MLLongThreshold   float64
	MLShortThreshold  float64
	MLMinTrainSamples int
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

	return cfg
}

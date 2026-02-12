package main

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"bug-free-umbrella/internal/config"
	"bug-free-umbrella/internal/domain"
	"bug-free-umbrella/internal/job"
	"bug-free-umbrella/internal/service"

	"github.com/gin-gonic/gin"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestMainBootstrap(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := stubServerDeps()
	defer restore()

	done := make(chan struct{})
	go func() {
		main()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("main did not exit")
	}
}

func stubServerDeps() func() {
	origLoadEnv := loadEnvFunc
	origLoadConfig := loadConfigFunc
	origInitPostgres := initPostgresFunc
	origInitRedis := initRedisFunc
	origInitTracer := initTracerFunc
	origNewProvider := newCoinGeckoProviderFunc
	origStartPoller := startPollerFunc
	origStartTelegram := startTelegramBotFunc
	origNewRouter := newRouterFunc
	origSetupSignal := setupSignalNotify
	origWait := waitForSignalFunc
	origStartHTTP := startHTTPServerFunc
	origShutdownHTTP := shutdownHTTPServerFunc

	loadEnvFunc = func(...string) error { return nil }
	loadConfigFunc = func() *config.Config {
		return &config.Config{RedisURL: "", DatabaseURL: "", CoinGeckoPollSecs: 1}
	}
	initPostgresFunc = func(context.Context) {}
	initRedisFunc = func(context.Context) {}
	initTracerFunc = func(ctx context.Context) (*sdktrace.TracerProvider, trace.Tracer, error) {
		tp := sdktrace.NewTracerProvider()
		return tp, tp.Tracer("test"), nil
	}
	newCoinGeckoProviderFunc = func(trace.Tracer) service.PriceProvider { return stubPriceProvider{} }
	startPollerFunc = func(*job.PricePoller, context.Context) {}
	startTelegramBotFunc = func(*service.PriceService) {}
	newRouterFunc = func(...gin.OptionFunc) *gin.Engine { return gin.New() }
	setupSignalNotify = func(c chan<- os.Signal, sig ...os.Signal) {}
	waitForSignalFunc = func(<-chan os.Signal) {}
	startHTTPServerFunc = func(*http.Server) error { return http.ErrServerClosed }
	shutdownHTTPServerFunc = func(*http.Server, context.Context) error { return nil }

	return func() {
		loadEnvFunc = origLoadEnv
		loadConfigFunc = origLoadConfig
		initPostgresFunc = origInitPostgres
		initRedisFunc = origInitRedis
		initTracerFunc = origInitTracer
		newCoinGeckoProviderFunc = origNewProvider
		startPollerFunc = origStartPoller
		startTelegramBotFunc = origStartTelegram
		newRouterFunc = origNewRouter
		setupSignalNotify = origSetupSignal
		waitForSignalFunc = origWait
		startHTTPServerFunc = origStartHTTP
		shutdownHTTPServerFunc = origShutdownHTTP
	}
}

type stubPriceProvider struct{}

func (stubPriceProvider) FetchPrices(ctx context.Context) (map[string]*domain.PriceSnapshot, error) {
	return map[string]*domain.PriceSnapshot{
		"BTC": {Symbol: "BTC", PriceUSD: 1},
	}, nil
}

func (stubPriceProvider) FetchMarketChart(ctx context.Context, symbol string, days int, intervals []string) ([]*domain.Candle, error) {
	return []*domain.Candle{}, nil
}

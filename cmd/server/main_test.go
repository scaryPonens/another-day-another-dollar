package main

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"bug-free-umbrella/internal/advisor"
	"bug-free-umbrella/internal/bot"
	"bug-free-umbrella/internal/config"
	"bug-free-umbrella/internal/domain"
	"bug-free-umbrella/internal/job"
	"bug-free-umbrella/internal/repository"
	"bug-free-umbrella/internal/service"
	signalengine "bug-free-umbrella/internal/signal"

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

func TestHTTPAddrFromEnv(t *testing.T) {
	t.Setenv("PORT", "")
	if got := httpAddrFromEnv(); got != ":8080" {
		t.Fatalf("expected default :8080, got %s", got)
	}

	t.Setenv("PORT", "9090")
	if got := httpAddrFromEnv(); got != ":9090" {
		t.Fatalf("expected :9090, got %s", got)
	}

	t.Setenv("PORT", ":7070")
	if got := httpAddrFromEnv(); got != ":7070" {
		t.Fatalf("expected :7070, got %s", got)
	}
}

func stubServerDeps() func() {
	origLoadEnv := loadEnvFunc
	origLoadConfig := loadConfigFunc
	origInitPostgres := initPostgresFunc
	origInitRedis := initRedisFunc
	origInitTracer := initTracerFunc
	origNewSignalRepo := newSignalRepoFunc
	origNewProvider := newCoinGeckoProviderFunc
	origNewSignalEngine := newSignalEngineFunc
	origNewSignalService := newSignalServiceFunc
	origStartPoller := startPollerFunc
	origNewSignalPoller := newSignalPollerFunc
	origStartSignalPoller := startSignalPollerFunc
	origNewConvRepo := newConversationRepoFunc
	origNewOpenAIClient := newOpenAIClientFunc
	origNewAdvisor := newAdvisorServiceFunc
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
	newSignalRepoFunc = func(repository.PgxPool, trace.Tracer) *repository.SignalRepository {
		return nil
	}
	newCoinGeckoProviderFunc = func(trace.Tracer) service.PriceProvider { return stubPriceProvider{} }
	newSignalEngineFunc = func(func() time.Time) *signalengine.Engine { return signalengine.NewEngine(nil) }
	newSignalServiceFunc = func(
		trace.Tracer,
		service.SignalCandleRepository,
		service.SignalRepository,
		service.SignalEngine,
	) *service.SignalService {
		return nil
	}
	startPollerFunc = func(*job.PricePoller, context.Context) {}
	newSignalPollerFunc = func(trace.Tracer, job.SignalGenerator, job.SignalAlertSink) *job.SignalPoller {
		return nil
	}
	startSignalPollerFunc = func(*job.SignalPoller, context.Context) {}
	newConversationRepoFunc = func(repository.PgxPool, trace.Tracer) *repository.ConversationRepository {
		return nil
	}
	newOpenAIClientFunc = func(string) advisor.LLMClient { return nil }
	newAdvisorServiceFunc = func(
		trace.Tracer, advisor.LLMClient, advisor.PriceQuerier, advisor.SignalQuerier,
		advisor.ConversationStore, string, int,
	) *advisor.AdvisorService {
		return nil
	}
	startTelegramBotFunc = func(bot.PriceQuerier, bot.SignalLister, bot.Advisor) *bot.AlertDispatcher { return nil }
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
		newSignalRepoFunc = origNewSignalRepo
		newCoinGeckoProviderFunc = origNewProvider
		newSignalEngineFunc = origNewSignalEngine
		newSignalServiceFunc = origNewSignalService
		startPollerFunc = origStartPoller
		newSignalPollerFunc = origNewSignalPoller
		startSignalPollerFunc = origStartSignalPoller
		newConversationRepoFunc = origNewConvRepo
		newOpenAIClientFunc = origNewOpenAIClient
		newAdvisorServiceFunc = origNewAdvisor
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

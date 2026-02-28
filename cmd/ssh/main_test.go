package main

import (
	"context"
	"os"
	"testing"
	"time"

	"bug-free-umbrella/internal/advisor"
	"bug-free-umbrella/internal/config"
	"bug-free-umbrella/internal/repository"
	"bug-free-umbrella/internal/service"
	signalengine "bug-free-umbrella/internal/signal"

	"github.com/charmbracelet/ssh"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestMainBootstrap(t *testing.T) {
	restore := stubSSHDeps()
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

func stubSSHDeps() func() {
	origLoadEnv := loadEnvFunc
	origLoadConfig := loadConfigFunc
	origInitPostgres := initPostgresFunc
	origInitRedis := initRedisFunc
	origInitTracer := initTracerFunc
	origNewCandleRepo := newCandleRepoFunc
	origNewSignalRepo := newSignalRepoFunc
	origNewSSHUserRepo := newSSHUserRepoFunc
	origNewBacktestRepo := newBacktestRepoFunc
	origNewConvRepo := newConversationRepoFunc
	origNewProvider := newCoinGeckoProviderFunc
	origNewSignalEngine := newSignalEngineFunc
	origNewPriceService := newPriceServiceFunc
	origNewSignalService := newSignalServiceWithImagesFunc
	origNewOpenAIClient := newOpenAIClientFunc
	origNewAdvisor := newAdvisorServiceFunc
	origNewWishServer := newWishServerFunc
	origSetupSignal := setupSignalNotify
	origWait := waitForSignalFunc

	loadEnvFunc = func(...string) error { return nil }
	loadConfigFunc = func() *config.Config {
		return &config.Config{
			RedisURL:       "",
			DatabaseURL:    "",
			SSHPort:        2222,
			SSHHostKeyPath: ".ssh/test_key",
		}
	}
	initPostgresFunc = func(context.Context) {}
	initRedisFunc = func(context.Context) {}
	initTracerFunc = func(ctx context.Context) (*sdktrace.TracerProvider, trace.Tracer, error) {
		tp := sdktrace.NewTracerProvider()
		return tp, tp.Tracer("test"), nil
	}
	newCandleRepoFunc = func(repository.PgxPool, trace.Tracer) *repository.CandleRepository {
		return nil
	}
	newSignalRepoFunc = func(repository.PgxPool, trace.Tracer) *repository.SignalRepository {
		return nil
	}
	newSSHUserRepoFunc = func(repository.PgxPool, trace.Tracer) *repository.SSHUserRepository {
		return nil
	}
	newBacktestRepoFunc = func(repository.PgxPool, trace.Tracer) *repository.BacktestRepository {
		return nil
	}
	newConversationRepoFunc = func(repository.PgxPool, trace.Tracer) *repository.ConversationRepository {
		return nil
	}
	newCoinGeckoProviderFunc = func(trace.Tracer) service.PriceProvider { return nil }
	newSignalEngineFunc = func(func() time.Time) *signalengine.Engine { return signalengine.NewEngine(nil) }
	newPriceServiceFunc = func(
		trace.Tracer,
		service.PriceProvider,
		service.CandleRepository,
		service.RedisClient,
	) *service.PriceService {
		return nil
	}
	newSignalServiceWithImagesFunc = func(
		trace.Tracer,
		service.SignalCandleRepository,
		service.SignalRepository,
		service.SignalEngine,
		service.SignalImageRepository,
		service.SignalChartRenderer,
	) *service.SignalService {
		return nil
	}
	newOpenAIClientFunc = func(string) advisor.LLMClient { return nil }
	newAdvisorServiceFunc = func(
		trace.Tracer, advisor.LLMClient, advisor.PriceQuerier, advisor.SignalQuerier,
		advisor.ConversationStore, string, int,
	) *advisor.AdvisorService {
		return nil
	}
	newWishServerFunc = func(ops ...ssh.Option) (*ssh.Server, error) {
		return nil, nil
	}
	setupSignalNotify = func(c chan<- os.Signal, sig ...os.Signal) {}
	waitForSignalFunc = func(<-chan os.Signal) {}

	return func() {
		loadEnvFunc = origLoadEnv
		loadConfigFunc = origLoadConfig
		initPostgresFunc = origInitPostgres
		initRedisFunc = origInitRedis
		initTracerFunc = origInitTracer
		newCandleRepoFunc = origNewCandleRepo
		newSignalRepoFunc = origNewSignalRepo
		newSSHUserRepoFunc = origNewSSHUserRepo
		newBacktestRepoFunc = origNewBacktestRepo
		newConversationRepoFunc = origNewConvRepo
		newCoinGeckoProviderFunc = origNewProvider
		newSignalEngineFunc = origNewSignalEngine
		newPriceServiceFunc = origNewPriceService
		newSignalServiceWithImagesFunc = origNewSignalService
		newOpenAIClientFunc = origNewOpenAIClient
		newAdvisorServiceFunc = origNewAdvisor
		newWishServerFunc = origNewWishServer
		setupSignalNotify = origSetupSignal
		waitForSignalFunc = origWait
	}
}

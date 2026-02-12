package main

import (
	"context"
	"log"
	"net/http"
	"os"
	ossignal "os/signal"
	"strings"
	"syscall"
	"time"

	"bug-free-umbrella/internal/advisor"
	"bug-free-umbrella/internal/bot"
	"bug-free-umbrella/internal/cache"
	"bug-free-umbrella/internal/chart"
	"bug-free-umbrella/internal/config"
	"bug-free-umbrella/internal/db"
	"bug-free-umbrella/internal/handler"
	"bug-free-umbrella/internal/job"
	"bug-free-umbrella/internal/ml/ensemble"
	"bug-free-umbrella/internal/ml/features"
	"bug-free-umbrella/internal/ml/inference"
	"bug-free-umbrella/internal/ml/predictions"
	"bug-free-umbrella/internal/ml/registry"
	"bug-free-umbrella/internal/ml/training"
	"bug-free-umbrella/internal/provider"
	"bug-free-umbrella/internal/repository"
	"bug-free-umbrella/internal/service"
	signalengine "bug-free-umbrella/internal/signal"
	"bug-free-umbrella/pkg/tracing"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/trace"

	_ "bug-free-umbrella/docs"
)

var (
	loadEnvFunc              = godotenv.Load
	loadConfigFunc           = config.Load
	initPostgresFunc         = db.InitPostgres
	initRedisFunc            = cache.InitRedis
	initTracerFunc           = tracing.InitTracer
	newCandleRepoFunc        = repository.NewCandleRepository
	newSignalRepoFunc        = repository.NewSignalRepository
	newSignalImageRepoFunc   = repository.NewSignalImageRepository
	newCoinGeckoProviderFunc = func(tracer trace.Tracer) service.PriceProvider {
		return provider.NewCoinGeckoProvider(tracer)
	}
	newSignalEngineFunc            = signalengine.NewEngine
	newPriceServiceFunc            = service.NewPriceService
	newSignalServiceWithImagesFunc = service.NewSignalServiceWithImages
	newChartRendererFunc           = chart.NewRenderer
	newPricePollerFunc             = job.NewPricePoller
	newSignalPollerFunc            = job.NewSignalPoller
	newSignalImageJobFunc          = job.NewSignalImageMaintenance
	startPollerFunc                = func(p *job.PricePoller, ctx context.Context) { go p.Start(ctx) }
	startSignalPollerFunc          = func(p *job.SignalPoller, ctx context.Context) { go p.Start(ctx) }
	startSignalImageJobFunc        = func(j *job.SignalImageMaintenance, ctx context.Context) { go j.Start(ctx) }
	newConversationRepoFunc        = repository.NewConversationRepository
	newOpenAIClientFunc            = advisor.NewOpenAIClient
	newAdvisorServiceFunc          = advisor.NewAdvisorService
	startTelegramBotFunc           = bot.StartTelegramBot
	newWorkServiceFunc             = service.NewWorkService
	newHandlerFunc                 = handler.New
	newRouterFunc                  = gin.Default
	setupSignalNotify              = ossignal.Notify
	waitForSignalFunc              = func(quit <-chan os.Signal) { <-quit }
	startHTTPServerFunc            = func(srv *http.Server) error { return srv.ListenAndServe() }
	shutdownHTTPServerFunc         = func(srv *http.Server, ctx context.Context) error { return srv.Shutdown(ctx) }
)

// @title           Bug Free Umbrella API
// @version         1.0
// @description     A Go service with OpenTelemetry tracing.

// @host      localhost:8080
// @BasePath  /
func main() {
	loadEnvFunc()

	cfg := loadConfigFunc()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Init Postgres and Redis
	os.Setenv("DATABASE_URL", cfg.DatabaseURL)
	os.Setenv("REDIS_URL", cfg.RedisURL)
	initPostgresFunc(ctx)
	initRedisFunc(ctx)

	// Init tracing
	tp, tracer, err := initTracerFunc(ctx)
	if err != nil {
		log.Fatalf("failed to initialize tracer: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Printf("error shutting down tracer provider: %v", err)
		}
	}()

	// Create repositories
	candleRepo := newCandleRepoFunc(db.Pool, tracer)
	signalRepo := newSignalRepoFunc(db.Pool, tracer)
	signalImageRepo := newSignalImageRepoFunc(db.Pool, tracer)

	// Create providers and services
	cgProvider := newCoinGeckoProviderFunc(tracer)
	priceService := newPriceServiceFunc(tracer, cgProvider, candleRepo, cache.Client)
	signalEngine := newSignalEngineFunc(nil)
	chartRenderer := newChartRendererFunc()
	signalService := newSignalServiceWithImagesFunc(tracer, candleRepo, signalRepo, signalEngine, signalImageRepo, chartRenderer)

	// Create conversation repository and advisor
	convRepo := newConversationRepoFunc(db.Pool, tracer)
	var advisorSvc *advisor.AdvisorService
	if cfg.OpenAIAPIKey != "" {
		llmClient := newOpenAIClientFunc(cfg.OpenAIAPIKey)
		advisorSvc = newAdvisorServiceFunc(tracer, llmClient, priceService, signalService,
			convRepo, cfg.OpenAIModel, cfg.AdvisorMaxHistory)
		log.Println("Advisor service enabled")
	}

	// Start Telegram bot
	os.Setenv("TELEGRAM_BOT_TOKEN", cfg.TelegramBotToken)
	alertDispatcher := startTelegramBotFunc(priceService, signalService, advisorSvc)

	// Start background pollers (stopped by ctx cancel)
	poller := newPricePollerFunc(tracer, priceService, cfg.CoinGeckoPollSecs)
	startPollerFunc(poller, ctx)
	signalPoller := newSignalPollerFunc(tracer, signalService, alertDispatcher)
	startSignalPollerFunc(signalPoller, ctx)
	signalImageJob := newSignalImageJobFunc(tracer, signalService)
	startSignalImageJobFunc(signalImageJob, ctx)
	var mlService *service.MLSignalService
	if cfg.MLEnabled {
		if db.Pool == nil {
			log.Println("ML jobs disabled: DATABASE_URL is required for ML feature/model storage")
		} else {
			mlFeatureRepo := features.NewRepository(db.Pool, tracer)
			mlRegistryRepo := registry.NewRepository(db.Pool, tracer)
			mlPredictionRepo := predictions.NewRepository(db.Pool, tracer)
			mlTrainingSvc := training.NewService(tracer, mlFeatureRepo, mlRegistryRepo, training.Config{
				Interval:        cfg.MLInterval,
				TrainWindowDays: cfg.MLTrainWindowDays,
				MinTrainSamples: cfg.MLMinTrainSamples,
			})
			mlInferenceSvc := inference.NewService(
				tracer,
				mlFeatureRepo,
				mlRegistryRepo,
				mlPredictionRepo,
				signalRepo,
				ensemble.NewService(),
				inference.Config{
					Interval:       cfg.MLInterval,
					TargetHours:    cfg.MLTargetHours,
					LongThreshold:  cfg.MLLongThreshold,
					ShortThreshold: cfg.MLShortThreshold,
				},
			)
			mlService = service.NewMLSignalService(
				tracer,
				candleRepo,
				features.NewEngine(nil),
				mlFeatureRepo,
				mlTrainingSvc,
				mlInferenceSvc,
				mlPredictionRepo,
				service.MLSignalServiceConfig{
					Interval:        cfg.MLInterval,
					TargetHours:     cfg.MLTargetHours,
					TrainWindowDays: cfg.MLTrainWindowDays,
				},
			)
			go job.NewMLFeatureInferenceJob(
				tracer,
				mlService,
				time.Duration(cfg.MLInferPollSecs)*time.Second,
			).Start(ctx)
			go job.NewMLTrainingJob(tracer, mlService, cfg.MLTrainHourUTC).Start(ctx)
			go job.NewMLOutcomeResolverJob(
				tracer,
				mlService,
				time.Duration(cfg.MLResolvePollSecs)*time.Second,
				200,
			).Start(ctx)
			log.Printf(
				"ML jobs enabled interval=%s target_hours=%d train_window_days=%d",
				cfg.MLInterval, cfg.MLTargetHours, cfg.MLTrainWindowDays,
			)
		}
	}

	// Create handlers and routes
	workService := newWorkServiceFunc(tracer)
	h := newHandlerFunc(tracer, workService, priceService, signalService)
	if mlService != nil {
		h.SetMLTrainingRunner(mlService)
	}

	r := newRouterFunc()
	r.Use(otelgin.Middleware("bug-free-umbrella"))

	h.RegisterRoutes(r)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	srv := &http.Server{
		Addr:    httpAddrFromEnv(),
		Handler: r,
	}

	go func() {
		if err := startHTTPServerFunc(srv); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	setupSignalNotify(quit, syscall.SIGINT, syscall.SIGTERM)
	waitForSignalFunc(quit)
	log.Println("Shutting down server...")

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := shutdownHTTPServerFunc(srv, shutdownCtx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}

func httpAddrFromEnv() string {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		return ":8080"
	}
	if strings.HasPrefix(port, ":") {
		return port
	}
	return ":" + port
}

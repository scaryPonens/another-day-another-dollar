package main

import (
	"context"
	"fmt"
	"log"
	"os"
	ossignal "os/signal"
	"syscall"
	"time"

	"bug-free-umbrella/internal/advisor"
	"bug-free-umbrella/internal/cache"
	"bug-free-umbrella/internal/config"
	"bug-free-umbrella/internal/db"
	"bug-free-umbrella/internal/provider"
	"bug-free-umbrella/internal/repository"
	"bug-free-umbrella/internal/service"
	signalengine "bug-free-umbrella/internal/signal"
	"bug-free-umbrella/internal/tui"
	"bug-free-umbrella/pkg/tracing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/joho/godotenv"
	gossh "golang.org/x/crypto/ssh"
	"go.opentelemetry.io/otel/trace"
)

// ctxKey is a typed context key to avoid collisions.
type ctxKey string

const sshUserKey ctxKey = "ssh_user"

var (
	loadEnvFunc      = godotenv.Load
	loadConfigFunc   = config.Load
	initPostgresFunc = db.InitPostgres
	initRedisFunc    = cache.InitRedis
	initTracerFunc   = tracing.InitTracer
	newCandleRepoFunc        = repository.NewCandleRepository
	newSignalRepoFunc        = repository.NewSignalRepository
	newSSHUserRepoFunc       = repository.NewSSHUserRepository
	newBacktestRepoFunc      = repository.NewBacktestRepository
	newConversationRepoFunc  = repository.NewConversationRepository
	newCoinGeckoProviderFunc = func(tracer trace.Tracer) service.PriceProvider {
		return provider.NewCoinGeckoProvider(tracer)
	}
	newSignalEngineFunc            = signalengine.NewEngine
	newPriceServiceFunc            = service.NewPriceService
	newSignalServiceWithImagesFunc = service.NewSignalServiceWithImages
	newOpenAIClientFunc            = advisor.NewOpenAIClient
	newAdvisorServiceFunc          = advisor.NewAdvisorService
	newWishServerFunc              = wish.NewServer
	setupSignalNotify              = ossignal.Notify
	waitForSignalFunc              = func(quit <-chan os.Signal) { <-quit }
)

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
	sshUserRepo := newSSHUserRepoFunc(db.Pool, tracer)
	backtestRepo := newBacktestRepoFunc(db.Pool, tracer)
	convRepo := newConversationRepoFunc(db.Pool, tracer)

	// Create services
	cgProvider := newCoinGeckoProviderFunc(tracer)
	priceService := newPriceServiceFunc(tracer, cgProvider, candleRepo, cache.Client)
	signalEngine := newSignalEngineFunc(nil)
	signalService := newSignalServiceWithImagesFunc(tracer, candleRepo, signalRepo, signalEngine, nil, nil)

	// Advisor (optional)
	var advisorSvc *advisor.AdvisorService
	if cfg.OpenAIAPIKey != "" {
		llmClient := newOpenAIClientFunc(cfg.OpenAIAPIKey)
		advisorSvc = newAdvisorServiceFunc(tracer, llmClient, priceService, signalService,
			convRepo, cfg.OpenAIModel, cfg.AdvisorMaxHistory)
		log.Println("SSH advisor service enabled")
	}

	// Build Wish SSH server
	addr := fmt.Sprintf("0.0.0.0:%d", cfg.SSHPort)

	srv, err := newWishServerFunc(
		wish.WithAddress(addr),
		wish.WithHostKeyPath(cfg.SSHHostKeyPath),
		wish.WithPublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
			fingerprint := gossh.FingerprintSHA256(key)
			user, err := sshUserRepo.FindByFingerprint(context.Background(), fingerprint)
			if err != nil || user == nil {
				log.Printf("SSH auth denied: fingerprint=%s err=%v", fingerprint, err)
				return false
			}
			ctx.SetValue(sshUserKey, user)
			_ = sshUserRepo.UpdateLastLogin(context.Background(), user.ID)
			log.Printf("SSH auth accepted: user=%s fingerprint=%s", user.Username, fingerprint)
			return true
		}),
		wish.WithMiddleware(
			bubbletea.Middleware(func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
				user, _ := s.Context().Value(sshUserKey).(*repository.SSHUser)

				username := "unknown"
				var userID int64
				if user != nil {
					username = user.Username
					userID = user.ID
				}

				var advisorQ tui.AdvisorQuerier
				if advisorSvc != nil {
					advisorQ = advisorSvc
				}

				svc := tui.Services{
					Prices:   priceService,
					Signals:  signalService,
					Advisor:  advisorQ,
					Backtest: backtestRepo,
					UserID:   userID,
					Username: username,
				}

				model := tui.NewAppModel(svc)
				pty, _, _ := s.Pty()
				model.SetSize(pty.Window.Width, pty.Window.Height)

				return model, []tea.ProgramOption{tea.WithAltScreen()}
			}),
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Fatalf("failed to create SSH server: %v", err)
	}

	if srv != nil {
		go func() {
			log.Printf("SSH server listening on %s", addr)
			if err := srv.ListenAndServe(); err != nil {
				log.Printf("SSH server stopped: %v", err)
			}
		}()
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	setupSignalNotify(quit, syscall.SIGINT, syscall.SIGTERM)
	waitForSignalFunc(quit)
	log.Println("Shutting down SSH server...")

	cancel()

	if srv != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("SSH server shutdown error: %v", err)
		}
	}

	log.Println("SSH server exited")
}

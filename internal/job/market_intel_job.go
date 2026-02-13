package job

import (
	"context"
	"log"
	"time"

	"bug-free-umbrella/internal/domain"

	"go.opentelemetry.io/otel/trace"
)

type MarketIntelRunner interface {
	RunMarketIntel(ctx context.Context) (domain.MarketIntelRunResult, error)
}

type MarketIntelJob struct {
	tracer       trace.Tracer
	runner       MarketIntelRunner
	pollInterval time.Duration
}

func NewMarketIntelJob(tracer trace.Tracer, runner MarketIntelRunner, pollInterval time.Duration) *MarketIntelJob {
	if pollInterval <= 0 {
		pollInterval = 15 * time.Minute
	}
	return &MarketIntelJob{tracer: tracer, runner: runner, pollInterval: pollInterval}
}

func (j *MarketIntelJob) Start(ctx context.Context) {
	if j.runner == nil {
		log.Println("Market intel job disabled: no runner")
		<-ctx.Done()
		return
	}

	j.runOnce(ctx)
	ticker := time.NewTicker(j.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			j.runOnce(ctx)
		}
	}
}

func (j *MarketIntelJob) runOnce(ctx context.Context) {
	_, span := j.tracer.Start(ctx, "market-intel-job.run-once")
	defer span.End()

	result, err := j.runner.RunMarketIntel(ctx)
	if err != nil {
		log.Printf("Market intel cycle error: %v", err)
		return
	}
	if result.ItemsIngested > 0 || result.SignalsWritten > 0 {
		log.Printf(
			"Market intel cycle complete ingested=%d scored=%d onchain=%d composites=%d signals=%d warnings=%d",
			result.ItemsIngested,
			result.ItemsScored,
			result.OnChainSnapshots,
			result.CompositesWritten,
			result.SignalsWritten,
			len(result.Errors),
		)
	}
}

package job

import (
	"context"
	"log"
	"time"

	"go.opentelemetry.io/otel/trace"
)

type MLOutcomeResolver interface {
	ResolveOutcomes(ctx context.Context, limit int) (int, error)
}

type MLOutcomeResolverJob struct {
	tracer       trace.Tracer
	service      MLOutcomeResolver
	pollInterval time.Duration
	batchSize    int
}

func NewMLOutcomeResolverJob(tracer trace.Tracer, service MLOutcomeResolver, pollInterval time.Duration, batchSize int) *MLOutcomeResolverJob {
	if pollInterval <= 0 {
		pollInterval = 30 * time.Minute
	}
	if batchSize <= 0 {
		batchSize = 200
	}
	return &MLOutcomeResolverJob{tracer: tracer, service: service, pollInterval: pollInterval, batchSize: batchSize}
}

func (j *MLOutcomeResolverJob) Start(ctx context.Context) {
	if j.service == nil {
		log.Println("ML outcome resolver job disabled: no service")
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

func (j *MLOutcomeResolverJob) runOnce(ctx context.Context) {
	_, span := j.tracer.Start(ctx, "ml-outcome-resolver-job.run-once")
	defer span.End()

	resolved, err := j.service.ResolveOutcomes(ctx, j.batchSize)
	if err != nil {
		log.Printf("ML outcome resolver error: %v", err)
		return
	}
	if resolved > 0 {
		log.Printf("ML outcome resolver updated %d predictions", resolved)
	}
}

package job

import (
	"context"
	"log"
	"time"

	"bug-free-umbrella/internal/ml/inference"

	"go.opentelemetry.io/otel/trace"
)

type MLFeatureInferencer interface {
	RefreshFeatures(ctx context.Context) (int, error)
	RunInference(ctx context.Context) (inference.RunResult, error)
}

type MLFeatureInferenceJob struct {
	tracer       trace.Tracer
	service      MLFeatureInferencer
	pollInterval time.Duration
}

func NewMLFeatureInferenceJob(tracer trace.Tracer, service MLFeatureInferencer, pollInterval time.Duration) *MLFeatureInferenceJob {
	if pollInterval <= 0 {
		pollInterval = 15 * time.Minute
	}
	return &MLFeatureInferenceJob{tracer: tracer, service: service, pollInterval: pollInterval}
}

func (j *MLFeatureInferenceJob) Start(ctx context.Context) {
	if j.service == nil {
		log.Println("ML feature/inference job disabled: no service")
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

func (j *MLFeatureInferenceJob) runOnce(ctx context.Context) {
	_, span := j.tracer.Start(ctx, "ml-feature-inference-job.run-once")
	defer span.End()

	rows, err := j.service.RefreshFeatures(ctx)
	if err != nil {
		log.Printf("ML feature refresh error: %v", err)
		return
	}
	_, err = j.service.RunInference(ctx)
	if err != nil {
		log.Printf("ML inference error: %v", err)
		return
	}
	if rows > 0 {
		log.Printf("ML feature/inference cycle complete (%d feature rows refreshed)", rows)
	}
}

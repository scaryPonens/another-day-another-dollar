package job

import (
	"context"
	"log"
	"time"

	"bug-free-umbrella/internal/ml/training"

	"go.opentelemetry.io/otel/trace"
)

type MLTrainer interface {
	RunTraining(ctx context.Context) ([]training.ModelTrainResult, error)
}

type MLTrainingJob struct {
	tracer    trace.Tracer
	service   MLTrainer
	trainHour int
}

func NewMLTrainingJob(tracer trace.Tracer, service MLTrainer, trainHourUTC int) *MLTrainingJob {
	if trainHourUTC < 0 || trainHourUTC > 23 {
		trainHourUTC = 0
	}
	return &MLTrainingJob{tracer: tracer, service: service, trainHour: trainHourUTC}
}

func (j *MLTrainingJob) Start(ctx context.Context) {
	if j.service == nil {
		log.Println("ML training job disabled: no service")
		<-ctx.Done()
		return
	}
	for {
		next := nextRunUTC(time.Now().UTC(), j.trainHour)
		wait := time.Until(next)
		if wait < time.Second {
			wait = time.Second
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			j.runOnce(ctx)
		}
	}
}

func (j *MLTrainingJob) runOnce(ctx context.Context) {
	_, span := j.tracer.Start(ctx, "ml-training-job.run-once")
	defer span.End()

	results, err := j.service.RunTraining(ctx)
	if err != nil {
		log.Printf("ML training error: %v", err)
		return
	}
	for _, r := range results {
		log.Printf("ML training result model=%s version=%d auc=%.4f promoted=%v", r.ModelKey, r.Version, r.AUC, r.Promoted)
	}
}

func nextRunUTC(now time.Time, hour int) time.Time {
	run := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, time.UTC)
	if !run.After(now) {
		run = run.Add(24 * time.Hour)
	}
	return run
}

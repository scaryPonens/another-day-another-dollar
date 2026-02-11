package service

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/trace"
)

type WorkService struct {
	tracer trace.Tracer
}

func NewWorkService(tracer trace.Tracer) *WorkService {
	return &WorkService{tracer: tracer}
}

func (s *WorkService) DoWork(ctx context.Context) {
	ctx, span := s.tracer.Start(ctx, "do-work")
	defer span.End()

	time.Sleep(100 * time.Millisecond)

	s.doMoreWork(ctx)
}

func (s *WorkService) doMoreWork(ctx context.Context) {
	_, span := s.tracer.Start(ctx, "do-more-work")
	defer span.End()

	time.Sleep(50 * time.Millisecond)
}

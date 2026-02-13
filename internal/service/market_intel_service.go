package service

import (
	"context"
	"time"

	"bug-free-umbrella/internal/domain"
	"bug-free-umbrella/internal/marketintel"

	"go.opentelemetry.io/otel/trace"
)

type MarketIntelService struct {
	tracer trace.Tracer
	svc    *marketintel.Service
}

func NewMarketIntelService(tracer trace.Tracer, svc *marketintel.Service) *MarketIntelService {
	return &MarketIntelService{tracer: tracer, svc: svc}
}

func (s *MarketIntelService) RunMarketIntel(ctx context.Context) (domain.MarketIntelRunResult, error) {
	_, span := s.tracer.Start(ctx, "market-intel-service.run")
	defer span.End()
	if s == nil || s.svc == nil {
		return domain.MarketIntelRunResult{}, nil
	}
	return s.svc.RunCycle(ctx, time.Now().UTC())
}

package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"bug-free-umbrella/internal/domain"
	"bug-free-umbrella/internal/ml/features"
	"bug-free-umbrella/internal/ml/inference"
	"bug-free-umbrella/internal/ml/predictions"
	"bug-free-umbrella/internal/ml/training"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/trace"
)

type MLCandleRepository interface {
	GetCandles(ctx context.Context, symbol, interval string, limit int) ([]*domain.Candle, error)
	GetCandlesInRange(ctx context.Context, symbol, interval string, from, to time.Time) ([]*domain.Candle, error)
}

type MLSignalService struct {
	tracer         trace.Tracer
	candleRepo     MLCandleRepository
	featureEngine  *features.Engine
	featureRepo    *features.Repository
	trainingSvc    *training.Service
	inferenceSvc   *inference.Service
	predictionRepo *predictions.Repository

	interval        string
	targetHours     int
	trainWindowDays int
}

type MLSignalServiceConfig struct {
	Interval        string
	TargetHours     int
	TrainWindowDays int
}

func NewMLSignalService(
	tracer trace.Tracer,
	candleRepo MLCandleRepository,
	featureEngine *features.Engine,
	featureRepo *features.Repository,
	trainingSvc *training.Service,
	inferenceSvc *inference.Service,
	predictionRepo *predictions.Repository,
	cfg MLSignalServiceConfig,
) *MLSignalService {
	if cfg.Interval == "" {
		cfg.Interval = "1h"
	}
	if cfg.TargetHours <= 0 {
		cfg.TargetHours = 4
	}
	if cfg.TrainWindowDays <= 0 {
		cfg.TrainWindowDays = 90
	}
	if featureEngine == nil {
		featureEngine = features.NewEngine(nil)
	}
	return &MLSignalService{
		tracer:          tracer,
		candleRepo:      candleRepo,
		featureEngine:   featureEngine,
		featureRepo:     featureRepo,
		trainingSvc:     trainingSvc,
		inferenceSvc:    inferenceSvc,
		predictionRepo:  predictionRepo,
		interval:        cfg.Interval,
		targetHours:     cfg.TargetHours,
		trainWindowDays: cfg.TrainWindowDays,
	}
}

func (s *MLSignalService) RefreshFeatures(ctx context.Context) (int, error) {
	_, span := s.tracer.Start(ctx, "ml-signal-service.refresh-features")
	defer span.End()

	if s.candleRepo == nil || s.featureRepo == nil || s.featureEngine == nil {
		return 0, fmt.Errorf("ml feature refresh dependencies are not initialized")
	}

	rowsCount := 0
	limit := (s.trainWindowDays * 24) + s.targetHours + 48
	if limit < 500 {
		limit = 500
	}
	for _, symbol := range domain.SupportedSymbols {
		candles, err := s.candleRepo.GetCandles(ctx, symbol, s.interval, limit)
		if err != nil {
			return rowsCount, fmt.Errorf("get candles for %s: %w", symbol, err)
		}
		if len(candles) == 0 {
			continue
		}
		rows := s.featureEngine.BuildRows(candles, s.targetHours)
		if len(rows) == 0 {
			continue
		}
		if err := s.featureRepo.UpsertRows(ctx, rows); err != nil {
			return rowsCount, fmt.Errorf("upsert feature rows for %s: %w", symbol, err)
		}
		rowsCount += len(rows)
	}
	return rowsCount, nil
}

func (s *MLSignalService) RunInference(ctx context.Context) (inference.RunResult, error) {
	_, span := s.tracer.Start(ctx, "ml-signal-service.run-inference")
	defer span.End()

	if s.inferenceSvc == nil {
		return inference.RunResult{}, nil
	}
	return s.inferenceSvc.RunLatest(ctx, time.Now().UTC())
}

func (s *MLSignalService) RunTraining(ctx context.Context) ([]training.ModelTrainResult, error) {
	_, span := s.tracer.Start(ctx, "ml-signal-service.run-training")
	defer span.End()

	if s.trainingSvc == nil {
		return nil, nil
	}
	return s.trainingSvc.TrainAll(ctx, time.Now().UTC())
}

func (s *MLSignalService) ResolveOutcomes(ctx context.Context, limit int) (int, error) {
	_, span := s.tracer.Start(ctx, "ml-signal-service.resolve-outcomes")
	defer span.End()

	if s.predictionRepo == nil || s.candleRepo == nil {
		return 0, nil
	}
	if limit <= 0 {
		limit = 200
	}

	pending, err := s.predictionRepo.ListUnresolvedDue(ctx, time.Now().UTC(), limit)
	if err != nil {
		return 0, err
	}

	resolved := 0
	for i := range pending {
		pred := pending[i]
		candles, err := s.candleRepo.GetCandlesInRange(ctx, pred.Symbol, pred.Interval, pred.OpenTime, pred.TargetTime)
		if err != nil {
			return resolved, err
		}
		openClose, targetClose, ok := extractOpenAndTargetClose(candles, pred.OpenTime, pred.TargetTime)
		if !ok || openClose == 0 {
			continue
		}
		actualUp := targetClose > openClose
		predictedUp := pred.ProbUp >= 0.5
		if pred.Direction == domain.DirectionLong {
			predictedUp = true
		} else if pred.Direction == domain.DirectionShort {
			predictedUp = false
		}
		realized := (targetClose / openClose) - 1
		isCorrect := predictedUp == actualUp
		if err := s.predictionRepo.ResolvePrediction(ctx, pred.ID, actualUp, isCorrect, realized); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return resolved, err
		}
		resolved++
	}
	return resolved, nil
}

func extractOpenAndTargetClose(candles []*domain.Candle, openTime, targetTime time.Time) (float64, float64, bool) {
	if len(candles) == 0 {
		return 0, 0, false
	}
	type row struct {
		time  int64
		close float64
	}
	values := make([]row, 0, len(candles))
	for _, c := range candles {
		if c == nil {
			continue
		}
		values = append(values, row{time: c.OpenTime.UTC().Unix(), close: c.Close})
	}
	sort.Slice(values, func(i, j int) bool { return values[i].time < values[j].time })

	openTS := openTime.UTC().Unix()
	targetTS := targetTime.UTC().Unix()
	openClose := 0.0
	targetClose := 0.0
	hasOpen := false
	hasTarget := false
	for _, v := range values {
		if v.time == openTS {
			hasOpen = true
			openClose = v.close
		}
		if v.time == targetTS {
			hasTarget = true
			targetClose = v.close
		}
	}
	return openClose, targetClose, hasOpen && hasTarget
}

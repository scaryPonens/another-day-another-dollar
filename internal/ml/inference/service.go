package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"bug-free-umbrella/internal/domain"
	"bug-free-umbrella/internal/ml/common"
	"bug-free-umbrella/internal/ml/ensemble"
	"bug-free-umbrella/internal/ml/models/logreg"
	"bug-free-umbrella/internal/ml/models/xgboost"

	"go.opentelemetry.io/otel/trace"
)

type FeatureReader interface {
	ListLatestByInterval(ctx context.Context, interval string) ([]domain.MLFeatureRow, error)
}

type ModelRegistry interface {
	GetActiveModel(ctx context.Context, modelKey string) (*domain.MLModelVersion, error)
}

type PredictionStore interface {
	UpsertPrediction(ctx context.Context, prediction domain.MLPrediction) (*domain.MLPrediction, error)
	AttachSignalID(ctx context.Context, predictionID, signalID int64) error
}

type SignalStore interface {
	InsertSignals(ctx context.Context, signals []domain.Signal) ([]domain.Signal, error)
	ListSignals(ctx context.Context, filter domain.SignalFilter) ([]domain.Signal, error)
}

type Config struct {
	Interval       string
	TargetHours    int
	LongThreshold  float64
	ShortThreshold float64
}

type Service struct {
	tracer      trace.Tracer
	features    FeatureReader
	registry    ModelRegistry
	predictions PredictionStore
	signals     SignalStore
	ensemble    *ensemble.Service
	cfg         Config
}

type RunResult struct {
	Predictions int
	Signals     int
}

func NewService(
	tracer trace.Tracer,
	features FeatureReader,
	registry ModelRegistry,
	predictions PredictionStore,
	signals SignalStore,
	ensembleSvc *ensemble.Service,
	cfg Config,
) *Service {
	if cfg.Interval == "" {
		cfg.Interval = "1h"
	}
	if cfg.TargetHours <= 0 {
		cfg.TargetHours = 4
	}
	if cfg.LongThreshold <= 0 || cfg.LongThreshold >= 1 {
		cfg.LongThreshold = 0.55
	}
	if cfg.ShortThreshold <= 0 || cfg.ShortThreshold >= 1 {
		cfg.ShortThreshold = 0.45
	}
	if ensembleSvc == nil {
		ensembleSvc = ensemble.NewService()
	}
	return &Service{
		tracer:      tracer,
		features:    features,
		registry:    registry,
		predictions: predictions,
		signals:     signals,
		ensemble:    ensembleSvc,
		cfg:         cfg,
	}
}

func (s *Service) RunLatest(ctx context.Context, now time.Time) (RunResult, error) {
	_, span := s.tracer.Start(ctx, "ml-inference.run-latest")
	defer span.End()

	if s.features == nil || s.registry == nil || s.predictions == nil || s.signals == nil {
		return RunResult{}, fmt.Errorf("ml inference service is not fully initialized")
	}

	logVersion, logPredict, err := s.loadLogReg(ctx)
	if err != nil {
		return RunResult{}, err
	}
	xgbVersion, xgbPredict, err := s.loadXGBoost(ctx)
	if err != nil {
		return RunResult{}, err
	}
	if logPredict == nil && xgbPredict == nil {
		return RunResult{}, nil
	}

	rows, err := s.features.ListLatestByInterval(ctx, s.cfg.Interval)
	if err != nil {
		return RunResult{}, err
	}
	result := RunResult{}
	for i := range rows {
		row := rows[i]
		targetTime := row.OpenTime.UTC().Add(time.Duration(s.cfg.TargetHours) * time.Hour)
		features := common.FeatureVector(row)

		classicScore := s.classicScore(ctx, row)
		logProb := 0.5
		xgbProb := 0.5

		if logPredict != nil {
			logProb = common.Clamp01(logPredict(features))
			pred, hasSignal, err := s.persistModelPrediction(ctx, row, common.ModelKeyLogReg, logVersion, logProb, targetTime, 0)
			if err != nil {
				return result, err
			}
			if pred != nil {
				result.Predictions++
			}
			if hasSignal {
				result.Signals++
			}
		}

		if xgbPredict != nil {
			xgbProb = common.Clamp01(xgbPredict(features))
			pred, hasSignal, err := s.persistModelPrediction(ctx, row, common.ModelKeyXGBoost, xgbVersion, xgbProb, targetTime, 0)
			if err != nil {
				return result, err
			}
			if pred != nil {
				result.Predictions++
			}
			if hasSignal {
				result.Signals++
			}
		}

		ensembleScore := s.ensemble.Score(ensemble.Components{
			ClassicScore: classicScore,
			LogRegProb:   logProb,
			XGBoostProb:  xgbProb,
		})
		if ensembleScore > 1 {
			ensembleScore = 1
		}
		if ensembleScore < -1 {
			ensembleScore = -1
		}
		ensembleProb := common.Clamp01((ensembleScore + 1) / 2)
		version := max(logVersion, xgbVersion)
		if version <= 0 {
			version = 1
		}
		pred, hasSignal, err := s.persistModelPrediction(ctx, row, common.ModelKeyEnsembleV1, version, ensembleProb, targetTime, ensembleScore)
		if err != nil {
			return result, err
		}
		if pred != nil {
			result.Predictions++
		}
		if hasSignal {
			result.Signals++
		}
	}

	return result, nil
}

func (s *Service) persistModelPrediction(
	ctx context.Context,
	row domain.MLFeatureRow,
	modelKey string,
	modelVersion int,
	probUp float64,
	targetTime time.Time,
	ensembleScore float64,
) (*domain.MLPrediction, bool, error) {
	confidence := common.Confidence(probUp)
	direction := common.DirectionFromProb(probUp, s.cfg.LongThreshold, s.cfg.ShortThreshold)
	if modelKey == common.ModelKeyEnsembleV1 {
		direction = ensemble.Direction(ensembleScore)
	}
	risk := common.RiskFromConfidence(confidence)
	detailsJSON := s.buildDetailsJSON(modelKey, modelVersion, probUp, confidence, ensembleScore)

	pred, err := s.predictions.UpsertPrediction(ctx, domain.MLPrediction{
		Symbol:       row.Symbol,
		Interval:     row.Interval,
		OpenTime:     row.OpenTime.UTC(),
		TargetTime:   targetTime.UTC(),
		ModelKey:     modelKey,
		ModelVersion: modelVersion,
		ProbUp:       probUp,
		Confidence:   confidence,
		Direction:    direction,
		Risk:         risk,
		DetailsJSON:  detailsJSON,
	})
	if err != nil {
		return nil, false, err
	}

	if direction == domain.DirectionHold {
		return pred, false, nil
	}
	indicator := indicatorForModelKey(modelKey)
	signalDetails := signalDetails(modelKey, modelVersion, probUp, confidence, ensembleScore)
	persistedSignals, err := s.signals.InsertSignals(ctx, []domain.Signal{{
		Symbol:    row.Symbol,
		Interval:  row.Interval,
		Indicator: indicator,
		Timestamp: row.OpenTime.UTC(),
		Risk:      risk,
		Direction: direction,
		Details:   signalDetails,
	}})
	if err != nil {
		return pred, false, err
	}
	if len(persistedSignals) > 0 && persistedSignals[0].ID > 0 {
		if err := s.predictions.AttachSignalID(ctx, pred.ID, persistedSignals[0].ID); err != nil {
			return pred, false, err
		}
	}
	return pred, true, nil
}

func (s *Service) loadLogReg(ctx context.Context) (int, func([]float64) float64, error) {
	active, err := s.registry.GetActiveModel(ctx, common.ModelKeyLogReg)
	if err != nil || active == nil {
		return 0, nil, err
	}
	model, err := logreg.UnmarshalBinary(active.ArtifactBlob)
	if err != nil {
		return 0, nil, err
	}
	return active.Version, model.PredictProb, nil
}

func (s *Service) loadXGBoost(ctx context.Context) (int, func([]float64) float64, error) {
	active, err := s.registry.GetActiveModel(ctx, common.ModelKeyXGBoost)
	if err != nil || active == nil {
		return 0, nil, err
	}
	model, err := xgboost.UnmarshalBinary(active.ArtifactBlob)
	if err != nil {
		return 0, nil, err
	}
	return active.Version, model.PredictProb, nil
}

func (s *Service) classicScore(ctx context.Context, row domain.MLFeatureRow) float64 {
	signals, err := s.signals.ListSignals(ctx, domain.SignalFilter{Symbol: row.Symbol, Limit: 100})
	if err != nil {
		return 0
	}
	targetTS := row.OpenTime.UTC().Unix()
	weighted := 0.0
	weightTotal := 0.0
	for i := range signals {
		sig := signals[i]
		if sig.Interval != row.Interval || sig.Timestamp.UTC().Unix() != targetTS {
			continue
		}
		if !isClassicIndicator(sig.Indicator) {
			continue
		}
		dir := 0.0
		switch sig.Direction {
		case domain.DirectionLong:
			dir = 1
		case domain.DirectionShort:
			dir = -1
		default:
			dir = 0
		}
		weight := (6.0 - float64(sig.Risk)) / 5.0
		if weight < 0 {
			weight = 0
		}
		weighted += dir * weight
		weightTotal += weight
	}
	if weightTotal == 0 {
		return 0
	}
	score := weighted / weightTotal
	if score > 1 {
		return 1
	}
	if score < -1 {
		return -1
	}
	return score
}

func (s *Service) buildDetailsJSON(modelKey string, version int, probUp, confidence, ensembleScore float64) string {
	payload := map[string]any{
		"model_key":     modelKey,
		"model_version": version,
		"prob_up":       roundFloat(probUp),
		"confidence":    roundFloat(confidence),
		"target":        "4h",
	}
	if modelKey == common.ModelKeyEnsembleV1 {
		payload["ensemble_score"] = roundFloat(ensembleScore)
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func signalDetails(modelKey string, version int, probUp, confidence, ensembleScore float64) string {
	if modelKey == common.ModelKeyEnsembleV1 {
		return fmt.Sprintf(
			"model_key=%s;model_version=%d;prob_up=%.4f;confidence=%.4f;target=4h;ensemble_score=%.4f",
			modelKey, version, probUp, confidence, ensembleScore,
		)
	}
	return fmt.Sprintf(
		"model_key=%s;model_version=%d;prob_up=%.4f;confidence=%.4f;target=4h",
		modelKey, version, probUp, confidence,
	)
}

func indicatorForModelKey(modelKey string) string {
	switch modelKey {
	case common.ModelKeyLogReg:
		return domain.IndicatorMLLogRegUp4H
	case common.ModelKeyXGBoost:
		return domain.IndicatorMLXGBoostUp4H
	default:
		return domain.IndicatorMLEnsembleUp4H
	}
}

func isClassicIndicator(indicator string) bool {
	switch indicator {
	case domain.IndicatorRSI, domain.IndicatorMACD, domain.IndicatorBollinger, domain.IndicatorVolumeZ:
		return true
	default:
		return false
	}
}

func roundFloat(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v*10000) / 10000
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

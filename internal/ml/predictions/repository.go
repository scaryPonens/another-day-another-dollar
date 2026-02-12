package predictions

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"bug-free-umbrella/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"go.opentelemetry.io/otel/trace"
)

type pool interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Repository struct {
	pool   pool
	tracer trace.Tracer
}

func NewRepository(pool pool, tracer trace.Tracer) *Repository {
	return &Repository{pool: pool, tracer: tracer}
}

func (r *Repository) UpsertPrediction(ctx context.Context, prediction domain.MLPrediction) (*domain.MLPrediction, error) {
	_, span := r.tracer.Start(ctx, "ml-predictions.upsert")
	defer span.End()

	details := prediction.DetailsJSON
	if details == "" {
		details = "{}"
	}
	if !json.Valid([]byte(details)) {
		details = `{"raw":"invalid"}`
	}

	row := r.pool.QueryRow(ctx, `
INSERT INTO ml_predictions (
    symbol, interval, open_time, target_time,
    model_key, model_version,
    prob_up, confidence, direction, risk,
    signal_id, details_json
) VALUES (
    $1, $2, $3, $4,
    $5, $6,
    $7, $8, $9, $10,
    $11, $12
)
ON CONFLICT (symbol, interval, open_time, model_key, model_version) DO UPDATE SET
    prob_up = EXCLUDED.prob_up,
    confidence = EXCLUDED.confidence,
    direction = EXCLUDED.direction,
    risk = EXCLUDED.risk,
    details_json = EXCLUDED.details_json,
    target_time = EXCLUDED.target_time
RETURNING id, symbol, interval, open_time, target_time,
          model_key, model_version,
          prob_up, confidence, direction, risk,
          signal_id, details_json,
          created_at, resolved_at, actual_up, is_correct, realized_return`,
		prediction.Symbol,
		prediction.Interval,
		prediction.OpenTime.UTC(),
		prediction.TargetTime.UTC(),
		prediction.ModelKey,
		prediction.ModelVersion,
		prediction.ProbUp,
		prediction.Confidence,
		string(prediction.Direction),
		int16(prediction.Risk),
		prediction.SignalID,
		details,
	)
	out, err := scanPredictionRow(row)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) AttachSignalID(ctx context.Context, predictionID, signalID int64) error {
	_, span := r.tracer.Start(ctx, "ml-predictions.attach-signal")
	defer span.End()

	tag, err := r.pool.Exec(ctx, `UPDATE ml_predictions SET signal_id = $2 WHERE id = $1`, predictionID, signalID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) ListUnresolvedDue(ctx context.Context, cutoff time.Time, limit int) ([]domain.MLPrediction, error) {
	_, span := r.tracer.Start(ctx, "ml-predictions.list-unresolved-due")
	defer span.End()

	if limit <= 0 {
		limit = 200
	}
	rows, err := r.pool.Query(ctx, `
SELECT id, symbol, interval, open_time, target_time,
       model_key, model_version,
       prob_up, confidence, direction, risk,
       signal_id, details_json,
       created_at, resolved_at, actual_up, is_correct, realized_return
FROM ml_predictions
WHERE resolved_at IS NULL
  AND target_time <= $1
ORDER BY target_time ASC
LIMIT $2`, cutoff.UTC(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.MLPrediction, 0, limit)
	for rows.Next() {
		p, err := scanPredictionRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *Repository) ResolvePrediction(ctx context.Context, predictionID int64, actualUp bool, isCorrect bool, realizedReturn float64) error {
	_, span := r.tracer.Start(ctx, "ml-predictions.resolve")
	defer span.End()

	tag, err := r.pool.Exec(ctx, `
UPDATE ml_predictions
SET resolved_at = NOW(),
    actual_up = $2,
    is_correct = $3,
    realized_return = $4
WHERE id = $1
  AND resolved_at IS NULL`, predictionID, actualUp, isCorrect, realizedReturn)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanPredictionRow(s scanner) (*domain.MLPrediction, error) {
	var out domain.MLPrediction
	var direction string
	var risk int16
	var resolvedAt pgtype.Timestamptz
	var actualUp pgtype.Bool
	var isCorrect pgtype.Bool
	var realizedReturn pgtype.Float8

	if err := s.Scan(
		&out.ID,
		&out.Symbol,
		&out.Interval,
		&out.OpenTime,
		&out.TargetTime,
		&out.ModelKey,
		&out.ModelVersion,
		&out.ProbUp,
		&out.Confidence,
		&direction,
		&risk,
		&out.SignalID,
		&out.DetailsJSON,
		&out.CreatedAt,
		&resolvedAt,
		&actualUp,
		&isCorrect,
		&realizedReturn,
	); err != nil {
		return nil, err
	}
	out.Direction = domain.SignalDirection(direction)
	out.Risk = domain.RiskLevel(risk)
	out.OpenTime = out.OpenTime.UTC()
	out.TargetTime = out.TargetTime.UTC()
	out.CreatedAt = out.CreatedAt.UTC()

	if resolvedAt.Valid {
		t := resolvedAt.Time.UTC()
		out.ResolvedAt = &t
	}
	if actualUp.Valid {
		v := actualUp.Bool
		out.ActualUp = &v
	}
	if isCorrect.Valid {
		v := isCorrect.Bool
		out.IsCorrect = &v
	}
	if realizedReturn.Valid {
		v := realizedReturn.Float64
		out.RealizedReturn = &v
	}
	return &out, nil
}

var errInvalidJSON = errors.New("invalid JSON payload")

func EnsureValidJSON(raw string) error {
	if raw == "" || json.Valid([]byte(raw)) {
		return nil
	}
	return errInvalidJSON
}

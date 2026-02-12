package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"bug-free-umbrella/internal/domain"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/trace"
)

type SignalRepository struct {
	pool   PgxPool
	tracer trace.Tracer
}

func NewSignalRepository(pool PgxPool, tracer trace.Tracer) *SignalRepository {
	return &SignalRepository{pool: pool, tracer: tracer}
}

func (r *SignalRepository) InsertSignals(ctx context.Context, signals []domain.Signal) error {
	if len(signals) == 0 {
		return nil
	}

	_, span := r.tracer.Start(ctx, "signal-repo.insert-signals")
	defer span.End()

	batch := &pgx.Batch{}
	for _, s := range signals {
		batch.Queue(
			`INSERT INTO signals (symbol, interval, indicator, direction, risk, timestamp, details)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 ON CONFLICT (symbol, interval, indicator, timestamp, direction) DO NOTHING`,
			s.Symbol,
			s.Interval,
			s.Indicator,
			string(s.Direction),
			int16(s.Risk),
			s.Timestamp.UTC(),
			s.Details,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range signals {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}

	return nil
}

func (r *SignalRepository) ListSignals(ctx context.Context, filter domain.SignalFilter) ([]domain.Signal, error) {
	_, span := r.tracer.Start(ctx, "signal-repo.list-signals")
	defer span.End()

	args := make([]any, 0, 4)
	var sb strings.Builder
	sb.WriteString(`SELECT symbol, interval, indicator, direction, risk, timestamp, details
		FROM signals
		WHERE 1=1`)

	if filter.Symbol != "" {
		args = append(args, strings.ToUpper(filter.Symbol))
		sb.WriteString(fmt.Sprintf(" AND symbol = $%d", len(args)))
	}
	if filter.Risk != nil {
		args = append(args, int16(*filter.Risk))
		sb.WriteString(fmt.Sprintf(" AND risk = $%d", len(args)))
	}
	if filter.Indicator != "" {
		args = append(args, strings.ToLower(filter.Indicator))
		sb.WriteString(fmt.Sprintf(" AND indicator = $%d", len(args)))
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	args = append(args, limit)
	sb.WriteString(fmt.Sprintf(" ORDER BY timestamp DESC LIMIT $%d", len(args)))

	rows, err := r.pool.Query(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	signals := make([]domain.Signal, 0, limit)
	for rows.Next() {
		var s domain.Signal
		var direction string
		var risk int16
		var ts time.Time

		if err := rows.Scan(&s.Symbol, &s.Interval, &s.Indicator, &direction, &risk, &ts, &s.Details); err != nil {
			return nil, err
		}
		s.Direction = domain.SignalDirection(direction)
		s.Risk = domain.RiskLevel(risk)
		s.Timestamp = ts.UTC()
		signals = append(signals, s)
	}

	return signals, rows.Err()
}

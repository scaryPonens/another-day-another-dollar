package repository

import (
	"context"
	"time"

	"bug-free-umbrella/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.opentelemetry.io/otel/trace"
)

const createCandlesTable = `
CREATE TABLE IF NOT EXISTS candles (
    symbol      TEXT        NOT NULL,
    interval    TEXT        NOT NULL,
    open_time   TIMESTAMPTZ NOT NULL,
    open        NUMERIC     NOT NULL,
    high        NUMERIC     NOT NULL,
    low         NUMERIC     NOT NULL,
    close       NUMERIC     NOT NULL,
    volume      NUMERIC     NOT NULL,
    PRIMARY KEY (symbol, interval, open_time)
);

CREATE INDEX IF NOT EXISTS idx_candles_symbol_interval_time
    ON candles (symbol, interval, open_time DESC);
`

type PgxPool interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type CandleRepository struct {
	pool   PgxPool
	tracer trace.Tracer
}

func NewCandleRepository(pool PgxPool, tracer trace.Tracer) *CandleRepository {
	return &CandleRepository{pool: pool, tracer: tracer}
}

func (r *CandleRepository) RunMigrations(ctx context.Context) error {
	_, span := r.tracer.Start(ctx, "candle-repo.run-migrations")
	defer span.End()

	_, err := r.pool.Exec(ctx, createCandlesTable)
	return err
}

func (r *CandleRepository) UpsertCandles(ctx context.Context, candles []*domain.Candle) error {
	if len(candles) == 0 {
		return nil
	}

	_, span := r.tracer.Start(ctx, "candle-repo.upsert-candles")
	defer span.End()

	batch := &pgx.Batch{}
	for _, c := range candles {
		batch.Queue(
			`INSERT INTO candles (symbol, interval, open_time, open, high, low, close, volume)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			 ON CONFLICT (symbol, interval, open_time) DO UPDATE SET
			     open = EXCLUDED.open,
			     high = EXCLUDED.high,
			     low = EXCLUDED.low,
			     close = EXCLUDED.close,
			     volume = EXCLUDED.volume`,
			c.Symbol, c.Interval, c.OpenTime, c.Open, c.High, c.Low, c.Close, c.Volume,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range candles {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (r *CandleRepository) GetCandles(ctx context.Context, symbol, interval string, limit int) ([]*domain.Candle, error) {
	_, span := r.tracer.Start(ctx, "candle-repo.get-candles")
	defer span.End()

	rows, err := r.pool.Query(ctx,
		`SELECT symbol, interval, open_time, open, high, low, close, volume
		 FROM candles
		 WHERE symbol = $1 AND interval = $2
		 ORDER BY open_time DESC
		 LIMIT $3`,
		symbol, interval, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candles []*domain.Candle
	for rows.Next() {
		c := &domain.Candle{}
		if err := rows.Scan(&c.Symbol, &c.Interval, &c.OpenTime, &c.Open, &c.High, &c.Low, &c.Close, &c.Volume); err != nil {
			return nil, err
		}
		candles = append(candles, c)
	}
	return candles, rows.Err()
}

func (r *CandleRepository) GetCandlesInRange(ctx context.Context, symbol, interval string, from, to time.Time) ([]*domain.Candle, error) {
	_, span := r.tracer.Start(ctx, "candle-repo.get-candles-in-range")
	defer span.End()

	rows, err := r.pool.Query(ctx,
		`SELECT symbol, interval, open_time, open, high, low, close, volume
		 FROM candles
		 WHERE symbol = $1 AND interval = $2 AND open_time >= $3 AND open_time <= $4
		 ORDER BY open_time DESC`,
		symbol, interval, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candles []*domain.Candle
	for rows.Next() {
		c := &domain.Candle{}
		if err := rows.Scan(&c.Symbol, &c.Interval, &c.OpenTime, &c.Open, &c.High, &c.Low, &c.Close, &c.Volume); err != nil {
			return nil, err
		}
		candles = append(candles, c)
	}
	return candles, rows.Err()
}

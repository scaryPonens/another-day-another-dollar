package features

import (
	"context"
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
}

type Repository struct {
	pool   pool
	tracer trace.Tracer
}

func NewRepository(pool pool, tracer trace.Tracer) *Repository {
	return &Repository{pool: pool, tracer: tracer}
}

func (r *Repository) UpsertRows(ctx context.Context, rows []domain.MLFeatureRow) error {
	if len(rows) == 0 {
		return nil
	}
	_, span := r.tracer.Start(ctx, "ml-feature-repo.upsert")
	defer span.End()

	for i := range rows {
		row := rows[i]
		_, err := r.pool.Exec(ctx, `
INSERT INTO ml_feature_rows (
    symbol, interval, open_time,
    ret_1h, ret_4h, ret_12h, ret_24h,
    volatility_6h, volatility_24h, volume_z_24h,
    rsi_14, macd_line, macd_signal, macd_hist,
    bb_pos, bb_width, target_up_4h, updated_at
) VALUES (
    $1, $2, $3,
    $4, $5, $6, $7,
    $8, $9, $10,
    $11, $12, $13, $14,
    $15, $16, $17, NOW()
)
ON CONFLICT (symbol, interval, open_time) DO UPDATE SET
    ret_1h = EXCLUDED.ret_1h,
    ret_4h = EXCLUDED.ret_4h,
    ret_12h = EXCLUDED.ret_12h,
    ret_24h = EXCLUDED.ret_24h,
    volatility_6h = EXCLUDED.volatility_6h,
    volatility_24h = EXCLUDED.volatility_24h,
    volume_z_24h = EXCLUDED.volume_z_24h,
    rsi_14 = EXCLUDED.rsi_14,
    macd_line = EXCLUDED.macd_line,
    macd_signal = EXCLUDED.macd_signal,
    macd_hist = EXCLUDED.macd_hist,
    bb_pos = EXCLUDED.bb_pos,
    bb_width = EXCLUDED.bb_width,
    target_up_4h = EXCLUDED.target_up_4h,
    updated_at = NOW()`,
			row.Symbol,
			row.Interval,
			row.OpenTime.UTC(),
			row.Ret1H,
			row.Ret4H,
			row.Ret12H,
			row.Ret24H,
			row.Volatility6H,
			row.Volatility24H,
			row.VolumeZ24H,
			row.RSI14,
			row.MACDLine,
			row.MACDSignal,
			row.MACDHist,
			row.BBPos,
			row.BBWidth,
			row.TargetUp4H,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) ListLabeledRows(ctx context.Context, interval string, from, to time.Time) ([]domain.MLFeatureRow, error) {
	_, span := r.tracer.Start(ctx, "ml-feature-repo.list-labeled")
	defer span.End()

	rows, err := r.pool.Query(ctx, `
SELECT symbol, interval, open_time,
       ret_1h, ret_4h, ret_12h, ret_24h,
       volatility_6h, volatility_24h, volume_z_24h,
       rsi_14, macd_line, macd_signal, macd_hist,
       bb_pos, bb_width, target_up_4h, created_at, updated_at
FROM ml_feature_rows
WHERE interval = $1
  AND open_time >= $2
  AND open_time <= $3
  AND target_up_4h IS NOT NULL
ORDER BY open_time ASC`, interval, from.UTC(), to.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFeatureRows(rows)
}

func (r *Repository) ListRows(ctx context.Context, interval string, from, to time.Time) ([]domain.MLFeatureRow, error) {
	_, span := r.tracer.Start(ctx, "ml-feature-repo.list")
	defer span.End()

	rows, err := r.pool.Query(ctx, `
SELECT symbol, interval, open_time,
       ret_1h, ret_4h, ret_12h, ret_24h,
       volatility_6h, volatility_24h, volume_z_24h,
       rsi_14, macd_line, macd_signal, macd_hist,
       bb_pos, bb_width, target_up_4h, created_at, updated_at
FROM ml_feature_rows
WHERE interval = $1
  AND open_time >= $2
  AND open_time <= $3
ORDER BY open_time ASC`, interval, from.UTC(), to.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFeatureRows(rows)
}

func (r *Repository) ListLatestByInterval(ctx context.Context, interval string) ([]domain.MLFeatureRow, error) {
	_, span := r.tracer.Start(ctx, "ml-feature-repo.list-latest")
	defer span.End()

	rows, err := r.pool.Query(ctx, `
SELECT DISTINCT ON (symbol)
       symbol, interval, open_time,
       ret_1h, ret_4h, ret_12h, ret_24h,
       volatility_6h, volatility_24h, volume_z_24h,
       rsi_14, macd_line, macd_signal, macd_hist,
       bb_pos, bb_width, target_up_4h, created_at, updated_at
FROM ml_feature_rows
WHERE interval = $1
ORDER BY symbol, open_time DESC`, interval)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFeatureRows(rows)
}

func scanFeatureRows(rows pgx.Rows) ([]domain.MLFeatureRow, error) {
	result := make([]domain.MLFeatureRow, 0)
	for rows.Next() {
		var row domain.MLFeatureRow
		var target pgtype.Bool
		if err := rows.Scan(
			&row.Symbol,
			&row.Interval,
			&row.OpenTime,
			&row.Ret1H,
			&row.Ret4H,
			&row.Ret12H,
			&row.Ret24H,
			&row.Volatility6H,
			&row.Volatility24H,
			&row.VolumeZ24H,
			&row.RSI14,
			&row.MACDLine,
			&row.MACDSignal,
			&row.MACDHist,
			&row.BBPos,
			&row.BBWidth,
			&target,
			&row.CreatedAt,
			&row.UpdatedAt,
		); err != nil {
			return nil, err
		}
		row.OpenTime = row.OpenTime.UTC()
		row.CreatedAt = row.CreatedAt.UTC()
		row.UpdatedAt = row.UpdatedAt.UTC()
		if target.Valid {
			v := target.Bool
			row.TargetUp4H = &v
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

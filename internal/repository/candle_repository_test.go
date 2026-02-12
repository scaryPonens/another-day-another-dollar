package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bug-free-umbrella/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.opentelemetry.io/otel/trace"
)

func TestUpsertCandlesBatchesStatements(t *testing.T) {
	batchResults := &stubBatchResults{}
	pool := &stubPool{batchResults: batchResults}
	repo := NewCandleRepository(pool, trace.NewNoopTracerProvider().Tracer("test"))

	candles := []*domain.Candle{
		{Symbol: "BTC", Interval: "1h", OpenTime: time.Unix(0, 0)},
		{Symbol: "ETH", Interval: "1h", OpenTime: time.Unix(3600, 0)},
	}
	if err := repo.UpsertCandles(context.Background(), candles); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pool.queuedBatch == nil || pool.queuedBatch.Len() != len(candles) {
		t.Fatalf("expected batch of size %d", len(candles))
	}
	if batchResults.execCalls != len(candles) {
		t.Fatalf("expected %d Exec calls, got %d", len(candles), batchResults.execCalls)
	}
}

func TestGetCandlesReturnsRows(t *testing.T) {
	rows := [][]any{{
		"BTC", "1h", time.Unix(0, 0), 1.0, 2.0, 0.5, 1.5, 100.0,
	}}
	pool := &stubPool{rowsData: rows}
	repo := NewCandleRepository(pool, trace.NewNoopTracerProvider().Tracer("test"))

	candles, err := repo.GetCandles(context.Background(), "BTC", "1h", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candles) != 1 || candles[0].Symbol != "BTC" {
		t.Fatalf("unexpected candles: %+v", candles)
	}
}

func TestGetCandlesInRange(t *testing.T) {
	now := time.Now().UTC()
	rows := [][]any{{
		"ETH", "4h", now, 10.0, 12.0, 8.0, 11.0, 200.0,
	}}
	pool := &stubPool{rowsData: rows}
	repo := NewCandleRepository(pool, trace.NewNoopTracerProvider().Tracer("test"))

	candles, err := repo.GetCandlesInRange(context.Background(), "ETH", "4h", now.Add(-time.Hour), now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candles) != 1 || candles[0].Interval != "4h" {
		t.Fatalf("unexpected candles: %+v", candles)
	}
}

type stubPool struct {
	batchResults pgx.BatchResults
	queuedBatch  *pgx.Batch
	rowsData     [][]any
}

func (s *stubPool) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (s *stubPool) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	s.queuedBatch = b
	if s.batchResults != nil {
		return s.batchResults
	}
	return &stubBatchResults{}
}

func (s *stubPool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if s.rowsData == nil {
		return &stubRows{}, nil
	}
	dataCopy := make([][]any, len(s.rowsData))
	for i := range s.rowsData {
		row := make([]any, len(s.rowsData[i]))
		copy(row, s.rowsData[i])
		dataCopy[i] = row
	}
	return &stubRows{data: dataCopy}, nil
}

type stubBatchResults struct {
	execCalls int
}

func (s *stubBatchResults) Exec() (pgconn.CommandTag, error) {
	s.execCalls++
	return pgconn.CommandTag{}, nil
}

func (s *stubBatchResults) Query() (pgx.Rows, error) { return &stubRows{}, nil }

func (s *stubBatchResults) QueryRow() pgx.Row { return &stubRow{} }

func (s *stubBatchResults) Close() error { return nil }

type stubRows struct {
	data [][]any
	idx  int
}

func (r *stubRows) Close() {}

func (r *stubRows) Err() error { return nil }

func (r *stubRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }

func (r *stubRows) FieldDescriptions() []pgconn.FieldDescription { return nil }

func (r *stubRows) Next() bool {
	if len(r.data) == 0 || r.idx >= len(r.data) {
		return false
	}
	r.idx++
	return true
}

func (r *stubRows) Scan(dest ...any) error {
	if r.idx == 0 || r.idx > len(r.data) {
		return fmt.Errorf("invalid scan index")
	}
	row := r.data[r.idx-1]
	for i, d := range dest {
		switch ptr := d.(type) {
		case *string:
			*ptr = row[i].(string)
		case *time.Time:
			*ptr = row[i].(time.Time)
		case *float64:
			*ptr = row[i].(float64)
		default:
			return fmt.Errorf("unsupported dest type %T", d)
		}
	}
	return nil
}

func (r *stubRows) Values() ([]any, error) { return nil, nil }

func (r *stubRows) RawValues() [][]byte { return nil }

func (r *stubRows) Conn() *pgx.Conn { return nil }

type stubRow struct{}

func (stubRow) Scan(dest ...any) error { return nil }

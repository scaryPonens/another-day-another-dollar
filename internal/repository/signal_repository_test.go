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

func TestSignalInsertSignalsBatchesStatements(t *testing.T) {
	batchResults := &signalStubBatchResults{}
	pool := &signalStubPool{batchResults: batchResults}
	repo := NewSignalRepository(pool, trace.NewNoopTracerProvider().Tracer("test"))

	signals := []domain.Signal{
		{
			Symbol:    "BTC",
			Interval:  "1h",
			Indicator: domain.IndicatorRSI,
			Direction: domain.DirectionLong,
			Risk:      domain.RiskLevel3,
			Timestamp: time.Unix(0, 0).UTC(),
		},
		{
			Symbol:    "ETH",
			Interval:  "15m",
			Indicator: domain.IndicatorMACD,
			Direction: domain.DirectionShort,
			Risk:      domain.RiskLevel4,
			Timestamp: time.Unix(3600, 0).UTC(),
		},
	}
	if err := repo.InsertSignals(context.Background(), signals); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pool.queuedBatch == nil || pool.queuedBatch.Len() != len(signals) {
		t.Fatalf("expected batch of size %d", len(signals))
	}
	if batchResults.execCalls != len(signals) {
		t.Fatalf("expected %d Exec calls, got %d", len(signals), batchResults.execCalls)
	}
}

func TestSignalListSignalsReturnsRows(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	rows := [][]any{{
		"BTC", "1h", domain.IndicatorRSI, string(domain.DirectionLong), int16(domain.RiskLevel2), now, "rsi crossed below 30",
	}}
	pool := &signalStubPool{rowsData: rows}
	repo := NewSignalRepository(pool, trace.NewNoopTracerProvider().Tracer("test"))

	risk := domain.RiskLevel2
	signals, err := repo.ListSignals(context.Background(), domain.SignalFilter{
		Symbol: "btc",
		Risk:   &risk,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(signals))
	}
	if signals[0].Symbol != "BTC" || signals[0].Direction != domain.DirectionLong || signals[0].Risk != domain.RiskLevel2 {
		t.Fatalf("unexpected signal payload: %+v", signals[0])
	}
}

type signalStubPool struct {
	batchResults pgx.BatchResults
	queuedBatch  *pgx.Batch
	rowsData     [][]any
}

func (s *signalStubPool) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (s *signalStubPool) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	s.queuedBatch = b
	if s.batchResults != nil {
		return s.batchResults
	}
	return &signalStubBatchResults{}
}

func (s *signalStubPool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if s.rowsData == nil {
		return &signalStubRows{}, nil
	}
	dataCopy := make([][]any, len(s.rowsData))
	for i := range s.rowsData {
		row := make([]any, len(s.rowsData[i]))
		copy(row, s.rowsData[i])
		dataCopy[i] = row
	}
	return &signalStubRows{data: dataCopy}, nil
}

type signalStubBatchResults struct {
	execCalls int
}

func (s *signalStubBatchResults) Exec() (pgconn.CommandTag, error) {
	s.execCalls++
	return pgconn.CommandTag{}, nil
}

func (s *signalStubBatchResults) Query() (pgx.Rows, error) { return &signalStubRows{}, nil }

func (s *signalStubBatchResults) QueryRow() pgx.Row { return &signalStubRow{} }

func (s *signalStubBatchResults) Close() error { return nil }

type signalStubRows struct {
	data [][]any
	idx  int
}

func (r *signalStubRows) Close() {}

func (r *signalStubRows) Err() error { return nil }

func (r *signalStubRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }

func (r *signalStubRows) FieldDescriptions() []pgconn.FieldDescription { return nil }

func (r *signalStubRows) Next() bool {
	if len(r.data) == 0 || r.idx >= len(r.data) {
		return false
	}
	r.idx++
	return true
}

func (r *signalStubRows) Scan(dest ...any) error {
	if r.idx == 0 || r.idx > len(r.data) {
		return fmt.Errorf("invalid scan index")
	}
	row := r.data[r.idx-1]
	for i, d := range dest {
		switch ptr := d.(type) {
		case *string:
			*ptr = row[i].(string)
		case *int16:
			*ptr = row[i].(int16)
		case *time.Time:
			*ptr = row[i].(time.Time)
		default:
			return fmt.Errorf("unsupported dest type %T", d)
		}
	}
	return nil
}

func (r *signalStubRows) Values() ([]any, error) { return nil, nil }

func (r *signalStubRows) RawValues() [][]byte { return nil }

func (r *signalStubRows) Conn() *pgx.Conn { return nil }

type signalStubRow struct{}

func (signalStubRow) Scan(dest ...any) error { return nil }

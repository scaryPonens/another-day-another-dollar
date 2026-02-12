package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.opentelemetry.io/otel/trace"
)

func TestConversationAppendMessageExecsInsert(t *testing.T) {
	pool := &convStubPool{}
	repo := NewConversationRepository(pool, trace.NewNoopTracerProvider().Tracer("test"))

	if err := repo.AppendMessage(context.Background(), 123, "user", "hello"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pool.execCount != 1 {
		t.Fatalf("expected 1 exec call, got %d", pool.execCount)
	}
}

func TestConversationRecentMessagesReturnsChronological(t *testing.T) {
	t1 := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 1, 1, 10, 1, 0, 0, time.UTC)
	// Rows come back newest-first from the query
	rows := [][]any{
		{"assistant", "hi there", t2},
		{"user", "hello", t1},
	}
	pool := &convStubPool{rowsData: rows}
	repo := NewConversationRepository(pool, trace.NewNoopTracerProvider().Tracer("test"))

	messages, err := repo.RecentMessages(context.Background(), 123, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
	// After reversal, oldest first
	if messages[0].Role != "user" || messages[0].Content != "hello" {
		t.Fatalf("expected first message to be user/hello, got %+v", messages[0])
	}
	if messages[1].Role != "assistant" || messages[1].Content != "hi there" {
		t.Fatalf("expected second message to be assistant/hi there, got %+v", messages[1])
	}
}

func TestConversationRecentMessagesEmptyResult(t *testing.T) {
	pool := &convStubPool{}
	repo := NewConversationRepository(pool, trace.NewNoopTracerProvider().Tracer("test"))

	messages, err := repo.RecentMessages(context.Background(), 999, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(messages))
	}
}

// --- stubs ---

type convStubPool struct {
	execCount int
	rowsData  [][]any
}

func (s *convStubPool) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	s.execCount++
	return pgconn.CommandTag{}, nil
}

func (s *convStubPool) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return &convStubBatchResults{}
}

func (s *convStubPool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if s.rowsData == nil {
		return &convStubRows{}, nil
	}
	dataCopy := make([][]any, len(s.rowsData))
	for i := range s.rowsData {
		row := make([]any, len(s.rowsData[i]))
		copy(row, s.rowsData[i])
		dataCopy[i] = row
	}
	return &convStubRows{data: dataCopy}, nil
}

type convStubBatchResults struct{}

func (convStubBatchResults) Exec() (pgconn.CommandTag, error) { return pgconn.CommandTag{}, nil }
func (convStubBatchResults) Query() (pgx.Rows, error)         { return &convStubRows{}, nil }
func (convStubBatchResults) QueryRow() pgx.Row                { return &convStubRow{} }
func (convStubBatchResults) Close() error                     { return nil }

type convStubRows struct {
	data [][]any
	idx  int
}

func (r *convStubRows) Close()                                    {}
func (r *convStubRows) Err() error                                { return nil }
func (r *convStubRows) CommandTag() pgconn.CommandTag              { return pgconn.CommandTag{} }
func (r *convStubRows) FieldDescriptions() []pgconn.FieldDescription { return nil }

func (r *convStubRows) Next() bool {
	if len(r.data) == 0 || r.idx >= len(r.data) {
		return false
	}
	r.idx++
	return true
}

func (r *convStubRows) Scan(dest ...any) error {
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
		default:
			return fmt.Errorf("unsupported dest type %T", d)
		}
	}
	return nil
}

func (r *convStubRows) Values() ([]any, error) { return nil, nil }
func (r *convStubRows) RawValues() [][]byte     { return nil }
func (r *convStubRows) Conn() *pgx.Conn        { return nil }

type convStubRow struct{}

func (convStubRow) Scan(dest ...any) error { return nil }

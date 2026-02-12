package job

import (
	"context"
	"testing"
	"time"

	"bug-free-umbrella/internal/domain"

	"go.opentelemetry.io/otel/trace"
)

func TestNewPricePollerInterval(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	poller := NewPricePoller(tracer, &stubPriceService{}, 2)
	if poller.pollInterval != 2*time.Second {
		t.Fatalf("expected 2s interval, got %v", poller.pollInterval)
	}
}

func TestPricePollerStart(t *testing.T) {
	t.Parallel()

	tracer := trace.NewNoopTracerProvider().Tracer("test")
	stub := &stubPriceService{}
	poller := NewPricePoller(tracer, stub, 1)

	ctx, cancel := context.WithCancel(context.Background())
	go poller.Start(ctx)

	eventually(t, func() bool { return stub.refreshPricesCalls > 0 })
	cancel()
}

func TestFetchShortBatch(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	stub := &stubPriceService{}
	poller := NewPricePoller(tracer, stub, 1)

	idx := 0
	poller.fetchShortBatch(context.Background(), &idx, 3)

	if len(stub.shortSymbols) != 3 {
		t.Fatalf("expected 3 symbols, got %d", len(stub.shortSymbols))
	}
	if stub.shortSymbols[0] != domain.SupportedSymbols[0] {
		t.Fatalf("unexpected symbol order: %+v", stub.shortSymbols)
	}
}

func TestFetchLongBatch(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	stub := &stubPriceService{}
	poller := NewPricePoller(tracer, stub, 1)

	idx := 0
	poller.fetchLongBatch(context.Background(), &idx)

	if len(stub.longSymbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(stub.longSymbols))
	}
	if stub.longSymbols[0] != domain.SupportedSymbols[0] {
		t.Fatalf("unexpected symbol: %+v", stub.longSymbols)
	}
}

func eventually(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(100 * time.Millisecond)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met")
}

type stubPriceService struct {
	refreshPricesCalls int
	shortSymbols       []string
	longSymbols        []string
}

func (s *stubPriceService) RefreshPrices(ctx context.Context) error {
	s.refreshPricesCalls++
	return nil
}

func (s *stubPriceService) RefreshShortCandles(ctx context.Context, symbol string) error {
	s.shortSymbols = append(s.shortSymbols, symbol)
	return nil
}

func (s *stubPriceService) RefreshLongCandles(ctx context.Context, symbol string) error {
	s.longSymbols = append(s.longSymbols, symbol)
	return nil
}

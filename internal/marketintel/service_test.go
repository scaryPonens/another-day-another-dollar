package marketintel

import (
	"context"
	"testing"
	"time"

	"bug-free-umbrella/internal/domain"
	"bug-free-umbrella/internal/provider"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/trace"
)

func TestServiceRunCycleWritesDirectionalSignalsOnly(t *testing.T) {
	now := time.Date(2026, 2, 13, 19, 30, 0, 0, time.UTC)
	store := &marketStoreStub{
		averagesBySymbol: map[string]map[string]SourceSentimentStats{
			"BTC": {
				"news":       {Score: 0.9, Confidence: 0.8, Count: 3},
				"reddit":     {Score: 0.3, Confidence: 0.7, Count: 2},
				"fear_greed": {Score: 0.1, Confidence: 0.6, Count: 1},
			},
		},
	}
	signals := &signalStoreStub{}
	svc := NewService(
		trace.NewNoopTracerProvider().Tracer("test"),
		store,
		NewScorer(nil, 8),
		signals,
		nil,
		nil,
		nil,
		nil,
		Config{
			Intervals:      []string{"1h"},
			LongThreshold:  0.20,
			ShortThreshold: -0.20,
		},
	)

	res, err := svc.RunCycle(context.Background(), now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.CompositesWritten != len(domain.SupportedSymbols) {
		t.Fatalf("expected one composite per symbol, got %d", res.CompositesWritten)
	}
	if res.SignalsWritten != 1 {
		t.Fatalf("expected one directional signal for BTC, got %d", res.SignalsWritten)
	}
	if len(signals.inserted) != 1 {
		t.Fatalf("expected one inserted signal row, got %d", len(signals.inserted))
	}
	if signals.inserted[0].Indicator != domain.IndicatorFundSentimentComposite {
		t.Fatalf("unexpected indicator %s", signals.inserted[0].Indicator)
	}
}

func TestServiceRunCycleDoesNotFailOnOnChainErrors(t *testing.T) {
	now := time.Date(2026, 2, 13, 19, 30, 0, 0, time.UTC)
	store := &marketStoreStub{}
	svc := NewService(
		trace.NewNoopTracerProvider().Tracer("test"),
		store,
		NewScorer(nil, 8),
		nil,
		nil,
		nil,
		nil,
		map[string]OnChainReader{"BTC": onchainReaderStub{err: context.DeadlineExceeded}},
		Config{Intervals: []string{"1h"}, EnableOnChain: true, OnChainSymbols: []string{"BTC"}},
	)

	res, err := svc.RunCycle(context.Background(), now)
	if err != nil {
		t.Fatalf("unexpected fatal error: %v", err)
	}
	if len(res.Errors) == 0 {
		t.Fatalf("expected non-fatal warning errors")
	}
}

type marketStoreStub struct {
	itemSeq          int64
	composites       []domain.MarketCompositeSnapshot
	averagesBySymbol map[string]map[string]SourceSentimentStats
}

func (s *marketStoreStub) UpsertItems(ctx context.Context, items []domain.MarketIntelItem) ([]domain.MarketIntelItem, error) {
	out := make([]domain.MarketIntelItem, len(items))
	for i := range items {
		s.itemSeq++
		out[i] = items[i]
		out[i].ID = s.itemSeq
	}
	return out, nil
}

func (s *marketStoreStub) UpsertItemSymbols(ctx context.Context, itemID int64, symbols []string) error {
	return nil
}

func (s *marketStoreStub) ListUnscoredItems(ctx context.Context, limit int) ([]domain.MarketIntelItem, error) {
	return nil, nil
}

func (s *marketStoreStub) UpdateItemSentiment(ctx context.Context, itemID int64, score float64, confidence float64, label string, model string, reason string, scoredAt time.Time) error {
	return nil
}

func (s *marketStoreStub) GetSentimentAverages(ctx context.Context, symbol string, from, to time.Time) (map[string]SourceSentimentStats, error) {
	if s.averagesBySymbol == nil {
		return map[string]SourceSentimentStats{}, nil
	}
	if stats, ok := s.averagesBySymbol[symbol]; ok {
		return stats, nil
	}
	return map[string]SourceSentimentStats{}, nil
}

func (s *marketStoreStub) UpsertOnChainSnapshot(ctx context.Context, snapshot domain.MarketOnChainSnapshot) (*domain.MarketOnChainSnapshot, error) {
	copy := snapshot
	return &copy, nil
}

func (s *marketStoreStub) UpsertCompositeSnapshot(ctx context.Context, snapshot domain.MarketCompositeSnapshot) (*domain.MarketCompositeSnapshot, error) {
	copy := snapshot
	s.composites = append(s.composites, copy)
	return &copy, nil
}

func (s *marketStoreStub) AttachCompositeSignalID(ctx context.Context, symbol, interval string, openTime time.Time, signalID int64) error {
	return nil
}

func (s *marketStoreStub) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	return 0, nil
}

type signalStoreStub struct {
	inserted []domain.Signal
}

func (s *signalStoreStub) InsertSignals(ctx context.Context, signals []domain.Signal) ([]domain.Signal, error) {
	out := append([]domain.Signal(nil), signals...)
	for i := range out {
		out[i].ID = int64(len(s.inserted) + i + 1)
	}
	s.inserted = append(s.inserted, out...)
	return out, nil
}

type onchainReaderStub struct {
	err error
}

func (s onchainReaderStub) FetchSnapshot(ctx context.Context, interval string, bucketTime time.Time) (*provider.OnChainSnapshot, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &provider.OnChainSnapshot{ProviderKey: "stub", Symbol: "BTC", Interval: interval, BucketTime: bucketTime, Score: 0.1, Confidence: 0.5}, nil
}

var _ Store = (*marketStoreStub)(nil)
var _ SignalStore = (*signalStoreStub)(nil)
var _ OnChainReader = (onchainReaderStub{})
var _ = pgx.ErrNoRows

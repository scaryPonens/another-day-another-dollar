package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"bug-free-umbrella/internal/domain"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/trace"
)

var testTracer = trace.NewNoopTracerProvider().Tracer("test")

func TestPriceService_GetCurrentPriceCacheHit(t *testing.T) {
	t.Parallel()

	redis := newFakeRedis()
	snap := &domain.PriceSnapshot{Symbol: "BTC", PriceUSD: 123.45}
	data, _ := json.Marshal(snap)
	_ = redis.Set(context.Background(), "price:BTC", data, 0)

	svc := NewPriceService(testTracer, &mockProvider{}, &mockCandleRepo{}, redis)

	got, err := svc.GetCurrentPrice(context.Background(), "BTC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.PriceUSD != snap.PriceUSD {
		t.Fatalf("expected %.2f, got %.2f", snap.PriceUSD, got.PriceUSD)
	}
}

func TestPriceService_GetCurrentPriceFetchesOnMiss(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		prices: map[string]*domain.PriceSnapshot{
			"BTC": {Symbol: "BTC", PriceUSD: 42},
		},
	}
	redis := newFakeRedis()
	svc := NewPriceService(testTracer, provider, &mockCandleRepo{}, redis)

	got, err := svc.GetCurrentPrice(context.Background(), "BTC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Symbol != "BTC" || got.PriceUSD != 42 {
		t.Fatalf("unexpected snapshot: %+v", got)
	}
	if provider.fetchPricesCalls != 1 {
		t.Fatalf("expected FetchPrices to be called once, got %d", provider.fetchPricesCalls)
	}
	if _, ok := redis.data["price:BTC"]; !ok {
		t.Fatalf("price not cached")
	}
}

func TestPriceService_GetCurrentPriceUnsupported(t *testing.T) {
	t.Parallel()

	svc := NewPriceService(testTracer, &mockProvider{}, &mockCandleRepo{}, nil)
	if _, err := svc.GetCurrentPrice(context.Background(), "FAKE"); err == nil {
		t.Fatal("expected error for unsupported symbol")
	}
}

func TestPriceService_GetCurrentPricesUsesCache(t *testing.T) {
	t.Parallel()

	redis := newFakeRedis()
	cached := &domain.PriceSnapshot{Symbol: "BTC", PriceUSD: 1}
	data, _ := json.Marshal(cached)
	_ = redis.Set(context.Background(), "price:BTC", data, 0)

	prices := make(map[string]*domain.PriceSnapshot)
	for _, symbol := range domain.SupportedSymbols {
		if symbol == "BTC" {
			continue
		}
		prices[symbol] = &domain.PriceSnapshot{Symbol: symbol, PriceUSD: float64(len(symbol))}
	}

	provider := &mockProvider{prices: prices}
	svc := NewPriceService(testTracer, provider, &mockCandleRepo{}, redis)

	snapshots, err := svc.GetCurrentPrices(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.fetchPricesCalls != 1 {
		t.Fatalf("expected fetch once, got %d", provider.fetchPricesCalls)
	}
	if len(snapshots) != len(domain.SupportedSymbols) {
		t.Fatalf("expected %d snapshots, got %d", len(domain.SupportedSymbols), len(snapshots))
	}
}

func TestPriceService_RefreshPricesCachesAll(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		prices: map[string]*domain.PriceSnapshot{
			"BTC": {Symbol: "BTC", PriceUSD: 10},
			"ETH": {Symbol: "ETH", PriceUSD: 20},
		},
	}
	redis := newFakeRedis()
	svc := NewPriceService(testTracer, provider, &mockCandleRepo{}, redis)

	if err := svc.RefreshPrices(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.fetchPricesCalls != 1 {
		t.Fatalf("expected fetch once, got %d", provider.fetchPricesCalls)
	}
	if len(redis.data) != 2 {
		t.Fatalf("expected cached entries, got %d", len(redis.data))
	}
}

func TestPriceService_RefreshShortCandles(t *testing.T) {
	t.Parallel()

	candles := []*domain.Candle{{Symbol: "BTC", Interval: "5m"}}
	provider := &mockProvider{marketCandles: candles}
	repo := &mockCandleRepo{}
	svc := NewPriceService(testTracer, provider, repo, nil)

	if err := svc.RefreshShortCandles(context.Background(), "BTC"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.lastMarketSymbol != "BTC" || provider.lastMarketDays != 1 {
		t.Fatalf("unexpected market chart args: %+v", provider)
	}
	if repo.upsertCalls != 1 || len(repo.upsertArg) != 1 {
		t.Fatalf("expected 1 upsert call, got %d", repo.upsertCalls)
	}
}

func TestPriceService_RefreshLongCandles(t *testing.T) {
	t.Parallel()

	candles := []*domain.Candle{{Symbol: "BTC", Interval: "1d"}}
	provider := &mockProvider{marketCandles: candles}
	repo := &mockCandleRepo{}
	svc := NewPriceService(testTracer, provider, repo, nil)

	if err := svc.RefreshLongCandles(context.Background(), "BTC"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.lastMarketDays != 30 {
		t.Fatalf("expected days=30, got %d", provider.lastMarketDays)
	}
	if repo.upsertCalls != 1 || repo.upsertArg[0].Interval != "1d" {
		t.Fatalf("unexpected upsert payload: %+v", repo.upsertArg)
	}
}

func TestPriceService_GetCandles(t *testing.T) {
	t.Parallel()

	repo := &mockCandleRepo{
		getResp: []*domain.Candle{{Symbol: "BTC", Interval: "1h"}},
	}
	svc := NewPriceService(testTracer, &mockProvider{}, repo, nil)

	candles, err := svc.GetCandles(context.Background(), "BTC", "1h", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastGetSymbol != "BTC" || repo.lastGetInterval != "1h" || repo.lastGetLimit != 5 {
		t.Fatalf("unexpected repo args: %s %s %d", repo.lastGetSymbol, repo.lastGetInterval, repo.lastGetLimit)
	}
	if len(candles) != 1 {
		t.Fatalf("expected 1 candle, got %d", len(candles))
	}
}

type mockProvider struct {
	prices        map[string]*domain.PriceSnapshot
	marketCandles []*domain.Candle
	priceErr      error
	marketErr     error

	fetchPricesCalls    int
	marketCalls         int
	lastMarketSymbol    string
	lastMarketDays      int
	lastMarketIntervals []string
}

func (m *mockProvider) FetchPrices(ctx context.Context) (map[string]*domain.PriceSnapshot, error) {
	m.fetchPricesCalls++
	if m.priceErr != nil {
		return nil, m.priceErr
	}
	return m.prices, nil
}

func (m *mockProvider) FetchMarketChart(ctx context.Context, symbol string, days int, intervals []string) ([]*domain.Candle, error) {
	m.marketCalls++
	m.lastMarketSymbol = symbol
	m.lastMarketDays = days
	m.lastMarketIntervals = append([]string(nil), intervals...)
	if m.marketErr != nil {
		return nil, m.marketErr
	}
	return m.marketCandles, nil
}

type mockCandleRepo struct {
	getResp []*domain.Candle
	getErr  error

	lastGetSymbol   string
	lastGetInterval string
	lastGetLimit    int

	upsertArg   []*domain.Candle
	upsertErr   error
	upsertCalls int
}

func (m *mockCandleRepo) GetCandles(ctx context.Context, symbol, interval string, limit int) ([]*domain.Candle, error) {
	m.lastGetSymbol = symbol
	m.lastGetInterval = interval
	m.lastGetLimit = limit
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.getResp, nil
}

func (m *mockCandleRepo) UpsertCandles(ctx context.Context, candles []*domain.Candle) error {
	m.upsertCalls++
	m.upsertArg = candles
	if m.upsertErr != nil {
		return m.upsertErr
	}
	return nil
}

type fakeRedis struct {
	data   map[string][]byte
	setErr error
	getErr error
}

func newFakeRedis() *fakeRedis {
	return &fakeRedis{data: make(map[string][]byte)}
}

func (f *fakeRedis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	if f.setErr != nil {
		return redis.NewStatusResult("", f.setErr)
	}
	switch v := value.(type) {
	case []byte:
		f.data[key] = append([]byte(nil), v...)
	case string:
		f.data[key] = []byte(v)
	default:
		bytes, _ := json.Marshal(v)
		f.data[key] = bytes
	}
	return redis.NewStatusResult("OK", nil)
}

func (f *fakeRedis) Get(ctx context.Context, key string) *redis.StringCmd {
	if f.getErr != nil {
		return redis.NewStringResult("", f.getErr)
	}
	if v, ok := f.data[key]; ok {
		return redis.NewStringResult(string(v), nil)
	}
	return redis.NewStringResult("", redis.Nil)
}

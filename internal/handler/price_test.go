package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bug-free-umbrella/internal/domain"
	"bug-free-umbrella/internal/service"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestGetPriceSuccess(t *testing.T) {
	handler := newTestHandler(map[string]*domain.PriceSnapshot{
		"BTC": {Symbol: "BTC", PriceUSD: 99.5},
	}, nil, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/prices/BTC", nil)

	router := gin.New()
	router.GET("/api/prices/:symbol", handler.GetPrice)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var snapshot domain.PriceSnapshot
	if err := json.Unmarshal(w.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if snapshot.Symbol != "BTC" {
		t.Fatalf("expected BTC snapshot, got %s", snapshot.Symbol)
	}
}

func TestGetPriceInvalidSymbol(t *testing.T) {
	handler := newTestHandler(map[string]*domain.PriceSnapshot{}, nil, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/prices/invalid", nil)

	router := gin.New()
	router.GET("/api/prices/:symbol", handler.GetPrice)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetPriceProviderError(t *testing.T) {
	handler := newTestHandler(map[string]*domain.PriceSnapshot{}, errFetch, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/prices/BTC", nil)

	router := gin.New()
	router.GET("/api/prices/:symbol", handler.GetPrice)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestGetAllPrices(t *testing.T) {
	prices := make(map[string]*domain.PriceSnapshot)
	for _, symbol := range domain.SupportedSymbols {
		prices[symbol] = &domain.PriceSnapshot{Symbol: symbol, PriceUSD: float64(len(symbol))}
	}
	handler := newTestHandler(prices, nil, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/prices", nil)

	router := gin.New()
	router.GET("/api/prices", handler.GetAllPrices)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Prices []domain.PriceSnapshot `json:"prices"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(resp.Prices) != len(domain.SupportedSymbols) {
		t.Fatalf("expected %d prices, got %d", len(domain.SupportedSymbols), len(resp.Prices))
	}
}

func TestGetCandlesInvalidInterval(t *testing.T) {
	handler := newTestHandler(nil, nil, &stubRepo{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/candles/BTC?interval=2h", nil)

	router := gin.New()
	router.GET("/api/candles/:symbol", handler.GetCandles)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetCandlesSuccess(t *testing.T) {
	candles := []*domain.Candle{{
		Symbol:   "ETH",
		Interval: "1h",
		OpenTime: time.Unix(0, 0).UTC(),
		Open:     10,
		High:     12,
		Low:      9,
		Close:    11,
		Volume:   1000,
	}}
	repo := &stubRepo{candles: candles}
	handler := newTestHandler(nil, nil, repo)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/candles/ETH?interval=1h&limit=1", nil)

	router := gin.New()
	router.GET("/api/candles/:symbol", handler.GetCandles)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Symbol   string          `json:"symbol"`
		Interval string          `json:"interval"`
		Candles  []domain.Candle `json:"candles"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if resp.Symbol != "ETH" || resp.Interval != "1h" || len(resp.Candles) != 1 {
		t.Fatalf("unexpected payload: %+v", resp)
	}
	if repo.lastLimit != 1 {
		t.Fatalf("expected limit=1, got %d", repo.lastLimit)
	}
}

type stubPriceProvider struct {
	prices   map[string]*domain.PriceSnapshot
	fetchErr error
}

func (s *stubPriceProvider) FetchPrices(ctx context.Context) (map[string]*domain.PriceSnapshot, error) {
	if s.fetchErr != nil {
		return nil, s.fetchErr
	}
	return s.prices, nil
}

func (s *stubPriceProvider) FetchMarketChart(ctx context.Context, symbol string, days int, intervals []string) ([]*domain.Candle, error) {
	return nil, nil
}

type stubRepo struct {
	candles []*domain.Candle

	lastSymbol   string
	lastInterval string
	lastLimit    int
}

func (s *stubRepo) GetCandles(ctx context.Context, symbol, interval string, limit int) ([]*domain.Candle, error) {
	s.lastSymbol = symbol
	s.lastInterval = interval
	s.lastLimit = limit
	return s.candles, nil
}

func (s *stubRepo) UpsertCandles(ctx context.Context, candles []*domain.Candle) error {
	s.candles = candles
	return nil
}

var errFetch = errors.New("fetch error")

func newTestHandler(prices map[string]*domain.PriceSnapshot, fetchErr error, repo service.CandleRepository) *Handler {
	tracer := trace.NewNoopTracerProvider().Tracer("handler-test")
	provider := &stubPriceProvider{prices: prices, fetchErr: fetchErr}
	if repo == nil {
		repo = &stubRepo{}
	}
	priceService := service.NewPriceService(tracer, provider, repo, nil)
	return &Handler{
		tracer:       tracer,
		workService:  service.NewWorkService(tracer),
		priceService: priceService,
	}
}

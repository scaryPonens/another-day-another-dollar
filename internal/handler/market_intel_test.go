package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"bug-free-umbrella/internal/domain"
	"bug-free-umbrella/internal/service"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

func TestTriggerMarketIntelRunServiceUnavailable(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("handler-test")
	h := &Handler{tracer: tracer, workService: service.NewWorkService(tracer)}

	router := gin.New()
	router.POST("/api/market-intel/run", h.TriggerMarketIntelRun)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/market-intel/run", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestTriggerMarketIntelRunSuccess(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("handler-test")
	h := &Handler{tracer: tracer, workService: service.NewWorkService(tracer)}
	h.SetMarketIntelRunner(marketIntelRunnerStub{result: domain.MarketIntelRunResult{
		ItemsIngested:     11,
		ItemsScored:       8,
		OnChainSnapshots:  4,
		CompositesWritten: 20,
		SignalsWritten:    6,
		Errors:            []string{"one warning"},
	}})

	router := gin.New()
	router.POST("/api/market-intel/run", h.TriggerMarketIntelRun)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/market-intel/run", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body struct {
		Status            string   `json:"status"`
		ItemsIngested     int      `json:"items_ingested"`
		ItemsScored       int      `json:"items_scored"`
		OnChainSnapshots  int      `json:"onchain_snapshots"`
		CompositesWritten int      `json:"composites_written"`
		SignalsWritten    int      `json:"signals_written"`
		Errors            []string `json:"errors"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if body.Status != "ok" || body.ItemsIngested != 11 || body.SignalsWritten != 6 {
		t.Fatalf("unexpected response payload: %+v", body)
	}
}

func TestTriggerMarketIntelRunFailure(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("handler-test")
	h := &Handler{tracer: tracer, workService: service.NewWorkService(tracer)}
	h.SetMarketIntelRunner(marketIntelRunnerStub{err: errors.New("run failed")})

	router := gin.New()
	router.POST("/api/market-intel/run", h.TriggerMarketIntelRun)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/market-intel/run", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

type marketIntelRunnerStub struct {
	result domain.MarketIntelRunResult
	err    error
}

func (s marketIntelRunnerStub) RunMarketIntel(ctx context.Context) (domain.MarketIntelRunResult, error) {
	if s.err != nil {
		return domain.MarketIntelRunResult{}, s.err
	}
	return s.result, nil
}

package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"bug-free-umbrella/internal/ml/training"
	"bug-free-umbrella/internal/service"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

func TestTriggerMLTrainingServiceUnavailable(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("handler-test")
	h := &Handler{tracer: tracer, workService: service.NewWorkService(tracer)}

	router := gin.New()
	router.POST("/api/ml/train", h.TriggerMLTraining)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/ml/train", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestTriggerMLTrainingSuccess(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("handler-test")
	h := &Handler{tracer: tracer, workService: service.NewWorkService(tracer)}
	h.SetMLTrainingRunner(mlTrainingRunnerStub{results: []training.ModelTrainResult{{ModelKey: "logreg", Version: 2, AUC: 0.72, Promoted: true}}})

	router := gin.New()
	router.POST("/api/ml/train", h.TriggerMLTraining)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/ml/train", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body struct {
		Status  string                      `json:"status"`
		Trained int                         `json:"trained"`
		Results []training.ModelTrainResult `json:"results"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if body.Status != "ok" || body.Trained != 1 || len(body.Results) != 1 {
		t.Fatalf("unexpected response payload: %+v", body)
	}
}

func TestTriggerMLTrainingFailure(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("handler-test")
	h := &Handler{tracer: tracer, workService: service.NewWorkService(tracer)}
	h.SetMLTrainingRunner(mlTrainingRunnerStub{err: errors.New("train failed")})

	router := gin.New()
	router.POST("/api/ml/train", h.TriggerMLTraining)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/ml/train", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

type mlTrainingRunnerStub struct {
	results []training.ModelTrainResult
	err     error
}

func (s mlTrainingRunnerStub) RunTraining(ctx context.Context) ([]training.ModelTrainResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return append([]training.ModelTrainResult(nil), s.results...), nil
}

package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bug-free-umbrella/internal/service"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

func TestHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	tracer := trace.NewNoopTracerProvider().Tracer("test")
	h := &Handler{
		tracer:      tracer,
		workService: service.NewWorkService(tracer),
	}
	r.GET("/health", h.Health)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	body := w.Body.String()
	if body != "{\"status\":\"healthy\"}\n" && body != "{\"status\":\"healthy\"}" {
		t.Errorf("unexpected body: %s", body)
	}
}

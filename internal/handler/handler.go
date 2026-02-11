package handler

import (
	"bug-free-umbrella/internal/service"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

type Handler struct {
	tracer      trace.Tracer
	workService *service.WorkService
}

func New(tracer trace.Tracer, workService *service.WorkService) *Handler {
	return &Handler{
		tracer:      tracer,
		workService: workService,
	}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/health", h.Health)
	r.GET("/api/hello", h.Hello)
	r.GET("/api/slow", h.Slow)
}

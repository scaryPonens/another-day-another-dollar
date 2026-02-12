package handler

import (
	"bug-free-umbrella/internal/service"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

type Handler struct {
	tracer       trace.Tracer
	workService  *service.WorkService
	priceService *service.PriceService
}

func New(tracer trace.Tracer, workService *service.WorkService, priceService *service.PriceService) *Handler {
	return &Handler{
		tracer:       tracer,
		workService:  workService,
		priceService: priceService,
	}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/health", h.Health)
	r.GET("/api/prices", h.GetAllPrices)
	r.GET("/api/prices/:symbol", h.GetPrice)
	r.GET("/api/candles/:symbol", h.GetCandles)
}

package handler

import (
	"bug-free-umbrella/internal/service"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

type Handler struct {
	tracer            trace.Tracer
	workService       *service.WorkService
	priceService      *service.PriceService
	signalService     *service.SignalService
	mlTrainer         MLTrainingRunner
	marketIntelRunner MarketIntelRunner
}

func New(
	tracer trace.Tracer,
	workService *service.WorkService,
	priceService *service.PriceService,
	signalService *service.SignalService,
) *Handler {
	return &Handler{
		tracer:        tracer,
		workService:   workService,
		priceService:  priceService,
		signalService: signalService,
	}
}

func (h *Handler) SetMLTrainingRunner(runner MLTrainingRunner) {
	h.mlTrainer = runner
}

func (h *Handler) SetMarketIntelRunner(runner MarketIntelRunner) {
	h.marketIntelRunner = runner
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/health", h.Health)
	r.GET("/api/prices", h.GetAllPrices)
	r.GET("/api/prices/:symbol", h.GetPrice)
	r.GET("/api/candles/:symbol", h.GetCandles)
	r.GET("/api/signals", h.GetSignals)
	r.GET("/api/signals/:id/image", h.GetSignalImage)
	r.POST("/api/ml/train", h.TriggerMLTraining)
	r.POST("/api/market-intel/run", h.TriggerMarketIntelRun)
}

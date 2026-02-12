package handler

import (
	"net/http"
	"strconv"
	"strings"

	"bug-free-umbrella/internal/domain"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
)

// GetSignals godoc
// @Summary      Get generated trading signals
// @Description  Returns recent signals, optionally filtered by symbol/risk/indicator
// @Tags         signals
// @Produce      json
// @Param        symbol     query  string  false  "Asset symbol (e.g., BTC, ETH)"
// @Param        risk       query  int     false  "Risk level (1-5)"
// @Param        indicator  query  string  false  "Indicator key (rsi, macd, bollinger, volume_zscore, ml_logreg_up4h, ml_xgboost_up4h, ml_ensemble_up4h)"
// @Param        limit      query  int     false  "Number of signals (default 50, max 200)"  default(50)
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Failure      503  {object}  map[string]string
// @Router       /api/signals [get]
func (h *Handler) GetSignals(c *gin.Context) {
	if h.signalService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "signal service unavailable"})
		return
	}

	ctx, span := h.tracer.Start(c.Request.Context(), "handler.get-signals")
	defer span.End()

	filter := domain.SignalFilter{
		Symbol:    strings.ToUpper(strings.TrimSpace(c.Query("symbol"))),
		Indicator: strings.ToLower(strings.TrimSpace(c.Query("indicator"))),
	}

	if filter.Symbol != "" {
		span.SetAttributes(attribute.String("symbol", filter.Symbol))
		if _, ok := domain.CoinGeckoID[filter.Symbol]; !ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":             "unsupported symbol: " + filter.Symbol,
				"supported_symbols": domain.SupportedSymbols,
			})
			return
		}
	}

	if rawRisk := strings.TrimSpace(c.Query("risk")); rawRisk != "" {
		r, err := strconv.Atoi(rawRisk)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "risk must be an integer between 1 and 5"})
			return
		}
		risk := domain.RiskLevel(r)
		if !risk.IsValid() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "risk must be between 1 and 5"})
			return
		}
		filter.Risk = &risk
	}

	limit := 50
	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		n, err := strconv.Atoi(rawLimit)
		if err != nil || n <= 0 || n > 200 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be between 1 and 200"})
			return
		}
		limit = n
	}
	filter.Limit = limit

	signals, err := h.signalService.ListSignals(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"signals": signals})
}

// GetSignalImage godoc
// @Summary      Get signal chart image
// @Description  Returns the rendered PNG chart image for a signal id
// @Tags         signals
// @Produce      png
// @Param        id  path  int  true  "Signal ID"
// @Success      200  {file}  binary
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      503  {object}  map[string]string
// @Router       /api/signals/{id}/image [get]
func (h *Handler) GetSignalImage(c *gin.Context) {
	if h.signalService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "signal service unavailable"})
		return
	}

	ctx, span := h.tracer.Start(c.Request.Context(), "handler.get-signal-image")
	defer span.End()

	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id must be a positive integer"})
		return
	}

	imageData, err := h.signalService.GetSignalImage(ctx, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if imageData == nil || len(imageData.Bytes) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "signal image not found"})
		return
	}

	c.Data(http.StatusOK, imageData.Ref.MimeType, imageData.Bytes)
}

package handler

import (
	"net/http"
	"strconv"
	"strings"

	"bug-free-umbrella/internal/domain"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
)

// GetPrice godoc
// @Summary      Get current price for a crypto asset
// @Description  Returns the latest cached price, 24h volume, and 24h change
// @Tags         prices
// @Produce      json
// @Param        symbol  path  string  true  "Asset symbol (e.g., BTC, ETH)"
// @Success      200  {object}  domain.PriceSnapshot
// @Failure      400  {object}  map[string]string
// @Router       /api/prices/{symbol} [get]
func (h *Handler) GetPrice(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "handler.get-price")
	defer span.End()

	symbol := strings.ToUpper(c.Param("symbol"))
	span.SetAttributes(attribute.String("symbol", symbol))

	if _, ok := domain.CoinGeckoID[symbol]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "unsupported symbol: " + symbol,
			"supported_symbols": domain.SupportedSymbols,
		})
		return
	}

	snapshot, err := h.priceService.GetCurrentPrice(ctx, symbol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, snapshot)
}

// GetAllPrices godoc
// @Summary      Get current prices for all supported assets
// @Description  Returns latest cached prices for all 10 tracked cryptocurrencies
// @Tags         prices
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/prices [get]
func (h *Handler) GetAllPrices(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "handler.get-all-prices")
	defer span.End()

	snapshots, err := h.priceService.GetCurrentPrices(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"prices": snapshots})
}

// GetCandles godoc
// @Summary      Get historical OHLCV candles
// @Description  Returns historical candle data for a given asset and interval
// @Tags         prices
// @Produce      json
// @Param        symbol    path   string  true   "Asset symbol (e.g., BTC, ETH)"
// @Param        interval  query  string  false  "Candle interval (5m, 15m, 1h, 4h, 1d)"  default(1h)
// @Param        limit     query  int     false  "Number of candles (default 100, max 500)"  default(100)
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Router       /api/candles/{symbol} [get]
func (h *Handler) GetCandles(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "handler.get-candles")
	defer span.End()

	symbol := strings.ToUpper(c.Param("symbol"))
	span.SetAttributes(attribute.String("symbol", symbol))

	if _, ok := domain.CoinGeckoID[symbol]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "unsupported symbol: " + symbol,
			"supported_symbols": domain.SupportedSymbols,
		})
		return
	}

	interval := c.DefaultQuery("interval", "1h")
	validInterval := false
	for _, si := range domain.SupportedIntervals {
		if interval == si {
			validInterval = true
			break
		}
	}
	if !validInterval {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":               "unsupported interval: " + interval,
			"supported_intervals": domain.SupportedIntervals,
		})
		return
	}

	limit := 100
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}

	candles, err := h.priceService.GetCandles(ctx, symbol, interval, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":   symbol,
		"interval": interval,
		"candles":  candles,
	})
}

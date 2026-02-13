package handler

import (
	"context"
	"net/http"

	"bug-free-umbrella/internal/domain"

	"github.com/gin-gonic/gin"
)

type MarketIntelRunner interface {
	RunMarketIntel(ctx context.Context) (domain.MarketIntelRunResult, error)
}

// TriggerMarketIntelRun godoc
// @Summary      Trigger fundamentals and sentiment ingestion/scoring manually
// @Description  Runs one Phase 7 market-intel cycle and returns ingest/score/composite counters
// @Tags         market-intel
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/market-intel/run [post]
func (h *Handler) TriggerMarketIntelRun(c *gin.Context) {
	if h.marketIntelRunner == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "market intel service unavailable"})
		return
	}

	ctx, span := h.tracer.Start(c.Request.Context(), "handler.trigger-market-intel-run")
	defer span.End()

	result, err := h.marketIntelRunner.RunMarketIntel(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":             "ok",
		"items_ingested":     result.ItemsIngested,
		"items_scored":       result.ItemsScored,
		"onchain_snapshots":  result.OnChainSnapshots,
		"composites_written": result.CompositesWritten,
		"signals_written":    result.SignalsWritten,
		"errors":             result.Errors,
	})
}

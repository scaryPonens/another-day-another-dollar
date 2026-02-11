package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Slow godoc
// @Summary      Slow endpoint
// @Description  Returns a response after a simulated delay
// @Tags         api
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /api/slow [get]
func (h *Handler) Slow(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "slow-handler")
	defer span.End()

	h.workService.DoWork(ctx)

	c.JSON(http.StatusOK, gin.H{"message": "Slow response completed"})
}

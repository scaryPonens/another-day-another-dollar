package handler

import (
	"context"
	"net/http"

	"bug-free-umbrella/internal/ml/training"

	"github.com/gin-gonic/gin"
)

type MLTrainingRunner interface {
	RunTraining(ctx context.Context) ([]training.ModelTrainResult, error)
}

// TriggerMLTraining godoc
// @Summary      Trigger ML model training manually
// @Description  Runs an immediate ML training cycle and returns model training outcomes
// @Tags         ml
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     ApiKeyAuth
// @Router       /api/ml/train [post]
func (h *Handler) TriggerMLTraining(c *gin.Context) {
	if h.mlTrainer == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "ml training service unavailable"})
		return
	}

	ctx, span := h.tracer.Start(c.Request.Context(), "handler.trigger-ml-training")
	defer span.End()

	results, err := h.mlTrainer.RunTraining(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"trained": len(results),
		"results": results,
	})
}

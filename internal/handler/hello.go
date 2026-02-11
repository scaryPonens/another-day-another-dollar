package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
)

// Hello godoc
// @Summary      Say hello
// @Description  Returns a hello world greeting
// @Tags         api
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /api/hello [get]
func (h *Handler) Hello(c *gin.Context) {
	_, span := h.tracer.Start(c.Request.Context(), "hello-handler")
	defer span.End()

	span.SetAttributes(attribute.String("custom.attribute", "hello-value"))

	c.JSON(http.StatusOK, gin.H{"message": "Hello, World!"})
}

package controller

import (
	"net/http"
	"os"

	"sales/src/sales/infrastructure/metrics"

	"github.com/gin-gonic/gin"
)

// MetricsController H8: endpoint para eventos de métricas desde frontend
// Protección: METRICS_EVENT_SECRET (opcional) — si está definido, requiere X-Metrics-Key
type MetricsController struct {
	secret string
}

// NewMetricsController crea una nueva instancia
func NewMetricsController() *MetricsController {
	return &MetricsController{
		secret: os.Getenv("METRICS_EVENT_SECRET"),
	}
}

// RegisterRoutes registra rutas bajo /sales/internal/metrics
func (c *MetricsController) RegisterRoutes(router *gin.RouterGroup) {
	sales := router.Group("/sales")
	internal := sales.Group("/internal")
	metricsGroup := internal.Group("/metrics")
	metricsGroup.POST("/event", c.requireInternalAuth(), c.RecordEvent)
}

// requireInternalAuth middleware: si METRICS_EVENT_SECRET está definido, exige X-Metrics-Key
func (c *MetricsController) requireInternalAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if c.secret == "" {
			ctx.Next()
			return
		}
		if ctx.GetHeader("X-Metrics-Key") != c.secret {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		ctx.Next()
	}
}

// EventRequest payload para POST /event
type EventRequest struct {
	Event string `json:"event" binding:"required"`
}

// RecordEvent POST /sales/internal/metrics/event
// Eventos: quickstart_started, quickstart_completed
func (c *MetricsController) RecordEvent(ctx *gin.Context) {
	var req EventRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "event is required"})
		return
	}
	switch req.Event {
	case "quickstart_started":
		metrics.QuickstartStartedTotal.Inc()
	case "quickstart_completed":
		metrics.QuickstartCompletedTotal.Inc()
	default:
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "unknown event: " + req.Event})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"ok": true})
}

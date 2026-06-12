package controller

import (
	"net/http"
	"os"

	"sales/src/sales/infrastructure/metrics"

	"github.com/gin-gonic/gin"
	"github.com/hornosg/go-shared/infrastructure/response"
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
			response.Abort(ctx, http.StatusForbidden, "forbidden")
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
		response.JSON(ctx, http.StatusBadRequest, "event is required")
		return
	}
	switch req.Event {
	case "quickstart_started":
		metrics.QuickstartStartedTotal.Inc()
	case "quickstart_completed":
		metrics.QuickstartCompletedTotal.Inc()
	default:
		response.JSON(ctx, http.StatusBadRequest, "unknown event: "+req.Event)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"ok": true})
}

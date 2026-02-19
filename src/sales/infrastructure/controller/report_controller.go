package controller

import (
	"log"
	"net/http"

	"sales/src/sales/application/usecase"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ReportController maneja las peticiones HTTP para reportes
// HITO C - Reportes Diarios
type ReportController struct {
	dailyReportUC *usecase.DailyReportUseCase
}

// NewReportController crea una nueva instancia del controlador
func NewReportController(dailyReportUC *usecase.DailyReportUseCase) *ReportController {
	return &ReportController{
		dailyReportUC: dailyReportUC,
	}
}

// RegisterRoutes registra las rutas del controlador
func (c *ReportController) RegisterRoutes(router *gin.RouterGroup) {
	reports := router.Group("/reports")
	{
		reports.GET("/daily", c.DailyReport)
	}

	log.Println("Rutas Report disponibles:")
	log.Println("  GET    /api/v1/reports/daily?date=YYYY-MM-DD")
}

// DailyReport maneja el reporte diario de ventas
func (c *ReportController) DailyReport(ctx *gin.Context) {
	// ========================================================================
	// PASO 1: Validar header X-Tenant-ID (OBLIGATORIO)
	// ========================================================================
	tenantID := ctx.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "X-Tenant-ID header is required",
		})
		return
	}

	// ========================================================================
	// PASO 2: Parsear tenant_id a UUID
	// ========================================================================
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid X-Tenant-ID format",
		})
		return
	}

	// ========================================================================
	// PASO 3: Leer query parameter 'date' (OBLIGATORIO)
	// ========================================================================
	date := ctx.Query("date")
	if date == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "date query parameter is required (format: YYYY-MM-DD)",
		})
		return
	}

	// ========================================================================
	// PASO 4: Ejecutar use case
	// ========================================================================
	resp, err := c.dailyReportUC.Execute(ctx.Request.Context(), tenantUUID, date)
	if err != nil {
		log.Printf("Error generating daily report: %v", err)

		// Si es error de formato de fecha → 400
		if contains(err.Error(), "invalid date format") {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid date format",
				"details": err.Error(),
			})
			return
		}

		// Otros errores → 500
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error generating daily report",
			"details": err.Error(),
		})
		return
	}

	// ========================================================================
	// PASO 5: Responder exitosamente
	// ========================================================================
	ctx.JSON(http.StatusOK, resp)
}

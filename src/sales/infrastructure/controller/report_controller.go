package controller

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"sales/src/sales/application/usecase"
	"sales/src/sales/application/usecase/reports"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hornosg/go-shared/infrastructure/response"
)

// ReportController maneja las peticiones HTTP para reportes
// HITO C - Reportes Diarios + Open Orders + Aging + Customer Balance
type ReportController struct {
	dailyReportUC      *usecase.DailyReportUseCase
	openOrdersReportUC *reports.GetOpenOrdersReportUseCase
	agingReportUC      *reports.GetAgingReportUseCase
	customerBalanceUC  *reports.GetCustomerBalanceUseCase
}

// NewReportController crea una nueva instancia del controlador
func NewReportController(
	dailyReportUC *usecase.DailyReportUseCase,
	openOrdersReportUC *reports.GetOpenOrdersReportUseCase,
	agingReportUC *reports.GetAgingReportUseCase,
	customerBalanceUC *reports.GetCustomerBalanceUseCase,
) *ReportController {
	return &ReportController{
		dailyReportUC:      dailyReportUC,
		openOrdersReportUC: openOrdersReportUC,
		agingReportUC:      agingReportUC,
		customerBalanceUC:  customerBalanceUC,
	}
}

// RegisterRoutes registra las rutas del controlador
func (c *ReportController) RegisterRoutes(router *gin.RouterGroup) {
	log.Println("🔍 DEBUG: RegisterRoutes llamado")
	log.Printf("🔍 DEBUG: openOrdersReportUC is nil? %v", c.openOrdersReportUC == nil)
	log.Printf("🔍 DEBUG: agingReportUC is nil? %v", c.agingReportUC == nil)
	log.Printf("🔍 DEBUG: customerBalanceUC is nil? %v", c.customerBalanceUC == nil)

	reports := router.Group("/reports")
	{
		reports.GET("/daily", c.DailyReport)
	}

	// Reportes financieros operativos (sales_orders + sales_payments; no ledger)
	log.Println("🔍 DEBUG: Registrando salesReports group...")
	salesReports := router.Group("/sales/reports")
	{
		log.Println("🔍 DEBUG: Registrando /open-orders")
		salesReports.GET("/open-orders", c.GetOpenOrdersReport)
		log.Println("🔍 DEBUG: Registrando /aging")
		salesReports.GET("/aging", c.GetAgingReport)
		log.Println("🔍 DEBUG: Registrando /customer-balance")
		salesReports.GET("/customer-balance", c.GetCustomerBalanceReport)
	}
	log.Println("🔍 DEBUG: salesReports group registrado")

	log.Println("Rutas Report disponibles:")
	log.Println("  GET    /api/v1/reports/daily?date=YYYY-MM-DD")
	log.Println("  GET    /api/v1/sales/reports/open-orders?page=1&page_size=50&aging_bucket=0_30&customer_id=...&sort=remaining_desc")
	log.Println("  GET    /api/v1/sales/reports/aging")
	log.Println("  GET    /api/v1/sales/reports/customer-balance?customer_id=...")
}

// DailyReport maneja el reporte diario de ventas
func (c *ReportController) DailyReport(ctx *gin.Context) {
	// ========================================================================
	// PASO 1: Validar header X-Tenant-ID (OBLIGATORIO)
	// ========================================================================
	tenantID := ctx.GetString("tenant_id")
	if tenantID == "" {
		response.JSON(ctx, http.StatusUnauthorized, "tenant_id missing from request context")
		return
	}

	// ========================================================================
	// PASO 2: Parsear tenant_id a UUID
	// ========================================================================
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		response.JSON(ctx, http.StatusBadRequest, "Invalid X-Tenant-ID format")
		return
	}

	// ========================================================================
	// PASO 3: Leer query parameter 'date' (OBLIGATORIO)
	// ========================================================================
	date := ctx.Query("date")
	if date == "" {
		response.JSON(ctx, http.StatusBadRequest, "date query parameter is required (format: YYYY-MM-DD)")
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
			response.JSONWithDetails(ctx, http.StatusBadRequest, "Invalid date format", err.Error())
			return
		}

		// Otros errores → 500
		response.JSONWithDetails(ctx, http.StatusInternalServerError, "Error generating daily report", err.Error())
		return
	}

	// ========================================================================
	// PASO 5: Responder exitosamente
	// ========================================================================
	ctx.JSON(http.StatusOK, resp)
}

// GetOpenOrdersReport maneja GET /api/v1/sales/reports/open-orders
// Parámetros: page, page_size, aging_bucket (0_30|31_60|61_90|90_plus), customer_id, sort (remaining_desc|remaining_asc|days_desc|days_asc)
func (c *ReportController) GetOpenOrdersReport(ctx *gin.Context) {
	start := time.Now()
	defer func() { reportOpenOrdersDuration.Observe(time.Since(start).Seconds()) }()

	tenantID := ctx.GetString("tenant_id")
	if tenantID == "" {
		response.JSON(ctx, http.StatusUnauthorized, "tenant_id missing from request context")
		return
	}

	page := 1
	pageSize := 50
	if p := ctx.Query("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n >= 1 {
			page = n
		}
	}
	if ps := ctx.Query("page_size"); ps != "" {
		if n, err := strconv.Atoi(ps); err == nil && n >= 1 && n <= 100 {
			pageSize = n
		}
	}
	agingBucket := ctx.Query("aging_bucket")
	customerID := ctx.Query("customer_id")
	sort := ctx.Query("sort")

	if c.openOrdersReportUC == nil {
		response.JSON(ctx, http.StatusServiceUnavailable, "Open orders report not available")
		return
	}

	resp, err := c.openOrdersReportUC.Execute(ctx.Request.Context(), tenantID, page, pageSize, agingBucket, customerID, sort)
	if err != nil {
		log.Printf("Error generating open orders report: %v", err)
		response.JSONWithDetails(ctx, http.StatusInternalServerError, "Error generating open orders report", err.Error())
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

// GetCustomerBalanceReport maneja GET /api/v1/sales/reports/customer-balance
// Query: customer_id (obligatorio), aging_bucket (0_30|31_60|61_90|90_plus), page, page_size (default 20, máx 100), sort
// summary y aging siempre completos; open_orders paginado y filtrado por aging_bucket si se envía.
func (c *ReportController) GetCustomerBalanceReport(ctx *gin.Context) {
	tenantID := ctx.GetString("tenant_id")
	if tenantID == "" {
		response.JSON(ctx, http.StatusUnauthorized, "tenant_id missing from request context")
		return
	}
	customerID := ctx.Query("customer_id")
	if customerID == "" {
		response.JSON(ctx, http.StatusBadRequest, "customer_id query parameter is required")
		return
	}

	page := 1
	pageSize := 20
	if p := ctx.Query("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n >= 1 {
			page = n
		}
	}
	if ps := ctx.Query("page_size"); ps != "" {
		if n, err := strconv.Atoi(ps); err == nil && n >= 1 && n <= 100 {
			pageSize = n
		}
	}
	agingBucket := ctx.Query("aging_bucket")
	sort := ctx.Query("sort")

	if c.customerBalanceUC == nil {
		response.JSON(ctx, http.StatusServiceUnavailable, "Customer balance report not available")
		return
	}

	resp, err := c.customerBalanceUC.Execute(ctx.Request.Context(), tenantID, customerID, page, pageSize, agingBucket, sort)
	if err != nil {
		log.Printf("Error generating customer balance report: %v", err)
		response.JSONWithDetails(ctx, http.StatusInternalServerError, "Error generating customer balance report", err.Error())
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

// GetAgingReport maneja GET /api/v1/sales/reports/aging
func (c *ReportController) GetAgingReport(ctx *gin.Context) {
	start := time.Now()
	defer func() { reportAgingDuration.Observe(time.Since(start).Seconds()) }()

	tenantID := ctx.GetString("tenant_id")
	if tenantID == "" {
		response.JSON(ctx, http.StatusUnauthorized, "tenant_id missing from request context")
		return
	}

	if c.agingReportUC == nil {
		response.JSON(ctx, http.StatusServiceUnavailable, "Aging report not available")
		return
	}

	resp, err := c.agingReportUC.Execute(ctx.Request.Context(), tenantID)
	if err != nil {
		log.Printf("Error generating aging report: %v", err)
		response.JSONWithDetails(ctx, http.StatusInternalServerError, "Error generating aging report", err.Error())
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

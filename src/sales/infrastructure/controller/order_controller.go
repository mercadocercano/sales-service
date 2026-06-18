package controller

import (
	"fmt"
	"log"
	"net/http"
	"sales/src/sales/application/request"
	"sales/src/sales/application/response"
	"sales/src/sales/application/usecase"
	"sales/src/sales/domain/entity"
	"sales/src/sales/infrastructure/metrics"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	httpresp "github.com/hornosg/go-shared/infrastructure/response"
	"github.com/shopspring/decimal"
)

// OrderController maneja las peticiones HTTP para orders
type OrderController struct {
	validateStockUC       *usecase.ValidateStockUseCase
	reserveStockUC        *usecase.ReserveStockUseCase
	releaseStockUC        *usecase.ReleaseStockUseCase
	createOrderUC         *usecase.CreateOrderUseCase
	confirmOrderUC        *usecase.ConfirmOrderUseCase
	cancelOrderUC         *usecase.CancelOrderUseCase
	listOrdersUC          *usecase.ListOrdersUseCase
	getOrderUC            *usecase.GetOrderUseCase
	getOrderFinancialUC   *usecase.GetOrderFinancialUseCase
	posSaleUC             *usecase.POSSaleUseCase
	listPosSalesUC        *usecase.ListPosSalesUseCase
	getPosSaleUC          *usecase.GetPosSaleUseCase
	generatePosSalePdfUC  *usecase.GeneratePosSalePdfUseCase
	registerPaymentUC     *usecase.RegisterPaymentUseCase
	applyCustomerCreditUC *usecase.ApplyCustomerCreditUseCase
}

// NewOrderController crea una nueva instancia del controlador
func NewOrderController(
	validateStockUC *usecase.ValidateStockUseCase,
	reserveStockUC *usecase.ReserveStockUseCase,
	releaseStockUC *usecase.ReleaseStockUseCase,
	createOrderUC *usecase.CreateOrderUseCase,
	confirmOrderUC *usecase.ConfirmOrderUseCase,
	cancelOrderUC *usecase.CancelOrderUseCase,
	listOrdersUC *usecase.ListOrdersUseCase,
	getOrderUC *usecase.GetOrderUseCase,
	getOrderFinancialUC *usecase.GetOrderFinancialUseCase,
	posSaleUC *usecase.POSSaleUseCase,
	listPosSalesUC *usecase.ListPosSalesUseCase,
	getPosSaleUC *usecase.GetPosSaleUseCase,
	generatePosSalePdfUC *usecase.GeneratePosSalePdfUseCase,
	registerPaymentUC *usecase.RegisterPaymentUseCase,
	applyCustomerCreditUC *usecase.ApplyCustomerCreditUseCase,
) *OrderController {
	return &OrderController{
		validateStockUC:       validateStockUC,
		reserveStockUC:        reserveStockUC,
		releaseStockUC:        releaseStockUC,
		createOrderUC:         createOrderUC,
		confirmOrderUC:        confirmOrderUC,
		cancelOrderUC:         cancelOrderUC,
		listOrdersUC:          listOrdersUC,
		getOrderUC:            getOrderUC,
		getOrderFinancialUC:   getOrderFinancialUC,
		posSaleUC:             posSaleUC,
		listPosSalesUC:        listPosSalesUC,
		getPosSaleUC:          getPosSaleUC,
		generatePosSalePdfUC:  generatePosSalePdfUC,
		registerPaymentUC:     registerPaymentUC,
		applyCustomerCreditUC: applyCustomerCreditUC,
	}
}

// RegisterRoutes registra las rutas del controlador
func (c *OrderController) RegisterRoutes(router *gin.RouterGroup) {
	// HITO v0.5: Rutas alineadas con dominio Sales
	sales := router.Group("/sales")
	{
		// Órdenes de venta
		orders := sales.Group("/orders")
		{
			orders.GET("", c.ListOrders)
			orders.GET("/:order_id/financial", c.GetOrderFinancial)
			orders.GET("/:order_id", c.GetOrder)
			orders.POST("", c.CreateOrder)
			orders.POST("/:order_id/confirm", c.ConfirmOrder)
			orders.POST("/:order_id/cancel", c.CancelOrder)
			orders.POST("/:order_id/payments", c.RegisterPayment)         // HITO Cobranza
			orders.POST("/:order_id/apply-credit", c.ApplyCustomerCredit) // Credit-based (Fase 1)
			orders.POST("/validate-stock", c.ValidateStock)
			orders.POST("/reserve-stock", c.ReserveStock)
			orders.POST("/release-stock", c.ReleaseStock)
		}

		// POS - Ventas directas
		pos := sales.Group("/pos")
		{
			pos.POST("", c.POSSale)
			pos.GET("", c.ListPosSales)
			pos.GET("/:id", c.GetPosSale)        // E18 Tramo B: detalle de comprobante
			pos.GET("/:id/pdf", c.GetPosSalePdf) // E18 Tramo C: comprobante PDF A4
		}
	}

	log.Println("✅ HITO v0.5 - Rutas Sales disponibles:")
	log.Println("  GET    /api/v1/sales/orders")
	log.Println("  GET    /api/v1/sales/orders/:order_id")
	log.Println("  POST   /api/v1/sales/orders")
	log.Println("  POST   /api/v1/sales/orders/:order_id/confirm")
	log.Println("  POST   /api/v1/sales/orders/:order_id/cancel")
	log.Println("  POST   /api/v1/sales/orders/:order_id/payments ⭐ (HITO Cobranza)")
	log.Println("  POST   /api/v1/sales/orders/:order_id/apply-credit (Credit-based)")
	log.Println("  POST   /api/v1/sales/orders/validate-stock")
	log.Println("  POST   /api/v1/sales/orders/reserve-stock")
	log.Println("  POST   /api/v1/sales/orders/release-stock")
	log.Println("  POST   /api/v1/sales/pos  ⭐ (POS Direct Sale)")
	log.Println("  GET    /api/v1/sales/pos  (POS Sales Report)")
}

// ListPosSales lista las ventas POS del tenant (para reporte)
func (c *OrderController) ListPosSales(ctx *gin.Context) {
	if c.listPosSalesUC == nil {
		httpresp.JSON(ctx, http.StatusServiceUnavailable, "POS sales list not available (database not configured)")
		return
	}

	tenantID := ctx.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		httpresp.JSON(ctx, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		httpresp.JSON(ctx, http.StatusBadRequest, "Invalid X-Tenant-ID format")
		return
	}

	items, err := c.listPosSalesUC.Execute(ctx.Request.Context(), tenantUUID)
	if err != nil {
		log.Printf("Error listing POS sales: %v", err)
		httpresp.JSON(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"items":       items,
		"total_count": len(items),
	})
}

// GetPosSale devuelve el detalle completo de una venta POS (comprobante).
// E18 Tramo B: respeta multi-tenancy — una venta de otro tenant responde 404.
func (c *OrderController) GetPosSale(ctx *gin.Context) {
	if c.getPosSaleUC == nil {
		httpresp.JSON(ctx, http.StatusServiceUnavailable, "POS sale detail not available (database not configured)")
		return
	}

	tenantID := ctx.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		httpresp.JSON(ctx, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}

	saleID := ctx.Param("id")
	if saleID == "" {
		httpresp.JSON(ctx, http.StatusBadRequest, "sale id is required")
		return
	}

	resp, err := c.getPosSaleUC.Execute(ctx.Request.Context(), tenantID, saleID)
	if err != nil {
		// No encontrada o de otro tenant → 404 (no se distingue para no filtrar info)
		if err == entity.ErrPosSaleNotFound {
			httpresp.JSON(ctx, http.StatusNotFound, "POS sale not found")
			return
		}
		// IDs malformados → 400
		if contains(err.Error(), "invalid tenant_id") || contains(err.Error(), "invalid sale id") {
			httpresp.JSONWithDetails(ctx, http.StatusBadRequest, "Invalid request", err.Error())
			return
		}
		log.Printf("Error getting POS sale detail: %v", err)
		httpresp.JSONWithDetails(ctx, http.StatusInternalServerError, "Error getting POS sale detail", err.Error())
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// GetPosSalePdf devuelve el comprobante de una venta POS como PDF A4.
// E18 Tramo C: respeta multi-tenancy — una venta de otro tenant responde 404.
func (c *OrderController) GetPosSalePdf(ctx *gin.Context) {
	if c.generatePosSalePdfUC == nil {
		httpresp.JSON(ctx, http.StatusServiceUnavailable, "POS sale PDF not available (database not configured)")
		return
	}

	tenantID := ctx.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		httpresp.JSON(ctx, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}

	saleID := ctx.Param("id")
	if saleID == "" {
		httpresp.JSON(ctx, http.StatusBadRequest, "sale id is required")
		return
	}

	pdfBytes, err := c.generatePosSalePdfUC.Execute(ctx.Request.Context(), tenantID, saleID)
	if err != nil {
		// No encontrada o de otro tenant → 404 (no se distingue para no filtrar info)
		if err == entity.ErrPosSaleNotFound {
			httpresp.JSON(ctx, http.StatusNotFound, "POS sale not found")
			return
		}
		// IDs malformados → 400
		if contains(err.Error(), "invalid tenant_id") || contains(err.Error(), "invalid sale id") {
			httpresp.JSONWithDetails(ctx, http.StatusBadRequest, "Invalid request", err.Error())
			return
		}
		log.Printf("Error generating POS sale PDF: %v", err)
		httpresp.JSONWithDetails(ctx, http.StatusInternalServerError, "Error generating POS sale PDF", err.Error())
		return
	}

	filename := fmt.Sprintf("comprobante-%s.pdf", saleID)
	ctx.Header("Content-Disposition", fmt.Sprintf("inline; filename=%q", filename))
	ctx.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// CancelOrder maneja la cancelación de una orden confirmada
func (c *OrderController) CancelOrder(ctx *gin.Context) {
	// Verificar que el use case esté disponible
	if c.cancelOrderUC == nil {
		httpresp.JSON(ctx, http.StatusServiceUnavailable, "Order cancellation not available (database not configured)")
		return
	}

	// 1. Validar header X-Tenant-ID (OBLIGATORIO)
	tenantID := ctx.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		httpresp.JSON(ctx, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}

	// 2. Obtener order_id del path
	orderID := ctx.Param("order_id")
	if orderID == "" {
		httpresp.JSON(ctx, http.StatusBadRequest, "order_id is required")
		return
	}

	// 3. Ejecutar use case
	order, err := c.cancelOrderUC.Execute(ctx.Request.Context(), tenantID, orderID)
	if err != nil {
		log.Printf("Error canceling order: %v", err)

		// Manejar errores específicos
		if err == entity.ErrOrderNotFound {
			httpresp.JSON(ctx, http.StatusNotFound, "Order not found")
			return
		}
		if err == entity.ErrOrderNotInConfirmedState {
			httpresp.JSON(ctx, http.StatusConflict, "Order is not in CONFIRMED state")
			return
		}

		// Otros errores
		httpresp.JSONWithDetails(ctx, http.StatusBadGateway, "Error canceling order", err.Error())
		return
	}

	// 5. Responder exitosamente
	ctx.JSON(http.StatusOK, gin.H{
		"order_id": order.OrderID,
		"status":   string(order.Status),
	})
}

// ConfirmOrder maneja la confirmación de una orden
func (c *OrderController) ConfirmOrder(ctx *gin.Context) {
	// Verificar que el use case esté disponible
	if c.confirmOrderUC == nil {
		httpresp.JSON(ctx, http.StatusServiceUnavailable, "Order confirmation not available (database not configured)")
		return
	}

	// 1. Validar header X-Tenant-ID (OBLIGATORIO)
	tenantID := ctx.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		httpresp.JSON(ctx, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}

	// 2. Obtener order_id del path
	orderID := ctx.Param("order_id")
	if orderID == "" {
		httpresp.JSON(ctx, http.StatusBadRequest, "order_id is required")
		return
	}

	// 3. Validar body
	var req request.ConfirmOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		httpresp.JSONWithDetails(ctx, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// 4. Ejecutar use case
	order, err := c.confirmOrderUC.Execute(ctx.Request.Context(), tenantID, orderID, req.Reference)
	if err != nil {
		log.Printf("Error confirming order: %v", err)

		// Manejar errores específicos
		if err == entity.ErrOrderNotFound {
			httpresp.JSON(ctx, http.StatusNotFound, "Order not found")
			return
		}
		if err == entity.ErrOrderNotInCreatedState {
			httpresp.JSON(ctx, http.StatusConflict, "Order is not in CREATED state")
			return
		}
		if contains(err.Error(), "insufficient_reserved_stock") {
			httpresp.JSON(ctx, http.StatusConflict, "Insufficient reserved stock")
			return
		}

		// Otros errores
		httpresp.JSONWithDetails(ctx, http.StatusInternalServerError, "Error confirming order", err.Error())
		return
	}

	// 6. Responder exitosamente
	ctx.JSON(http.StatusOK, gin.H{
		"order_id": order.OrderID,
		"status":   string(order.Status),
	})
}

// CreateOrder maneja la creación de una orden
func (c *OrderController) CreateOrder(ctx *gin.Context) {
	// Verificar que el use case esté disponible
	if c.createOrderUC == nil {
		httpresp.JSON(ctx, http.StatusServiceUnavailable, "Order creation not available (database not configured)")
		return
	}

	// 1. Validar header X-Tenant-ID (OBLIGATORIO)
	tenantID := ctx.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		httpresp.JSON(ctx, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}

	// 2. Validar body
	var req request.CreateOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		httpresp.JSONWithDetails(ctx, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// 3. Ejecutar use case con snapshots de PIM
	resp, err := c.createOrderUC.Execute(ctx.Request.Context(), tenantID, &req)
	if err != nil {
		log.Printf("Error creating order: %v", err)

		// HITO v0.6: Manejar error de customer no encontrado
		if contains(err.Error(), "customer_not_found") {
			httpresp.JSON(ctx, http.StatusNotFound, "Customer not found")
			return
		}

		// Otros errores
		httpresp.JSONWithDetails(ctx, http.StatusInternalServerError, "Error creating order", err.Error())
		return
	}

	// 5. Responder exitosamente con 201 Created
	ctx.JSON(http.StatusCreated, resp)
}

// ReleaseStock maneja la liberación de stock reservado
func (c *OrderController) ReleaseStock(ctx *gin.Context) {
	// 1. Validar header X-Tenant-ID (OBLIGATORIO)
	tenantID := ctx.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		httpresp.JSON(ctx, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}

	// 2. Validar body
	var req request.ReleaseStockRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		httpresp.JSONWithDetails(ctx, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// 3. Ejecutar use case
	resp, err := c.releaseStockUC.Execute(tenantID, &req)
	if err != nil {
		log.Printf("Error releasing stock: %v", err)

		// Si es error de stock reservado insuficiente → 409
		if contains(err.Error(), "insufficient_reserved_stock") {
			httpresp.JSON(ctx, http.StatusConflict, "Insufficient reserved stock to release")
			return
		}

		// Otros errores → 502
		httpresp.JSONWithDetails(ctx, http.StatusBadGateway, "Error communicating with stock service", err.Error())
		return
	}

	// 5. Responder exitosamente
	ctx.JSON(http.StatusOK, resp)
}

// ReserveStock maneja la reserva de stock para una orden
func (c *OrderController) ReserveStock(ctx *gin.Context) {
	// 1. Validar header X-Tenant-ID (OBLIGATORIO)
	tenantID := ctx.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		httpresp.JSON(ctx, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}

	// 2. Validar body
	var req request.ReserveStockRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		httpresp.JSONWithDetails(ctx, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// 3. Ejecutar use case
	resp, err := c.reserveStockUC.Execute(tenantID, &req)
	if err != nil {
		log.Printf("Error reserving stock: %v", err)

		// Si es error de stock insuficiente → 409
		if contains(err.Error(), "insufficient_stock") {
			httpresp.JSON(ctx, http.StatusConflict, "Insufficient stock available")
			return
		}

		// Otros errores → 502
		httpresp.JSONWithDetails(ctx, http.StatusBadGateway, "Error communicating with stock service", err.Error())
		return
	}

	// 5. Responder exitosamente
	ctx.JSON(http.StatusOK, resp)
}

// contains helper para verificar substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if len(s[i:]) >= len(substr) && s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ListOrders maneja el listado de órdenes con paginación
func (c *OrderController) ListOrders(ctx *gin.Context) {
	// Verificar que el use case esté disponible
	if c.listOrdersUC == nil {
		httpresp.JSON(ctx, http.StatusServiceUnavailable, "Order listing not available (database not configured)")
		return
	}

	// 1. Validar header X-Tenant-ID (OBLIGATORIO)
	tenantID := ctx.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		httpresp.JSON(ctx, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}

	// 2. Obtener parámetros de paginación
	page := 1
	pageSize := 10

	if pageStr := ctx.Query("page"); pageStr != "" {
		if p, err := ctx.GetQuery("page"); err {
			if n, parseErr := parsePageParam(p); parseErr == nil {
				page = n
			}
		}
	}

	if pageSizeStr := ctx.Query("page_size"); pageSizeStr != "" {
		if ps, err := ctx.GetQuery("page_size"); err {
			if n, parseErr := parsePageParam(ps); parseErr == nil {
				pageSize = n
			}
		}
	}

	// Filtro: status (CREATED|CONFIRMED|PAID|CANCELED) o financial_status (OPEN|PAID); financial_status tiene prioridad
	statusFilter := ctx.Query("status")
	if fs := ctx.Query("financial_status"); fs != "" {
		switch fs {
		case "OPEN":
			statusFilter = "OPEN" // órdenes con remaining > 0 (CREATED + CONFIRMED)
		case "PAID":
			statusFilter = "PAID" // remaining == 0
		}
	}

	// 3. Ejecutar use case
	resp, err := c.listOrdersUC.Execute(ctx.Request.Context(), tenantID, page, pageSize, statusFilter)
	if err != nil {
		log.Printf("Error listing orders: %v", err)
		httpresp.JSONWithDetails(ctx, http.StatusInternalServerError, "Error listing orders", err.Error())
		return
	}

	// 4. Responder exitosamente
	ctx.JSON(http.StatusOK, resp)
}

// GetOrder maneja la obtención de una orden por ID
func (c *OrderController) GetOrder(ctx *gin.Context) {
	// Verificar que el use case esté disponible
	if c.getOrderUC == nil {
		httpresp.JSON(ctx, http.StatusServiceUnavailable, "Order retrieval not available (database not configured)")
		return
	}

	// 1. Validar header X-Tenant-ID (OBLIGATORIO)
	tenantID := ctx.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		httpresp.JSON(ctx, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}

	// 2. Obtener order_id del path
	orderID := ctx.Param("order_id")
	if orderID == "" {
		httpresp.JSON(ctx, http.StatusBadRequest, "order_id is required")
		return
	}

	// 3. Ejecutar use case
	resp, err := c.getOrderUC.Execute(ctx.Request.Context(), tenantID, orderID)
	if err != nil {
		log.Printf("Error getting order: %v", err)

		// Manejar errores específicos
		if err == entity.ErrOrderNotFound {
			httpresp.JSON(ctx, http.StatusNotFound, "Order not found")
			return
		}

		// Otros errores
		httpresp.JSONWithDetails(ctx, http.StatusInternalServerError, "Error getting order", err.Error())
		return
	}

	// 4. Responder exitosamente
	ctx.JSON(http.StatusOK, resp)
}

// GetOrderFinancial maneja GET /orders/:order_id/financial (resumen financiero: total, paid, remaining, status)
func (c *OrderController) GetOrderFinancial(ctx *gin.Context) {
	if c.getOrderFinancialUC == nil {
		httpresp.JSON(ctx, http.StatusServiceUnavailable, "Order financial view not available (database not configured)")
		return
	}
	tenantID := ctx.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		httpresp.JSON(ctx, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}
	orderID := ctx.Param("order_id")
	if orderID == "" {
		httpresp.JSON(ctx, http.StatusBadRequest, "order_id is required")
		return
	}
	resp, err := c.getOrderFinancialUC.Execute(ctx.Request.Context(), tenantID, orderID)
	if err != nil {
		if err == entity.ErrOrderNotFound {
			httpresp.JSON(ctx, http.StatusNotFound, "Order not found")
			return
		}
		log.Printf("Error getting order financial: %v", err)
		httpresp.JSONWithDetails(ctx, http.StatusInternalServerError, "Error getting order financial", err.Error())
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

// parsePageParam parsea parámetros numéricos
func parsePageParam(s string) (int, error) {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return 0, err
	}
	return n, nil
}

// POSSale maneja venta directa POS sin crear orden
// H8: Instrumentado con métricas Prometheus (pos_sales_total, pos_sales_success, pos_sales_failed, pos_sale_latency_ms)
func (c *OrderController) POSSale(ctx *gin.Context) {
	start := time.Now()
	metrics.POSSalesTotal.Inc()

	// 1. Validar header X-Tenant-ID (OBLIGATORIO)
	tenantID := ctx.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		metrics.POSSalesFailed.Inc()
		httpresp.JSON(ctx, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}

	// 2. Validar body
	var req request.POSSaleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		metrics.POSSalesFailed.Inc()
		httpresp.JSONWithDetails(ctx, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// 3. Ejecutar use case
	authToken := ctx.GetHeader("Authorization")
	resp, err := c.posSaleUC.Execute(tenantID, &req, authToken)
	durationMs := time.Since(start).Milliseconds()
	metrics.POSSaleLatency.Observe(float64(durationMs))

	if err != nil {
		metrics.POSSalesFailed.Inc()
		log.Printf("Error processing POS sale: %v", err)

		// Mapeo de errores de negocio a códigos 4xx. El 502 queda reservado
		// SOLO para fallos reales de downstream/persistencia (Bad Gateway),
		// para que el front pueda distinguir una regla de negocio de una caída.
		msg := err.Error()
		switch {
		// HITO v0.6: Error de customer no encontrado → 404
		case contains(msg, "customer_not_found"):
			httpresp.JSON(ctx, http.StatusNotFound, "Customer not found")

		// Método de pago inválido (p. ej. el front manda "efectivo" en vez de "cash") → 400
		case contains(msg, "payment_method_not_found"):
			httpresp.JSONWithDetails(ctx, http.StatusBadRequest, "Invalid payment method", msg)

		// Stock insuficiente, no inicializado o rechazado por stock-service → 409
		// (se chequea antes que las validaciones genéricas para no confundir
		// "stock rejected: ..." con un error de input)
		case contains(msg, "insufficient_stock"),
			contains(msg, "stock rejected"):
			httpresp.JSONWithDetails(ctx, http.StatusConflict, "Stock unavailable for POS sale", msg)

		// Validaciones de input / reglas de dominio del request → 400.
		// Cubre todas las validaciones del use case y de las entidades POS:
		// "*_id is required", "amount_paid must be greater than 0",
		// "quantity/price must be greater than...", "at least one item", etc.
		case contains(msg, "is required"),
			contains(msg, "must be greater"),
			contains(msg, "must be at least"),
			contains(msg, "must be positive"),
			contains(msg, "amount_paid"),
			contains(msg, "at least one item"),
			contains(msg, "invalid tenant_id"):
			httpresp.JSONWithDetails(ctx, http.StatusBadRequest, "Invalid request", msg)

		// Fallos reales de downstream / persistencia → 502
		default:
			httpresp.JSONWithDetails(ctx, http.StatusBadGateway, "Error processing POS sale", msg)
		}
		return
	}

	metrics.POSSalesSuccess.Inc()
	// TTFS se registra en POSSaleUseCase (solo primera venta del tenant)
	ctx.JSON(http.StatusCreated, resp)
}

// ValidateStock maneja la validación de stock para items de orden
func (c *OrderController) ValidateStock(ctx *gin.Context) {
	// 1. Validar header X-Tenant-ID (OBLIGATORIO)
	tenantID := ctx.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		httpresp.JSON(ctx, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}

	// 2. Validar body
	var req request.ValidateStockRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		httpresp.JSONWithDetails(ctx, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// 3. Validar que solo venga 1 item
	if len(req.Items) != 1 {
		httpresp.JSON(ctx, http.StatusBadRequest, "Only 1 item is allowed")
		return
	}

	// 4. Ejecutar use case
	resp, err := c.validateStockUC.Execute(tenantID, &req)
	if err != nil {
		log.Printf("Error validating stock: %v", err)
		httpresp.JSONWithDetails(ctx, http.StatusBadGateway, "Error communicating with stock service", err.Error())
		return
	}

	// 6. Responder exitosamente
	ctx.JSON(http.StatusOK, resp)
}

// RegisterPayment maneja el registro de un pago contra una orden confirmada
// HITO Cobranza: Endpoint con validación transaccional
func (c *OrderController) RegisterPayment(ctx *gin.Context) {
	// Verificar que el use case esté disponible
	if c.registerPaymentUC == nil {
		httpresp.JSON(ctx, http.StatusServiceUnavailable, "Payment registration not available (database not configured)")
		return
	}

	// 1. Validar header X-Tenant-ID (OBLIGATORIO)
	tenantID := ctx.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		httpresp.JSON(ctx, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}

	// 2. Obtener order_id del path
	orderID := ctx.Param("order_id")
	if orderID == "" {
		httpresp.JSON(ctx, http.StatusBadRequest, "order_id is required")
		return
	}

	// 3. Validar body
	var req request.RegisterPaymentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		httpresp.JSONWithDetails(ctx, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// 4. Ejecutar use case
	paymentID, err := c.registerPaymentUC.Execute(ctx.Request.Context(), tenantID, orderID, req.Amount)
	if err != nil {
		log.Printf("Error registering payment: %v", err)

		// Manejar errores específicos
		if err == entity.ErrOrderNotFound {
			httpresp.JSON(ctx, http.StatusNotFound, "Order not found")
			return
		}
		if contains(err.Error(), "order must be confirmed") {
			httpresp.JSON(ctx, http.StatusConflict, "Order must be confirmed before accepting payments")
			return
		}
		if contains(err.Error(), "payment exceeds order total") {
			httpresp.JSONWithDetails(ctx, http.StatusBadRequest, "Payment exceeds order total", err.Error())
			return
		}

		// Otros errores
		httpresp.JSONWithDetails(ctx, http.StatusInternalServerError, "Error registering payment", err.Error())
		return
	}

	// 5. Responder exitosamente con 201 Created
	resp := response.RegisterPaymentResponse{
		PaymentID: paymentID,
		Amount:    req.Amount,
		Currency:  "ARS",
	}
	ctx.JSON(http.StatusCreated, resp)
}

// ApplyCustomerCredit aplica un crédito (parcial o total) a una orden.
// POST /api/v1/sales/orders/:order_id/apply-credit
// Body: { "customer_credit_id": "uuid", "amount": number }
func (c *OrderController) ApplyCustomerCredit(ctx *gin.Context) {
	if c.applyCustomerCreditUC == nil {
		httpresp.JSON(ctx, http.StatusServiceUnavailable, "Apply credit not available")
		return
	}
	tenantID := ctx.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		httpresp.JSON(ctx, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}
	orderID := ctx.Param("order_id")
	if orderID == "" {
		httpresp.JSON(ctx, http.StatusBadRequest, "order_id is required")
		return
	}
	var req request.ApplyCreditRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		httpresp.JSONWithDetails(ctx, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}
	amountDec := decimal.NewFromFloat(req.Amount)
	applicationID, err := c.applyCustomerCreditUC.Execute(ctx.Request.Context(), tenantID, req.CustomerCreditID, orderID, amountDec)
	if err != nil {
		log.Printf("Error applying credit: %v", err)
		if contains(err.Error(), "not found") {
			httpresp.JSON(ctx, http.StatusNotFound, err.Error())
			return
		}
		if contains(err.Error(), "must be CREATED or CONFIRMED") || contains(err.Error(), "fully applied") ||
			contains(err.Error(), "less than amount") || contains(err.Error(), "same customer") {
			httpresp.JSON(ctx, http.StatusBadRequest, err.Error())
			return
		}
		httpresp.JSONWithDetails(ctx, http.StatusInternalServerError, "Failed to apply credit", err.Error())
		return
	}
	ctx.JSON(http.StatusCreated, response.ApplyCreditResponse{ApplicationID: applicationID, Amount: req.Amount})
}

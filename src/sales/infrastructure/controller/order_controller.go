package controller

import (
	"fmt"
	"log"
	"net/http"
	"sales/src/sales/application/request"
	"sales/src/sales/application/usecase"
	"sales/src/sales/domain/entity"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// OrderController maneja las peticiones HTTP para orders
type OrderController struct {
	validateStockUC *usecase.ValidateStockUseCase
	reserveStockUC  *usecase.ReserveStockUseCase
	releaseStockUC  *usecase.ReleaseStockUseCase
	createOrderUC   *usecase.CreateOrderUseCase
	confirmOrderUC  *usecase.ConfirmOrderUseCase
	cancelOrderUC   *usecase.CancelOrderUseCase
	listOrdersUC    *usecase.ListOrdersUseCase
	getOrderUC      *usecase.GetOrderUseCase
	posSaleUC       *usecase.POSSaleUseCase
	listPosSalesUC  *usecase.ListPosSalesUseCase
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
	posSaleUC *usecase.POSSaleUseCase,
	listPosSalesUC *usecase.ListPosSalesUseCase,
) *OrderController {
	return &OrderController{
		validateStockUC: validateStockUC,
		reserveStockUC:  reserveStockUC,
		releaseStockUC:  releaseStockUC,
		createOrderUC:   createOrderUC,
		confirmOrderUC:  confirmOrderUC,
		cancelOrderUC:   cancelOrderUC,
		listOrdersUC:    listOrdersUC,
		getOrderUC:      getOrderUC,
		posSaleUC:       posSaleUC,
		listPosSalesUC:  listPosSalesUC,
	}
}

// RegisterRoutes registra las rutas del controlador
func (c *OrderController) RegisterRoutes(router *gin.RouterGroup) {
	orders := router.Group("/orders")
	{
		orders.GET("", c.ListOrders)
		orders.GET("/:order_id", c.GetOrder)
		orders.POST("", c.CreateOrder)
		orders.POST("/:order_id/confirm", c.ConfirmOrder)
		orders.POST("/:order_id/cancel", c.CancelOrder)
		orders.POST("/validate-stock", c.ValidateStock)
		orders.POST("/reserve-stock", c.ReserveStock)
		orders.POST("/release-stock", c.ReleaseStock)
	}

	// Grupo POS para ventas directas
	pos := router.Group("/pos")
	{
		pos.POST("/sale", c.POSSale)
		pos.GET("/sales", c.ListPosSales)
	}

	log.Println("Rutas Order disponibles:")
	log.Println("  GET    /api/v1/orders")
	log.Println("  GET    /api/v1/orders/:order_id")
	log.Println("  POST   /api/v1/orders")
	log.Println("  POST   /api/v1/orders/:order_id/confirm")
	log.Println("  POST   /api/v1/orders/:order_id/cancel")
	log.Println("  POST   /api/v1/orders/validate-stock")
	log.Println("  POST   /api/v1/orders/reserve-stock")
	log.Println("  POST   /api/v1/orders/release-stock")
	log.Println("  POST   /api/v1/pos/sale  ⭐ (POS Direct Sale)")
	log.Println("  GET    /api/v1/pos/sales  (POS Sales Report)")
}

// ListPosSales lista las ventas POS del tenant (para reporte)
func (c *OrderController) ListPosSales(ctx *gin.Context) {
	if c.listPosSalesUC == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "POS sales list not available (database not configured)",
		})
		return
	}

	tenantID := ctx.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid X-Tenant-ID format"})
		return
	}

	items, err := c.listPosSalesUC.Execute(ctx.Request.Context(), tenantUUID)
	if err != nil {
		log.Printf("Error listing POS sales: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"items":       items,
		"total_count": len(items),
	})
}

// CancelOrder maneja la cancelación de una orden confirmada
func (c *OrderController) CancelOrder(ctx *gin.Context) {
	// Verificar que el use case esté disponible
	if c.cancelOrderUC == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Order cancellation not available (database not configured)",
		})
		return
	}

	// 1. Validar header X-Tenant-ID (OBLIGATORIO)
	tenantID := ctx.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "X-Tenant-ID header is required",
		})
		return
	}

	// 2. Obtener Authorization header
	authToken := ctx.GetHeader("Authorization")

	// 3. Obtener order_id del path
	orderID := ctx.Param("order_id")
	if orderID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "order_id is required",
		})
		return
	}

	// 4. Ejecutar use case
	order, err := c.cancelOrderUC.Execute(ctx.Request.Context(), tenantID, authToken, orderID)
	if err != nil {
		log.Printf("Error canceling order: %v", err)

		// Manejar errores específicos
		if err == entity.ErrOrderNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error": "Order not found",
			})
			return
		}
		if err == entity.ErrOrderNotInConfirmedState {
			ctx.JSON(http.StatusConflict, gin.H{
				"error": "Order is not in CONFIRMED state",
			})
			return
		}

		// Otros errores
		ctx.JSON(http.StatusBadGateway, gin.H{
			"error":   "Error canceling order",
			"details": err.Error(),
		})
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
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Order confirmation not available (database not configured)",
		})
		return
	}

	// 1. Validar header X-Tenant-ID (OBLIGATORIO)
	tenantID := ctx.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "X-Tenant-ID header is required",
		})
		return
	}

	// 2. Obtener Authorization header
	authToken := ctx.GetHeader("Authorization")

	// 3. Obtener order_id del path
	orderID := ctx.Param("order_id")
	if orderID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "order_id is required",
		})
		return
	}

	// 4. Validar body
	var req request.ConfirmOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// 5. Ejecutar use case
	order, err := c.confirmOrderUC.Execute(ctx.Request.Context(), tenantID, authToken, orderID, req.Reference)
	if err != nil {
		log.Printf("Error confirming order: %v", err)

		// Manejar errores específicos
		if err == entity.ErrOrderNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error": "Order not found",
			})
			return
		}
		if err == entity.ErrOrderNotInCreatedState {
			ctx.JSON(http.StatusConflict, gin.H{
				"error": "Order is not in CREATED state",
			})
			return
		}
		if contains(err.Error(), "insufficient_reserved_stock") {
			ctx.JSON(http.StatusConflict, gin.H{
				"error": "Insufficient reserved stock",
			})
			return
		}

		// Otros errores
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error confirming order",
			"details": err.Error(),
		})
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
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Order creation not available (database not configured)",
		})
		return
	}

	// 1. Validar header X-Tenant-ID (OBLIGATORIO)
	tenantID := ctx.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "X-Tenant-ID header is required",
		})
		return
	}

	// 2. Obtener Authorization header
	authToken := ctx.GetHeader("Authorization")

	// 3. Validar body
	var req request.CreateOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// 4. Ejecutar use case con snapshots de PIM
	resp, err := c.createOrderUC.Execute(ctx.Request.Context(), tenantID, authToken, &req)
	if err != nil {
		log.Printf("Error creating order: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error creating order",
			"details": err.Error(),
		})
		return
	}

	// 4. Responder exitosamente con 201 Created
	ctx.JSON(http.StatusCreated, resp)
}

// ReleaseStock maneja la liberación de stock reservado
func (c *OrderController) ReleaseStock(ctx *gin.Context) {
	// 1. Validar header X-Tenant-ID (OBLIGATORIO)
	tenantID := ctx.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "X-Tenant-ID header is required",
		})
		return
	}

	// 2. Obtener Authorization header (pasarlo tal cual)
	authToken := ctx.GetHeader("Authorization")

	// 3. Validar body
	var req request.ReleaseStockRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// 4. Ejecutar use case
	resp, err := c.releaseStockUC.Execute(tenantID, authToken, &req)
	if err != nil {
		log.Printf("Error releasing stock: %v", err)

		// Si es error de stock reservado insuficiente → 409
		if contains(err.Error(), "insufficient_reserved_stock") {
			ctx.JSON(http.StatusConflict, gin.H{
				"error": "Insufficient reserved stock to release",
			})
			return
		}

		// Otros errores → 502
		ctx.JSON(http.StatusBadGateway, gin.H{
			"error":   "Error communicating with stock service",
			"details": err.Error(),
		})
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
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "X-Tenant-ID header is required",
		})
		return
	}

	// 2. Obtener Authorization header (pasarlo tal cual)
	authToken := ctx.GetHeader("Authorization")

	// 3. Validar body
	var req request.ReserveStockRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// 4. Ejecutar use case
	resp, err := c.reserveStockUC.Execute(tenantID, authToken, &req)
	if err != nil {
		log.Printf("Error reserving stock: %v", err)

		// Si es error de stock insuficiente → 409
		if contains(err.Error(), "insufficient_stock") {
			ctx.JSON(http.StatusConflict, gin.H{
				"error": "Insufficient stock available",
			})
			return
		}

		// Otros errores → 502
		ctx.JSON(http.StatusBadGateway, gin.H{
			"error":   "Error communicating with stock service",
			"details": err.Error(),
		})
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
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Order listing not available (database not configured)",
		})
		return
	}

	// 1. Validar header X-Tenant-ID (OBLIGATORIO)
	tenantID := ctx.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "X-Tenant-ID header is required",
		})
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

	// 3. Ejecutar use case
	resp, err := c.listOrdersUC.Execute(ctx.Request.Context(), tenantID, page, pageSize)
	if err != nil {
		log.Printf("Error listing orders: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error listing orders",
			"details": err.Error(),
		})
		return
	}

	// 4. Responder exitosamente
	ctx.JSON(http.StatusOK, resp)
}

// GetOrder maneja la obtención de una orden por ID
func (c *OrderController) GetOrder(ctx *gin.Context) {
	// Verificar que el use case esté disponible
	if c.getOrderUC == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Order retrieval not available (database not configured)",
		})
		return
	}

	// 1. Validar header X-Tenant-ID (OBLIGATORIO)
	tenantID := ctx.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "X-Tenant-ID header is required",
		})
		return
	}

	// 2. Obtener order_id del path
	orderID := ctx.Param("order_id")
	if orderID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "order_id is required",
		})
		return
	}

	// 3. Ejecutar use case
	resp, err := c.getOrderUC.Execute(ctx.Request.Context(), tenantID, orderID)
	if err != nil {
		log.Printf("Error getting order: %v", err)

		// Manejar errores específicos
		if err == entity.ErrOrderNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error": "Order not found",
			})
			return
		}

		// Otros errores
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error getting order",
			"details": err.Error(),
		})
		return
	}

	// 4. Responder exitosamente
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
func (c *OrderController) POSSale(ctx *gin.Context) {
	// 1. Validar header X-Tenant-ID (OBLIGATORIO)
	tenantID := ctx.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "X-Tenant-ID header is required",
		})
		return
	}

	// 2. Obtener Authorization header
	authToken := ctx.GetHeader("Authorization")

	// 3. Validar body
	var req request.POSSaleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// 4. Ejecutar use case
	resp, err := c.posSaleUC.Execute(tenantID, authToken, &req)
	if err != nil {
		log.Printf("Error processing POS sale: %v", err)

		// Si es error de stock insuficiente → 409
		if contains(err.Error(), "insufficient_stock") {
			ctx.JSON(http.StatusConflict, gin.H{
				"error": "Insufficient stock for POS sale",
			})
			return
		}

		// Otros errores → 502
		ctx.JSON(http.StatusBadGateway, gin.H{
			"error":   "Error processing POS sale",
			"details": err.Error(),
		})
		return
	}

	// 5. Responder exitosamente
	ctx.JSON(http.StatusCreated, resp)
}

// ValidateStock maneja la validación de stock para items de orden
func (c *OrderController) ValidateStock(ctx *gin.Context) {
	// 1. Validar header X-Tenant-ID (OBLIGATORIO)
	tenantID := ctx.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "X-Tenant-ID header is required",
		})
		return
	}

	// 2. Obtener Authorization header (pasarlo tal cual, no validar)
	authToken := ctx.GetHeader("Authorization")

	// 3. Validar body
	var req request.ValidateStockRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// 4. Validar que solo venga 1 item
	if len(req.Items) != 1 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Only 1 item is allowed",
		})
		return
	}

	// 5. Ejecutar use case
	resp, err := c.validateStockUC.Execute(tenantID, authToken, &req)
	if err != nil {
		log.Printf("Error validating stock: %v", err)
		ctx.JSON(http.StatusBadGateway, gin.H{
			"error":   "Error communicating with stock service",
			"details": err.Error(),
		})
		return
	}

	// 6. Responder exitosamente
	ctx.JSON(http.StatusOK, resp)
}

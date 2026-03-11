package controller

import (
	"log"
	"net/http"

	"sales/src/sales/application/request"
	"sales/src/sales/application/response"
	"sales/src/sales/application/usecase"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// CustomerController maneja las peticiones HTTP para operaciones por cliente (créditos a cuenta).
type CustomerController struct {
	createCreditUC *usecase.CreateCustomerCreditUseCase
}

// NewCustomerController crea el controlador.
func NewCustomerController(createCreditUC *usecase.CreateCustomerCreditUseCase) *CustomerController {
	return &CustomerController{
		createCreditUC: createCreditUC,
	}
}

// RegisterRoutes registra las rutas bajo /sales/customers (router = v1, path final /api/v1/sales/customers/...)
func (c *CustomerController) RegisterRoutes(router *gin.RouterGroup) {
	sales := router.Group("/sales")
	customers := sales.Group("/customers")
	{
		customers.POST("/:customer_id/credits", c.CreateCredit)
	}
	log.Println("  POST   /api/v1/sales/customers/:customer_id/credits (Credit-based Fase 1)")
}

// CreateCredit maneja POST /api/v1/sales/customers/:customer_id/credits
// Body: { "amount": number }. Respuesta 201: { "credit_id", "amount", "status": "AVAILABLE" }
func (c *CustomerController) CreateCredit(ctx *gin.Context) {
	if c.createCreditUC == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "Create credit not available (database or eventbus not configured)"})
		return
	}
	tenantID := ctx.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}
	customerID := ctx.Param("customer_id")
	if customerID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "customer_id is required"})
		return
	}
	var req request.CreateCustomerCreditRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}
	amountDec := decimal.NewFromFloat(req.Amount)
	creditID, err := c.createCreditUC.Execute(ctx.Request.Context(), tenantID, customerID, amountDec)
	if err != nil {
		log.Printf("Error creating customer credit: %v", err)
		if err.Error() == "amount must be greater than zero" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create credit", "details": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, response.CreateCustomerCreditResponse{
		CreditID: creditID,
		Amount:   req.Amount,
		Status:   "AVAILABLE",
	})
}

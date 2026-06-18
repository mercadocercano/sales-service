package controller

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	sharedmw "github.com/hornosg/go-shared/infrastructure/middleware"
	httpresp "github.com/hornosg/go-shared/infrastructure/response"

	"sales/src/sales/application/request"
	"sales/src/sales/application/response"
	"sales/src/sales/application/usecase"
	"sales/src/sales/domain/entity"
)

// CashSessionController expone los endpoints de caja POS (ADR-003 §8). Maneja efectivo
// real (L4): toda ruta pasa por RequireRole; el actor (user_id) sale del JWT, nunca del
// body. approve-review exige rol supervisor EXACTO (separación de funciones, C3).
type CashSessionController struct {
	openUC     *usecase.OpenCashSessionUseCase
	currentUC  *usecase.GetCurrentCashSessionUseCase
	getUC      *usecase.GetCashSessionUseCase
	movementUC *usecase.RegisterCashMovementUseCase
	closeUC    *usecase.CloseCashSessionUseCase
	approveUC  *usecase.ApproveCashReviewUseCase
}

func NewCashSessionController(
	openUC *usecase.OpenCashSessionUseCase,
	currentUC *usecase.GetCurrentCashSessionUseCase,
	getUC *usecase.GetCashSessionUseCase,
	movementUC *usecase.RegisterCashMovementUseCase,
	closeUC *usecase.CloseCashSessionUseCase,
	approveUC *usecase.ApproveCashReviewUseCase,
) *CashSessionController {
	return &CashSessionController{openUC, currentUC, getUC, movementUC, closeUC, approveUC}
}

func (c *CashSessionController) RegisterRoutes(router *gin.RouterGroup) {
	cashier := sharedmw.RequireRole("cashier", "supervisor")
	supervisorOnly := sharedmw.RequireRole("supervisor") // C3: exacto, sin comodín

	grp := router.Group("/sales/cash-sessions")
	{
		grp.POST("", cashier, c.Open)
		grp.GET("/current", cashier, c.GetCurrent)
		grp.GET("/:id", cashier, c.GetByID)
		grp.POST("/:id/movements", cashier, c.RegisterMovement)
		grp.POST("/:id/close", cashier, c.Close)
		grp.POST("/:id/approve-review", supervisorOnly, c.ApproveReview)
	}
	log.Println("✅ E18 Tramo A - Rutas Caja disponibles bajo /api/v1/sales/cash-sessions (RBAC cashier/supervisor)")
}

func (c *CashSessionController) Open(ctx *gin.Context) {
	if c.openUC == nil {
		httpresp.JSON(ctx, http.StatusServiceUnavailable, "cash sessions not available (database not configured)")
		return
	}
	tenantID, ok := tenantFromHeader(ctx)
	if !ok {
		return
	}
	userID, ok := userIDFromContext(ctx)
	if !ok {
		return
	}
	var req request.OpenCashSessionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		httpresp.JSON(ctx, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	session, err := c.openUC.Execute(ctx.Request.Context(), tenantID, userID, req)
	if err != nil {
		mapCashError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, response.NewCashSessionResponse(session))
}

func (c *CashSessionController) GetCurrent(ctx *gin.Context) {
	if c.currentUC == nil {
		httpresp.JSON(ctx, http.StatusServiceUnavailable, "cash sessions not available (database not configured)")
		return
	}
	tenantID, ok := tenantFromHeader(ctx)
	if !ok {
		return
	}
	pointOfSaleID := ctx.Query("point_of_sale_id")
	session, err := c.currentUC.Execute(ctx.Request.Context(), tenantID, pointOfSaleID)
	if err != nil {
		mapCashError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, response.NewCashSessionResponse(session))
}

func (c *CashSessionController) GetByID(ctx *gin.Context) {
	if c.getUC == nil {
		httpresp.JSON(ctx, http.StatusServiceUnavailable, "cash sessions not available (database not configured)")
		return
	}
	tenantID, ok := tenantFromHeader(ctx)
	if !ok {
		return
	}
	id, ok := pathUUID(ctx, "id")
	if !ok {
		return
	}
	session, expected, err := c.getUC.Execute(ctx.Request.Context(), tenantID, id)
	if err != nil {
		mapCashError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, response.NewCashSessionDetailResponse(session, expected))
}

func (c *CashSessionController) RegisterMovement(ctx *gin.Context) {
	if c.movementUC == nil {
		httpresp.JSON(ctx, http.StatusServiceUnavailable, "cash sessions not available (database not configured)")
		return
	}
	tenantID, ok := tenantFromHeader(ctx)
	if !ok {
		return
	}
	userID, ok := userIDFromContext(ctx)
	if !ok {
		return
	}
	id, ok := pathUUID(ctx, "id")
	if !ok {
		return
	}
	var req request.CashMovementRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		httpresp.JSON(ctx, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	session, err := c.movementUC.Execute(ctx.Request.Context(), tenantID, id, userID, req)
	if err != nil {
		mapCashError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, response.NewCashSessionResponse(session))
}

func (c *CashSessionController) Close(ctx *gin.Context) {
	if c.closeUC == nil {
		httpresp.JSON(ctx, http.StatusServiceUnavailable, "cash sessions not available (database not configured)")
		return
	}
	tenantID, ok := tenantFromHeader(ctx)
	if !ok {
		return
	}
	userID, ok := userIDFromContext(ctx)
	if !ok {
		return
	}
	id, ok := pathUUID(ctx, "id")
	if !ok {
		return
	}
	var req request.CloseCashSessionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		httpresp.JSON(ctx, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	session, err := c.closeUC.Execute(ctx.Request.Context(), tenantID, id, userID, req.CountedAmount)
	if err != nil {
		mapCashError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, response.NewCashSessionResponse(session))
}

func (c *CashSessionController) ApproveReview(ctx *gin.Context) {
	if c.approveUC == nil {
		httpresp.JSON(ctx, http.StatusServiceUnavailable, "cash sessions not available (database not configured)")
		return
	}
	tenantID, ok := tenantFromHeader(ctx)
	if !ok {
		return
	}
	userID, ok := userIDFromContext(ctx)
	if !ok {
		return
	}
	id, ok := pathUUID(ctx, "id")
	if !ok {
		return
	}
	session, err := c.approveUC.Execute(ctx.Request.Context(), tenantID, id, userID)
	if err != nil {
		mapCashError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, response.NewCashSessionResponse(session))
}

// --- helpers ---

func tenantFromHeader(ctx *gin.Context) (uuid.UUID, bool) {
	raw := ctx.GetHeader("X-Tenant-ID")
	if raw == "" {
		httpresp.JSON(ctx, http.StatusBadRequest, "X-Tenant-ID header is required")
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		httpresp.JSON(ctx, http.StatusBadRequest, "invalid X-Tenant-ID format")
		return uuid.Nil, false
	}
	return id, true
}

// userIDFromContext extrae el user_id del JWT (claims dejados por TenantValidation),
// NUNCA del body. Fail-closed: sin claims válidos ⇒ 401.
func userIDFromContext(ctx *gin.Context) (uuid.UUID, bool) {
	v, exists := ctx.Get("jwt_claims")
	if !exists {
		httpresp.JSON(ctx, http.StatusUnauthorized, "missing authentication claims")
		return uuid.Nil, false
	}
	claims, ok := v.(jwt.MapClaims)
	if !ok {
		httpresp.JSON(ctx, http.StatusUnauthorized, "invalid authentication claims")
		return uuid.Nil, false
	}
	uidStr, _ := claims["user_id"].(string)
	id, err := uuid.Parse(uidStr)
	if err != nil {
		httpresp.JSON(ctx, http.StatusUnauthorized, "invalid user_id in claims")
		return uuid.Nil, false
	}
	return id, true
}

func pathUUID(ctx *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(ctx.Param(name))
	if err != nil {
		httpresp.JSON(ctx, http.StatusBadRequest, "invalid "+name+" format")
		return uuid.Nil, false
	}
	return id, true
}

func mapCashError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, entity.ErrCashSessionNotFound):
		httpresp.JSON(ctx, http.StatusNotFound, "cash register session not found")
	case errors.Is(err, entity.ErrCashSessionAlreadyOpen):
		httpresp.JSON(ctx, http.StatusConflict, err.Error())
	case errors.Is(err, entity.ErrCashSessionNotOpen),
		errors.Is(err, entity.ErrCashSessionNotInReview),
		errors.Is(err, entity.ErrCashSessionConflict),
		errors.Is(err, entity.ErrInvalidCashStatus):
		httpresp.JSON(ctx, http.StatusConflict, err.Error())
	case errors.Is(err, entity.ErrApproverIsOperator):
		httpresp.JSON(ctx, http.StatusForbidden, err.Error())
	case errors.Is(err, entity.ErrCashMethodUnresolved):
		// Misconfiguración del servicio (no del cliente): no se puede arquear.
		httpresp.JSON(ctx, http.StatusServiceUnavailable, err.Error())
	case errors.Is(err, entity.ErrPointOfSaleRequired),
		errors.Is(err, entity.ErrOpenedByRequired),
		errors.Is(err, entity.ErrOpeningAmountNegative),
		errors.Is(err, entity.ErrInvalidMovementType),
		errors.Is(err, entity.ErrMovementAmountInvalid),
		errors.Is(err, entity.ErrMovementReasonRequired),
		errors.Is(err, entity.ErrCountedAmountNegative),
		errors.Is(err, entity.ErrTenantIDRequired):
		httpresp.JSON(ctx, http.StatusBadRequest, err.Error())
	default:
		log.Printf("cash session error: %v", err)
		httpresp.JSON(ctx, http.StatusInternalServerError, "internal error")
	}
}

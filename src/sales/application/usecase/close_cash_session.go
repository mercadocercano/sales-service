package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"sales/src/sales/domain/entity"
	"sales/src/sales/domain/port"
)

// CloseCashSessionUseCase cierra la caja con arqueo (flujo monolítico). El expected se
// calcula server-side dentro de la tx; si |difference| > umbral queda en PENDING_REVIEW.
type CloseCashSessionUseCase struct {
	repo           port.CashRegisterRepository
	paymentMethod  port.PaymentMethodPort
	eventPublisher port.EventPublisher
}

func NewCloseCashSessionUseCase(repo port.CashRegisterRepository, paymentMethod port.PaymentMethodPort, eventPublisher port.EventPublisher) *CloseCashSessionUseCase {
	return &CloseCashSessionUseCase{repo: repo, paymentMethod: paymentMethod, eventPublisher: eventPublisher}
}

// Execute cierra la caja. closedBy sale del JWT. counted es lo único que aporta el
// cliente; expected y difference se calculan/persisten server-side.
func (uc *CloseCashSessionUseCase) Execute(ctx context.Context, tenantID, sessionID, closedBy uuid.UUID, counted decimal.Decimal) (*entity.CashRegisterSession, error) {
	cashID := resolveCashPaymentMethodID(uc.paymentMethod)
	// Fail-closed (sign-off C1): sin método cash resoluble, el expected omitiría las
	// ventas en efectivo y el arqueo daría un faltante falso. No se cierra a ciegas.
	if cashID == uuid.Nil {
		return nil, entity.ErrCashMethodUnresolved
	}

	session, err := uc.repo.CloseSession(ctx, tenantID, sessionID, closedBy, cashID, counted)
	if err != nil {
		return nil, err
	}

	publishCashEvent(ctx, uc.eventPublisher, session, "sales.cash_register.closed")
	return session, nil
}

// ApproveCashReviewUseCase aprueba un cierre en PENDING_REVIEW. El control de rol
// (supervisor) lo hace el middleware; la regla aprobador≠operador la valida la entidad.
type ApproveCashReviewUseCase struct {
	repo           port.CashRegisterRepository
	eventPublisher port.EventPublisher
}

func NewApproveCashReviewUseCase(repo port.CashRegisterRepository, eventPublisher port.EventPublisher) *ApproveCashReviewUseCase {
	return &ApproveCashReviewUseCase{repo: repo, eventPublisher: eventPublisher}
}

func (uc *ApproveCashReviewUseCase) Execute(ctx context.Context, tenantID, sessionID, approverID uuid.UUID) (*entity.CashRegisterSession, error) {
	session, err := uc.repo.ApproveReview(ctx, tenantID, sessionID, approverID)
	if err != nil {
		return nil, err
	}

	publishCashEvent(ctx, uc.eventPublisher, session, "sales.cash_register.review_approved")
	return session, nil
}

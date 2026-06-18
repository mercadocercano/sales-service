package usecase

import (
	"context"

	"github.com/google/uuid"

	"sales/src/sales/application/request"
	"sales/src/sales/domain/entity"
	"sales/src/sales/domain/port"
)

// OpenCashSessionUseCase abre una caja para una terminal (ADR-003 §3).
type OpenCashSessionUseCase struct {
	repo           port.CashRegisterRepository
	config         port.CashConfigProvider
	eventPublisher port.EventPublisher
}

func NewOpenCashSessionUseCase(repo port.CashRegisterRepository, config port.CashConfigProvider, eventPublisher port.EventPublisher) *OpenCashSessionUseCase {
	return &OpenCashSessionUseCase{repo: repo, config: config, eventPublisher: eventPublisher}
}

// Execute abre la caja. openedBy sale del JWT (no del body). El umbral de descuadre se
// snapshotea en la sesión al abrir. Si ya hay una caja abierta para la terminal, el
// repositorio devuelve entity.ErrCashSessionAlreadyOpen.
func (uc *OpenCashSessionUseCase) Execute(ctx context.Context, tenantID, openedBy uuid.UUID, req request.OpenCashSessionRequest) (*entity.CashRegisterSession, error) {
	threshold := uc.config.GetReviewThreshold(ctx, tenantID)

	session, err := entity.NewCashRegisterSession(tenantID, req.PointOfSaleID, openedBy, req.OpeningAmount, threshold)
	if err != nil {
		return nil, err
	}
	if err := uc.repo.OpenSession(ctx, session); err != nil {
		return nil, err
	}

	publishCashEvent(ctx, uc.eventPublisher, session, "sales.cash_register.opened")
	return session, nil
}

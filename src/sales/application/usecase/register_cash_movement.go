package usecase

import (
	"context"

	"github.com/google/uuid"

	"sales/src/sales/application/request"
	"sales/src/sales/domain/entity"
	"sales/src/sales/domain/port"
	"sales/src/sales/domain/value_object"
)

// RegisterCashMovementUseCase registra un movimiento manual de efectivo (append-only).
type RegisterCashMovementUseCase struct {
	repo           port.CashRegisterRepository
	eventPublisher port.EventPublisher
}

func NewRegisterCashMovementUseCase(repo port.CashRegisterRepository, eventPublisher port.EventPublisher) *RegisterCashMovementUseCase {
	return &RegisterCashMovementUseCase{repo: repo, eventPublisher: eventPublisher}
}

// Execute valida y registra el movimiento. createdBy sale del JWT. El repositorio
// bloquea la sesión y revalida que esté abierta dentro de la transacción.
func (uc *RegisterCashMovementUseCase) Execute(ctx context.Context, tenantID, sessionID, createdBy uuid.UUID, req request.CashMovementRequest) (*entity.CashRegisterSession, error) {
	mv, err := entity.NewCashMovement(
		tenantID, sessionID,
		value_object.CashMovementType(req.Type),
		req.Amount, req.Reason, createdBy,
	)
	if err != nil {
		return nil, err
	}

	session, err := uc.repo.RegisterMovement(ctx, mv)
	if err != nil {
		return nil, err
	}

	publishCashEvent(ctx, uc.eventPublisher, session, "sales.cash_register.movement_registered")
	return session, nil
}

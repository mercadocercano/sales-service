package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"sales/src/sales/domain/entity"
	"sales/src/sales/domain/port"
)

// GetCurrentCashSessionUseCase devuelve la caja abierta de una terminal.
type GetCurrentCashSessionUseCase struct {
	repo port.CashRegisterRepository
}

func NewGetCurrentCashSessionUseCase(repo port.CashRegisterRepository) *GetCurrentCashSessionUseCase {
	return &GetCurrentCashSessionUseCase{repo: repo}
}

func (uc *GetCurrentCashSessionUseCase) Execute(ctx context.Context, tenantID uuid.UUID, pointOfSaleID string) (*entity.CashRegisterSession, error) {
	if pointOfSaleID == "" {
		return nil, entity.ErrPointOfSaleRequired
	}
	return uc.repo.GetOpenByTerminal(ctx, tenantID, pointOfSaleID)
}

// GetCashSessionUseCase devuelve el detalle de una caja + su arqueo. Para una caja
// abierta calcula el expected en vivo; para una cerrada/en revisión usa el snapshot.
type GetCashSessionUseCase struct {
	repo          port.CashRegisterRepository
	paymentMethod port.PaymentMethodPort
}

func NewGetCashSessionUseCase(repo port.CashRegisterRepository, paymentMethod port.PaymentMethodPort) *GetCashSessionUseCase {
	return &GetCashSessionUseCase{repo: repo, paymentMethod: paymentMethod}
}

// Execute devuelve la sesión y el expected vigente. Si la caja está cerrada/en revisión,
// expected es el snapshot persistido; si está abierta, se calcula server-side.
func (uc *GetCashSessionUseCase) Execute(ctx context.Context, tenantID, id uuid.UUID) (*entity.CashRegisterSession, decimal.Decimal, error) {
	session, err := uc.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, decimal.Zero, err
	}

	if session.ExpectedAmount != nil {
		return session, *session.ExpectedAmount, nil
	}

	cashID := resolveCashPaymentMethodID(uc.paymentMethod)
	expected, err := uc.repo.ComputeExpected(ctx, session, cashID)
	if err != nil {
		return nil, decimal.Zero, err
	}
	return session, expected, nil
}

package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"sales/src/sales/domain/value_object"
)

// CashMovement es un movimiento MANUAL de efectivo dentro de una sesión de caja.
// Append-only: no se actualiza ni borra; las correcciones son contra-asientos.
type CashMovement struct {
	ID            uuid.UUID
	TenantID      uuid.UUID
	SessionID     uuid.UUID
	Type          value_object.CashMovementType
	Amount        decimal.Decimal
	Reason        string
	CreatedBy     uuid.UUID
	CreatedAt     time.Time
}

// NewCashMovement valida y crea un movimiento manual. El monto debe ser positivo; el
// signo en el arqueo lo determina el Type.
func NewCashMovement(
	tenantID, sessionID uuid.UUID,
	movementType value_object.CashMovementType,
	amount decimal.Decimal,
	reason string,
	createdBy uuid.UUID,
) (*CashMovement, error) {
	if tenantID == uuid.Nil {
		return nil, ErrTenantIDRequired
	}
	if !movementType.IsValid() {
		return nil, ErrInvalidMovementType
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, ErrMovementAmountInvalid
	}
	if reason == "" {
		return nil, ErrMovementReasonRequired
	}
	if createdBy == uuid.Nil {
		return nil, ErrOpenedByRequired
	}

	return &CashMovement{
		ID:        uuid.New(),
		TenantID:  tenantID,
		SessionID: sessionID,
		Type:      movementType,
		Amount:    amount,
		Reason:    reason,
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
	}, nil
}

package entity

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Acciones auditables sobre efectivo (ADR-003 gate #5).
const (
	CashAuditOpened             = "opened"
	CashAuditMovementRegistered = "movement_registered"
	CashAuditClosed             = "closed"
	CashAuditReviewApproved     = "review_approved"
)

// CashAuditEntry es un asiento inmutable de la bitácora de efectivo. El user_id es
// SIEMPRE el del JWT (el actor), nunca un valor del body. Los punteros van nil cuando
// no aplican a la acción.
type CashAuditEntry struct {
	TenantID       uuid.UUID
	SessionID      uuid.UUID
	Action         string
	UserID         uuid.UUID
	PointOfSaleID  string
	OpeningAmount  *decimal.Decimal
	ExpectedAmount *decimal.Decimal
	CountedAmount  *decimal.Decimal
	Difference     *decimal.Decimal
	MovementType   string
	MovementAmount *decimal.Decimal
	Detail         string
}

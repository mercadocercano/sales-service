package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"sales/src/sales/domain/value_object"
)

// CashRegisterSession es la raíz de agregado de la caja POS (ADR-003).
// Maneja efectivo real (L4). La lógica de transición y de arqueo vive acá (en Go),
// no en triggers de DB. El cálculo del expected (que requiere agregar pos_sales y
// movimientos) lo hace el repositorio y se lo pasa a Close().
type CashRegisterSession struct {
	ID             uuid.UUID
	TenantID       uuid.UUID
	PointOfSaleID  string
	Status         value_object.CashSessionStatus
	OpenedBy       uuid.UUID
	OpeningAmount  decimal.Decimal
	OpenedAt       time.Time
	ReviewThreshold decimal.Decimal

	// Snapshot de cierre (nil hasta cerrar)
	ClosedBy       *uuid.UUID
	CountedAmount  *decimal.Decimal
	ExpectedAmount *decimal.Decimal
	Difference     *decimal.Decimal
	ClosedAt       *time.Time

	// Aprobación de descuadre (nil salvo que haya pasado por PENDING_REVIEW)
	ApprovedBy *uuid.UUID
	ApprovedAt *time.Time

	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewCashRegisterSession abre una nueva sesión de caja (estado open).
func NewCashRegisterSession(
	tenantID uuid.UUID,
	pointOfSaleID string,
	openedBy uuid.UUID,
	openingAmount decimal.Decimal,
	reviewThreshold decimal.Decimal,
) (*CashRegisterSession, error) {
	if tenantID == uuid.Nil {
		return nil, ErrTenantIDRequired
	}
	if pointOfSaleID == "" {
		return nil, ErrPointOfSaleRequired
	}
	if openedBy == uuid.Nil {
		return nil, ErrOpenedByRequired
	}
	if openingAmount.LessThan(decimal.Zero) {
		return nil, ErrOpeningAmountNegative
	}
	if reviewThreshold.LessThan(decimal.Zero) {
		reviewThreshold = decimal.Zero
	}

	now := time.Now()
	return &CashRegisterSession{
		ID:              uuid.New(),
		TenantID:        tenantID,
		PointOfSaleID:   pointOfSaleID,
		Status:          value_object.CashStatusOpen,
		OpenedBy:        openedBy,
		OpeningAmount:   openingAmount,
		OpenedAt:        now,
		ReviewThreshold: reviewThreshold,
		Version:         1,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

// Close ejecuta el cierre con arqueo (flujo monolítico, ADR-003 + decisión owner).
// El expected ya viene calculado server-side por el repositorio. Calcula la diferencia
// y decide el estado final: si |difference| <= umbral -> closed; si no -> pending_review
// (requiere aprobación de un supervisor distinto del operador).
func (s *CashRegisterSession) Close(closedBy uuid.UUID, counted, expected decimal.Decimal) error {
	if !s.Status.IsOpen() {
		return ErrCashSessionNotOpen
	}
	if closedBy == uuid.Nil {
		return ErrOpenedByRequired
	}
	if counted.LessThan(decimal.Zero) {
		return ErrCountedAmountNegative
	}

	difference := counted.Sub(expected)
	now := time.Now()

	s.ClosedBy = &closedBy
	s.CountedAmount = &counted
	s.ExpectedAmount = &expected
	s.Difference = &difference
	s.ClosedAt = &now
	s.UpdatedAt = now

	// |difference| <= umbral -> cierre OK; si no, queda pendiente de revisión.
	if difference.Abs().LessThanOrEqual(s.ReviewThreshold) {
		s.Status = value_object.CashStatusClosed
	} else {
		s.Status = value_object.CashStatusPendingReview
	}
	return nil
}

// ApproveReview aprueba un cierre con descuadre (PENDING_REVIEW -> closed).
// Separación de funciones (ADR-003 gate #2): el aprobador NO puede ser quien abrió ni
// quien cerró la caja. El control de rol (supervisor) lo hace el middleware; esta es la
// regla de negocio transaccional.
func (s *CashRegisterSession) ApproveReview(approverID uuid.UUID) error {
	if s.Status != value_object.CashStatusPendingReview {
		return ErrCashSessionNotInReview
	}
	if approverID == uuid.Nil {
		return ErrOpenedByRequired
	}
	if approverID == s.OpenedBy {
		return ErrApproverIsOperator
	}
	if s.ClosedBy != nil && approverID == *s.ClosedBy {
		return ErrApproverIsOperator
	}

	now := time.Now()
	s.ApprovedBy = &approverID
	s.ApprovedAt = &now
	s.Status = value_object.CashStatusClosed
	s.UpdatedAt = now
	return nil
}

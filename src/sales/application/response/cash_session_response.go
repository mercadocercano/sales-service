package response

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"sales/src/sales/domain/entity"
)

// CashSessionResponse es la vista pública de una sesión de caja + su arqueo.
type CashSessionResponse struct {
	ID             uuid.UUID        `json:"id"`
	TenantID       uuid.UUID        `json:"tenant_id"`
	PointOfSaleID  string           `json:"point_of_sale_id"`
	Status         string           `json:"status"`
	OpenedBy       uuid.UUID        `json:"opened_by"`
	OpeningAmount  decimal.Decimal  `json:"opening_amount"`
	OpenedAt       time.Time        `json:"opened_at"`
	ReviewThreshold decimal.Decimal `json:"review_threshold"`

	// Arqueo. Para una caja abierta, ExpectedAmount es el cálculo en vivo; para una
	// cerrada/en revisión, es el snapshot persistido. Difference = counted − expected.
	ExpectedAmount *decimal.Decimal `json:"expected_amount,omitempty"`
	CountedAmount  *decimal.Decimal `json:"counted_amount,omitempty"`
	Difference     *decimal.Decimal `json:"difference,omitempty"`

	ClosedBy   *uuid.UUID `json:"closed_by,omitempty"`
	ClosedAt   *time.Time `json:"closed_at,omitempty"`
	ApprovedBy *uuid.UUID `json:"approved_by,omitempty"`
	ApprovedAt *time.Time `json:"approved_at,omitempty"`
	Version    int        `json:"version"`
}

// NewCashSessionResponse mapea una sesión usando su snapshot persistido (expected, etc.).
func NewCashSessionResponse(s *entity.CashRegisterSession) *CashSessionResponse {
	return &CashSessionResponse{
		ID:              s.ID,
		TenantID:        s.TenantID,
		PointOfSaleID:   s.PointOfSaleID,
		Status:          s.Status.String(),
		OpenedBy:        s.OpenedBy,
		OpeningAmount:   s.OpeningAmount,
		OpenedAt:        s.OpenedAt,
		ReviewThreshold: s.ReviewThreshold,
		ExpectedAmount:  s.ExpectedAmount,
		CountedAmount:   s.CountedAmount,
		Difference:      s.Difference,
		ClosedBy:        s.ClosedBy,
		ClosedAt:        s.ClosedAt,
		ApprovedBy:      s.ApprovedBy,
		ApprovedAt:      s.ApprovedAt,
		Version:         s.Version,
	}
}

// NewCashSessionDetailResponse mapea una sesión forzando el expected en vivo (vista de
// detalle de una caja abierta, donde el snapshot aún no existe).
func NewCashSessionDetailResponse(s *entity.CashRegisterSession, expected decimal.Decimal) *CashSessionResponse {
	resp := NewCashSessionResponse(s)
	if s.ExpectedAmount == nil {
		exp := expected
		resp.ExpectedAmount = &exp
	}
	return resp
}

package request

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// POSSaleItemRequest representa un item dentro de una venta POS
// HITO B - Multi-item support
type POSSaleItemRequest struct {
	SKU       string          `json:"sku" binding:"required"`
	Quantity  int             `json:"quantity" binding:"required,gt=0"`
	UnitPrice decimal.Decimal `json:"unit_price" binding:"required"`
}

// POSSaleRequest request para venta directa POS multi-item
// customer_id es OPCIONAL — nil = venta anónima (Consumidor Final)
// payment_method_id acepta UUID ("uuid-...") o código ("cash", "efectivo")
type POSSaleRequest struct {
	CustomerID      *uuid.UUID           `json:"customer_id,omitempty"`
	Items           []POSSaleItemRequest `json:"items" binding:"required,min=1,dive"`
	PaymentMethodID string               `json:"payment_method_id" binding:"required"` // UUID o código
	DiscountAmount  decimal.Decimal      `json:"discount_amount,omitempty"`
	AmountPaid      decimal.Decimal      `json:"amount_paid" binding:"required"`
	Currency        string               `json:"currency,omitempty"`
	Notes           string               `json:"notes,omitempty"`
	// E18 Tramo A: caja a la que asociar la venta (modo DEGRADADO, ADR-003 §4). Opcional:
	// si no viene, o la sesión no está abierta, la venta procede con FK NULL (no se bloquea).
	CashRegisterSessionID *uuid.UUID `json:"cash_register_session_id,omitempty"`
}

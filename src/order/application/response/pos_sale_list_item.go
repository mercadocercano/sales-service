package response

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PosSaleListItem representa un item de la lista de ventas POS
// HITO B - Actualizado para multi-item
type PosSaleListItem struct {
	ID              uuid.UUID       `json:"id"`
	CustomerID      *uuid.UUID      `json:"customer_id,omitempty"`
	PaymentMethodID uuid.UUID       `json:"payment_method_id"`
	TotalAmount     decimal.Decimal `json:"total_amount"`     // Suma de subtotales
	DiscountAmount  decimal.Decimal `json:"discount_amount"`  // Descuento aplicado
	FinalAmount     decimal.Decimal `json:"final_amount"`     // Total - descuento
	Currency        string          `json:"currency"`
	TotalItems      int             `json:"total_items"`      // Cantidad de items
	CreatedAt       time.Time       `json:"created_at"`
}

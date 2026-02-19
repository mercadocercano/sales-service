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
	UnitPrice decimal.Decimal `json:"unit_price" binding:"required"` // Precio unitario
}

// POSSaleRequest request para venta directa POS multi-item
// HITO B - Refactorizado para multi-item + descuentos
// HITO: POST /pos/sale devuelve DTO listo para imprimir
type POSSaleRequest struct {
	Items           []POSSaleItemRequest `json:"items" binding:"required,min=1,dive"` // Mínimo 1 item
	CustomerID      *uuid.UUID           `json:"customer_id"`                         // Opcional (NULL = consumidor final)
	PaymentMethodID uuid.UUID            `json:"payment_method_id" binding:"required"`
	DiscountAmount  decimal.Decimal      `json:"discount_amount,omitempty"` // Descuento fijo (default: 0)
	AmountPaid      decimal.Decimal      `json:"amount_paid" binding:"required"`      // Monto pagado por el cliente
	Currency        string               `json:"currency,omitempty"`                  // Default: "ARS"
	Notes           string               `json:"notes,omitempty"`
}

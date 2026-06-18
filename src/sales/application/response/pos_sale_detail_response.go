package response

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// POSSaleDetailItemResponse representa un ítem en el detalle del comprobante POS.
// E18 Tramo B.
type POSSaleDetailItemResponse struct {
	SKU         string          `json:"sku"`
	ProductName string          `json:"product_name"`
	Quantity    int             `json:"quantity"`
	UnitPrice   decimal.Decimal `json:"unit_price"`
	Subtotal    decimal.Decimal `json:"subtotal"`
}

// POSSaleDetailResponse es el detalle completo de una venta POS para el comprobante.
// E18 Tramo B: incluye número de comprobante INTERNO (no fiscal/AFIP).
//
// Contrato: usa la clave "items" (no campos legacy) para alinearse con el resto
// del ecosistema. sale_number puede ser null para ventas previas a la migración 018.
type POSSaleDetailResponse struct {
	PosSaleID         uuid.UUID                   `json:"pos_sale_id"`
	SaleNumber        *int                        `json:"sale_number"` // comprobante INTERNO; null si la venta es previa a la numeración
	Items             []POSSaleDetailItemResponse `json:"items"`
	TotalItems        int                         `json:"total_items"`
	SubtotalAmount    decimal.Decimal             `json:"subtotal_amount"` // suma de subtotales (antes de descuento)
	DiscountAmount    decimal.Decimal             `json:"discount_amount"`
	FinalAmount       decimal.Decimal             `json:"final_amount"`
	AmountPaid        decimal.Decimal             `json:"amount_paid"`
	Change            decimal.Decimal             `json:"change"`
	Currency          string                      `json:"currency"`
	PaymentMethodID   uuid.UUID                   `json:"payment_method_id"`
	PaymentMethodName string                      `json:"payment_method_name"`
	CustomerID        uuid.UUID                   `json:"customer_id"`   // uuid.Nil = Consumidor Final
	CustomerName      string                      `json:"customer_name"` // "Consumidor Final" si Nil; "Cliente" si no se pudo resolver
	CreatedAt         time.Time                   `json:"created_at"`
}

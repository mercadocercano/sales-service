package response

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// POSSaleItemResponse representa un item en la respuesta de venta POS
// HITO B - Multi-item support
type POSSaleItemResponse struct {
	ItemID         uuid.UUID       `json:"item_id"`
	SKU            string          `json:"sku"`
	ProductName    string          `json:"product_name"`
	Quantity       int             `json:"quantity"`
	UnitPrice      decimal.Decimal `json:"unit_price"`
	Subtotal       decimal.Decimal `json:"subtotal"`
	StockEntryID   uuid.UUID       `json:"stock_entry_id"`
}

// POSSaleResponse respuesta de venta directa POS multi-item
// HITO B - Refactorizado para multi-item + descuentos
// HITO: POST /pos/sale devuelve DTO listo para imprimir
type POSSaleResponse struct {
	PosSaleID         uuid.UUID              `json:"pos_sale_id"`
	SaleNumber        string                 `json:"sale_number"`       // UUID como número de venta
	Items             []POSSaleItemResponse  `json:"items"`
	TotalItems        int                    `json:"total_items"`
	SubtotalAmount    decimal.Decimal        `json:"subtotal_amount"`   // Suma de subtotales (antes: total_amount)
	DiscountAmount    decimal.Decimal        `json:"discount_amount"`   // Descuento aplicado
	FinalAmount       decimal.Decimal        `json:"final_amount"`      // Total - descuento
	PaymentMethodID   uuid.UUID              `json:"payment_method_id"`
	PaymentMethodName string                 `json:"payment_method_name"` // Nombre legible del método
	AmountPaid        decimal.Decimal        `json:"amount_paid"`       // Monto pagado
	Change            decimal.Decimal        `json:"change"`            // Vuelto
	Currency          string                 `json:"currency"`
	CustomerID        *uuid.UUID             `json:"customer_id,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
}

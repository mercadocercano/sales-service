package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PosSale representa una venta POS (Aggregate Root)
// HITO B - Refactorizado para soportar multi-item + descuentos
// HITO: POST /pos/sale devuelve DTO listo para imprimir
type PosSale struct {
	ID              uuid.UUID       `json:"id"`
	TenantID        uuid.UUID       `json:"tenant_id"`
	CustomerID      *uuid.UUID      `json:"customer_id"`       // NULL = consumidor final
	PaymentMethodID uuid.UUID       `json:"payment_method_id"` // Obligatorio
	TotalAmount     decimal.Decimal `json:"total_amount"`      // Suma de subtotales
	DiscountAmount  decimal.Decimal `json:"discount_amount"`   // Descuento fijo
	FinalAmount     decimal.Decimal `json:"final_amount"`      // total - discount
	AmountPaid      decimal.Decimal `json:"amount_paid"`       // Monto pagado por el cliente
	Change          decimal.Decimal `json:"change"`            // Vuelto (amount_paid - final_amount)
	Currency        string          `json:"currency"`
	CreatedAt       time.Time       `json:"created_at"`
	Items           []PosSaleItem   `json:"items"` // DDD: Collection of entities
}

// NewPosSale crea una nueva venta POS con múltiples items (DDD Aggregate Root)
// HITO B - Constructor multi-item
// HITO: POST /pos/sale devuelve DTO listo para imprimir
func NewPosSale(
	tenantID uuid.UUID,
	customerID *uuid.UUID,
	paymentMethodID uuid.UUID,
	items []PosSaleItem,
	discountAmount decimal.Decimal,
	amountPaid decimal.Decimal,
	currency string,
) (*PosSale, error) {
	// Validaciones básicas
	if tenantID == uuid.Nil {
		return nil, ErrTenantIDRequired
	}
	if paymentMethodID == uuid.Nil {
		return nil, ErrTenantIDRequired // TODO: Crear ErrPaymentMethodRequired
	}
	if len(items) == 0 {
		return nil, ErrPosSaleMustHaveItems
	}
	if discountAmount.LessThan(decimal.Zero) {
		return nil, ErrInvalidDiscount
	}

	// Default currency
	if currency == "" {
		currency = "ARS"
	}

	// Calcular total_amount (suma de subtotales)
	totalAmount := decimal.Zero
	for _, item := range items {
		totalAmount = totalAmount.Add(item.Subtotal)
	}

	// Calcular final_amount
	finalAmount := totalAmount.Sub(discountAmount)
	if finalAmount.LessThan(decimal.Zero) {
		finalAmount = decimal.Zero // Descuento no puede generar monto negativo
	}

	// HITO: Validar amount_paid >= final_amount
	if amountPaid.LessThan(finalAmount) {
		return nil, ErrInsufficientPayment
	}

	// HITO: Calcular change (vuelto)
	change := amountPaid.Sub(finalAmount)

	posSaleID := uuid.New()

	// Asignar pos_sale_id a todos los items
	for i := range items {
		items[i].PosSaleID = posSaleID
	}

	return &PosSale{
		ID:              posSaleID,
		TenantID:        tenantID,
		CustomerID:      customerID,
		PaymentMethodID: paymentMethodID,
		TotalAmount:     totalAmount,
		DiscountAmount:  discountAmount,
		FinalAmount:     finalAmount,
		AmountPaid:      amountPaid,
		Change:          change,
		Currency:        currency,
		CreatedAt:       time.Now(),
		Items:           items,
	}, nil
}

// TotalItems retorna el número total de items
func (ps *PosSale) TotalItems() int {
	return len(ps.Items)
}

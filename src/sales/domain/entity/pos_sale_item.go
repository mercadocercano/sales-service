package entity

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PosSaleItem representa un item dentro de una venta POS (Entity dentro del Aggregate)
// HITO B - Multi-item support
type PosSaleItem struct {
	ID           uuid.UUID       `json:"id"`
	PosSaleID    uuid.UUID       `json:"pos_sale_id"`
	SKU          string          `json:"sku"`
	ProductName  string          `json:"product_name"`
	Quantity     int             `json:"quantity"`
	UnitPrice    decimal.Decimal `json:"unit_price"`
	Subtotal     decimal.Decimal `json:"subtotal"`
	StockEntryID uuid.UUID       `json:"stock_entry_id"`
}

// NewPosSaleItem crea un nuevo item de venta POS
// Validaciones mínimas, cálculo de subtotal
func NewPosSaleItem(
	posSaleID uuid.UUID,
	sku string,
	productName string,
	quantity int,
	unitPrice decimal.Decimal,
	stockEntryID uuid.UUID,
) (*PosSaleItem, error) {
	// Validaciones básicas
	if sku == "" {
		return nil, ErrSKURequired
	}
	if productName == "" {
		return nil, ErrProductNameRequired
	}
	if quantity <= 0 {
		return nil, ErrInvalidQuantity
	}
	if unitPrice.LessThan(decimal.Zero) {
		return nil, ErrInvalidPrice
	}
	if stockEntryID == uuid.Nil {
		return nil, ErrStockEntryIDRequired
	}

	// Calcular subtotal
	subtotal := unitPrice.Mul(decimal.NewFromInt(int64(quantity)))

	return &PosSaleItem{
		ID:           uuid.New(),
		PosSaleID:    posSaleID,
		SKU:          sku,
		ProductName:  productName,
		Quantity:     quantity,
		UnitPrice:    unitPrice,
		Subtotal:     subtotal,
		StockEntryID: stockEntryID,
	}, nil
}

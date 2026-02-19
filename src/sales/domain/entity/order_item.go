package entity

import (
	"encoding/json"
	"github.com/google/uuid"
)

// OrderItem representa un item dentro de una orden (Entity dentro del Aggregate)
type OrderItem struct {
	ItemID          string          `json:"item_id"`
	OrderID         string          `json:"order_id"`
	SKU             string          `json:"sku"`
	Quantity        int             `json:"quantity"`
	ProductSnapshot json.RawMessage `json:"product_snapshot,omitempty"`
	VariantSnapshot json.RawMessage `json:"variant_snapshot,omitempty"`
}

// NewOrderItem crea un nuevo item de orden
func NewOrderItem(orderID, sku string, quantity int) (*OrderItem, error) {
	if sku == "" {
		return nil, ErrSKURequired
	}
	if quantity <= 0 {
		return nil, ErrInvalidQuantity
	}

	return &OrderItem{
		ItemID:   uuid.New().String(),
		OrderID:  orderID,
		SKU:      sku,
		Quantity: quantity,
	}, nil
}

// NewOrderItemWithSnapshots crea un item de orden con snapshots inmutables
func NewOrderItemWithSnapshots(orderID, sku string, quantity int, productSnapshot, variantSnapshot json.RawMessage) (*OrderItem, error) {
	if sku == "" {
		return nil, ErrSKURequired
	}
	if quantity <= 0 {
		return nil, ErrInvalidQuantity
	}

	return &OrderItem{
		ItemID:          uuid.New().String(),
		OrderID:         orderID,
		SKU:             sku,
		Quantity:        quantity,
		ProductSnapshot: productSnapshot,
		VariantSnapshot: variantSnapshot,
	}, nil
}

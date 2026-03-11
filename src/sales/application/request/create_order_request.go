package request

import "github.com/google/uuid"

// CreateOrderItemRequest representa un item dentro de una orden
type CreateOrderItemRequest struct {
	SKU       string  `json:"sku" binding:"required"`
	Quantity  int     `json:"quantity" binding:"required,min=1"`
	UnitPrice float64 `json:"unit_price" binding:"required,gt=0"`
}

// CreateOrderRequest representa la petición para crear una orden (multi-item)
// HITO v0.6: customer_id ahora es OBLIGATORIO (validado contra customer-service)
type CreateOrderRequest struct {
	CustomerID uuid.UUID                `json:"customer_id" binding:"required"` // HITO v0.6: Obligatorio
	Items      []CreateOrderItemRequest `json:"items" binding:"required,min=1,dive"`
	Reference  string                   `json:"reference,omitempty"`
}

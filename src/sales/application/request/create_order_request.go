package request

// CreateOrderItemRequest representa un item dentro de una orden
type CreateOrderItemRequest struct {
	SKU      string `json:"sku" binding:"required"`
	Quantity int    `json:"quantity" binding:"required,min=1"`
}

// CreateOrderRequest representa la petición para crear una orden (multi-item)
type CreateOrderRequest struct {
	Items     []CreateOrderItemRequest `json:"items" binding:"required,min=1,dive"`
	Reference string                   `json:"reference,omitempty"`
}

package response

// CreateOrderItemResponse representa un item en la respuesta
type CreateOrderItemResponse struct {
	ItemID   string `json:"item_id"`
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

// CreateOrderResponse representa la respuesta de creación de orden (multi-item)
type CreateOrderResponse struct {
	OrderID    string                    `json:"order_id"`
	Items      []CreateOrderItemResponse `json:"items"`
	TotalItems int                       `json:"total_items"`
	Status     string                    `json:"status"`
}

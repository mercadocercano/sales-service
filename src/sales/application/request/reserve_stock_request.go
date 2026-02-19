package request

// ReserveStockItem representa un item a reservar
type ReserveStockItem struct {
	SKU      string `json:"sku" binding:"required"`
	Quantity int    `json:"quantity" binding:"required,min=1"`
}

// ReserveStockRequest representa la petición para reservar stock (multi-item)
type ReserveStockRequest struct {
	Items []ReserveStockItem `json:"items" binding:"required,min=1,dive"`
}

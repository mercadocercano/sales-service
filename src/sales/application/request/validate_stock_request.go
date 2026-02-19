package request

// ValidateStockItem representa un item a validar
type ValidateStockItem struct {
	SKU      string `json:"sku" binding:"required"`
	Quantity int    `json:"quantity" binding:"required,min=1"`
}

// ValidateStockRequest representa la petición para validar stock (multi-item)
type ValidateStockRequest struct {
	Items []ValidateStockItem `json:"items" binding:"required,min=1,dive"`
}

package request

// ReleaseStockRequest representa la petición para liberar stock reservado
type ReleaseStockRequest struct {
	SKU       string `json:"sku" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required,min=1"`
	Reference string `json:"reference" binding:"required"`
}

package response

// ReleaseStockResponse representa la respuesta de liberación de stock
type ReleaseStockResponse struct {
	Released  bool   `json:"released"`
	SKU       string `json:"sku"`
	Quantity  int    `json:"quantity"`
	Reference string `json:"reference"`
}

package response

// ReserveStockItemResponse representa la respuesta de un item reservado
type ReserveStockItemResponse struct {
	SKU       string `json:"sku"`
	Quantity  int    `json:"quantity"`
	Reference string `json:"reference"`
}

// ReserveStockResponse representa la respuesta de reserva de stock (multi-item)
type ReserveStockResponse struct {
	Reserved bool                       `json:"reserved"`
	Items    []ReserveStockItemResponse `json:"items"`
}

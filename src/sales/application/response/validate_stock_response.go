package response

// ValidateStockItemResponse representa la respuesta de validación de un item
type ValidateStockItemResponse struct {
	SKU          string `json:"sku"`
	RequestedQty int    `json:"requested_qty"`
	Available    bool   `json:"available"`
	AvailableQty int    `json:"available_qty"`
}

// ValidateStockResponse representa la respuesta completa de validación
type ValidateStockResponse struct {
	Valid bool                        `json:"valid"`
	Items []ValidateStockItemResponse `json:"items"`
}

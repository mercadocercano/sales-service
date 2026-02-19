package response

import "encoding/json"

// GetOrderResponse representa la respuesta de obtención de una orden
type GetOrderResponse struct {
	OrderID   string              `json:"order_id"`
	TenantID  string              `json:"tenant_id"`
	Status    string              `json:"status"`
	CreatedAt string              `json:"created_at"`
	Items     []OrderItemResponse `json:"items"`
}

// OrderItemResponse representa un item dentro de la orden
type OrderItemResponse struct {
	ItemID          string          `json:"item_id"`
	SKU             string          `json:"sku"`
	Quantity        int             `json:"quantity"`
	ProductSnapshot json.RawMessage `json:"product_snapshot,omitempty"`
	VariantSnapshot json.RawMessage `json:"variant_snapshot,omitempty"`
}

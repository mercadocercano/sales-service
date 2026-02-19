package request

// ConfirmOrderRequest representa la petición para confirmar una orden
type ConfirmOrderRequest struct {
	Reference string `json:"reference" binding:"required"`
}

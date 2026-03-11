package response

// ApplyCreditResponse representa la respuesta de aplicar crédito a una orden
type ApplyCreditResponse struct {
	ApplicationID string  `json:"application_id"`
	Amount        float64 `json:"amount"`
}

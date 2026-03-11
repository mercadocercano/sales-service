package request

// ApplyCreditRequest representa la solicitud de aplicar crédito a una orden
type ApplyCreditRequest struct {
	CustomerCreditID string  `json:"customer_credit_id" binding:"required"`
	Amount           float64 `json:"amount" binding:"required,gt=0"`
}

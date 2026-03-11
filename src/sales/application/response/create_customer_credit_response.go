package response

// CreateCustomerCreditResponse representa la respuesta de crear crédito a cuenta
type CreateCustomerCreditResponse struct {
	CreditID string  `json:"credit_id"`
	Amount   float64 `json:"amount"`
	Status   string  `json:"status"`
}

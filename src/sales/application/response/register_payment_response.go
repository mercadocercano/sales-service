package response

// RegisterPaymentResponse representa la respuesta del registro de pago
type RegisterPaymentResponse struct {
	PaymentID string  `json:"payment_id"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
}

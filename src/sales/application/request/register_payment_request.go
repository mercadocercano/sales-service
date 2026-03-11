package request

// RegisterPaymentRequest representa la solicitud de registro de pago
type RegisterPaymentRequest struct {
	Amount float64 `json:"amount" binding:"required,gt=0"`
}

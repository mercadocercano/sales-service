package request

// CreateCustomerCreditRequest representa la solicitud de crear crédito a cuenta del cliente
type CreateCustomerCreditRequest struct {
	Amount float64 `json:"amount" binding:"required,gt=0"`
}

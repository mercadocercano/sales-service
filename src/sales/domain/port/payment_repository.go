package port

import "context"

// PaymentRegistration contiene el resultado de registrar un pago
type PaymentRegistration struct {
	PaymentID   string
	OrderStatus string
	CustomerID  string
}

// PaymentRepository define las operaciones de persistencia de pagos
type PaymentRepository interface {
	RegisterPayment(ctx context.Context, tenantID, orderID string, amount float64) (*PaymentRegistration, error)
}

package port

import "github.com/google/uuid"

// PaymentMethodInfo contiene la información de un método de pago
type PaymentMethodInfo struct {
	ID   uuid.UUID
	Code string
	Name string
}

// PaymentMethodPort define las operaciones de consulta de métodos de pago
type PaymentMethodPort interface {
	Get(id uuid.UUID) (PaymentMethodInfo, bool)
	GetByCode(code string) (PaymentMethodInfo, bool)
	GetName(id uuid.UUID) string
}

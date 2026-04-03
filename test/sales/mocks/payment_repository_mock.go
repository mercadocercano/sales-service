package mocks

import (
	"context"

	"sales/src/sales/domain/port"

	"github.com/stretchr/testify/mock"
)

// MockPaymentRepository es un mock de port.PaymentRepository
type MockPaymentRepository struct {
	mock.Mock
}

func (m *MockPaymentRepository) RegisterPayment(ctx context.Context, tenantID, orderID string, amount float64) (*port.PaymentRegistration, error) {
	args := m.Called(ctx, tenantID, orderID, amount)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*port.PaymentRegistration), args.Error(1)
}

package mocks

import (
	"sales/src/sales/domain/port"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockPaymentMethodPort es un mock de port.PaymentMethodPort
type MockPaymentMethodPort struct {
	mock.Mock
}

func (m *MockPaymentMethodPort) Get(id uuid.UUID) (port.PaymentMethodInfo, bool) {
	args := m.Called(id)
	return args.Get(0).(port.PaymentMethodInfo), args.Bool(1)
}

func (m *MockPaymentMethodPort) GetByCode(code string) (port.PaymentMethodInfo, bool) {
	args := m.Called(code)
	return args.Get(0).(port.PaymentMethodInfo), args.Bool(1)
}

func (m *MockPaymentMethodPort) GetName(id uuid.UUID) string {
	args := m.Called(id)
	return args.String(0)
}

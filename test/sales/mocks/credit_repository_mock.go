package mocks

import (
	"context"

	"sales/src/sales/domain/port"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
)

// MockCreditRepository es un mock de port.CreditRepository
type MockCreditRepository struct {
	mock.Mock
}

func (m *MockCreditRepository) CreateCredit(ctx context.Context, tenantID, customerID string, amount decimal.Decimal) (string, error) {
	args := m.Called(ctx, tenantID, customerID, amount)
	return args.String(0), args.Error(1)
}

func (m *MockCreditRepository) ApplyCredit(ctx context.Context, tenantID, creditID, orderID string, amount decimal.Decimal) (*port.CreditApplication, error) {
	args := m.Called(ctx, tenantID, creditID, orderID, amount)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*port.CreditApplication), args.Error(1)
}

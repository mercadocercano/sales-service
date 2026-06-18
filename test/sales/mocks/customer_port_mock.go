package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockCustomerPort es un mock de port.CustomerPort
type MockCustomerPort struct {
	mock.Mock
}

func (m *MockCustomerPort) Exists(ctx context.Context, tenantID uuid.UUID, customerID uuid.UUID) (bool, error) {
	args := m.Called(ctx, tenantID, customerID)
	return args.Bool(0), args.Error(1)
}

func (m *MockCustomerPort) GetName(ctx context.Context, tenantID uuid.UUID, customerID uuid.UUID) (string, error) {
	args := m.Called(ctx, tenantID, customerID)
	return args.String(0), args.Error(1)
}

package mocks

import (
	"context"

	"sales/src/sales/domain/port"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockTenantPort es un mock de port.TenantPort
type MockTenantPort struct {
	mock.Mock
}

func (m *MockTenantPort) GetTenant(ctx context.Context, tenantID uuid.UUID) (*port.TenantInfo, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*port.TenantInfo), args.Error(1)
}

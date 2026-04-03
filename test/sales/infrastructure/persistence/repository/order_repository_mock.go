package repository

import (
	"context"
	"sales/src/sales/domain/entity"

	"github.com/stretchr/testify/mock"
)

// MockOrderRepository es un mock del port OrderRepository
type MockOrderRepository struct {
	mock.Mock
}

func (m *MockOrderRepository) Save(ctx context.Context, order *entity.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockOrderRepository) FindByID(ctx context.Context, orderID, tenantID string) (*entity.Order, error) {
	args := m.Called(ctx, orderID, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Order), args.Error(1)
}

func (m *MockOrderRepository) List(ctx context.Context, tenantID string, page, pageSize int, statusFilter string) ([]*entity.Order, int, error) {
	args := m.Called(ctx, tenantID, page, pageSize, statusFilter)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*entity.Order), args.Int(1), args.Error(2)
}

func (m *MockOrderRepository) GetOrderFinancialSummary(ctx context.Context, orderID, tenantID string) (*entity.OrderFinancialSummary, error) {
	args := m.Called(ctx, orderID, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.OrderFinancialSummary), args.Error(1)
}

func (m *MockOrderRepository) Confirm(ctx context.Context, orderID, tenantID string) error {
	args := m.Called(ctx, orderID, tenantID)
	return args.Error(0)
}

func (m *MockOrderRepository) Cancel(ctx context.Context, orderID, tenantID string) error {
	args := m.Called(ctx, orderID, tenantID)
	return args.Error(0)
}

func (m *MockOrderRepository) UpdateOrderNumber(ctx context.Context, orderID, tenantID string, orderNumber int) error {
	args := m.Called(ctx, orderID, tenantID, orderNumber)
	return args.Error(0)
}

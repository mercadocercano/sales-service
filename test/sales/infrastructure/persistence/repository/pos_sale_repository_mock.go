package repository

import (
	"context"
	"sales/src/sales/domain/entity"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockPosSaleRepository es un mock del port PosSaleRepository
type MockPosSaleRepository struct {
	mock.Mock
}

func (m *MockPosSaleRepository) Create(ctx context.Context, sale *entity.PosSale) error {
	args := m.Called(ctx, sale)
	return args.Error(0)
}

func (m *MockPosSaleRepository) GetByID(ctx context.Context, tenantID, saleID uuid.UUID) (*entity.PosSale, error) {
	args := m.Called(ctx, tenantID, saleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.PosSale), args.Error(1)
}

func (m *MockPosSaleRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*entity.PosSale, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.PosSale), args.Error(1)
}

func (m *MockPosSaleRepository) CountByTenant(ctx context.Context, tenantID uuid.UUID) (int, error) {
	args := m.Called(ctx, tenantID)
	return args.Int(0), args.Error(1)
}

func (m *MockPosSaleRepository) CountSalesTodayByTenant(ctx context.Context, tenantID uuid.UUID) (int, error) {
	args := m.Called(ctx, tenantID)
	return args.Int(0), args.Error(1)
}

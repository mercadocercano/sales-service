package mocks

import (
	"sales/src/sales/domain/port"

	"github.com/stretchr/testify/mock"
)

// MockStockPort es un mock de port.StockPort
type MockStockPort struct {
	mock.Mock
}

func (m *MockStockPort) ValidateStock(tenantID, sku string, quantity int) (*port.StockAvailability, bool, error) {
	args := m.Called(tenantID, sku, quantity)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).(*port.StockAvailability), args.Bool(1), args.Error(2)
}

func (m *MockStockPort) ReserveStock(tenantID, sku string, quantity int, reference string) (*port.StockReserveResult, error) {
	args := m.Called(tenantID, sku, quantity, reference)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*port.StockReserveResult), args.Error(1)
}

func (m *MockStockPort) ReleaseStock(tenantID, sku string, quantity int, reference string) (*port.StockReleaseResult, error) {
	args := m.Called(tenantID, sku, quantity, reference)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*port.StockReleaseResult), args.Error(1)
}

func (m *MockStockPort) ConsumeStock(tenantID, sku string, quantity int, reference string) (*port.StockConsumeResult, error) {
	args := m.Called(tenantID, sku, quantity, reference)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*port.StockConsumeResult), args.Error(1)
}

func (m *MockStockPort) RevertConsume(tenantID, sku string, quantity int, reference string) (*port.StockRevertResult, error) {
	args := m.Called(tenantID, sku, quantity, reference)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*port.StockRevertResult), args.Error(1)
}

func (m *MockStockPort) ProcessSaleAtomic(tenantID, sku string, quantity float64, reference string) (*port.ProcessSaleAtomicResult, error) {
	args := m.Called(tenantID, sku, quantity, reference)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*port.ProcessSaleAtomicResult), args.Error(1)
}

func (m *MockStockPort) CompensateSale(tenantID, stockEntryID, reason string) error {
	args := m.Called(tenantID, stockEntryID, reason)
	return args.Error(0)
}

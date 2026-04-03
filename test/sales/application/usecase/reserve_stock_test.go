package usecase_test

import (
	"errors"
	"testing"

	"sales/src/sales/application/request"
	"sales/src/sales/application/usecase"
	"sales/src/sales/domain/port"
	mocks "sales/test/sales/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestReserveStockUseCase_Execute_WithAllItemsReserved_ReturnsReserved(t *testing.T) {
	// Arrange
	stockPort := new(mocks.MockStockPort)
	uc := usecase.NewReserveStockUseCase(stockPort)

	req := &request.ReserveStockRequest{
		Items: []request.ReserveStockItem{
			{SKU: "SKU-001", Quantity: 2},
			{SKU: "SKU-002", Quantity: 3},
		},
	}

	stockPort.On("ReserveStock", "tenant-1", "SKU-001", 2, mock.Anything).
		Return(&port.StockReserveResult{
			SKU:          "SKU-001",
			ReservedQty:  2,
			RemainingQty: 8,
			Reference:    "ref-001",
		}, nil)

	stockPort.On("ReserveStock", "tenant-1", "SKU-002", 3, mock.Anything).
		Return(&port.StockReserveResult{
			SKU:          "SKU-002",
			ReservedQty:  3,
			RemainingQty: 7,
			Reference:    "ref-002",
		}, nil)

	// Act
	resp, err := uc.Execute("tenant-1", req)

	// Assert
	require.NoError(t, err)
	assert.True(t, resp.Reserved)
	assert.Len(t, resp.Items, 2)
	assert.Equal(t, "SKU-001", resp.Items[0].SKU)
	assert.Equal(t, 2, resp.Items[0].Quantity)
	assert.Equal(t, "SKU-002", resp.Items[1].SKU)
	assert.Equal(t, 3, resp.Items[1].Quantity)
	stockPort.AssertExpectations(t)
}

func TestReserveStockUseCase_Execute_WithSecondItemFailing_RollsBackAndReturnsError(t *testing.T) {
	// Arrange
	stockPort := new(mocks.MockStockPort)
	uc := usecase.NewReserveStockUseCase(stockPort)

	req := &request.ReserveStockRequest{
		Items: []request.ReserveStockItem{
			{SKU: "SKU-001", Quantity: 2},
			{SKU: "SKU-002", Quantity: 100},
		},
	}

	stockPort.On("ReserveStock", "tenant-1", "SKU-001", 2, mock.Anything).
		Return(&port.StockReserveResult{
			SKU:          "SKU-001",
			ReservedQty:  2,
			RemainingQty: 8,
			Reference:    "ref-001",
		}, nil)

	stockPort.On("ReserveStock", "tenant-1", "SKU-002", 100, mock.Anything).
		Return(nil, errors.New("insufficient stock for SKU SKU-002"))

	// Rollback: se libera el primer item reservado
	stockPort.On("ReleaseStock", "tenant-1", "SKU-001", 2, "ref-001").
		Return(&port.StockReleaseResult{
			SKU:         "SKU-001",
			ReleasedQty: 2,
		}, nil)

	// Act
	resp, err := uc.Execute("tenant-1", req)

	// Assert
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient_stock")
	assert.Contains(t, err.Error(), "SKU-002")
	stockPort.AssertExpectations(t)
}

func TestReserveStockUseCase_Execute_WithStockServiceError_ReturnsError(t *testing.T) {
	// Arrange
	stockPort := new(mocks.MockStockPort)
	uc := usecase.NewReserveStockUseCase(stockPort)

	req := &request.ReserveStockRequest{
		Items: []request.ReserveStockItem{
			{SKU: "SKU-001", Quantity: 2},
		},
	}

	stockPort.On("ReserveStock", "tenant-1", "SKU-001", 2, mock.Anything).
		Return(nil, errors.New("stock service unavailable"))

	// Act
	resp, err := uc.Execute("tenant-1", req)

	// Assert
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stock service unavailable")
	stockPort.AssertExpectations(t)
}

package usecase_test

import (
	"errors"
	"testing"

	"sales/src/sales/application/request"
	"sales/src/sales/application/usecase"
	"sales/src/sales/domain/port"
	mocks "sales/test/sales/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateStockUseCase_Execute_WithAllItemsAvailable_ReturnsValid(t *testing.T) {
	// Arrange
	stockPort := new(mocks.MockStockPort)
	uc := usecase.NewValidateStockUseCase(stockPort)

	req := &request.ValidateStockRequest{
		Items: []request.ValidateStockItem{
			{SKU: "SKU-001", Quantity: 2},
			{SKU: "SKU-002", Quantity: 3},
		},
	}

	stockPort.On("ValidateStock", "tenant-1", "SKU-001", 2).
		Return(&port.StockAvailability{
			VariantSKU:        "SKU-001",
			AvailableQuantity: 10,
			IsOutOfStock:      false,
		}, true, nil)

	stockPort.On("ValidateStock", "tenant-1", "SKU-002", 3).
		Return(&port.StockAvailability{
			VariantSKU:        "SKU-002",
			AvailableQuantity: 5,
			IsOutOfStock:      false,
		}, true, nil)

	// Act
	resp, err := uc.Execute("tenant-1", req)

	// Assert
	require.NoError(t, err)
	assert.True(t, resp.Valid)
	assert.Len(t, resp.Items, 2)
	assert.True(t, resp.Items[0].Available)
	assert.Equal(t, "SKU-001", resp.Items[0].SKU)
	assert.Equal(t, 2, resp.Items[0].RequestedQty)
	assert.Equal(t, 10, resp.Items[0].AvailableQty)
	assert.True(t, resp.Items[1].Available)
	stockPort.AssertExpectations(t)
}

func TestValidateStockUseCase_Execute_WithOneItemOutOfStock_ReturnsInvalid(t *testing.T) {
	// Arrange
	stockPort := new(mocks.MockStockPort)
	uc := usecase.NewValidateStockUseCase(stockPort)

	req := &request.ValidateStockRequest{
		Items: []request.ValidateStockItem{
			{SKU: "SKU-001", Quantity: 2},
			{SKU: "SKU-002", Quantity: 100},
		},
	}

	stockPort.On("ValidateStock", "tenant-1", "SKU-001", 2).
		Return(&port.StockAvailability{
			VariantSKU:        "SKU-001",
			AvailableQuantity: 10,
			IsOutOfStock:      false,
		}, true, nil)

	stockPort.On("ValidateStock", "tenant-1", "SKU-002", 100).
		Return(&port.StockAvailability{
			VariantSKU:        "SKU-002",
			AvailableQuantity: 5,
			IsOutOfStock:      true,
		}, false, nil)

	// Act
	resp, err := uc.Execute("tenant-1", req)

	// Assert
	require.NoError(t, err)
	assert.False(t, resp.Valid)
	assert.Len(t, resp.Items, 2)
	assert.True(t, resp.Items[0].Available)
	assert.False(t, resp.Items[1].Available)
	assert.Equal(t, 5, resp.Items[1].AvailableQty)
	stockPort.AssertExpectations(t)
}

func TestValidateStockUseCase_Execute_WithStockServiceError_ReturnsError(t *testing.T) {
	// Arrange
	stockPort := new(mocks.MockStockPort)
	uc := usecase.NewValidateStockUseCase(stockPort)

	req := &request.ValidateStockRequest{
		Items: []request.ValidateStockItem{
			{SKU: "SKU-001", Quantity: 2},
		},
	}

	stockPort.On("ValidateStock", "tenant-1", "SKU-001", 2).
		Return(nil, false, errors.New("stock service unavailable"))

	// Act
	resp, err := uc.Execute("tenant-1", req)

	// Assert
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stock service unavailable")
	stockPort.AssertExpectations(t)
}

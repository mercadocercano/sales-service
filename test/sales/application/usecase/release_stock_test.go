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

func TestReleaseStockUseCase_Execute_WithValidRelease_ReturnsReleased(t *testing.T) {
	// Arrange
	stockPort := new(mocks.MockStockPort)
	uc := usecase.NewReleaseStockUseCase(stockPort)

	req := &request.ReleaseStockRequest{
		SKU:       "SKU-001",
		Quantity:  5,
		Reference: "ref-001",
	}

	stockPort.On("ReleaseStock", "tenant-1", "SKU-001", 5, "ref-001").
		Return(&port.StockReleaseResult{
			SKU:          "SKU-001",
			ReleasedQty:  5,
			AvailableQty: 15,
			ReservedQty:  0,
			Reference:    "ref-001",
		}, nil)

	// Act
	resp, err := uc.Execute("tenant-1", req)

	// Assert
	require.NoError(t, err)
	assert.True(t, resp.Released)
	assert.Equal(t, "SKU-001", resp.SKU)
	assert.Equal(t, 5, resp.Quantity)
	assert.Equal(t, "ref-001", resp.Reference)
	stockPort.AssertExpectations(t)
}

func TestReleaseStockUseCase_Execute_WithInsufficientReservedStock_ReturnsError(t *testing.T) {
	// Arrange
	stockPort := new(mocks.MockStockPort)
	uc := usecase.NewReleaseStockUseCase(stockPort)

	req := &request.ReleaseStockRequest{
		SKU:       "SKU-001",
		Quantity:  100,
		Reference: "ref-001",
	}

	stockPort.On("ReleaseStock", "tenant-1", "SKU-001", 100, "ref-001").
		Return(nil, errors.New("insufficient reserved stock"))

	// Act
	resp, err := uc.Execute("tenant-1", req)

	// Assert
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient_reserved_stock")
	stockPort.AssertExpectations(t)
}

func TestReleaseStockUseCase_Execute_WithStockServiceError_ReturnsError(t *testing.T) {
	// Arrange
	stockPort := new(mocks.MockStockPort)
	uc := usecase.NewReleaseStockUseCase(stockPort)

	req := &request.ReleaseStockRequest{
		SKU:       "SKU-001",
		Quantity:  5,
		Reference: "ref-001",
	}

	stockPort.On("ReleaseStock", "tenant-1", "SKU-001", 5, "ref-001").
		Return(nil, errors.New("connection timeout"))

	// Act
	resp, err := uc.Execute("tenant-1", req)

	// Assert
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error releasing stock")
	stockPort.AssertExpectations(t)
}

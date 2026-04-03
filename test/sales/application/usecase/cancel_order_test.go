package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"sales/src/sales/application/usecase"
	"sales/src/sales/domain/entity"
	"sales/src/sales/domain/port"
	mockRepo "sales/test/sales/infrastructure/persistence/repository"
	mocks "sales/test/sales/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCancelOrderUseCase_Execute_WithConfirmedOrder_CancelsAndRevertsStock(t *testing.T) {
	// Arrange
	orderRepo := new(mockRepo.MockOrderRepository)
	stockPort := new(mocks.MockStockPort)
	uc := usecase.NewCancelOrderUseCase(orderRepo, stockPort)

	order := &entity.Order{
		OrderID:    "order-123",
		TenantID:   "tenant-1",
		CustomerID: "customer-1",
		Status:     entity.OrderStatusConfirmed,
		CreatedAt:  time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		Items: []entity.OrderItem{
			{ItemID: "item-1", SKU: "SKU-001", Quantity: 2},
			{ItemID: "item-2", SKU: "SKU-002", Quantity: 3},
		},
	}

	orderRepo.On("FindByID", context.Background(), "order-123", "tenant-1").
		Return(order, nil)

	stockPort.On("RevertConsume", "tenant-1", "SKU-001", 2, "order-123").
		Return(&port.StockRevertResult{SKU: "SKU-001", RevertedQty: 2}, nil)

	stockPort.On("RevertConsume", "tenant-1", "SKU-002", 3, "order-123").
		Return(&port.StockRevertResult{SKU: "SKU-002", RevertedQty: 3}, nil)

	orderRepo.On("Cancel", context.Background(), "order-123", "tenant-1").
		Return(nil)

	// Act
	result, err := uc.Execute(context.Background(), "tenant-1", "order-123")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, entity.OrderStatusCanceled, result.Status)
	assert.Equal(t, "order-123", result.OrderID)
	orderRepo.AssertExpectations(t)
	stockPort.AssertExpectations(t)
}

func TestCancelOrderUseCase_Execute_WithNonExistentOrder_ReturnsNotFoundError(t *testing.T) {
	// Arrange
	orderRepo := new(mockRepo.MockOrderRepository)
	stockPort := new(mocks.MockStockPort)
	uc := usecase.NewCancelOrderUseCase(orderRepo, stockPort)

	orderRepo.On("FindByID", context.Background(), "order-999", "tenant-1").
		Return(nil, errors.New("order not found"))

	// Act
	result, err := uc.Execute(context.Background(), "tenant-1", "order-999")

	// Assert
	assert.Nil(t, result)
	assert.ErrorIs(t, err, entity.ErrOrderNotFound)
	orderRepo.AssertExpectations(t)
	stockPort.AssertNotCalled(t, "RevertConsume")
}

func TestCancelOrderUseCase_Execute_WithNonConfirmedOrder_ReturnsStateError(t *testing.T) {
	// Arrange
	orderRepo := new(mockRepo.MockOrderRepository)
	stockPort := new(mocks.MockStockPort)
	uc := usecase.NewCancelOrderUseCase(orderRepo, stockPort)

	order := &entity.Order{
		OrderID:    "order-123",
		TenantID:   "tenant-1",
		CustomerID: "customer-1",
		Status:     entity.OrderStatusCreated,
		CreatedAt:  time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		Items: []entity.OrderItem{
			{ItemID: "item-1", SKU: "SKU-001", Quantity: 2},
		},
	}

	orderRepo.On("FindByID", context.Background(), "order-123", "tenant-1").
		Return(order, nil)

	// Act
	result, err := uc.Execute(context.Background(), "tenant-1", "order-123")

	// Assert
	assert.Nil(t, result)
	assert.ErrorIs(t, err, entity.ErrOrderNotInConfirmedState)
	orderRepo.AssertExpectations(t)
	stockPort.AssertNotCalled(t, "RevertConsume")
}

func TestCancelOrderUseCase_Execute_WithStockRevertError_PropagatesError(t *testing.T) {
	// Arrange
	orderRepo := new(mockRepo.MockOrderRepository)
	stockPort := new(mocks.MockStockPort)
	uc := usecase.NewCancelOrderUseCase(orderRepo, stockPort)

	order := &entity.Order{
		OrderID:    "order-123",
		TenantID:   "tenant-1",
		CustomerID: "customer-1",
		Status:     entity.OrderStatusConfirmed,
		CreatedAt:  time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		Items: []entity.OrderItem{
			{ItemID: "item-1", SKU: "SKU-001", Quantity: 2},
		},
	}

	orderRepo.On("FindByID", context.Background(), "order-123", "tenant-1").
		Return(order, nil)

	stockPort.On("RevertConsume", "tenant-1", "SKU-001", 2, "order-123").
		Return(nil, errors.New("stock service unavailable"))

	// Act
	result, err := uc.Execute(context.Background(), "tenant-1", "order-123")

	// Assert
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error reverting stock")
	assert.Contains(t, err.Error(), "SKU-001")
	orderRepo.AssertExpectations(t)
	stockPort.AssertExpectations(t)
	orderRepo.AssertNotCalled(t, "Cancel")
}

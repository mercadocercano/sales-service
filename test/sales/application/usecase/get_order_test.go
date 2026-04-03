package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"sales/src/sales/application/usecase"
	"sales/src/sales/domain/entity"
	mockRepo "sales/test/sales/infrastructure/persistence/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOrderUseCase_Execute_WithExistingOrder_ReturnsResponse(t *testing.T) {
	// Arrange
	repo := new(mockRepo.MockOrderRepository)
	uc := usecase.NewGetOrderUseCase(repo)

	order := &entity.Order{
		OrderID:    "order-123",
		TenantID:   "tenant-1",
		CustomerID: "customer-1",
		Status:     entity.OrderStatusCreated,
		CreatedAt:  time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		Items: []entity.OrderItem{
			{ItemID: "item-1", SKU: "SKU-001", Quantity: 2},
			{ItemID: "item-2", SKU: "SKU-002", Quantity: 3},
		},
	}

	repo.On("FindByID", context.Background(), "order-123", "tenant-1").Return(order, nil)

	// Act
	resp, err := uc.Execute(context.Background(), "tenant-1", "order-123")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "order-123", resp.OrderID)
	assert.Equal(t, "tenant-1", resp.TenantID)
	assert.Equal(t, "CREATED", resp.Status)
	assert.Len(t, resp.Items, 2)
	assert.Equal(t, "SKU-001", resp.Items[0].SKU)
	assert.Equal(t, 2, resp.Items[0].Quantity)
	repo.AssertExpectations(t)
}

func TestGetOrderUseCase_Execute_WithNonExistentOrder_ReturnsNotFoundError(t *testing.T) {
	// Arrange
	repo := new(mockRepo.MockOrderRepository)
	uc := usecase.NewGetOrderUseCase(repo)

	repo.On("FindByID", context.Background(), "order-999", "tenant-1").
		Return(nil, errors.New("order not found"))

	// Act
	resp, err := uc.Execute(context.Background(), "tenant-1", "order-999")

	// Assert
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, entity.ErrOrderNotFound)
	repo.AssertExpectations(t)
}

func TestGetOrderUseCase_Execute_WithRepositoryError_ReturnsError(t *testing.T) {
	// Arrange
	repo := new(mockRepo.MockOrderRepository)
	uc := usecase.NewGetOrderUseCase(repo)

	repo.On("FindByID", context.Background(), "order-123", "tenant-1").
		Return(nil, errors.New("database connection failed"))

	// Act
	resp, err := uc.Execute(context.Background(), "tenant-1", "order-123")

	// Assert
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection failed")
	repo.AssertExpectations(t)
}

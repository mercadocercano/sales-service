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

func TestListOrdersUseCase_Execute_WithOrders_ReturnsPaginatedResponse(t *testing.T) {
	// Arrange
	repo := new(mockRepo.MockOrderRepository)
	uc := usecase.NewListOrdersUseCase(repo)

	orders := []*entity.Order{
		{
			OrderID:   "order-1",
			TenantID:  "tenant-1",
			Status:    entity.OrderStatusCreated,
			CreatedAt: time.Now(),
			Items:     []entity.OrderItem{{ItemID: "item-1", SKU: "SKU-001", Quantity: 1}},
		},
		{
			OrderID:   "order-2",
			TenantID:  "tenant-1",
			Status:    entity.OrderStatusConfirmed,
			CreatedAt: time.Now(),
			Items:     []entity.OrderItem{{ItemID: "item-2", SKU: "SKU-002", Quantity: 2}},
		},
	}

	repo.On("List", context.Background(), "tenant-1", 1, 10, "").Return(orders, 2, nil)

	// Act
	resp, err := uc.Execute(context.Background(), "tenant-1", 1, 10, "")

	// Assert
	require.NoError(t, err)
	assert.Len(t, resp.Items, 2)
	assert.Equal(t, 2, resp.TotalCount)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 10, resp.PageSize)
	assert.Equal(t, 1, resp.TotalPages)
	assert.Equal(t, "CREATED", resp.Items[0].Status)
	assert.Equal(t, "CONFIRMED", resp.Items[1].Status)
	repo.AssertExpectations(t)
}

func TestListOrdersUseCase_Execute_WithEmptyResult_ReturnsEmptyList(t *testing.T) {
	// Arrange
	repo := new(mockRepo.MockOrderRepository)
	uc := usecase.NewListOrdersUseCase(repo)

	repo.On("List", context.Background(), "tenant-1", 1, 10, "").Return([]*entity.Order{}, 0, nil)

	// Act
	resp, err := uc.Execute(context.Background(), "tenant-1", 1, 10, "")

	// Assert
	require.NoError(t, err)
	assert.Empty(t, resp.Items)
	assert.Equal(t, 0, resp.TotalCount)
	assert.Equal(t, 0, resp.TotalPages)
	repo.AssertExpectations(t)
}

func TestListOrdersUseCase_Execute_NormalizesInvalidPage(t *testing.T) {
	// Arrange
	repo := new(mockRepo.MockOrderRepository)
	uc := usecase.NewListOrdersUseCase(repo)

	// page < 1 se normaliza a 1
	repo.On("List", context.Background(), "tenant-1", 1, 10, "").Return([]*entity.Order{}, 0, nil)

	// Act
	resp, err := uc.Execute(context.Background(), "tenant-1", 0, 10, "")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Page)
	repo.AssertExpectations(t)
}

func TestListOrdersUseCase_Execute_NormalizesInvalidPageSize(t *testing.T) {
	// Arrange
	repo := new(mockRepo.MockOrderRepository)
	uc := usecase.NewListOrdersUseCase(repo)

	// pageSize > 100 se normaliza a 10
	repo.On("List", context.Background(), "tenant-1", 1, 10, "").Return([]*entity.Order{}, 0, nil)

	// Act
	resp, err := uc.Execute(context.Background(), "tenant-1", 1, 200, "")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 10, resp.PageSize)
	repo.AssertExpectations(t)
}

func TestListOrdersUseCase_Execute_NormalizesZeroPageSize(t *testing.T) {
	// Arrange
	repo := new(mockRepo.MockOrderRepository)
	uc := usecase.NewListOrdersUseCase(repo)

	// pageSize < 1 se normaliza a 10
	repo.On("List", context.Background(), "tenant-1", 1, 10, "").Return([]*entity.Order{}, 0, nil)

	// Act
	resp, err := uc.Execute(context.Background(), "tenant-1", 1, 0, "")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 10, resp.PageSize)
	repo.AssertExpectations(t)
}

func TestListOrdersUseCase_Execute_CalculatesTotalPagesCorrectly(t *testing.T) {
	// Arrange
	repo := new(mockRepo.MockOrderRepository)
	uc := usecase.NewListOrdersUseCase(repo)

	// 25 registros con pageSize 10 = 3 páginas
	repo.On("List", context.Background(), "tenant-1", 1, 10, "").Return([]*entity.Order{}, 25, nil)

	// Act
	resp, err := uc.Execute(context.Background(), "tenant-1", 1, 10, "")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 3, resp.TotalPages)
	repo.AssertExpectations(t)
}

func TestListOrdersUseCase_Execute_WithStatusFilter_PassesFilter(t *testing.T) {
	// Arrange
	repo := new(mockRepo.MockOrderRepository)
	uc := usecase.NewListOrdersUseCase(repo)

	repo.On("List", context.Background(), "tenant-1", 1, 10, "CONFIRMED").Return([]*entity.Order{}, 0, nil)

	// Act
	resp, err := uc.Execute(context.Background(), "tenant-1", 1, 10, "CONFIRMED")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, resp)
	repo.AssertExpectations(t)
}

func TestListOrdersUseCase_Execute_WithRepositoryError_ReturnsError(t *testing.T) {
	// Arrange
	repo := new(mockRepo.MockOrderRepository)
	uc := usecase.NewListOrdersUseCase(repo)

	repo.On("List", context.Background(), "tenant-1", 1, 10, "").
		Return(nil, 0, errors.New("database error"))

	// Act
	resp, err := uc.Execute(context.Background(), "tenant-1", 1, 10, "")

	// Assert
	assert.Nil(t, resp)
	assert.Error(t, err)
	repo.AssertExpectations(t)
}

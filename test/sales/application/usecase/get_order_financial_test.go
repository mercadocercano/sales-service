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

func TestGetOrderFinancialUseCase_Execute_WithSummary_ReturnsResponse(t *testing.T) {
	// Arrange
	repo := new(mockRepo.MockOrderRepository)
	uc := usecase.NewGetOrderFinancialUseCase(repo)

	lastPayment := time.Date(2025, 3, 10, 14, 0, 0, 0, time.UTC)
	summary := &entity.OrderFinancialSummary{
		OrderID:       "order-123",
		TenantID:      "tenant-1",
		TotalAmount:   1000.0,
		TotalPaid:     600.0,
		Remaining:     400.0,
		Status:        "CONFIRMED",
		PaymentCount:  2,
		LastPaymentAt: &lastPayment,
	}

	repo.On("GetOrderFinancialSummary", context.Background(), "order-123", "tenant-1").Return(summary, nil)

	// Act
	resp, err := uc.Execute(context.Background(), "tenant-1", "order-123")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "order-123", resp.OrderID)
	assert.Equal(t, "tenant-1", resp.TenantID)
	assert.Equal(t, 1000.0, resp.TotalAmount)
	assert.Equal(t, 600.0, resp.TotalPaid)
	assert.Equal(t, 400.0, resp.Remaining)
	assert.Equal(t, "CONFIRMED", resp.Status)
	assert.Equal(t, 2, resp.PaymentCount)
	require.NotNil(t, resp.LastPaymentAt)
	assert.Contains(t, *resp.LastPaymentAt, "2025-03-10")
	repo.AssertExpectations(t)
}

func TestGetOrderFinancialUseCase_Execute_WithNoPayments_NilLastPaymentAt(t *testing.T) {
	// Arrange
	repo := new(mockRepo.MockOrderRepository)
	uc := usecase.NewGetOrderFinancialUseCase(repo)

	summary := &entity.OrderFinancialSummary{
		OrderID:       "order-123",
		TenantID:      "tenant-1",
		TotalAmount:   500.0,
		TotalPaid:     0.0,
		Remaining:     500.0,
		Status:        "CREATED",
		PaymentCount:  0,
		LastPaymentAt: nil,
	}

	repo.On("GetOrderFinancialSummary", context.Background(), "order-123", "tenant-1").Return(summary, nil)

	// Act
	resp, err := uc.Execute(context.Background(), "tenant-1", "order-123")

	// Assert
	require.NoError(t, err)
	assert.Nil(t, resp.LastPaymentAt)
	assert.Equal(t, 0, resp.PaymentCount)
	repo.AssertExpectations(t)
}

func TestGetOrderFinancialUseCase_Execute_WithRepositoryError_ReturnsError(t *testing.T) {
	// Arrange
	repo := new(mockRepo.MockOrderRepository)
	uc := usecase.NewGetOrderFinancialUseCase(repo)

	repo.On("GetOrderFinancialSummary", context.Background(), "order-123", "tenant-1").
		Return(nil, errors.New("database error"))

	// Act
	resp, err := uc.Execute(context.Background(), "tenant-1", "order-123")

	// Assert
	assert.Nil(t, resp)
	assert.Error(t, err)
	repo.AssertExpectations(t)
}

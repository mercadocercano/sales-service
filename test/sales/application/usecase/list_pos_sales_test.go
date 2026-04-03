package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"sales/src/sales/application/usecase"
	"sales/src/sales/domain/entity"
	mockRepo "sales/test/sales/infrastructure/persistence/repository"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListPosSalesUseCase_Execute_WithSales_ReturnsMappedList(t *testing.T) {
	// Arrange
	repo := new(mockRepo.MockPosSaleRepository)
	uc := usecase.NewListPosSalesUseCase(repo)

	tenantID := uuid.New()
	customerID := uuid.New()
	paymentMethodID := uuid.New()

	sales := []*entity.PosSale{
		{
			ID:              uuid.New(),
			TenantID:        tenantID,
			CustomerID:      customerID,
			PaymentMethodID: paymentMethodID,
			TotalAmount:     decimal.NewFromInt(500),
			DiscountAmount:  decimal.NewFromInt(50),
			FinalAmount:     decimal.NewFromInt(450),
			Currency:        "ARS",
			CreatedAt:       time.Now(),
			Items: []entity.PosSaleItem{
				{ID: uuid.New(), SKU: "SKU-1"},
				{ID: uuid.New(), SKU: "SKU-2"},
			},
		},
	}

	repo.On("ListByTenant", context.Background(), tenantID).Return(sales, nil)

	// Act
	resp, err := uc.Execute(context.Background(), tenantID)

	// Assert
	require.NoError(t, err)
	require.Len(t, resp, 1)
	assert.Equal(t, sales[0].ID, resp[0].ID)
	assert.Equal(t, customerID, resp[0].CustomerID)
	assert.Equal(t, paymentMethodID, resp[0].PaymentMethodID)
	assert.True(t, decimal.NewFromInt(500).Equal(resp[0].TotalAmount))
	assert.True(t, decimal.NewFromInt(50).Equal(resp[0].DiscountAmount))
	assert.True(t, decimal.NewFromInt(450).Equal(resp[0].FinalAmount))
	assert.Equal(t, "ARS", resp[0].Currency)
	assert.Equal(t, 2, resp[0].TotalItems)
	repo.AssertExpectations(t)
}

func TestListPosSalesUseCase_Execute_WithEmptyResult_ReturnsEmptyList(t *testing.T) {
	// Arrange
	repo := new(mockRepo.MockPosSaleRepository)
	uc := usecase.NewListPosSalesUseCase(repo)
	tenantID := uuid.New()

	repo.On("ListByTenant", context.Background(), tenantID).Return([]*entity.PosSale{}, nil)

	// Act
	resp, err := uc.Execute(context.Background(), tenantID)

	// Assert
	require.NoError(t, err)
	assert.Empty(t, resp)
	repo.AssertExpectations(t)
}

func TestListPosSalesUseCase_Execute_WithRepositoryError_ReturnsError(t *testing.T) {
	// Arrange
	repo := new(mockRepo.MockPosSaleRepository)
	uc := usecase.NewListPosSalesUseCase(repo)
	tenantID := uuid.New()

	repo.On("ListByTenant", context.Background(), tenantID).
		Return(nil, errors.New("database error"))

	// Act
	resp, err := uc.Execute(context.Background(), tenantID)

	// Assert
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
	repo.AssertExpectations(t)
}

func TestListPosSalesUseCase_Execute_WithMultipleSales_ReturnsMappedList(t *testing.T) {
	// Arrange
	repo := new(mockRepo.MockPosSaleRepository)
	uc := usecase.NewListPosSalesUseCase(repo)
	tenantID := uuid.New()

	sales := []*entity.PosSale{
		{
			ID:             uuid.New(),
			TenantID:       tenantID,
			CustomerID:     uuid.New(),
			PaymentMethodID: uuid.New(),
			TotalAmount:    decimal.NewFromInt(100),
			DiscountAmount: decimal.Zero,
			FinalAmount:    decimal.NewFromInt(100),
			Currency:       "ARS",
			CreatedAt:      time.Now(),
			Items:          []entity.PosSaleItem{{ID: uuid.New()}},
		},
		{
			ID:             uuid.New(),
			TenantID:       tenantID,
			CustomerID:     uuid.New(),
			PaymentMethodID: uuid.New(),
			TotalAmount:    decimal.NewFromInt(200),
			DiscountAmount: decimal.NewFromInt(20),
			FinalAmount:    decimal.NewFromInt(180),
			Currency:       "USD",
			CreatedAt:      time.Now(),
			Items:          []entity.PosSaleItem{{ID: uuid.New()}, {ID: uuid.New()}, {ID: uuid.New()}},
		},
	}

	repo.On("ListByTenant", context.Background(), tenantID).Return(sales, nil)

	// Act
	resp, err := uc.Execute(context.Background(), tenantID)

	// Assert
	require.NoError(t, err)
	require.Len(t, resp, 2)
	assert.Equal(t, 1, resp[0].TotalItems)
	assert.Equal(t, 3, resp[1].TotalItems)
	assert.Equal(t, "USD", resp[1].Currency)
	repo.AssertExpectations(t)
}

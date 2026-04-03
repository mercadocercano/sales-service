package usecase_test

import (
	"context"
	"errors"
	"testing"

	"sales/src/sales/application/usecase"
	"sales/test/sales/mocks"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateCustomerCreditUseCase_Execute_HappyPath_ReturnsCreditID(t *testing.T) {
	// Arrange
	creditRepo := new(mocks.MockCreditRepository)
	eventPublisher := new(mocks.MockEventPublisher)
	uc := usecase.NewCreateCustomerCreditUseCase(creditRepo, eventPublisher)

	amount := decimal.NewFromFloat(500.00)
	creditRepo.On("CreateCredit", mock.Anything, "tenant-1", "cust-001", amount).
		Return("credit-001", nil)
	eventPublisher.On("Execute", mock.Anything, "credit-001", "customer_credit", "sales.customer.credit.created", mock.Anything, "sales-service").
		Return(nil)

	// Act
	creditID, err := uc.Execute(context.Background(), "tenant-1", "cust-001", amount)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "credit-001", creditID)
	creditRepo.AssertExpectations(t)
	eventPublisher.AssertExpectations(t)
}

func TestCreateCustomerCreditUseCase_Execute_WithZeroAmount_ReturnsError(t *testing.T) {
	// Arrange
	creditRepo := new(mocks.MockCreditRepository)
	eventPublisher := new(mocks.MockEventPublisher)
	uc := usecase.NewCreateCustomerCreditUseCase(creditRepo, eventPublisher)

	// Act
	creditID, err := uc.Execute(context.Background(), "tenant-1", "cust-001", decimal.Zero)

	// Assert
	assert.Empty(t, creditID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be greater than zero")
	creditRepo.AssertNotCalled(t, "CreateCredit")
}

func TestCreateCustomerCreditUseCase_Execute_WithNegativeAmount_ReturnsError(t *testing.T) {
	// Arrange
	creditRepo := new(mocks.MockCreditRepository)
	eventPublisher := new(mocks.MockEventPublisher)
	uc := usecase.NewCreateCustomerCreditUseCase(creditRepo, eventPublisher)

	// Act
	creditID, err := uc.Execute(context.Background(), "tenant-1", "cust-001", decimal.NewFromFloat(-100))

	// Assert
	assert.Empty(t, creditID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be greater than zero")
}

func TestCreateCustomerCreditUseCase_Execute_WithRepoError_ReturnsError(t *testing.T) {
	// Arrange
	creditRepo := new(mocks.MockCreditRepository)
	eventPublisher := new(mocks.MockEventPublisher)
	uc := usecase.NewCreateCustomerCreditUseCase(creditRepo, eventPublisher)

	amount := decimal.NewFromFloat(300.00)
	creditRepo.On("CreateCredit", mock.Anything, "tenant-1", "cust-001", amount).
		Return("", errors.New("database error"))

	// Act
	creditID, err := uc.Execute(context.Background(), "tenant-1", "cust-001", amount)

	// Assert
	assert.Empty(t, creditID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
	creditRepo.AssertExpectations(t)
	eventPublisher.AssertNotCalled(t, "Execute")
}

package usecase_test

import (
	"context"
	"errors"
	"testing"

	"sales/src/sales/application/usecase"
	"sales/src/sales/domain/port"
	"sales/test/sales/mocks"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestApplyCustomerCreditUseCase_Execute_HappyPath_ReturnsApplicationID(t *testing.T) {
	// Arrange
	creditRepo := new(mocks.MockCreditRepository)
	eventPublisher := new(mocks.MockEventPublisher)
	uc := usecase.NewApplyCustomerCreditUseCase(creditRepo, eventPublisher)

	amount := decimal.NewFromFloat(200.00)
	application := &port.CreditApplication{
		ApplicationID: "app-001",
		CustomerID:    "cust-001",
	}

	creditRepo.On("ApplyCredit", mock.Anything, "tenant-1", "credit-001", "order-001", amount).
		Return(application, nil)
	eventPublisher.On("Execute", mock.Anything, "app-001", "customer_credit_application", "sales.credit.applied", mock.Anything, "sales-service").
		Return(nil)

	// Act
	appID, err := uc.Execute(context.Background(), "tenant-1", "credit-001", "order-001", amount)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "app-001", appID)
	creditRepo.AssertExpectations(t)
	eventPublisher.AssertExpectations(t)
}

func TestApplyCustomerCreditUseCase_Execute_WithZeroAmount_ReturnsError(t *testing.T) {
	// Arrange
	creditRepo := new(mocks.MockCreditRepository)
	eventPublisher := new(mocks.MockEventPublisher)
	uc := usecase.NewApplyCustomerCreditUseCase(creditRepo, eventPublisher)

	// Act
	appID, err := uc.Execute(context.Background(), "tenant-1", "credit-001", "order-001", decimal.Zero)

	// Assert
	assert.Empty(t, appID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be positive")
	creditRepo.AssertNotCalled(t, "ApplyCredit")
}

func TestApplyCustomerCreditUseCase_Execute_WithNegativeAmount_ReturnsError(t *testing.T) {
	// Arrange
	creditRepo := new(mocks.MockCreditRepository)
	eventPublisher := new(mocks.MockEventPublisher)
	uc := usecase.NewApplyCustomerCreditUseCase(creditRepo, eventPublisher)

	// Act
	appID, err := uc.Execute(context.Background(), "tenant-1", "credit-001", "order-001", decimal.NewFromFloat(-50))

	// Assert
	assert.Empty(t, appID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be positive")
}

func TestApplyCustomerCreditUseCase_Execute_WithRepoError_ReturnsError(t *testing.T) {
	// Arrange
	creditRepo := new(mocks.MockCreditRepository)
	eventPublisher := new(mocks.MockEventPublisher)
	uc := usecase.NewApplyCustomerCreditUseCase(creditRepo, eventPublisher)

	amount := decimal.NewFromFloat(999.99)
	creditRepo.On("ApplyCredit", mock.Anything, "tenant-1", "credit-001", "order-001", amount).
		Return(nil, errors.New("credit fully applied"))

	// Act
	appID, err := uc.Execute(context.Background(), "tenant-1", "credit-001", "order-001", amount)

	// Assert
	assert.Empty(t, appID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "credit fully applied")
	creditRepo.AssertExpectations(t)
	eventPublisher.AssertNotCalled(t, "Execute")
}

func TestApplyCustomerCreditUseCase_Execute_WithCustomerMismatch_ReturnsError(t *testing.T) {
	// Arrange
	creditRepo := new(mocks.MockCreditRepository)
	eventPublisher := new(mocks.MockEventPublisher)
	uc := usecase.NewApplyCustomerCreditUseCase(creditRepo, eventPublisher)

	amount := decimal.NewFromFloat(100.00)
	creditRepo.On("ApplyCredit", mock.Anything, "tenant-1", "credit-002", "order-001", amount).
		Return(nil, errors.New("customer mismatch: credit belongs to different customer"))

	// Act
	appID, err := uc.Execute(context.Background(), "tenant-1", "credit-002", "order-001", amount)

	// Assert
	assert.Empty(t, appID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "customer mismatch")
	creditRepo.AssertExpectations(t)
}

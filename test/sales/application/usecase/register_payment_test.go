package usecase_test

import (
	"context"
	"errors"
	"testing"

	"sales/src/sales/application/usecase"
	"sales/src/sales/domain/port"
	"sales/test/sales/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRegisterPaymentUseCase_Execute_HappyPath_ReturnsPaymentID(t *testing.T) {
	// Arrange
	paymentRepo := new(mocks.MockPaymentRepository)
	eventPublisher := new(mocks.MockEventPublisher)
	uc := usecase.NewRegisterPaymentUseCase(paymentRepo, eventPublisher)

	registration := &port.PaymentRegistration{
		PaymentID:   "pay-001",
		OrderStatus: "paid",
		CustomerID:  "cust-001",
	}

	paymentRepo.On("RegisterPayment", mock.Anything, "tenant-1", "order-1", 150.0).
		Return(registration, nil)
	eventPublisher.On("Execute", mock.Anything, "pay-001", "sales_payment", "sales.payment.registered", mock.Anything, "sales-service").
		Return(nil)

	// Act
	paymentID, err := uc.Execute(context.Background(), "tenant-1", "order-1", 150.0)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "pay-001", paymentID)
	paymentRepo.AssertExpectations(t)
	eventPublisher.AssertExpectations(t)
}

func TestRegisterPaymentUseCase_Execute_WithNegativeAmount_ReturnsError(t *testing.T) {
	// Arrange
	paymentRepo := new(mocks.MockPaymentRepository)
	eventPublisher := new(mocks.MockEventPublisher)
	uc := usecase.NewRegisterPaymentUseCase(paymentRepo, eventPublisher)

	// Act
	paymentID, err := uc.Execute(context.Background(), "tenant-1", "order-1", -10.0)

	// Assert
	assert.Empty(t, paymentID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be positive")
	paymentRepo.AssertNotCalled(t, "RegisterPayment")
}

func TestRegisterPaymentUseCase_Execute_WithZeroAmount_ReturnsError(t *testing.T) {
	// Arrange
	paymentRepo := new(mocks.MockPaymentRepository)
	eventPublisher := new(mocks.MockEventPublisher)
	uc := usecase.NewRegisterPaymentUseCase(paymentRepo, eventPublisher)

	// Act
	paymentID, err := uc.Execute(context.Background(), "tenant-1", "order-1", 0)

	// Assert
	assert.Empty(t, paymentID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be positive")
}

func TestRegisterPaymentUseCase_Execute_WithRepoError_ReturnsError(t *testing.T) {
	// Arrange
	paymentRepo := new(mocks.MockPaymentRepository)
	eventPublisher := new(mocks.MockEventPublisher)
	uc := usecase.NewRegisterPaymentUseCase(paymentRepo, eventPublisher)

	paymentRepo.On("RegisterPayment", mock.Anything, "tenant-1", "order-999", 100.0).
		Return(nil, errors.New("order not found"))

	// Act
	paymentID, err := uc.Execute(context.Background(), "tenant-1", "order-999", 100.0)

	// Assert
	assert.Empty(t, paymentID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "order not found")
	paymentRepo.AssertExpectations(t)
	eventPublisher.AssertNotCalled(t, "Execute")
}

func TestRegisterPaymentUseCase_Execute_WithEventPublishFailure_StillReturnsPaymentID(t *testing.T) {
	// Arrange
	paymentRepo := new(mocks.MockPaymentRepository)
	eventPublisher := new(mocks.MockEventPublisher)
	uc := usecase.NewRegisterPaymentUseCase(paymentRepo, eventPublisher)

	registration := &port.PaymentRegistration{
		PaymentID:   "pay-002",
		OrderStatus: "paid",
		CustomerID:  "cust-002",
	}

	paymentRepo.On("RegisterPayment", mock.Anything, "tenant-1", "order-2", 200.0).
		Return(registration, nil)
	eventPublisher.On("Execute", mock.Anything, "pay-002", "sales_payment", "sales.payment.registered", mock.Anything, "sales-service").
		Return(errors.New("event bus unavailable"))

	// Act
	paymentID, err := uc.Execute(context.Background(), "tenant-1", "order-2", 200.0)

	// Assert — payment still registered despite event failure
	require.NoError(t, err)
	assert.Equal(t, "pay-002", paymentID)
	paymentRepo.AssertExpectations(t)
	eventPublisher.AssertExpectations(t)
}

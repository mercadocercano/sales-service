package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"sales/src/sales/application/usecase"
	"sales/src/sales/domain/entity"
	"sales/src/sales/domain/port"
	"sales/test/sales/mocks"

	mockRepo "sales/test/sales/infrastructure/persistence/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ---------- helpers ----------

func buildCreatedOrder(tenantID string) *entity.Order {
	orderNumber := 0
	_ = orderNumber
	return &entity.Order{
		OrderID:    "order-123",
		TenantID:   tenantID,
		CustomerID: "customer-abc",
		Status:     entity.OrderStatusCreated,
		CreatedAt:  time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		Items: []entity.OrderItem{
			{ItemID: "item-1", OrderID: "order-123", SKU: "SKU-001", Quantity: 2, UnitPrice: 100.0},
			{ItemID: "item-2", OrderID: "order-123", SKU: "SKU-002", Quantity: 1, UnitPrice: 250.0},
		},
	}
}

// ---------- tests ----------

func TestConfirmOrderUseCase_Execute_HappyPath_ReturnsConfirmedOrder(t *testing.T) {
	// Arrange
	orderRepo := new(mockRepo.MockOrderRepository)
	stockPort := new(mocks.MockStockPort)
	eventPublisher := new(mocks.MockEventPublisher)
	sequencePort := new(mocks.MockSequencePort)

	uc := usecase.NewConfirmOrderUseCase(orderRepo, stockPort, eventPublisher, sequencePort)

	tenantID := "tenant-1"
	order := buildCreatedOrder(tenantID)

	orderRepo.On("FindByID", mock.Anything, "order-123", tenantID).Return(order, nil)

	// Stock consume for each item
	stockPort.On("ConsumeStock", tenantID, "SKU-001", 2, "ref-pay-001").
		Return(&port.StockConsumeResult{SKU: "SKU-001", ConsumedQty: 2}, nil)
	stockPort.On("ConsumeStock", tenantID, "SKU-002", 1, "ref-pay-001").
		Return(&port.StockConsumeResult{SKU: "SKU-002", ConsumedQty: 1}, nil)

	// Sequence assigns order number
	sequencePort.On("NextNumber", mock.Anything, tenantID, "SALES_ORDER").Return(42, nil)

	// Confirm in DB
	orderRepo.On("Confirm", mock.Anything, "order-123", tenantID).Return(nil)

	// Persist order number
	orderRepo.On("UpdateOrderNumber", mock.Anything, "order-123", tenantID, 42).Return(nil)

	// Event published (payload contains timestamps, use mock.Anything)
	eventPublisher.On("Execute", mock.Anything, "order-123", "sales_order", "sales.order.confirmed", mock.Anything, "order-service").
		Return(nil)

	// Act
	result, err := uc.Execute(context.Background(), tenantID, "order-123", "ref-pay-001")

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, entity.OrderStatusConfirmed, result.Status)
	assert.Equal(t, "order-123", result.OrderID)
	assert.NotNil(t, result.OrderNumber)
	assert.Equal(t, 42, *result.OrderNumber)

	orderRepo.AssertExpectations(t)
	stockPort.AssertExpectations(t)
	sequencePort.AssertExpectations(t)
	eventPublisher.AssertExpectations(t)
}

func TestConfirmOrderUseCase_Execute_WithOrderNotFound_ReturnsError(t *testing.T) {
	// Arrange
	orderRepo := new(mockRepo.MockOrderRepository)
	stockPort := new(mocks.MockStockPort)
	eventPublisher := new(mocks.MockEventPublisher)
	sequencePort := new(mocks.MockSequencePort)

	uc := usecase.NewConfirmOrderUseCase(orderRepo, stockPort, eventPublisher, sequencePort)

	orderRepo.On("FindByID", mock.Anything, "order-999", "tenant-1").
		Return(nil, errors.New("not found"))

	// Act
	result, err := uc.Execute(context.Background(), "tenant-1", "order-999", "ref-001")

	// Assert
	assert.Nil(t, result)
	assert.ErrorIs(t, err, entity.ErrOrderNotFound)
	orderRepo.AssertExpectations(t)
}

func TestConfirmOrderUseCase_Execute_WithOrderNotInCreatedState_ReturnsError(t *testing.T) {
	// Arrange
	orderRepo := new(mockRepo.MockOrderRepository)
	stockPort := new(mocks.MockStockPort)
	eventPublisher := new(mocks.MockEventPublisher)
	sequencePort := new(mocks.MockSequencePort)

	uc := usecase.NewConfirmOrderUseCase(orderRepo, stockPort, eventPublisher, sequencePort)

	confirmedOrder := &entity.Order{
		OrderID:    "order-123",
		TenantID:   "tenant-1",
		CustomerID: "customer-abc",
		Status:     entity.OrderStatusConfirmed,
		Items:      []entity.OrderItem{{ItemID: "item-1", SKU: "SKU-001", Quantity: 1}},
	}

	orderRepo.On("FindByID", mock.Anything, "order-123", "tenant-1").Return(confirmedOrder, nil)

	// Act
	result, err := uc.Execute(context.Background(), "tenant-1", "order-123", "ref-001")

	// Assert
	assert.Nil(t, result)
	assert.ErrorIs(t, err, entity.ErrOrderNotInCreatedState)
	orderRepo.AssertExpectations(t)
}

func TestConfirmOrderUseCase_Execute_WithStockConsumeError_ReturnsError(t *testing.T) {
	// Arrange
	orderRepo := new(mockRepo.MockOrderRepository)
	stockPort := new(mocks.MockStockPort)
	eventPublisher := new(mocks.MockEventPublisher)
	sequencePort := new(mocks.MockSequencePort)

	uc := usecase.NewConfirmOrderUseCase(orderRepo, stockPort, eventPublisher, sequencePort)

	tenantID := "tenant-1"
	order := buildCreatedOrder(tenantID)

	orderRepo.On("FindByID", mock.Anything, "order-123", tenantID).Return(order, nil)

	// First item consume fails with insufficient reserved stock
	stockPort.On("ConsumeStock", tenantID, "SKU-001", 2, "ref-001").
		Return(nil, errors.New("insufficient reserved stock for SKU-001"))

	// Act
	result, err := uc.Execute(context.Background(), tenantID, "order-123", "ref-001")

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient_reserved_stock")
	orderRepo.AssertExpectations(t)
	stockPort.AssertExpectations(t)
}

func TestConfirmOrderUseCase_Execute_WithSequenceFailure_ReturnsError(t *testing.T) {
	// Arrange
	orderRepo := new(mockRepo.MockOrderRepository)
	stockPort := new(mocks.MockStockPort)
	eventPublisher := new(mocks.MockEventPublisher)
	sequencePort := new(mocks.MockSequencePort)

	uc := usecase.NewConfirmOrderUseCase(orderRepo, stockPort, eventPublisher, sequencePort)

	tenantID := "tenant-1"
	order := buildCreatedOrder(tenantID)

	orderRepo.On("FindByID", mock.Anything, "order-123", tenantID).Return(order, nil)

	// Stock consume succeeds
	stockPort.On("ConsumeStock", tenantID, "SKU-001", 2, "ref-001").
		Return(&port.StockConsumeResult{SKU: "SKU-001", ConsumedQty: 2}, nil)
	stockPort.On("ConsumeStock", tenantID, "SKU-002", 1, "ref-001").
		Return(&port.StockConsumeResult{SKU: "SKU-002", ConsumedQty: 1}, nil)

	// Sequence fails
	sequencePort.On("NextNumber", mock.Anything, tenantID, "SALES_ORDER").
		Return(0, errors.New("sequence table locked"))

	// Act
	result, err := uc.Execute(context.Background(), tenantID, "order-123", "ref-001")

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error assigning order number")
	// Order should NOT be confirmed
	orderRepo.AssertNotCalled(t, "Confirm")
	sequencePort.AssertExpectations(t)
}

func TestConfirmOrderUseCase_Execute_WithEventPublishFailure_StillConfirmsOrder(t *testing.T) {
	// Arrange
	orderRepo := new(mockRepo.MockOrderRepository)
	stockPort := new(mocks.MockStockPort)
	eventPublisher := new(mocks.MockEventPublisher)
	sequencePort := new(mocks.MockSequencePort)

	uc := usecase.NewConfirmOrderUseCase(orderRepo, stockPort, eventPublisher, sequencePort)

	tenantID := "tenant-1"
	order := buildCreatedOrder(tenantID)

	orderRepo.On("FindByID", mock.Anything, "order-123", tenantID).Return(order, nil)

	// Stock consume succeeds
	stockPort.On("ConsumeStock", tenantID, "SKU-001", 2, "ref-001").
		Return(&port.StockConsumeResult{SKU: "SKU-001", ConsumedQty: 2}, nil)
	stockPort.On("ConsumeStock", tenantID, "SKU-002", 1, "ref-001").
		Return(&port.StockConsumeResult{SKU: "SKU-002", ConsumedQty: 1}, nil)

	// Sequence succeeds
	sequencePort.On("NextNumber", mock.Anything, tenantID, "SALES_ORDER").Return(99, nil)

	// Confirm in DB succeeds
	orderRepo.On("Confirm", mock.Anything, "order-123", tenantID).Return(nil)

	// Persist order number succeeds
	orderRepo.On("UpdateOrderNumber", mock.Anything, "order-123", tenantID, 99).Return(nil)

	// Event publish FAILS (warning only, should not block)
	eventPublisher.On("Execute", mock.Anything, "order-123", "sales_order", "sales.order.confirmed", mock.Anything, "order-service").
		Return(errors.New("rabbitmq connection refused"))

	// Act
	result, err := uc.Execute(context.Background(), tenantID, "order-123", "ref-001")

	// Assert — order is still confirmed despite event failure
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, entity.OrderStatusConfirmed, result.Status)
	assert.NotNil(t, result.OrderNumber)
	assert.Equal(t, 99, *result.OrderNumber)

	orderRepo.AssertExpectations(t)
	eventPublisher.AssertExpectations(t)
}

package usecase_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"sales/src/sales/application/request"
	"sales/src/sales/application/usecase"
	"sales/src/sales/domain/port"
	"sales/test/sales/mocks"

	mockRepo "sales/test/sales/infrastructure/persistence/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ---------- helpers ----------

func validTenantID() string {
	return uuid.New().String()
}

func validCustomerID() uuid.UUID {
	return uuid.New()
}

func validCreateOrderRequest(customerID uuid.UUID) *request.CreateOrderRequest {
	return &request.CreateOrderRequest{
		CustomerID: customerID,
		Items: []request.CreateOrderItemRequest{
			{SKU: "SKU-001", Quantity: 2, UnitPrice: 100.0},
			{SKU: "SKU-002", Quantity: 1, UnitPrice: 250.0},
		},
	}
}

func stockSuccessResult(sku, entryID string) *port.ProcessSaleAtomicResult {
	return &port.ProcessSaleAtomicResult{
		Success:      true,
		Message:      "ok",
		VariantSKU:   sku,
		QuantitySold: 1,
		StockEntryID: entryID,
	}
}

// ---------- tests ----------

func TestCreateOrderUseCase_Execute_HappyPath_ReturnsResponse(t *testing.T) {
	// Arrange
	orderRepo := new(mockRepo.MockOrderRepository)
	pimPort := new(mocks.MockPIMPort)
	stockPort := new(mocks.MockStockPort)
	customerPort := new(mocks.MockCustomerPort)

	uc := usecase.NewCreateOrderUseCase(orderRepo, pimPort, stockPort, customerPort)

	tenantID := validTenantID()
	customerID := validCustomerID()
	req := validCreateOrderRequest(customerID)

	tenantUUID := uuid.MustParse(tenantID)

	// Customer exists
	customerPort.On("Exists", mock.Anything, tenantUUID, customerID).Return(true, nil)

	// PIM snapshots (best effort)
	productSnap := json.RawMessage(`{"name":"Product 1"}`)
	variantSnap := json.RawMessage(`{"color":"red"}`)
	pimPort.On("GetSnapshotForSKU", tenantID, "SKU-001").Return(productSnap, variantSnap, nil)
	pimPort.On("GetSnapshotForSKU", tenantID, "SKU-002").Return(productSnap, variantSnap, nil)

	// Stock ProcessSaleAtomic succeeds for both items
	stockPort.On("ProcessSaleAtomic", tenantID, "SKU-001", mock.AnythingOfType("float64"), mock.AnythingOfType("string")).
		Return(stockSuccessResult("SKU-001", "entry-1"), nil)
	stockPort.On("ProcessSaleAtomic", tenantID, "SKU-002", mock.AnythingOfType("float64"), mock.AnythingOfType("string")).
		Return(stockSuccessResult("SKU-002", "entry-2"), nil)

	// Save succeeds
	orderRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

	// Act
	resp, err := uc.Execute(context.Background(), tenantID, req)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.OrderID)
	assert.Equal(t, 2, resp.TotalItems)
	assert.Equal(t, "CREATED", resp.Status)
	assert.Len(t, resp.Items, 2)
	assert.Equal(t, "SKU-001", resp.Items[0].SKU)
	assert.Equal(t, 2, resp.Items[0].Quantity)
	assert.Equal(t, "SKU-002", resp.Items[1].SKU)
	assert.Equal(t, 1, resp.Items[1].Quantity)

	orderRepo.AssertExpectations(t)
	stockPort.AssertExpectations(t)
	customerPort.AssertExpectations(t)
}

func TestCreateOrderUseCase_Execute_WithEmptyItems_ReturnsError(t *testing.T) {
	// Arrange
	orderRepo := new(mockRepo.MockOrderRepository)
	pimPort := new(mocks.MockPIMPort)
	stockPort := new(mocks.MockStockPort)
	customerPort := new(mocks.MockCustomerPort)

	uc := usecase.NewCreateOrderUseCase(orderRepo, pimPort, stockPort, customerPort)

	req := &request.CreateOrderRequest{
		CustomerID: validCustomerID(),
		Items:      []request.CreateOrderItemRequest{},
	}

	// Act
	resp, err := uc.Execute(context.Background(), validTenantID(), req)

	// Assert
	assert.Nil(t, resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "order must contain at least one item")
}

func TestCreateOrderUseCase_Execute_WithCustomerNotFound_ReturnsError(t *testing.T) {
	// Arrange
	orderRepo := new(mockRepo.MockOrderRepository)
	pimPort := new(mocks.MockPIMPort)
	stockPort := new(mocks.MockStockPort)
	customerPort := new(mocks.MockCustomerPort)

	uc := usecase.NewCreateOrderUseCase(orderRepo, pimPort, stockPort, customerPort)

	tenantID := validTenantID()
	customerID := validCustomerID()
	req := validCreateOrderRequest(customerID)

	tenantUUID := uuid.MustParse(tenantID)
	customerPort.On("Exists", mock.Anything, tenantUUID, customerID).Return(false, nil)

	// Act
	resp, err := uc.Execute(context.Background(), tenantID, req)

	// Assert
	assert.Nil(t, resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "customer_not_found")
	customerPort.AssertExpectations(t)
}

func TestCreateOrderUseCase_Execute_WithCustomerServiceError_ReturnsError(t *testing.T) {
	// Arrange
	orderRepo := new(mockRepo.MockOrderRepository)
	pimPort := new(mocks.MockPIMPort)
	stockPort := new(mocks.MockStockPort)
	customerPort := new(mocks.MockCustomerPort)

	uc := usecase.NewCreateOrderUseCase(orderRepo, pimPort, stockPort, customerPort)

	tenantID := validTenantID()
	customerID := validCustomerID()
	req := validCreateOrderRequest(customerID)

	tenantUUID := uuid.MustParse(tenantID)
	customerPort.On("Exists", mock.Anything, tenantUUID, customerID).
		Return(false, errors.New("connection refused"))

	// Act
	resp, err := uc.Execute(context.Background(), tenantID, req)

	// Assert
	assert.Nil(t, resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error validating customer")
	customerPort.AssertExpectations(t)
}

func TestCreateOrderUseCase_Execute_WithPIMFailure_StillCreatesOrder(t *testing.T) {
	// Arrange
	orderRepo := new(mockRepo.MockOrderRepository)
	pimPort := new(mocks.MockPIMPort)
	stockPort := new(mocks.MockStockPort)
	customerPort := new(mocks.MockCustomerPort)

	uc := usecase.NewCreateOrderUseCase(orderRepo, pimPort, stockPort, customerPort)

	tenantID := validTenantID()
	customerID := validCustomerID()
	req := &request.CreateOrderRequest{
		CustomerID: customerID,
		Items: []request.CreateOrderItemRequest{
			{SKU: "SKU-001", Quantity: 1, UnitPrice: 50.0},
		},
	}

	tenantUUID := uuid.MustParse(tenantID)
	customerPort.On("Exists", mock.Anything, tenantUUID, customerID).Return(true, nil)

	// PIM fails (best effort - should not block)
	pimPort.On("GetSnapshotForSKU", tenantID, "SKU-001").
		Return(nil, nil, errors.New("pim unavailable"))

	// Stock succeeds
	stockPort.On("ProcessSaleAtomic", tenantID, "SKU-001", mock.AnythingOfType("float64"), mock.AnythingOfType("string")).
		Return(stockSuccessResult("SKU-001", "entry-1"), nil)

	// Save succeeds
	orderRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

	// Act
	resp, err := uc.Execute(context.Background(), tenantID, req)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 1, resp.TotalItems)
	assert.Equal(t, "CREATED", resp.Status)

	orderRepo.AssertExpectations(t)
	stockPort.AssertExpectations(t)
}

func TestCreateOrderUseCase_Execute_WithStockFailOnSecondItem_CompensatesFirst(t *testing.T) {
	// Arrange
	orderRepo := new(mockRepo.MockOrderRepository)
	pimPort := new(mocks.MockPIMPort)
	stockPort := new(mocks.MockStockPort)
	customerPort := new(mocks.MockCustomerPort)

	uc := usecase.NewCreateOrderUseCase(orderRepo, pimPort, stockPort, customerPort)

	tenantID := validTenantID()
	customerID := validCustomerID()
	req := validCreateOrderRequest(customerID)

	tenantUUID := uuid.MustParse(tenantID)
	customerPort.On("Exists", mock.Anything, tenantUUID, customerID).Return(true, nil)

	pimPort.On("GetSnapshotForSKU", tenantID, "SKU-001").Return(nil, nil, nil)
	pimPort.On("GetSnapshotForSKU", tenantID, "SKU-002").Return(nil, nil, nil)

	// First item succeeds
	stockPort.On("ProcessSaleAtomic", tenantID, "SKU-001", mock.AnythingOfType("float64"), mock.AnythingOfType("string")).
		Return(stockSuccessResult("SKU-001", "entry-1"), nil)

	// Second item fails (HTTP/network error)
	stockPort.On("ProcessSaleAtomic", tenantID, "SKU-002", mock.AnythingOfType("float64"), mock.AnythingOfType("string")).
		Return(nil, errors.New("stock service unavailable"))

	// Compensation for first item
	stockPort.On("CompensateSale", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Act
	resp, err := uc.Execute(context.Background(), tenantID, req)

	// Assert
	assert.Nil(t, resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error processing stock for SKU SKU-002")
	stockPort.AssertCalled(t, "CompensateSale", tenantID, "entry-1", "order_creation_failed")
	stockPort.AssertExpectations(t)
}

func TestCreateOrderUseCase_Execute_WithStockRejected_CompensatesAndReturnsError(t *testing.T) {
	// Arrange
	orderRepo := new(mockRepo.MockOrderRepository)
	pimPort := new(mocks.MockPIMPort)
	stockPort := new(mocks.MockStockPort)
	customerPort := new(mocks.MockCustomerPort)

	uc := usecase.NewCreateOrderUseCase(orderRepo, pimPort, stockPort, customerPort)

	tenantID := validTenantID()
	customerID := validCustomerID()
	req := validCreateOrderRequest(customerID)

	tenantUUID := uuid.MustParse(tenantID)
	customerPort.On("Exists", mock.Anything, tenantUUID, customerID).Return(true, nil)

	pimPort.On("GetSnapshotForSKU", tenantID, "SKU-001").Return(nil, nil, nil)
	pimPort.On("GetSnapshotForSKU", tenantID, "SKU-002").Return(nil, nil, nil)

	// First item succeeds
	stockPort.On("ProcessSaleAtomic", tenantID, "SKU-001", mock.AnythingOfType("float64"), mock.AnythingOfType("string")).
		Return(stockSuccessResult("SKU-001", "entry-1"), nil)

	// Second item returns success=false (business rejection)
	stockPort.On("ProcessSaleAtomic", tenantID, "SKU-002", mock.AnythingOfType("float64"), mock.AnythingOfType("string")).
		Return(&port.ProcessSaleAtomicResult{
			Success: false,
			Message: "insufficient stock for SKU-002",
		}, nil)

	// Compensation
	stockPort.On("CompensateSale", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Act
	resp, err := uc.Execute(context.Background(), tenantID, req)

	// Assert
	assert.Nil(t, resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stock rejected for SKU SKU-002")
	stockPort.AssertCalled(t, "CompensateSale", tenantID, "entry-1", "insufficient_stock")
}

func TestCreateOrderUseCase_Execute_WithSaveFailure_CompensatesAllStock(t *testing.T) {
	// Arrange
	orderRepo := new(mockRepo.MockOrderRepository)
	pimPort := new(mocks.MockPIMPort)
	stockPort := new(mocks.MockStockPort)
	customerPort := new(mocks.MockCustomerPort)

	uc := usecase.NewCreateOrderUseCase(orderRepo, pimPort, stockPort, customerPort)

	tenantID := validTenantID()
	customerID := validCustomerID()
	req := validCreateOrderRequest(customerID)

	tenantUUID := uuid.MustParse(tenantID)
	customerPort.On("Exists", mock.Anything, tenantUUID, customerID).Return(true, nil)

	pimPort.On("GetSnapshotForSKU", tenantID, "SKU-001").Return(nil, nil, nil)
	pimPort.On("GetSnapshotForSKU", tenantID, "SKU-002").Return(nil, nil, nil)

	// Both items succeed
	stockPort.On("ProcessSaleAtomic", tenantID, "SKU-001", mock.AnythingOfType("float64"), mock.AnythingOfType("string")).
		Return(stockSuccessResult("SKU-001", "entry-1"), nil)
	stockPort.On("ProcessSaleAtomic", tenantID, "SKU-002", mock.AnythingOfType("float64"), mock.AnythingOfType("string")).
		Return(stockSuccessResult("SKU-002", "entry-2"), nil)

	// Save fails
	orderRepo.On("Save", mock.Anything, mock.Anything).Return(errors.New("database error"))

	// Compensation for both items
	stockPort.On("CompensateSale", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Act
	resp, err := uc.Execute(context.Background(), tenantID, req)

	// Assert
	assert.Nil(t, resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error saving order (stock compensated)")
	stockPort.AssertCalled(t, "CompensateSale", tenantID, "entry-1", "order_persistence_failed")
	stockPort.AssertCalled(t, "CompensateSale", tenantID, "entry-2", "order_persistence_failed")
}

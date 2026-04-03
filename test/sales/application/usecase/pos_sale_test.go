package usecase_test

import (
	"errors"
	"testing"

	"sales/src/sales/application/request"
	"sales/src/sales/application/usecase"
	"sales/src/sales/domain/port"
	mockRepo "sales/test/sales/infrastructure/persistence/repository"
	"sales/test/sales/mocks"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- helpers ---

func validPOSSaleRequest() *request.POSSaleRequest {
	return &request.POSSaleRequest{
		PaymentMethodID: uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
		CustomerID:      uuid.MustParse("11111111-2222-3333-4444-555555555555"),
		Items: []request.POSSaleItemRequest{
			{
				SKU:       "SKU-001",
				Quantity:  2,
				UnitPrice: decimal.NewFromFloat(100.00),
			},
		},
		DiscountAmount: decimal.Zero,
		AmountPaid:     decimal.NewFromFloat(200.00),
		Currency:       "ARS",
	}
}

func posSaleTenantID() string {
	return "aaaaaaaa-bbbb-cccc-dddd-ffffffffffff"
}

func setupPOSSaleUseCase(
	stockPort *mocks.MockStockPort,
	posSaleRepo *mockRepo.MockPosSaleRepository,
	paymentMethodPort *mocks.MockPaymentMethodPort,
	eventPublisher *mocks.MockEventPublisher,
	customerPort *mocks.MockCustomerPort,
) *usecase.POSSaleUseCase {
	return usecase.NewPOSSaleUseCase(
		stockPort,
		posSaleRepo,
		nil, // tenantMetricsRepo — not needed for core business logic tests
		paymentMethodPort,
		eventPublisher,
		customerPort,
		nil, // tenantPort — not needed for core business logic tests
	)
}

// --- tests ---

func TestPOSSaleUseCase_Execute_HappyPath_ReturnsResponse(t *testing.T) {
	// Arrange
	stockPort := new(mocks.MockStockPort)
	posSaleRepo := new(mockRepo.MockPosSaleRepository)
	paymentMethodPort := new(mocks.MockPaymentMethodPort)
	eventPublisher := new(mocks.MockEventPublisher)
	customerPort := new(mocks.MockCustomerPort)

	uc := setupPOSSaleUseCase(stockPort, posSaleRepo, paymentMethodPort, eventPublisher, customerPort)

	req := validPOSSaleRequest()
	tenantID := posSaleTenantID()
	tenantUUID := uuid.MustParse(tenantID)
	stockEntryID := uuid.New().String()

	paymentMethodPort.On("Get", req.PaymentMethodID).
		Return(port.PaymentMethodInfo{ID: req.PaymentMethodID, Code: "cash", Name: "Efectivo"}, true)
	customerPort.On("Exists", mock.Anything, tenantUUID, req.CustomerID).
		Return(true, nil)
	stockPort.On("ProcessSaleAtomic", tenantID, "SKU-001", mock.AnythingOfType("float64"), mock.AnythingOfType("string")).
		Return(&port.ProcessSaleAtomicResult{
			Success:        true,
			Message:        "ok",
			VariantSKU:     "SKU-001",
			QuantitySold:   2,
			RemainingStock: 8,
			TotalQuantity:  10,
			StockEntryID:   stockEntryID,
		}, nil)
	posSaleRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	posSaleRepo.On("CountSalesTodayByTenant", mock.Anything, tenantUUID).Return(1, nil)
	eventPublisher.On("Execute", mock.Anything, mock.Anything, "pos_sale", "sales.pos.confirmed", mock.Anything, "order-service").
		Return(nil)
	paymentMethodPort.On("GetName", req.PaymentMethodID).Return("Efectivo")

	// Act
	resp, err := uc.Execute(tenantID, req)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "SKU-001", resp.Items[0].SKU)
	assert.Equal(t, 2, resp.Items[0].Quantity)
	assert.Equal(t, "Efectivo", resp.PaymentMethodName)
	assert.Equal(t, req.CustomerID, resp.CustomerID)
	assert.True(t, resp.FinalAmount.Equal(decimal.NewFromFloat(200.00)))
	stockPort.AssertExpectations(t)
	posSaleRepo.AssertExpectations(t)
}

func TestPOSSaleUseCase_Execute_WithNilPaymentMethodID_ReturnsError(t *testing.T) {
	// Arrange
	stockPort := new(mocks.MockStockPort)
	posSaleRepo := new(mockRepo.MockPosSaleRepository)
	paymentMethodPort := new(mocks.MockPaymentMethodPort)
	eventPublisher := new(mocks.MockEventPublisher)
	customerPort := new(mocks.MockCustomerPort)

	uc := setupPOSSaleUseCase(stockPort, posSaleRepo, paymentMethodPort, eventPublisher, customerPort)

	req := validPOSSaleRequest()
	req.PaymentMethodID = uuid.Nil

	// Act
	resp, err := uc.Execute(posSaleTenantID(), req)

	// Assert
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "payment_method_id is required")
}

func TestPOSSaleUseCase_Execute_WithPaymentMethodNotFound_ReturnsError(t *testing.T) {
	// Arrange
	stockPort := new(mocks.MockStockPort)
	posSaleRepo := new(mockRepo.MockPosSaleRepository)
	paymentMethodPort := new(mocks.MockPaymentMethodPort)
	eventPublisher := new(mocks.MockEventPublisher)
	customerPort := new(mocks.MockCustomerPort)

	uc := setupPOSSaleUseCase(stockPort, posSaleRepo, paymentMethodPort, eventPublisher, customerPort)

	req := validPOSSaleRequest()
	paymentMethodPort.On("Get", req.PaymentMethodID).
		Return(port.PaymentMethodInfo{}, false)

	// Act
	resp, err := uc.Execute(posSaleTenantID(), req)

	// Assert
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "payment_method_not_found")
}

func TestPOSSaleUseCase_Execute_WithEmptyItems_ReturnsError(t *testing.T) {
	// Arrange
	stockPort := new(mocks.MockStockPort)
	posSaleRepo := new(mockRepo.MockPosSaleRepository)
	paymentMethodPort := new(mocks.MockPaymentMethodPort)
	eventPublisher := new(mocks.MockEventPublisher)
	customerPort := new(mocks.MockCustomerPort)

	uc := setupPOSSaleUseCase(stockPort, posSaleRepo, paymentMethodPort, eventPublisher, customerPort)

	req := validPOSSaleRequest()
	req.Items = []request.POSSaleItemRequest{}

	paymentMethodPort.On("Get", req.PaymentMethodID).
		Return(port.PaymentMethodInfo{ID: req.PaymentMethodID, Code: "cash", Name: "Efectivo"}, true)

	// Act
	resp, err := uc.Execute(posSaleTenantID(), req)

	// Assert
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one item is required")
}

func TestPOSSaleUseCase_Execute_WithZeroAmountPaid_ReturnsError(t *testing.T) {
	// Arrange
	stockPort := new(mocks.MockStockPort)
	posSaleRepo := new(mockRepo.MockPosSaleRepository)
	paymentMethodPort := new(mocks.MockPaymentMethodPort)
	eventPublisher := new(mocks.MockEventPublisher)
	customerPort := new(mocks.MockCustomerPort)

	uc := setupPOSSaleUseCase(stockPort, posSaleRepo, paymentMethodPort, eventPublisher, customerPort)

	req := validPOSSaleRequest()
	req.AmountPaid = decimal.Zero

	paymentMethodPort.On("Get", req.PaymentMethodID).
		Return(port.PaymentMethodInfo{ID: req.PaymentMethodID, Code: "cash", Name: "Efectivo"}, true)

	// Act
	resp, err := uc.Execute(posSaleTenantID(), req)

	// Assert
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount_paid must be greater than 0")
}

func TestPOSSaleUseCase_Execute_WithCustomerNotFound_ReturnsError(t *testing.T) {
	// Arrange
	stockPort := new(mocks.MockStockPort)
	posSaleRepo := new(mockRepo.MockPosSaleRepository)
	paymentMethodPort := new(mocks.MockPaymentMethodPort)
	eventPublisher := new(mocks.MockEventPublisher)
	customerPort := new(mocks.MockCustomerPort)

	uc := setupPOSSaleUseCase(stockPort, posSaleRepo, paymentMethodPort, eventPublisher, customerPort)

	req := validPOSSaleRequest()
	tenantID := posSaleTenantID()
	tenantUUID := uuid.MustParse(tenantID)

	paymentMethodPort.On("Get", req.PaymentMethodID).
		Return(port.PaymentMethodInfo{ID: req.PaymentMethodID, Code: "cash", Name: "Efectivo"}, true)
	customerPort.On("Exists", mock.Anything, tenantUUID, req.CustomerID).
		Return(false, nil)

	// Act
	resp, err := uc.Execute(tenantID, req)

	// Assert
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "customer_not_found")
}

func TestPOSSaleUseCase_Execute_WithStockProcessError_CompensatesAndReturnsError(t *testing.T) {
	// Arrange
	stockPort := new(mocks.MockStockPort)
	posSaleRepo := new(mockRepo.MockPosSaleRepository)
	paymentMethodPort := new(mocks.MockPaymentMethodPort)
	eventPublisher := new(mocks.MockEventPublisher)
	customerPort := new(mocks.MockCustomerPort)

	uc := setupPOSSaleUseCase(stockPort, posSaleRepo, paymentMethodPort, eventPublisher, customerPort)

	req := validPOSSaleRequest()
	tenantID := posSaleTenantID()
	tenantUUID := uuid.MustParse(tenantID)

	paymentMethodPort.On("Get", req.PaymentMethodID).
		Return(port.PaymentMethodInfo{ID: req.PaymentMethodID, Code: "cash", Name: "Efectivo"}, true)
	customerPort.On("Exists", mock.Anything, tenantUUID, req.CustomerID).
		Return(true, nil)
	stockPort.On("ProcessSaleAtomic", tenantID, "SKU-001", mock.AnythingOfType("float64"), mock.AnythingOfType("string")).
		Return(nil, errors.New("stock-service unavailable"))
	// CompensateSale should be called with empty list (no entries processed yet)
	// No CompensateSale expectation needed since processedStockEntries is empty

	// Act
	resp, err := uc.Execute(tenantID, req)

	// Assert
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error processing stock for SKU SKU-001")
	stockPort.AssertExpectations(t)
}

func TestPOSSaleUseCase_Execute_WithPersistenceError_CompensatesAndReturnsError(t *testing.T) {
	// Arrange
	stockPort := new(mocks.MockStockPort)
	posSaleRepo := new(mockRepo.MockPosSaleRepository)
	paymentMethodPort := new(mocks.MockPaymentMethodPort)
	eventPublisher := new(mocks.MockEventPublisher)
	customerPort := new(mocks.MockCustomerPort)

	uc := setupPOSSaleUseCase(stockPort, posSaleRepo, paymentMethodPort, eventPublisher, customerPort)

	req := validPOSSaleRequest()
	tenantID := posSaleTenantID()
	tenantUUID := uuid.MustParse(tenantID)
	stockEntryID := uuid.New().String()

	paymentMethodPort.On("Get", req.PaymentMethodID).
		Return(port.PaymentMethodInfo{ID: req.PaymentMethodID, Code: "cash", Name: "Efectivo"}, true)
	customerPort.On("Exists", mock.Anything, tenantUUID, req.CustomerID).
		Return(true, nil)
	stockPort.On("ProcessSaleAtomic", tenantID, "SKU-001", mock.AnythingOfType("float64"), mock.AnythingOfType("string")).
		Return(&port.ProcessSaleAtomicResult{
			Success:        true,
			Message:        "ok",
			VariantSKU:     "SKU-001",
			QuantitySold:   2,
			RemainingStock: 8,
			TotalQuantity:  10,
			StockEntryID:   stockEntryID,
		}, nil)
	posSaleRepo.On("Create", mock.Anything, mock.Anything).
		Return(errors.New("database connection lost"))
	// CompensateSale should be called for the processed stock entry
	stockPort.On("CompensateSale", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Act
	resp, err := uc.Execute(tenantID, req)

	// Assert
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error saving pos_sale")
	stockPort.AssertCalled(t, "CompensateSale", tenantID, stockEntryID, "pos_sale_persistence_failed")
}

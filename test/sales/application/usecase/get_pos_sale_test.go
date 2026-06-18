package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"sales/src/sales/application/usecase"
	"sales/src/sales/domain/entity"
	mockRepo "sales/test/sales/infrastructure/persistence/repository"
	"sales/test/sales/mocks"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func buildPersistedPosSale(tenantID, saleID uuid.UUID) *entity.PosSale {
	number := 7
	return &entity.PosSale{
		ID:              saleID,
		TenantID:        tenantID,
		CustomerID:      uuid.Nil, // consumidor final
		PaymentMethodID: testPMUUID(),
		TotalAmount:     decimal.NewFromFloat(200.00),
		DiscountAmount:  decimal.NewFromFloat(20.00),
		FinalAmount:     decimal.NewFromFloat(180.00),
		AmountPaid:      decimal.NewFromFloat(200.00),
		Change:          decimal.NewFromFloat(20.00),
		Currency:        "ARS",
		SaleNumber:      &number,
		CreatedAt:       time.Now(),
		Items: []entity.PosSaleItem{
			{
				ID:          uuid.New(),
				SKU:         "SKU-001",
				ProductName: "Yerba 1kg",
				Quantity:    2,
				UnitPrice:   decimal.NewFromFloat(100.00),
				Subtotal:    decimal.NewFromFloat(200.00),
			},
		},
	}
}

// T-001: detalle de una venta existente del tenant → DTO completo con sale_number.
func TestGetPosSaleUseCase_Execute_ExistingSale_ReturnsDetailWithNumber(t *testing.T) {
	posSaleRepo := new(mockRepo.MockPosSaleRepository)
	pmPort := new(mocks.MockPaymentMethodPort)

	tenantUUID := uuid.MustParse(posSaleTenantID())
	saleUUID := uuid.New()
	persisted := buildPersistedPosSale(tenantUUID, saleUUID)

	posSaleRepo.On("GetByID", mock.Anything, tenantUUID, saleUUID).Return(persisted, nil)
	pmPort.On("GetName", testPMUUID()).Return("Efectivo")

	// La venta de prueba es Consumidor Final (CustomerID = Nil): no se consulta customer-service.
	uc := usecase.NewGetPosSaleUseCase(posSaleRepo, pmPort, nil)

	resp, err := uc.Execute(context.Background(), posSaleTenantID(), saleUUID.String())

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, saleUUID, resp.PosSaleID)
	assert.Equal(t, "Consumidor Final", resp.CustomerName)
	require.NotNil(t, resp.SaleNumber)
	assert.Equal(t, 7, *resp.SaleNumber)
	assert.Equal(t, 1, resp.TotalItems)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "SKU-001", resp.Items[0].SKU)
	assert.Equal(t, "Yerba 1kg", resp.Items[0].ProductName)
	assert.True(t, resp.FinalAmount.Equal(decimal.NewFromFloat(180.00)))
	assert.True(t, resp.Change.Equal(decimal.NewFromFloat(20.00)))
	assert.Equal(t, "Efectivo", resp.PaymentMethodName)
	assert.Equal(t, "ARS", resp.Currency)
	posSaleRepo.AssertExpectations(t)
}

// T-002: venta de otro tenant / inexistente → ErrPosSaleNotFound (mapeado a 404).
func TestGetPosSaleUseCase_Execute_NotFoundOrOtherTenant_ReturnsErr(t *testing.T) {
	posSaleRepo := new(mockRepo.MockPosSaleRepository)
	pmPort := new(mocks.MockPaymentMethodPort)

	tenantUUID := uuid.MustParse(posSaleTenantID())
	saleUUID := uuid.New()

	// El repo filtra por tenant en el WHERE: una venta de otro tenant es indistinguible
	// de inexistente y devuelve ErrPosSaleNotFound.
	posSaleRepo.On("GetByID", mock.Anything, tenantUUID, saleUUID).Return(nil, entity.ErrPosSaleNotFound)

	uc := usecase.NewGetPosSaleUseCase(posSaleRepo, pmPort, nil)

	resp, err := uc.Execute(context.Background(), posSaleTenantID(), saleUUID.String())

	require.Error(t, err)
	assert.ErrorIs(t, err, entity.ErrPosSaleNotFound)
	assert.Nil(t, resp)
	posSaleRepo.AssertExpectations(t)
}

// T-002b: tenant id malformado → error de input (no llega al repo).
func TestGetPosSaleUseCase_Execute_InvalidTenantID_ReturnsErr(t *testing.T) {
	posSaleRepo := new(mockRepo.MockPosSaleRepository)
	pmPort := new(mocks.MockPaymentMethodPort)

	uc := usecase.NewGetPosSaleUseCase(posSaleRepo, pmPort, nil)

	resp, err := uc.Execute(context.Background(), "not-a-uuid", uuid.New().String())

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid tenant_id")
	posSaleRepo.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything, mock.Anything)
}

// T-CN-001: venta con cliente nombrado → CustomerName resuelto vía CustomerPort.
func TestGetPosSaleUseCase_Execute_NamedCustomer_ResolvesCustomerName(t *testing.T) {
	posSaleRepo := new(mockRepo.MockPosSaleRepository)
	pmPort := new(mocks.MockPaymentMethodPort)
	customerPort := new(mocks.MockCustomerPort)

	tenantUUID := uuid.MustParse(posSaleTenantID())
	saleUUID := uuid.New()
	customerUUID := uuid.New()
	persisted := buildPersistedPosSale(tenantUUID, saleUUID)
	persisted.CustomerID = customerUUID

	posSaleRepo.On("GetByID", mock.Anything, tenantUUID, saleUUID).Return(persisted, nil)
	pmPort.On("GetName", testPMUUID()).Return("Efectivo")
	customerPort.On("GetName", mock.Anything, tenantUUID, customerUUID).Return("Juana Pérez", nil)

	uc := usecase.NewGetPosSaleUseCase(posSaleRepo, pmPort, customerPort)

	resp, err := uc.Execute(context.Background(), posSaleTenantID(), saleUUID.String())

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, customerUUID, resp.CustomerID)
	assert.Equal(t, "Juana Pérez", resp.CustomerName)
	customerPort.AssertExpectations(t)
}

// T-CN-002: Consumidor Final (CustomerID Nil) → "Consumidor Final" sin llamar a customer-service.
func TestGetPosSaleUseCase_Execute_ConsumidorFinal_DoesNotCallCustomerPort(t *testing.T) {
	posSaleRepo := new(mockRepo.MockPosSaleRepository)
	pmPort := new(mocks.MockPaymentMethodPort)
	customerPort := new(mocks.MockCustomerPort)

	tenantUUID := uuid.MustParse(posSaleTenantID())
	saleUUID := uuid.New()
	persisted := buildPersistedPosSale(tenantUUID, saleUUID) // CustomerID = Nil

	posSaleRepo.On("GetByID", mock.Anything, tenantUUID, saleUUID).Return(persisted, nil)
	pmPort.On("GetName", testPMUUID()).Return("Efectivo")

	uc := usecase.NewGetPosSaleUseCase(posSaleRepo, pmPort, customerPort)

	resp, err := uc.Execute(context.Background(), posSaleTenantID(), saleUUID.String())

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Consumidor Final", resp.CustomerName)
	customerPort.AssertNotCalled(t, "GetName", mock.Anything, mock.Anything, mock.Anything)
}

// T-CN-003: cliente nombrado pero el port falla (o es nil) → fallback "Cliente", sin romper el detalle.
func TestGetPosSaleUseCase_Execute_CustomerPortError_FallsBack(t *testing.T) {
	tenantUUID := uuid.MustParse(posSaleTenantID())
	saleUUID := uuid.New()
	customerUUID := uuid.New()

	// Caso A: port presente pero retorna error técnico.
	{
		posSaleRepo := new(mockRepo.MockPosSaleRepository)
		pmPort := new(mocks.MockPaymentMethodPort)
		customerPort := new(mocks.MockCustomerPort)

		persisted := buildPersistedPosSale(tenantUUID, saleUUID)
		persisted.CustomerID = customerUUID

		posSaleRepo.On("GetByID", mock.Anything, tenantUUID, saleUUID).Return(persisted, nil)
		pmPort.On("GetName", testPMUUID()).Return("Efectivo")
		customerPort.On("GetName", mock.Anything, tenantUUID, customerUUID).
			Return("", errors.New("customer-service down"))

		uc := usecase.NewGetPosSaleUseCase(posSaleRepo, pmPort, customerPort)
		resp, err := uc.Execute(context.Background(), posSaleTenantID(), saleUUID.String())

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "Cliente", resp.CustomerName)
	}

	// Caso B: customerPort nil → mismo fallback.
	{
		posSaleRepo := new(mockRepo.MockPosSaleRepository)
		pmPort := new(mocks.MockPaymentMethodPort)

		persisted := buildPersistedPosSale(tenantUUID, saleUUID)
		persisted.CustomerID = customerUUID

		posSaleRepo.On("GetByID", mock.Anything, tenantUUID, saleUUID).Return(persisted, nil)
		pmPort.On("GetName", testPMUUID()).Return("Efectivo")

		uc := usecase.NewGetPosSaleUseCase(posSaleRepo, pmPort, nil)
		resp, err := uc.Execute(context.Background(), posSaleTenantID(), saleUUID.String())

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "Cliente", resp.CustomerName)
	}
}

// T-005: NewPosSale no asigna número — queda nil hasta que el repo lo persiste.
func TestNewPosSale_DoesNotAssignSaleNumber(t *testing.T) {
	item, err := entity.NewPosSaleItem(
		uuid.Nil, "SKU-001", "Yerba 1kg", 1, decimal.NewFromFloat(100.00), uuid.New(),
	)
	require.NoError(t, err)

	sale, err := entity.NewPosSale(
		uuid.New(),
		uuid.Nil,
		testPMUUID(),
		[]entity.PosSaleItem{*item},
		decimal.Zero,
		decimal.NewFromFloat(100.00),
		"ARS",
	)
	require.NoError(t, err)
	assert.Nil(t, sale.SaleNumber, "sale_number debe asignarse en la persistencia, no en el constructor")
}

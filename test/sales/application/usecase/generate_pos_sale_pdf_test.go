package usecase_test

import (
	"bytes"
	"context"
	"testing"

	"sales/src/sales/application/usecase"
	"sales/src/sales/domain/entity"
	"sales/src/sales/domain/port"
	"sales/src/sales/infrastructure/adapter"
	mockRepo "sales/test/sales/infrastructure/persistence/repository"
	"sales/test/sales/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// pdfMagic es la firma de un archivo PDF válido ("%PDF").
var pdfMagic = []byte("%PDF")

// T-C-001: venta existente del tenant → 200 con bytes de un PDF válido (header %PDF).
func TestGeneratePosSalePdfUseCase_Execute_ExistingSale_ReturnsValidPDF(t *testing.T) {
	posSaleRepo := new(mockRepo.MockPosSaleRepository)
	pmPort := new(mocks.MockPaymentMethodPort)
	tenantPort := new(mocks.MockTenantPort)

	tenantUUID := uuid.MustParse(posSaleTenantID())
	saleUUID := uuid.New()
	persisted := buildPersistedPosSale(tenantUUID, saleUUID)

	posSaleRepo.On("GetByID", mock.Anything, tenantUUID, saleUUID).Return(persisted, nil)
	pmPort.On("GetName", testPMUUID()).Return("Efectivo")
	tenantPort.On("GetTenant", mock.Anything, tenantUUID).
		Return(&port.TenantInfo{ID: tenantUUID, Name: "Almacén Don José"}, nil)

	getPosSaleUC := usecase.NewGetPosSaleUseCase(posSaleRepo, pmPort, nil)
	renderer := adapter.NewGofpdfPosReceiptRenderer()
	uc := usecase.NewGeneratePosSalePdfUseCase(getPosSaleUC, tenantPort, renderer)

	pdfBytes, err := uc.Execute(context.Background(), posSaleTenantID(), saleUUID.String())

	require.NoError(t, err)
	require.NotEmpty(t, pdfBytes)
	assert.True(t, bytes.HasPrefix(pdfBytes, pdfMagic), "el contenido debe empezar con la firma %%PDF")
	posSaleRepo.AssertExpectations(t)
}

// T-C-002: venta de otro tenant / inexistente → ErrPosSaleNotFound (mapeado a 404 en el borde).
func TestGeneratePosSalePdfUseCase_Execute_NotFoundOrOtherTenant_ReturnsErr(t *testing.T) {
	posSaleRepo := new(mockRepo.MockPosSaleRepository)
	pmPort := new(mocks.MockPaymentMethodPort)
	tenantPort := new(mocks.MockTenantPort)

	tenantUUID := uuid.MustParse(posSaleTenantID())
	saleUUID := uuid.New()

	// El repo filtra por tenant en el WHERE: una venta de otro tenant es
	// indistinguible de inexistente y devuelve ErrPosSaleNotFound.
	posSaleRepo.On("GetByID", mock.Anything, tenantUUID, saleUUID).Return(nil, entity.ErrPosSaleNotFound)

	getPosSaleUC := usecase.NewGetPosSaleUseCase(posSaleRepo, pmPort, nil)
	renderer := adapter.NewGofpdfPosReceiptRenderer()
	uc := usecase.NewGeneratePosSalePdfUseCase(getPosSaleUC, tenantPort, renderer)

	pdfBytes, err := uc.Execute(context.Background(), posSaleTenantID(), saleUUID.String())

	require.Error(t, err)
	assert.ErrorIs(t, err, entity.ErrPosSaleNotFound)
	assert.Nil(t, pdfBytes)
	posSaleRepo.AssertExpectations(t)
	// No se debe consultar el tenant si la venta no existe.
	tenantPort.AssertNotCalled(t, "GetTenant", mock.Anything, mock.Anything)
}

// T-C-003: el comprobante se genera aunque tenantPort sea nil (encabezado sin nombre de comercio).
func TestGeneratePosSalePdfUseCase_Execute_NilTenantPort_StillRenders(t *testing.T) {
	posSaleRepo := new(mockRepo.MockPosSaleRepository)
	pmPort := new(mocks.MockPaymentMethodPort)

	tenantUUID := uuid.MustParse(posSaleTenantID())
	saleUUID := uuid.New()
	persisted := buildPersistedPosSale(tenantUUID, saleUUID)

	posSaleRepo.On("GetByID", mock.Anything, tenantUUID, saleUUID).Return(persisted, nil)
	pmPort.On("GetName", testPMUUID()).Return("Efectivo")

	getPosSaleUC := usecase.NewGetPosSaleUseCase(posSaleRepo, pmPort, nil)
	renderer := adapter.NewGofpdfPosReceiptRenderer()
	uc := usecase.NewGeneratePosSalePdfUseCase(getPosSaleUC, nil, renderer)

	pdfBytes, err := uc.Execute(context.Background(), posSaleTenantID(), saleUUID.String())

	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(pdfBytes, pdfMagic))
}

package usecase

import (
	"context"
	"fmt"

	"sales/src/sales/domain/port"

	"github.com/google/uuid"
)

// GeneratePosSalePdfUseCase genera el PDF A4 del comprobante de una venta POS.
// E18 Tramo C.
//
// Orquesta: reusa GetPosSaleUseCase para obtener el detalle (que ya aplica
// multi-tenancy: una venta de otro tenant retorna entity.ErrPosSaleNotFound),
// resuelve el nombre del comercio vía TenantPort y delega el render al
// PosReceiptRenderer (port de infraestructura).
type GeneratePosSalePdfUseCase struct {
	getPosSale *GetPosSaleUseCase
	tenantPort port.TenantPort        // puede ser nil → encabezado sin nombre de comercio
	renderer   port.PosReceiptRenderer
}

// NewGeneratePosSalePdfUseCase crea el caso de uso.
// tenantPort es opcional (best-effort para el encabezado); renderer es obligatorio.
func NewGeneratePosSalePdfUseCase(
	getPosSale *GetPosSaleUseCase,
	tenantPort port.TenantPort,
	renderer port.PosReceiptRenderer,
) *GeneratePosSalePdfUseCase {
	return &GeneratePosSalePdfUseCase{
		getPosSale: getPosSale,
		tenantPort: tenantPort,
		renderer:   renderer,
	}
}

// Execute retorna los bytes del PDF del comprobante del tenant.
// Multi-tenancy: una venta de otro tenant (o inexistente) propaga
// entity.ErrPosSaleNotFound desde GetPosSaleUseCase (mapeado a 404 en el borde).
func (uc *GeneratePosSalePdfUseCase) Execute(ctx context.Context, tenantID, saleID string) ([]byte, error) {
	detail, err := uc.getPosSale.Execute(ctx, tenantID, saleID)
	if err != nil {
		return nil, err // incluye entity.ErrPosSaleNotFound y errores de input
	}

	// Resolver nombre del comercio (best-effort: no debe romper la generación
	// del comprobante si el tenant-service no responde).
	tenant := port.PosReceiptTenant{}
	if uc.tenantPort != nil {
		if tenantUUID, parseErr := uuid.Parse(tenantID); parseErr == nil {
			if info, tErr := uc.tenantPort.GetTenant(ctx, tenantUUID); tErr == nil && info != nil {
				tenant.Name = info.Name
			}
		}
	}

	pdf, err := uc.renderer.RenderPDF(detail, tenant)
	if err != nil {
		return nil, fmt.Errorf("rendering pos sale pdf: %w", err)
	}
	return pdf, nil
}

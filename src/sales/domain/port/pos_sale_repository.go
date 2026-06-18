package port

import (
	"context"
	"sales/src/sales/domain/entity"

	"github.com/google/uuid"
)

// PosSaleRepository define el contrato para persistir ventas POS
// Solo operaciones mínimas: Create, ListByTenant, CountByTenant
// Sin GetByID, sin Updates, sin Deletes
// Hito: POS-SALE-02.BE - Paso 2
type PosSaleRepository interface {
	// Create persiste una nueva venta POS.
	// E18 Tramo B: asigna sale_number (comprobante INTERNO correlativo por tenant)
	// dentro de la MISMA transacción que inserta pos_sales, tomando el siguiente
	// número desde document_sequences (POS_SALE) con SELECT ... FOR UPDATE.
	// Tras un Create exitoso, sale.SaleNumber queda poblado.
	Create(ctx context.Context, sale *entity.PosSale) error

	// GetByID retorna el detalle completo de una venta POS del tenant (con items).
	// E18 Tramo B: respeta multi-tenancy — si la venta no existe o pertenece a
	// otro tenant, retorna entity.ErrPosSaleNotFound (no se distingue para no filtrar info).
	GetByID(ctx context.Context, tenantID, saleID uuid.UUID) (*entity.PosSale, error)

	// ListByTenant retorna todas las ventas POS de un tenant
	// Sin paginación, sin filtros, sin ordenamiento
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*entity.PosSale, error)

	// CountByTenant cuenta las ventas POS de un tenant
	CountByTenant(ctx context.Context, tenantID uuid.UUID) (int, error)

	// CountSalesTodayByTenant cuenta ventas del tenant en el día actual (UTC)
	// H8 DAT: Si count == 1 tras Create, es la primera venta del día
	CountSalesTodayByTenant(ctx context.Context, tenantID uuid.UUID) (int, error)
}

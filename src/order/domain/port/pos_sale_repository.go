package port

import (
	"context"
	"order/src/order/domain/entity"

	"github.com/google/uuid"
)

// PosSaleRepository define el contrato para persistir ventas POS
// Solo operaciones mínimas: Create y ListByTenant
// Sin GetByID, sin Updates, sin Deletes
// Hito: POS-SALE-02.BE - Paso 2
type PosSaleRepository interface {
	// Create persiste una nueva venta POS
	// No valida, solo inserta
	Create(ctx context.Context, sale *entity.PosSale) error

	// ListByTenant retorna todas las ventas POS de un tenant
	// Sin paginación, sin filtros, sin ordenamiento
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*entity.PosSale, error)
}

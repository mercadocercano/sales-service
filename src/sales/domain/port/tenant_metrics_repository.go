package port

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// TenantMetricsRepository registra métricas por tenant (H8 TTFS edge case)
// INSERT ON CONFLICT DO NOTHING para evitar TTFS duplicado cuando se eliminan ventas
type TenantMetricsRepository interface {
	// RecordFirstSaleIfNew inserta first_sale_at solo si el tenant no tiene registro.
	// Retorna true si se insertó (primera venta real), false si ya existía.
	RecordFirstSaleIfNew(ctx context.Context, tenantID uuid.UUID, firstSaleAt time.Time) (inserted bool, err error)
}

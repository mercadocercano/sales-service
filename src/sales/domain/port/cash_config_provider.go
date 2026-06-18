package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CashConfigProvider expone la configuración de caja por tenant. Hoy solo el umbral de
// descuadre que dispara PENDING_REVIEW; el valor se snapshotea en la sesión al abrir.
// El adapter por defecto devuelve una tolerancia configurable por env, con un seam para
// leer de tenant-settings por tenant (deuda documentada).
type CashConfigProvider interface {
	GetReviewThreshold(ctx context.Context, tenantID uuid.UUID) decimal.Decimal
}

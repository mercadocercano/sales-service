package adapter

import (
	"context"
	"os"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// EnvCashConfigAdapter implementa port.CashConfigProvider. Por ahora devuelve una
// tolerancia de descuadre global configurable por env (CASH_REVIEW_THRESHOLD, default
// ARS 100). El método recibe tenantID como seam para, a futuro, leer el umbral de
// tenant-settings por tenant (DEUDA: hoy no consulta tenant-settings).
type EnvCashConfigAdapter struct {
	defaultThreshold decimal.Decimal
}

func NewEnvCashConfigAdapter() *EnvCashConfigAdapter {
	threshold := decimal.NewFromInt(100)
	if v := os.Getenv("CASH_REVIEW_THRESHOLD"); v != "" {
		if parsed, err := decimal.NewFromString(v); err == nil && parsed.GreaterThanOrEqual(decimal.Zero) {
			threshold = parsed
		}
	}
	return &EnvCashConfigAdapter{defaultThreshold: threshold}
}

func (a *EnvCashConfigAdapter) GetReviewThreshold(_ context.Context, _ uuid.UUID) decimal.Decimal {
	return a.defaultThreshold
}

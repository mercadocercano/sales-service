package port

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// DailyReportData contiene los datos del reporte diario
type DailyReportData struct {
	PosSalesCount      int
	OrdersCount        int
	PosGrossTotal      decimal.Decimal
	PosDiscounts       decimal.Decimal
	PosNetTotal        decimal.Decimal
	FirstTransactionAt *time.Time
	LastTransactionAt  *time.Time
}

// DailyReportRepository define las operaciones de consulta de reportes diarios
type DailyReportRepository interface {
	GetDailyReport(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (*DailyReportData, error)
}

package reports

import (
	"context"
	"sales/src/sales/domain/port"
)

// GetAgingReportUseCase genera el reporte de aging por cliente (cuentas por cobrar).
// Buckets: 0-30, 31-60, 61-90, 90+ días desde confirmed_at (o created_at si no confirmada).
type GetAgingReportUseCase struct {
	repo port.ReportRepository
}

// NewGetAgingReportUseCase crea el caso de uso.
func NewGetAgingReportUseCase(repo port.ReportRepository) *GetAgingReportUseCase {
	return &GetAgingReportUseCase{repo: repo}
}

// Execute ejecuta el reporte. Sin paginación (agrupado por cliente).
func (uc *GetAgingReportUseCase) Execute(ctx context.Context, tenantID string) (*AgingReportResponse, error) {
	rows, err := uc.repo.GetAgingReport(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var totalOutstanding float64
	for i := range rows {
		totalOutstanding += rows[i].Total
	}

	return &AgingReportResponse{
		TotalOutstanding: totalOutstanding,
		Customers:       rows,
	}, nil
}

package reports

import (
	"context"
	"sales/src/sales/domain/port"
)

// GetOpenOrdersReportUseCase genera el reporte de órdenes abiertas (outstanding).
// Soporta aging_bucket (tabs), customer_id, sort y bucket_counts para UI.
type GetOpenOrdersReportUseCase struct {
	repo port.ReportRepository
}

// NewGetOpenOrdersReportUseCase crea el caso de uso.
func NewGetOpenOrdersReportUseCase(repo port.ReportRepository) *GetOpenOrdersReportUseCase {
	return &GetOpenOrdersReportUseCase{repo: repo}
}

// Execute ejecuta el reporte.
// agingBucket: "" | "0_30" | "31_60" | "61_90" | "90_plus"
// sort: "" (remaining_desc) | "remaining_asc" | "days_desc" | "days_asc"
func (uc *GetOpenOrdersReportUseCase) Execute(
	ctx context.Context,
	tenantID string,
	page, pageSize int,
	agingBucket, customerID, sort string,
) (*OpenOrdersReportResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	rows, err := uc.repo.GetOpenOrders(ctx, tenantID, pageSize, offset, agingBucket, customerID, sort)
	if err != nil {
		return nil, err
	}
	summary, err := uc.repo.GetOpenOrdersSummary(ctx, tenantID, agingBucket, customerID)
	if err != nil {
		return nil, err
	}
	bucketCounts, err := uc.repo.GetOpenOrdersBucketCounts(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return &OpenOrdersReportResponse{
		Summary:      *summary,
		BucketCounts: *bucketCounts,
		Orders:       rows,
	}, nil
}

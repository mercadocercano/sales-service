package reports

import (
	"context"
	"sales/src/sales/domain/port"
)

// GetCustomerBalanceUseCase genera la ficha financiera de un cliente.
// Resumen, aging y órdenes abiertas; sin consultar ledger.
type GetCustomerBalanceUseCase struct {
	repo port.ReportRepository
}

// NewGetCustomerBalanceUseCase crea el caso de uso.
func NewGetCustomerBalanceUseCase(repo port.ReportRepository) *GetCustomerBalanceUseCase {
	return &GetCustomerBalanceUseCase{repo: repo}
}

// Execute ejecuta el reporte. summary y aging siempre completos; open_orders paginado y filtrable por aging_bucket.
// page/pageSize: default 1/20, page_size máx 100.
func (uc *GetCustomerBalanceUseCase) Execute(
	ctx context.Context,
	tenantID, customerID string,
	page, pageSize int,
	agingBucket, sort string,
) (*CustomerBalanceResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	summary, err := uc.repo.GetCustomerBalanceSummary(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}
	aging, err := uc.repo.GetCustomerAging(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}
	bucketCounts, err := uc.repo.GetCustomerBalanceBucketCounts(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}
	orders, err := uc.repo.GetCustomerOpenOrders(ctx, tenantID, customerID, pageSize, offset, agingBucket, sort)
	if err != nil {
		return nil, err
	}
	return &CustomerBalanceResponse{
		CustomerID:   customerID,
		Summary:      *summary,
		Aging:        *aging,
		BucketCounts: *bucketCounts,
		OpenOrders:   orders,
	}, nil
}

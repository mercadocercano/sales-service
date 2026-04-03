package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"sales/src/sales/domain/port"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// DailyReportPostgresRepository implementa port.DailyReportRepository con PostgreSQL
type DailyReportPostgresRepository struct {
	db *sql.DB
}

// NewDailyReportPostgresRepository crea una nueva instancia
func NewDailyReportPostgresRepository(db *sql.DB) *DailyReportPostgresRepository {
	return &DailyReportPostgresRepository{db: db}
}

// GetDailyReport obtiene los datos del reporte diario combinando POS sales y orders
func (r *DailyReportPostgresRepository) GetDailyReport(
	ctx context.Context,
	tenantID uuid.UUID,
	from, to time.Time,
) (*port.DailyReportData, error) {
	// Query POS Sales
	queryPOS := `
		SELECT
			COUNT(*) as sales_count,
			COALESCE(SUM(total_amount), 0) as gross_total,
			COALESCE(SUM(discount_amount), 0) as total_discounts,
			COALESCE(SUM(final_amount), 0) as net_total,
			MIN(created_at) as first_sale,
			MAX(created_at) as last_sale
		FROM pos_sales
		WHERE tenant_id = $1
			AND created_at >= $2
			AND created_at < $3
	`

	var posSalesCount int
	var grossTotal, totalDiscounts, netTotal decimal.Decimal
	var firstSale, lastSale sql.NullTime

	err := r.db.QueryRowContext(ctx, queryPOS, tenantID, from, to).Scan(
		&posSalesCount,
		&grossTotal,
		&totalDiscounts,
		&netTotal,
		&firstSale,
		&lastSale,
	)
	if err != nil {
		return nil, fmt.Errorf("error querying pos_sales: %w", err)
	}

	// Query Orders
	queryOrders := `
		SELECT COUNT(*)
		FROM orders
		WHERE tenant_id = $1
			AND created_at >= $2
			AND created_at < $3
	`

	var ordersCount int
	err = r.db.QueryRowContext(ctx, queryOrders, tenantID, from, to).Scan(&ordersCount)
	if err != nil {
		return nil, fmt.Errorf("error querying orders: %w", err)
	}

	result := &port.DailyReportData{
		PosSalesCount: posSalesCount,
		OrdersCount:   ordersCount,
		PosGrossTotal: grossTotal,
		PosDiscounts:  totalDiscounts,
		PosNetTotal:   netTotal,
	}

	if firstSale.Valid {
		result.FirstTransactionAt = &firstSale.Time
	}
	if lastSale.Valid {
		result.LastTransactionAt = &lastSale.Time
	}

	return result, nil
}

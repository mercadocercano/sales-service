package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/hornosg/go-shared/infrastructure/postgres"

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
// GetDailyReport obtiene los datos del reporte diario combinando POS sales y orders.
// PLAT-E25 T10: corre dentro de WithRLSInTransaction (SET LOCAL app.tenant_id) — el WHERE
// tenant_id explícito se mantiene como defensa en profundidad (mismo criterio que T4-T9).
// La query de "orders" apuntaba a la tabla `orders`, eliminada por la migración 010
// (`DROP TABLE IF EXISTS orders CASCADE`, reemplazada por `sales_orders`) — corregida acá.
func (r *DailyReportPostgresRepository) GetDailyReport(
	ctx context.Context,
	tenantID uuid.UUID,
	from, to time.Time,
) (*port.DailyReportData, error) {
	rc := postgres.RLSContext{TenantID: tenantID.String()}

	var posSalesCount, ordersCount int
	var grossTotal, totalDiscounts, netTotal decimal.Decimal
	var firstSale, lastSale sql.NullTime

	err := postgres.WithRLSInTransaction(ctx, r.db, rc, func(ctx context.Context, tx *sql.Tx) error {
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

		if err := tx.QueryRowContext(ctx, queryPOS, tenantID, from, to).Scan(
			&posSalesCount,
			&grossTotal,
			&totalDiscounts,
			&netTotal,
			&firstSale,
			&lastSale,
		); err != nil {
			return fmt.Errorf("error querying pos_sales: %w", err)
		}

		// Query Orders
		queryOrders := `
			SELECT COUNT(*)
			FROM sales_orders
			WHERE tenant_id = $1
				AND created_at >= $2
				AND created_at < $3
		`

		if err := tx.QueryRowContext(ctx, queryOrders, tenantID, from, to).Scan(&ordersCount); err != nil {
			return fmt.Errorf("error querying sales_orders: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
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

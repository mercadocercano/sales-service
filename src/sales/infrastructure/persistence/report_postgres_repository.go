package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/hornosg/go-shared/infrastructure/postgres"

	"sales/src/sales/domain/port"
)

// ReportPostgresRepository implementa port.ReportRepository con consultas optimizadas.
// Solo sales_orders + sales_payments; no consulta ledger.
type ReportPostgresRepository struct {
	db *sql.DB
}

// NewReportPostgresRepository crea el repositorio.
func NewReportPostgresRepository(db *sql.DB) *ReportPostgresRepository {
	return &ReportPostgresRepository{db: db}
}

func agingConditionSQL(agingBucket string) string {
	days := "DATE_PART('day', NOW() - COALESCE(so.confirmed_at, so.created_at))"
	switch agingBucket {
	case "0_30":
		return " AND " + days + " <= 30"
	case "31_60":
		return " AND " + days + " BETWEEN 31 AND 60"
	case "61_90":
		return " AND " + days + " BETWEEN 61 AND 90"
	case "90_plus":
		return " AND " + days + " > 90"
	default:
		return ""
	}
}

func orderBySQL(sort string) string {
	switch sort {
	case "remaining_asc":
		return "remaining ASC"
	case "days_desc":
		return "days_since_confirmed DESC"
	case "days_asc":
		return "days_since_confirmed ASC"
	default:
		return "remaining DESC"
	}
}

const openOrdersSelect = `
		SELECT
		    so.id,
		    so.customer_id,
		    so.total_amount,
		    COALESCE(sp.total_paid, 0) AS total_paid,
		    so.total_amount - COALESCE(sp.total_paid, 0) AS remaining,
		    so.status,
		    so.confirmed_at,
		    DATE_PART('day', NOW() - COALESCE(so.confirmed_at, so.created_at)) AS days_since_confirmed
		FROM sales_orders so
		LEFT JOIN (
		    SELECT sales_order_id, SUM(amount) AS total_paid
		    FROM sales_payments
		    GROUP BY sales_order_id
		) sp ON sp.sales_order_id = so.id
		WHERE so.tenant_id = $1
		  AND so.status IN ('CREATED', 'CONFIRMED')
		  AND (so.total_amount - COALESCE(sp.total_paid, 0)) > 0`

// GetOpenOrders devuelve órdenes abiertas con paginación, filtros opcionales (aging_bucket, customer_id) y orden.
// PLAT-E25 T10: corre dentro de WithRLSInTransaction (SET LOCAL app.tenant_id) — el WHERE
// tenant_id explícito se mantiene como defensa en profundidad (mismo criterio que T4-T9).
func (r *ReportPostgresRepository) GetOpenOrders(
	ctx context.Context,
	tenantID string,
	limit, offset int,
	agingBucket, customerID, sort string,
) ([]port.OpenOrderRow, error) {
	rc := postgres.RLSContext{TenantID: tenantID}

	var result []port.OpenOrderRow
	err := postgres.WithRLSInTransaction(ctx, r.db, rc, func(ctx context.Context, tx *sql.Tx) error {
		args := []interface{}{tenantID}
		n := 1
		customerClause := ""
		if customerID != "" {
			n++
			customerClause = " AND so.customer_id = $" + strconv.Itoa(n)
			args = append(args, customerID)
		}
		agingClause := agingConditionSQL(agingBucket)
		args = append(args, limit, offset)
		limitParam := n + 1
		offsetParam := n + 2
		query := openOrdersSelect + customerClause + agingClause +
			" ORDER BY " + orderBySQL(sort) +
			fmt.Sprintf(" LIMIT $%d OFFSET $%d", limitParam, offsetParam)
		rows, err := tx.QueryContext(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("get open orders: %w", err)
		}
		defer rows.Close()
		result, err = scanOpenOrderRows(rows)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetOpenOrdersSummary devuelve resumen (filtrado por aging_bucket y customer_id si aplica).
func (r *ReportPostgresRepository) GetOpenOrdersSummary(ctx context.Context, tenantID, agingBucket, customerID string) (*port.OpenOrdersSummary, error) {
	rc := postgres.RLSContext{TenantID: tenantID}

	var s port.OpenOrdersSummary
	err := postgres.WithRLSInTransaction(ctx, r.db, rc, func(ctx context.Context, tx *sql.Tx) error {
		args := []interface{}{tenantID}
		n := 1
		customerClause := ""
		if customerID != "" {
			n++
			customerClause = " AND so.customer_id = $" + strconv.Itoa(n)
			args = append(args, customerID)
		}
		agingClause := agingConditionSQL(agingBucket)
		query := `
		SELECT
		    COUNT(so.id) AS total_open_orders,
		    COALESCE(SUM(so.total_amount - COALESCE(sp.total_paid, 0)), 0) AS total_outstanding_amount
		FROM sales_orders so
		LEFT JOIN (
		    SELECT sales_order_id, SUM(amount) AS total_paid
		    FROM sales_payments
		    GROUP BY sales_order_id
		) sp ON sp.sales_order_id = so.id
		WHERE so.tenant_id = $1
		  AND so.status IN ('CREATED', 'CONFIRMED')
		  AND (so.total_amount - COALESCE(sp.total_paid, 0)) > 0` + customerClause + agingClause
		return tx.QueryRowContext(ctx, query, args...).Scan(
			&s.TotalOpenOrders,
			&s.TotalOutstandingAmount,
		)
	})
	if err != nil {
		return nil, fmt.Errorf("get open orders summary: %w", err)
	}
	return &s, nil
}

// GetOpenOrdersBucketCounts devuelve conteos por bucket (sin filtrar) para badges en tabs.
func (r *ReportPostgresRepository) GetOpenOrdersBucketCounts(ctx context.Context, tenantID string) (*port.BucketCounts, error) {
	rc := postgres.RLSContext{TenantID: tenantID}

	var c port.BucketCounts
	err := postgres.WithRLSInTransaction(ctx, r.db, rc, func(ctx context.Context, tx *sql.Tx) error {
		query := `
		WITH open AS (
		    SELECT
		        so.id,
		        DATE_PART('day', NOW() - COALESCE(so.confirmed_at, so.created_at)) AS days
		    FROM sales_orders so
		    LEFT JOIN (
		        SELECT sales_order_id, SUM(amount) AS total_paid
		        FROM sales_payments
		        GROUP BY sales_order_id
		    ) sp ON sp.sales_order_id = so.id
		    WHERE so.tenant_id = $1
		      AND so.status IN ('CREATED', 'CONFIRMED')
		      AND (so.total_amount - COALESCE(sp.total_paid, 0)) > 0
		)
		SELECT
		    COUNT(*) FILTER (WHERE days <= 30) AS c_0_30,
		    COUNT(*) FILTER (WHERE days > 30 AND days <= 60) AS c_31_60,
		    COUNT(*) FILTER (WHERE days > 60 AND days <= 90) AS c_61_90,
		    COUNT(*) FILTER (WHERE days > 90) AS c_90_plus
		FROM open
	`
		return tx.QueryRowContext(ctx, query, tenantID).Scan(
			&c.Bucket0_30,
			&c.Bucket31_60,
			&c.Bucket61_90,
			&c.Bucket90Plus,
		)
	})
	if err != nil {
		return nil, fmt.Errorf("get open orders bucket counts: %w", err)
	}
	return &c, nil
}

func scanOpenOrderRows(rows *sql.Rows) ([]port.OpenOrderRow, error) {
	var result []port.OpenOrderRow
	for rows.Next() {
		var row port.OpenOrderRow
		var daysFloat float64
		var confirmedAt sql.NullTime
		err := rows.Scan(
			&row.OrderID,
			&row.CustomerID,
			&row.TotalAmount,
			&row.TotalPaid,
			&row.Remaining,
			&row.Status,
			&confirmedAt,
			&daysFloat,
		)
		if err != nil {
			return nil, fmt.Errorf("scan open order row: %w", err)
		}
		row.DaysSinceConfirmed = int(daysFloat)
		if confirmedAt.Valid {
			t := confirmedAt.Time
			row.ConfirmedAt = &t
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating open orders: %w", err)
	}
	return result, nil
}

// GetCustomerBalanceSummary devuelve resumen financiero de un cliente (solo órdenes abiertas).
func (r *ReportPostgresRepository) GetCustomerBalanceSummary(ctx context.Context, tenantID, customerID string) (*port.CustomerBalanceSummary, error) {
	rc := postgres.RLSContext{TenantID: tenantID}

	var s port.CustomerBalanceSummary
	err := postgres.WithRLSInTransaction(ctx, r.db, rc, func(ctx context.Context, tx *sql.Tx) error {
		query := `
		SELECT
		    COUNT(so.id) AS open_orders_count,
		    COALESCE(SUM(so.total_amount - COALESCE(sp.total_paid, 0)), 0) AS total_outstanding,
		    MAX(sp_last.payment_date) AS last_payment_at,
		    MAX(DATE_PART('day', NOW() - COALESCE(so.confirmed_at, so.created_at)))::int AS oldest_debt_days
		FROM sales_orders so
		LEFT JOIN (
		    SELECT sales_order_id, SUM(amount) AS total_paid
		    FROM sales_payments
		    GROUP BY sales_order_id
		) sp ON sp.sales_order_id = so.id
		LEFT JOIN (
		    SELECT sales_order_id, MAX(created_at) AS payment_date
		    FROM sales_payments
		    GROUP BY sales_order_id
		) sp_last ON sp_last.sales_order_id = so.id
		WHERE so.tenant_id = $1
		  AND so.customer_id = $2
		  AND so.status IN ('CREATED', 'CONFIRMED')
		  AND (so.total_amount - COALESCE(sp.total_paid, 0)) > 0
	`
		var lastAt sql.NullTime
		err := tx.QueryRowContext(ctx, query, tenantID, customerID).Scan(
			&s.OpenOrdersCount,
			&s.TotalOutstanding,
			&lastAt,
			&s.OldestDebtDays,
		)
		if err == sql.ErrNoRows {
			s = port.CustomerBalanceSummary{}
			return nil
		}
		if err != nil {
			return err
		}
		if lastAt.Valid {
			t := lastAt.Time
			s.LastPaymentAt = &t
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("get customer balance summary: %w", err)
	}
	return &s, nil
}

// GetCustomerBalanceBucketCounts devuelve conteos por bucket para las órdenes abiertas del cliente (para tabs).
func (r *ReportPostgresRepository) GetCustomerBalanceBucketCounts(ctx context.Context, tenantID, customerID string) (*port.BucketCounts, error) {
	rc := postgres.RLSContext{TenantID: tenantID}

	var c port.BucketCounts
	err := postgres.WithRLSInTransaction(ctx, r.db, rc, func(ctx context.Context, tx *sql.Tx) error {
		query := `
		WITH open AS (
		    SELECT so.id,
		           DATE_PART('day', NOW() - COALESCE(so.confirmed_at, so.created_at)) AS days
		    FROM sales_orders so
		    LEFT JOIN (
		        SELECT sales_order_id, SUM(amount) AS total_paid
		        FROM sales_payments
		        GROUP BY sales_order_id
		    ) sp ON sp.sales_order_id = so.id
		    WHERE so.tenant_id = $1 AND so.customer_id = $2
		      AND so.status IN ('CREATED', 'CONFIRMED')
		      AND (so.total_amount - COALESCE(sp.total_paid, 0)) > 0
		)
		SELECT
		    COUNT(*) FILTER (WHERE days <= 30) AS c_0_30,
		    COUNT(*) FILTER (WHERE days > 30 AND days <= 60) AS c_31_60,
		    COUNT(*) FILTER (WHERE days > 60 AND days <= 90) AS c_61_90,
		    COUNT(*) FILTER (WHERE days > 90) AS c_90_plus
		FROM open
	`
		return tx.QueryRowContext(ctx, query, tenantID, customerID).Scan(
			&c.Bucket0_30,
			&c.Bucket31_60,
			&c.Bucket61_90,
			&c.Bucket90Plus,
		)
	})
	if err != nil {
		return nil, fmt.Errorf("get customer balance bucket counts: %w", err)
	}
	return &c, nil
}

// GetCustomerOpenOrders devuelve órdenes abiertas del cliente con paginación y filtro opcional por aging_bucket.
func (r *ReportPostgresRepository) GetCustomerOpenOrders(
	ctx context.Context,
	tenantID, customerID string,
	limit, offset int,
	agingBucket, sort string,
) ([]port.OpenOrderRow, error) {
	rc := postgres.RLSContext{TenantID: tenantID}

	var result []port.OpenOrderRow
	err := postgres.WithRLSInTransaction(ctx, r.db, rc, func(ctx context.Context, tx *sql.Tx) error {
		agingClause := agingConditionSQL(agingBucket)
		query := openOrdersSelect + ` AND so.customer_id = $2` + agingClause +
			` ORDER BY ` + orderBySQL(sort) +
			fmt.Sprintf(" LIMIT $3 OFFSET $4")
		rows, err := tx.QueryContext(ctx, query, tenantID, customerID, limit, offset)
		if err != nil {
			return fmt.Errorf("get customer open orders: %w", err)
		}
		defer rows.Close()
		result, err = scanOpenOrderRows(rows)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetCustomerAging devuelve aging de un solo cliente (un row o vacío).
func (r *ReportPostgresRepository) GetCustomerAging(ctx context.Context, tenantID, customerID string) (*port.AgingRow, error) {
	rc := postgres.RLSContext{TenantID: tenantID}

	var row port.AgingRow
	err := postgres.WithRLSInTransaction(ctx, r.db, rc, func(ctx context.Context, tx *sql.Tx) error {
		query := `
		SELECT
		    so.customer_id,
		    SUM(CASE WHEN NOW() - COALESCE(so.confirmed_at, so.created_at) <= INTERVAL '30 days' THEN so.total_amount - COALESCE(sp.total_paid, 0) ELSE 0 END) AS bucket_0_30,
		    SUM(CASE WHEN NOW() - COALESCE(so.confirmed_at, so.created_at) > INTERVAL '30 days' AND NOW() - COALESCE(so.confirmed_at, so.created_at) <= INTERVAL '60 days' THEN so.total_amount - COALESCE(sp.total_paid, 0) ELSE 0 END) AS bucket_31_60,
		    SUM(CASE WHEN NOW() - COALESCE(so.confirmed_at, so.created_at) > INTERVAL '60 days' AND NOW() - COALESCE(so.confirmed_at, so.created_at) <= INTERVAL '90 days' THEN so.total_amount - COALESCE(sp.total_paid, 0) ELSE 0 END) AS bucket_61_90,
		    SUM(CASE WHEN NOW() - COALESCE(so.confirmed_at, so.created_at) > INTERVAL '90 days' THEN so.total_amount - COALESCE(sp.total_paid, 0) ELSE 0 END) AS bucket_90_plus
		FROM sales_orders so
		LEFT JOIN (SELECT sales_order_id, SUM(amount) AS total_paid FROM sales_payments GROUP BY sales_order_id) sp ON sp.sales_order_id = so.id
		WHERE so.tenant_id = $1 AND so.customer_id = $2 AND so.status IN ('CREATED', 'CONFIRMED')
		GROUP BY so.customer_id
		HAVING SUM(so.total_amount - COALESCE(sp.total_paid, 0)) > 0
	`
		err := tx.QueryRowContext(ctx, query, tenantID, customerID).Scan(
			&row.CustomerID,
			&row.Bucket0_30,
			&row.Bucket31_60,
			&row.Bucket61_90,
			&row.Bucket90Plus,
		)
		if err == sql.ErrNoRows {
			// Cliente sin deuda: devolver row en ceros
			row.CustomerID = customerID
			return nil
		}
		if err != nil {
			return err
		}
		row.Total = row.Bucket0_30 + row.Bucket31_60 + row.Bucket61_90 + row.Bucket90Plus
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("get customer aging: %w", err)
	}
	return &row, nil
}

// GetAgingReport devuelve aging por cliente (buckets 0-30, 31-60, 61-90, 90+ días).
func (r *ReportPostgresRepository) GetAgingReport(ctx context.Context, tenantID string) ([]port.AgingRow, error) {
	rc := postgres.RLSContext{TenantID: tenantID}

	var result []port.AgingRow
	err := postgres.WithRLSInTransaction(ctx, r.db, rc, func(ctx context.Context, tx *sql.Tx) error {
		query := `
		SELECT
		    so.customer_id,

		    SUM(
		        CASE
		            WHEN NOW() - COALESCE(so.confirmed_at, so.created_at) <= INTERVAL '30 days'
		            THEN so.total_amount - COALESCE(sp.total_paid, 0)
		            ELSE 0
		        END
		    ) AS bucket_0_30,

		    SUM(
		        CASE
		            WHEN NOW() - COALESCE(so.confirmed_at, so.created_at) > INTERVAL '30 days'
		             AND NOW() - COALESCE(so.confirmed_at, so.created_at) <= INTERVAL '60 days'
		            THEN so.total_amount - COALESCE(sp.total_paid, 0)
		            ELSE 0
		        END
		    ) AS bucket_31_60,

		    SUM(
		        CASE
		            WHEN NOW() - COALESCE(so.confirmed_at, so.created_at) > INTERVAL '60 days'
		             AND NOW() - COALESCE(so.confirmed_at, so.created_at) <= INTERVAL '90 days'
		            THEN so.total_amount - COALESCE(sp.total_paid, 0)
		            ELSE 0
		        END
		    ) AS bucket_61_90,

		    SUM(
		        CASE
		            WHEN NOW() - COALESCE(so.confirmed_at, so.created_at) > INTERVAL '90 days'
		            THEN so.total_amount - COALESCE(sp.total_paid, 0)
		            ELSE 0
		        END
		    ) AS bucket_90_plus

		FROM sales_orders so
		LEFT JOIN (
		    SELECT
		        sales_order_id,
		        SUM(amount) AS total_paid
		    FROM sales_payments
		    GROUP BY sales_order_id
		) sp ON sp.sales_order_id = so.id
		WHERE so.tenant_id = $1
		  AND so.status IN ('CREATED', 'CONFIRMED')
		GROUP BY so.customer_id
		HAVING SUM(so.total_amount - COALESCE(sp.total_paid, 0)) > 0
		ORDER BY bucket_90_plus DESC
	`
		rows, err := tx.QueryContext(ctx, query, tenantID)
		if err != nil {
			return fmt.Errorf("get aging report: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var row port.AgingRow
			err := rows.Scan(
				&row.CustomerID,
				&row.Bucket0_30,
				&row.Bucket31_60,
				&row.Bucket61_90,
				&row.Bucket90Plus,
			)
			if err != nil {
				return fmt.Errorf("scan aging row: %w", err)
			}
			row.Total = row.Bucket0_30 + row.Bucket31_60 + row.Bucket61_90 + row.Bucket90Plus
			result = append(result, row)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterating aging report: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

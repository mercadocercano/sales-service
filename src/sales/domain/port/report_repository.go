package port

import (
	"context"
	"time"
)

// OpenOrderRow representa una fila del reporte de órdenes abiertas (outstanding).
// Solo órdenes con status CREATED/CONFIRMED y remaining > 0.
type OpenOrderRow struct {
	OrderID            string     `json:"order_id"`
	CustomerID         string     `json:"customer_id"`
	TotalAmount        float64    `json:"total_amount"`
	TotalPaid          float64    `json:"total_paid"`
	Remaining          float64    `json:"remaining"`
	Status             string     `json:"status"`
	ConfirmedAt        *time.Time `json:"confirmed_at,omitempty"`
	DaysSinceConfirmed int        `json:"days_since_confirmed"`
}

// OpenOrdersSummary es el resumen (filtrado) del reporte open-orders.
type OpenOrdersSummary struct {
	TotalOpenOrders        int     `json:"total_open_orders"`
	TotalOutstandingAmount float64 `json:"total_outstanding_amount"`
}

// BucketCounts son los conteos por bucket de aging para badges en tabs UI.
// Siempre sin filtrar por aging_bucket (totales por bucket).
type BucketCounts struct {
	Bucket0_30   int `json:"0_30"`
	Bucket31_60  int `json:"31_60"`
	Bucket61_90  int `json:"61_90"`
	Bucket90Plus int `json:"90_plus"`
}

// CustomerBalanceSummary es el resumen de ficha financiera de un cliente.
type CustomerBalanceSummary struct {
	TotalOutstanding  float64    `json:"total_outstanding"`
	OpenOrdersCount   int        `json:"open_orders_count"`
	OldestDebtDays   int        `json:"oldest_debt_days"`
	LastPaymentAt    *time.Time `json:"last_payment_at,omitempty"`
}

// AgingRow representa el aging por cliente (cuentas por cobrar por bucket).
type AgingRow struct {
	CustomerID   string  `json:"customer_id"`
	Bucket0_30   float64 `json:"bucket_0_30"`
	Bucket31_60  float64 `json:"bucket_31_60"`
	Bucket61_90  float64 `json:"bucket_61_90"`
	Bucket90Plus float64 `json:"bucket_90_plus"`
	Total        float64 `json:"total"`
}

// ReportRepository expone consultas de reporting sobre sales_orders + sales_payments.
// No consulta ledger. Solo datos operativos para cashflow y aging.
//
// agingBucket: "" | "0_30" | "31_60" | "61_90" | "90_plus"
// sort: "" (remaining_desc) | "remaining_asc" | "days_desc" | "days_asc"
type ReportRepository interface {
	GetOpenOrders(ctx context.Context, tenantID string, limit, offset int, agingBucket, customerID, sort string) ([]OpenOrderRow, error)
	GetOpenOrdersSummary(ctx context.Context, tenantID string, agingBucket, customerID string) (*OpenOrdersSummary, error)
	GetOpenOrdersBucketCounts(ctx context.Context, tenantID string) (*BucketCounts, error)
	GetAgingReport(ctx context.Context, tenantID string) ([]AgingRow, error)

	GetCustomerBalanceSummary(ctx context.Context, tenantID, customerID string) (*CustomerBalanceSummary, error)
	GetCustomerBalanceBucketCounts(ctx context.Context, tenantID, customerID string) (*BucketCounts, error)
	GetCustomerOpenOrders(ctx context.Context, tenantID, customerID string, limit, offset int, agingBucket, sort string) ([]OpenOrderRow, error)
	GetCustomerAging(ctx context.Context, tenantID, customerID string) (*AgingRow, error)
}

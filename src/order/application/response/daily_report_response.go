package response

import (
	"time"

	"github.com/shopspring/decimal"
)

// DailyReportResponse representa el reporte diario de ventas
// HITO C - Reportes Diarios
type DailyReportResponse struct {
	Date               string          `json:"date"`                          // YYYY-MM-DD
	PosSalesCount      int             `json:"pos_sales_count"`               // Cantidad ventas POS
	OrdersCount        int             `json:"orders_count"`                  // Cantidad órdenes
	TotalTransactions  int             `json:"total_transactions"`            // pos + orders
	PosGrossTotal      decimal.Decimal `json:"pos_gross_total"`               // Suma total_amount
	PosDiscounts       decimal.Decimal `json:"pos_discounts"`                 // Suma discount_amount
	PosNetTotal        decimal.Decimal `json:"pos_net_total"`                 // Suma final_amount
	FirstTransactionAt *time.Time      `json:"first_transaction_at,omitempty"` // Primera venta del día
	LastTransactionAt  *time.Time      `json:"last_transaction_at,omitempty"`  // Última venta del día
}

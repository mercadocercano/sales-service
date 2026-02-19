package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"order/src/order/application/response"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// DailyReportUseCase caso de uso para reporte diario de ventas
// HITO C - Reportes Diarios
type DailyReportUseCase struct {
	db *sql.DB
}

// NewDailyReportUseCase crea una nueva instancia del caso de uso
func NewDailyReportUseCase(db *sql.DB) *DailyReportUseCase {
	return &DailyReportUseCase{
		db: db,
	}
}

// Execute genera el reporte diario para una fecha específica
// Ejecuta dos queries separadas y combina resultados en memoria
func (uc *DailyReportUseCase) Execute(ctx context.Context, tenantID uuid.UUID, date string) (*response.DailyReportResponse, error) {
	// ========================================================================
	// PASO 1: VALIDAR FORMATO DE FECHA (YYYY-MM-DD)
	// ========================================================================
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)
	}

	// ========================================================================
	// PASO 2: CALCULAR RANGO [from, to) - NO usar DATE(created_at)
	// ========================================================================
	// Importante: Usar >= from AND < to para aprovechar índice
	from := parsedDate // 2026-02-12 00:00:00
	to := parsedDate.AddDate(0, 0, 1) // 2026-02-13 00:00:00

	// ========================================================================
	// PASO 3: QUERY POS SALES (Agregaciones)
	// ========================================================================
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

	err = uc.db.QueryRowContext(ctx, queryPOS, tenantID, from, to).Scan(
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

	// ========================================================================
	// PASO 4: QUERY ORDERS (Solo count, sin amounts)
	// ========================================================================
	queryOrders := `
		SELECT COUNT(*)
		FROM orders
		WHERE tenant_id = $1
			AND created_at >= $2
			AND created_at < $3
	`

	var ordersCount int
	err = uc.db.QueryRowContext(ctx, queryOrders, tenantID, from, to).Scan(&ordersCount)
	if err != nil {
		return nil, fmt.Errorf("error querying orders: %w", err)
	}

	// ========================================================================
	// PASO 5: CONSTRUIR RESPONSE (Combinación en memoria)
	// ========================================================================
	resp := &response.DailyReportResponse{
		Date:              date,
		PosSalesCount:     posSalesCount,
		OrdersCount:       ordersCount,
		TotalTransactions: posSalesCount + ordersCount,
		PosGrossTotal:     grossTotal,
		PosDiscounts:      totalDiscounts,
		PosNetTotal:       netTotal,
	}

	// Agregar timestamps solo si existen ventas
	if firstSale.Valid {
		resp.FirstTransactionAt = &firstSale.Time
	}
	if lastSale.Valid {
		resp.LastTransactionAt = &lastSale.Time
	}

	return resp, nil
}

package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"sales/src/sales/domain/entity"
	"sales/src/sales/domain/port"

	"github.com/google/uuid"
)

// PosSalePostgresRepository implementa PosSaleRepository usando PostgreSQL
// Sin transacciones, sin lógica, solo insert y select
// Hito: POS-SALE-02.BE - Paso 2
type PosSalePostgresRepository struct {
	db *sql.DB
}

// NewPosSalePostgresRepository crea una nueva instancia del repositorio
func NewPosSalePostgresRepository(db *sql.DB) port.PosSaleRepository {
	return &PosSalePostgresRepository{
		db: db,
	}
}

// Create persiste una nueva venta POS con sus items (atomically)
// HITO B - Refactorizado para multi-item
func (r *PosSalePostgresRepository) Create(ctx context.Context, sale *entity.PosSale) error {
	// Iniciar transacción para garantizar atomicidad
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Insertar pos_sale (aggregate root)
	// HITO: POST /pos/sale devuelve DTO listo para imprimir
	querySale := `
		INSERT INTO pos_sales (
			id, tenant_id, customer_id, payment_method_id,
			total_amount, discount_amount, final_amount,
			amount_paid, change, currency, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`

	_, err = tx.ExecContext(ctx, querySale,
		sale.ID,
		sale.TenantID,
		sale.CustomerID, // NULL permitido
		sale.PaymentMethodID,
		sale.TotalAmount,
		sale.DiscountAmount,
		sale.FinalAmount,
		sale.AmountPaid,
		sale.Change,
		sale.Currency,
		sale.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("error creating pos_sale: %w", err)
	}

	// 2. Insertar pos_sale_items (entities)
	queryItem := `
		INSERT INTO pos_sale_items (
			id, pos_sale_id, sku, product_name,
			quantity, unit_price, subtotal, stock_entry_id, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, NOW()
		)
	`

	for _, item := range sale.Items {
		_, err = tx.ExecContext(ctx, queryItem,
			item.ID,
			item.PosSaleID,
			item.SKU,
			item.ProductName,
			item.Quantity,
			item.UnitPrice,
			item.Subtotal,
			item.StockEntryID,
		)

		if err != nil {
			return fmt.Errorf("error creating pos_sale_item for SKU %s: %w", item.SKU, err)
		}
	}

	// Commit transacción
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	return nil
}

// ListByTenant retorna todas las ventas POS de un tenant CON sus items
// HITO B - Refactorizado para cargar items
func (r *PosSalePostgresRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*entity.PosSale, error) {
	// 1. Obtener pos_sales
	// HITO: POST /pos/sale devuelve DTO listo para imprimir
	querySales := `
		SELECT 
			id, tenant_id, customer_id, payment_method_id,
			total_amount, discount_amount, final_amount,
			amount_paid, change, currency, created_at
		FROM pos_sales
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, querySales, tenantID)
	if err != nil {
		return nil, fmt.Errorf("error querying pos_sales: %w", err)
	}
	defer rows.Close()

	var sales []*entity.PosSale

	for rows.Next() {
		sale := &entity.PosSale{}
		err := rows.Scan(
			&sale.ID,
			&sale.TenantID,
			&sale.CustomerID,
			&sale.PaymentMethodID,
			&sale.TotalAmount,
			&sale.DiscountAmount,
			&sale.FinalAmount,
			&sale.AmountPaid,
			&sale.Change,
			&sale.Currency,
			&sale.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning pos_sale: %w", err)
		}
		sales = append(sales, sale)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pos_sales: %w", err)
	}

	// 2. Obtener items para cada venta (N+1 query - simple para HITO B)
	queryItems := `
		SELECT 
			id, pos_sale_id, sku, product_name,
			quantity, unit_price, subtotal, stock_entry_id
		FROM pos_sale_items
		WHERE pos_sale_id = $1
		ORDER BY created_at
	`

	for _, sale := range sales {
		itemRows, err := r.db.QueryContext(ctx, queryItems, sale.ID)
		if err != nil {
			return nil, fmt.Errorf("error querying pos_sale_items: %w", err)
		}

		var items []entity.PosSaleItem

		for itemRows.Next() {
			item := entity.PosSaleItem{}
			err := itemRows.Scan(
				&item.ID,
				&item.PosSaleID,
				&item.SKU,
				&item.ProductName,
				&item.Quantity,
				&item.UnitPrice,
				&item.Subtotal,
				&item.StockEntryID,
			)
			if err != nil {
				itemRows.Close()
				return nil, fmt.Errorf("error scanning pos_sale_item: %w", err)
			}
			items = append(items, item)
		}

		itemRows.Close()

		if err = itemRows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating pos_sale_items: %w", err)
		}

		sale.Items = items
	}

	return sales, nil
}

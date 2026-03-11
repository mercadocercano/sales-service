package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"sales/src/sales/domain/entity"
)

// OrderPostgresRepository implementa OrderRepository usando PostgreSQL
type OrderPostgresRepository struct {
	db *sql.DB
}

// NewOrderPostgresRepository crea una nueva instancia del repositorio
func NewOrderPostgresRepository(db *sql.DB) *OrderPostgresRepository {
	return &OrderPostgresRepository{
		db: db,
	}
}

// Save persiste una orden con sus items en la base de datos (DDD Aggregate)
func (r *OrderPostgresRepository) Save(ctx context.Context, order *entity.Order) error {
	// Iniciar transacción para garantizar atomicidad del aggregate
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Insertar orden (aggregate root)
	// Calcular total desde los items con snapshots
	totalAmount := order.CalculateTotal()
	
	queryOrder := `
		INSERT INTO sales_orders (
			id, tenant_id, customer_id, status, total_amount, created_at, updated_at, version
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
	`

	_, err = tx.ExecContext(ctx, queryOrder,
		order.OrderID,
		order.TenantID,
		order.CustomerID,
		order.Status,
		totalAmount, // total calculado desde snapshots
		order.CreatedAt,
		order.CreatedAt, // updated_at
		1, // version
	)

	if err != nil {
		return fmt.Errorf("error saving order: %w", err)
	}

	// 2. Insertar items (entities dentro del aggregate) con snapshots
	queryItem := `
		INSERT INTO sales_order_items (
			id, sales_order_id, sku, quantity, unit_price, product_snapshot, variant_snapshot, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
	`

	for _, item := range order.Items {
		_, err = tx.ExecContext(ctx, queryItem,
			item.ItemID,
			order.OrderID,
			item.SKU,
			item.Quantity,
			item.UnitPrice,
			item.ProductSnapshot,
			item.VariantSnapshot,
			order.CreatedAt,
		)

		if err != nil {
			return fmt.Errorf("error saving order item: %w", err)
		}
	}

	// Commit transacción
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	return nil
}

// FindByID busca una orden con sus items por su ID (DDD: load aggregate)
func (r *OrderPostgresRepository) FindByID(ctx context.Context, orderID, tenantID string) (*entity.Order, error) {
	// 1. Buscar orden (aggregate root)
	queryOrder := `
		SELECT id, tenant_id, customer_id, status, order_number, total_amount, created_at
		FROM sales_orders
		WHERE id = $1 AND tenant_id = $2
	`

	var totalAmount float64
	order := &entity.Order{}
	err := r.db.QueryRowContext(ctx, queryOrder, orderID, tenantID).Scan(
		&order.OrderID,
		&order.TenantID,
		&order.CustomerID,
		&order.Status,
		&order.OrderNumber,
		&totalAmount,
		&order.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("order not found")
	}
	if err != nil {
		return nil, fmt.Errorf("error finding order: %w", err)
	}

	// 2. Cargar items (entities dentro del aggregate) con snapshots
	queryItems := `
		SELECT id, sales_order_id, sku, quantity, unit_price, product_snapshot, variant_snapshot
		FROM sales_order_items
		WHERE sales_order_id = $1
		ORDER BY created_at
	`

	rows, err := r.db.QueryContext(ctx, queryItems, orderID)
	if err != nil {
		return nil, fmt.Errorf("error finding order items: %w", err)
	}
	defer rows.Close()

	var items []entity.OrderItem
	for rows.Next() {
		var item entity.OrderItem
		var quantity float64 // DB usa NUMERIC
		err := rows.Scan(
			&item.ItemID,
			&item.OrderID,
			&item.SKU,
			&quantity,
			&item.UnitPrice,
			&item.ProductSnapshot,
			&item.VariantSnapshot,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning order item: %w", err)
		}
		item.Quantity = int(quantity) // Cast de NUMERIC a int
		items = append(items, item)
	}

	order.Items = items

	return order, nil
}

// Confirm actualiza el estado de una orden a CONFIRMED, setea confirmed_at y updated_at
func (r *OrderPostgresRepository) Confirm(ctx context.Context, orderID, tenantID string) error {
	query := `
		UPDATE sales_orders
		SET status = 'CONFIRMED', confirmed_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND status = 'CREATED'
	`

	result, err := r.db.ExecContext(ctx, query, orderID, tenantID)
	if err != nil {
		return fmt.Errorf("error confirming order: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("order not found or not in CREATED state")
	}

	return nil
}

// UpdateOrderNumber actualiza el número de orden (HITO v0.4)
func (r *OrderPostgresRepository) UpdateOrderNumber(ctx context.Context, orderID, tenantID string, orderNumber int) error {
	query := `
		UPDATE sales_orders
		SET order_number = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3
	`

	_, err := r.db.ExecContext(ctx, query, orderNumber, orderID, tenantID)
	if err != nil {
		return fmt.Errorf("error updating order number: %w", err)
	}

	return nil
}

// Cancel actualiza el estado de una orden a CANCELED
func (r *OrderPostgresRepository) Cancel(ctx context.Context, orderID, tenantID string) error {
	query := `
		UPDATE sales_orders
		SET status = 'CANCELED'
		WHERE id = $1 AND tenant_id = $2 AND status = 'CONFIRMED'
	`

	result, err := r.db.ExecContext(ctx, query, orderID, tenantID)
	if err != nil {
		return fmt.Errorf("error canceling order: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("order not found or not in CONFIRMED state")
	}

	return nil
}

// GetOrderFinancialSummary devuelve el resumen financiero de una orden (solo sales_orders + sales_payments)
func (r *OrderPostgresRepository) GetOrderFinancialSummary(ctx context.Context, orderID, tenantID string) (*entity.OrderFinancialSummary, error) {
	query := `
		SELECT
			so.id,
			so.tenant_id,
			so.total_amount,
			COALESCE(SUM(sp.amount), 0) AS total_paid,
			so.total_amount - COALESCE(SUM(sp.amount), 0) AS remaining,
			so.status,
			COUNT(sp.id)::int AS payment_count,
			MAX(sp.created_at) AS last_payment_at
		FROM sales_orders so
		LEFT JOIN sales_payments sp ON sp.sales_order_id = so.id AND sp.tenant_id = so.tenant_id
		WHERE so.id = $1 AND so.tenant_id = $2
		GROUP BY so.id, so.tenant_id, so.total_amount, so.status
	`
	var s entity.OrderFinancialSummary
	var lastAt sql.NullTime
	err := r.db.QueryRowContext(ctx, query, orderID, tenantID).Scan(
		&s.OrderID,
		&s.TenantID,
		&s.TotalAmount,
		&s.TotalPaid,
		&s.Remaining,
		&s.Status,
		&s.PaymentCount,
		&lastAt,
	)
	if err == sql.ErrNoRows {
		return nil, entity.ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error getting order financial summary: %w", err)
	}
	if lastAt.Valid {
		s.LastPaymentAt = &lastAt.Time
	}
	return &s, nil
}

// List retorna todas las órdenes de un tenant con paginación y filtro opcional por status
func (r *OrderPostgresRepository) List(ctx context.Context, tenantID string, page, pageSize int, statusFilter string) ([]*entity.Order, int, error) {
	// statusFilter: "" (todos) | CREATED | CONFIRMED | PAID | CANCELED | OPEN (CREATED+CONFIRMED = remaining > 0)
	// financial_status=OPEN -> OPEN, financial_status=PAID -> PAID
	countArgs := []interface{}{tenantID}
	queryCount := `SELECT COUNT(*) FROM sales_orders WHERE tenant_id = $1`
	if statusFilter == "OPEN" {
		queryCount += ` AND status IN ('CREATED', 'CONFIRMED')`
	} else if statusFilter != "" {
		queryCount += ` AND status = $2`
		countArgs = append(countArgs, statusFilter)
	}
	var totalCount int
	err := r.db.QueryRowContext(ctx, queryCount, countArgs...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting orders: %w", err)
	}

	// 2. Calcular offset
	offset := (page - 1) * pageSize

	// 3. Obtener órdenes paginadas
	queryOrders := `
		SELECT id, tenant_id, status, created_at
		FROM sales_orders
		WHERE tenant_id = $1
	`
	queryArgs := []interface{}{tenantID}
	if statusFilter == "OPEN" {
		queryOrders += ` AND status IN ('CREATED', 'CONFIRMED')`
	} else if statusFilter != "" {
		queryOrders += ` AND status = $2`
		queryArgs = append(queryArgs, statusFilter)
	}
	queryArgs = append(queryArgs, pageSize, offset)
	queryOrders += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, len(queryArgs)-1, len(queryArgs))

	rows, err := r.db.QueryContext(ctx, queryOrders, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("error listing orders: %w", err)
	}
	defer rows.Close()

	var orders []*entity.Order
	for rows.Next() {
		order := &entity.Order{}
		err := rows.Scan(
			&order.OrderID,
			&order.TenantID,
			&order.Status,
			&order.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("error scanning order: %w", err)
		}

		// 4. Cargar items de cada orden con snapshots
		queryItems := `
			SELECT id, sales_order_id, sku, quantity, unit_price, product_snapshot, variant_snapshot
			FROM sales_order_items
			WHERE sales_order_id = $1
			ORDER BY created_at
		`

		itemRows, err := r.db.QueryContext(ctx, queryItems, order.OrderID)
		if err != nil {
			return nil, 0, fmt.Errorf("error loading items for order %s: %w", order.OrderID, err)
		}

		var items []entity.OrderItem
		for itemRows.Next() {
			var item entity.OrderItem
			var quantity float64 // DB usa NUMERIC
			err := itemRows.Scan(
				&item.ItemID,
				&item.OrderID,
				&item.SKU,
				&quantity,
				&item.UnitPrice,
				&item.ProductSnapshot,
				&item.VariantSnapshot,
			)
			if err != nil {
				itemRows.Close()
				return nil, 0, fmt.Errorf("error scanning order item: %w", err)
			}
			item.Quantity = int(quantity) // Cast de NUMERIC a int
			items = append(items, item)
		}
		itemRows.Close()

		order.Items = items
		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating orders: %w", err)
	}

	return orders, totalCount, nil
}

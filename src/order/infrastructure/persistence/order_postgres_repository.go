package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"order/src/order/domain/entity"
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
	queryOrder := `
		INSERT INTO orders (
			order_id, tenant_id, status, created_at
		) VALUES (
			$1, $2, $3, $4
		)
	`

	_, err = tx.ExecContext(ctx, queryOrder,
		order.OrderID,
		order.TenantID,
		order.Status,
		order.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("error saving order: %w", err)
	}

	// 2. Insertar items (entities dentro del aggregate) con snapshots
	queryItem := `
		INSERT INTO order_items (
			item_id, order_id, sku, quantity, product_snapshot, variant_snapshot, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)
	`

	for _, item := range order.Items {
		_, err = tx.ExecContext(ctx, queryItem,
			item.ItemID,
			order.OrderID,
			item.SKU,
			item.Quantity,
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
		SELECT order_id, tenant_id, status, created_at
		FROM orders
		WHERE order_id = $1 AND tenant_id = $2
	`

	order := &entity.Order{}
	err := r.db.QueryRowContext(ctx, queryOrder, orderID, tenantID).Scan(
		&order.OrderID,
		&order.TenantID,
		&order.Status,
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
		SELECT item_id, order_id, sku, quantity, product_snapshot, variant_snapshot
		FROM order_items
		WHERE order_id = $1
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
		err := rows.Scan(
			&item.ItemID,
			&item.OrderID,
			&item.SKU,
			&item.Quantity,
			&item.ProductSnapshot,
			&item.VariantSnapshot,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning order item: %w", err)
		}
		items = append(items, item)
	}

	order.Items = items

	return order, nil
}

// Confirm actualiza el estado de una orden a CONFIRMED
func (r *OrderPostgresRepository) Confirm(ctx context.Context, orderID, tenantID string) error {
	query := `
		UPDATE orders
		SET status = 'CONFIRMED'
		WHERE order_id = $1 AND tenant_id = $2 AND status = 'CREATED'
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

// Cancel actualiza el estado de una orden a CANCELED
func (r *OrderPostgresRepository) Cancel(ctx context.Context, orderID, tenantID string) error {
	query := `
		UPDATE orders
		SET status = 'CANCELED'
		WHERE order_id = $1 AND tenant_id = $2 AND status = 'CONFIRMED'
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

// List retorna todas las órdenes de un tenant con paginación
func (r *OrderPostgresRepository) List(ctx context.Context, tenantID string, page, pageSize int) ([]*entity.Order, int, error) {
	// 1. Contar total de órdenes
	var totalCount int
	queryCount := `
		SELECT COUNT(*)
		FROM orders
		WHERE tenant_id = $1
	`
	err := r.db.QueryRowContext(ctx, queryCount, tenantID).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting orders: %w", err)
	}

	// 2. Calcular offset
	offset := (page - 1) * pageSize

	// 3. Obtener órdenes paginadas
	queryOrders := `
		SELECT order_id, tenant_id, status, created_at
		FROM orders
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, queryOrders, tenantID, pageSize, offset)
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
			SELECT item_id, order_id, sku, quantity, product_snapshot, variant_snapshot
			FROM order_items
			WHERE order_id = $1
			ORDER BY created_at
		`

		itemRows, err := r.db.QueryContext(ctx, queryItems, order.OrderID)
		if err != nil {
			return nil, 0, fmt.Errorf("error loading items for order %s: %w", order.OrderID, err)
		}

		var items []entity.OrderItem
		for itemRows.Next() {
			var item entity.OrderItem
			err := itemRows.Scan(
				&item.ItemID,
				&item.OrderID,
				&item.SKU,
				&item.Quantity,
				&item.ProductSnapshot,
				&item.VariantSnapshot,
			)
			if err != nil {
				itemRows.Close()
				return nil, 0, fmt.Errorf("error scanning order item: %w", err)
			}
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

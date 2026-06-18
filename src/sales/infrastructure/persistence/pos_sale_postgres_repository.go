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

	// 0. E18 Tramo B: asignar número de comprobante INTERNO (no fiscal) de forma
	// atómica con la creación de la venta. El número se toma de document_sequences
	// (POS_SALE) con SELECT ... FOR UPDATE dentro de ESTA transacción: si el INSERT
	// de pos_sales falla, el rollback revierte también el incremento de la secuencia,
	// evitando saltos. El lock de fila serializa a los concurrentes del mismo tenant.
	saleNumber, err := r.nextPosSaleNumberTx(ctx, tx, sale.TenantID)
	if err != nil {
		return fmt.Errorf("error assigning sale_number: %w", err)
	}
	sale.SaleNumber = &saleNumber

	// 0.b E18 Tramo A: asociación venta↔caja en modo DEGRADADO (ADR-003 §4). Solo se
	// asocia si la sesión pedida está ABIERTA para el tenant; el FOR UPDATE serializa
	// contra un cierre concurrente (si el cierre ya commiteó, la venta la verá no-open y
	// procede con NULL — nunca se pierde, no se fuerza reapertura porque el cierre es
	// monolítico atómico). Si no viene sesión o no está abierta ⇒ FK NULL.
	var cashSessionID interface{} = nil
	if sale.CashRegisterSessionID != nil {
		var openID uuid.UUID
		lockErr := tx.QueryRowContext(ctx,
			`SELECT id FROM cash_register_sessions WHERE id=$1 AND tenant_id=$2 AND status='open' FOR UPDATE`,
			*sale.CashRegisterSessionID, sale.TenantID,
		).Scan(&openID)
		switch lockErr {
		case nil:
			cashSessionID = openID
		case sql.ErrNoRows:
			sale.CashRegisterSessionID = nil // degradado: la sesión no está abierta
		default:
			return fmt.Errorf("error validating cash session: %w", lockErr)
		}
	}

	// 1. Insertar pos_sale (aggregate root)
	// HITO: POST /pos/sale devuelve DTO listo para imprimir
	querySale := `
		INSERT INTO pos_sales (
			id, tenant_id, customer_id, payment_method_id,
			total_amount, discount_amount, final_amount,
			amount_paid, change, currency, sale_number, cash_register_session_id, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
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
		saleNumber,
		cashSessionID,
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

// nextPosSaleNumberTx asigna el siguiente número correlativo de comprobante POS
// para el tenant, DENTRO de la transacción recibida.
//
// Concurrencia/atomicidad (E18 Tramo B):
//   - Asegura la fila de secuencia con INSERT ... ON CONFLICT DO NOTHING. No es un
//     UPSERT defensivo de negocio: la fila es un contador y su valor inicial 0 es
//     semánticamente correcto para un tenant que aún no emitió comprobantes.
//   - Toma la fila con SELECT ... FOR UPDATE: serializa a los concurrentes del mismo
//     tenant/document_type. El segundo espera hasta que el primero haga COMMIT/ROLLBACK.
//   - Incrementa con un único UPDATE ... RETURNING dentro de la misma tx, de modo que
//     el número y la venta se confirman o se revierten juntos (sin saltos por fallo).
func (r *PosSalePostgresRepository) nextPosSaleNumberTx(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID) (int, error) {
	const documentType = "POS_SALE"

	// Garantizar la existencia de la fila de secuencia para este tenant.
	ensure := `
		INSERT INTO document_sequences (id, tenant_id, document_type, current_number, version, updated_at)
		VALUES (gen_random_uuid(), $1, $2, 0, 1, NOW())
		ON CONFLICT DO NOTHING
	`
	if _, err := tx.ExecContext(ctx, ensure, tenantID, documentType); err != nil {
		return 0, fmt.Errorf("error ensuring sequence row: %w", err)
	}

	// Bloquear la fila y leer el número actual (serializa concurrentes del mismo tenant).
	lock := `
		SELECT current_number
		FROM document_sequences
		WHERE tenant_id = $1 AND document_type = $2
		FOR UPDATE
	`
	var current int
	if err := tx.QueryRowContext(ctx, lock, tenantID, documentType).Scan(&current); err != nil {
		return 0, fmt.Errorf("error locking sequence row: %w", err)
	}

	// Incrementar atómicamente y devolver el nuevo número.
	next := current + 1
	upd := `
		UPDATE document_sequences
		SET current_number = $1, version = version + 1, updated_at = NOW()
		WHERE tenant_id = $2 AND document_type = $3
	`
	if _, err := tx.ExecContext(ctx, upd, next, tenantID, documentType); err != nil {
		return 0, fmt.Errorf("error incrementing sequence: %w", err)
	}

	return next, nil
}

// GetByID retorna el detalle completo de una venta POS del tenant (con items).
// E18 Tramo B: el filtro por tenant_id va en el WHERE — una venta de otro tenant
// devuelve ErrPosSaleNotFound (no se distingue de inexistente para no filtrar info).
func (r *PosSalePostgresRepository) GetByID(ctx context.Context, tenantID, saleID uuid.UUID) (*entity.PosSale, error) {
	querySale := `
		SELECT
			id, tenant_id, customer_id, payment_method_id,
			total_amount, discount_amount, final_amount,
			amount_paid, change, currency, sale_number, created_at
		FROM pos_sales
		WHERE id = $1 AND tenant_id = $2
	`

	sale := &entity.PosSale{}
	err := r.db.QueryRowContext(ctx, querySale, saleID, tenantID).Scan(
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
		&sale.SaleNumber,
		&sale.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, entity.ErrPosSaleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error querying pos_sale: %w", err)
	}

	// Cargar items del aggregate
	queryItems := `
		SELECT
			id, pos_sale_id, sku, product_name,
			quantity, unit_price, subtotal, stock_entry_id
		FROM pos_sale_items
		WHERE pos_sale_id = $1
		ORDER BY created_at
	`
	rows, err := r.db.QueryContext(ctx, queryItems, sale.ID)
	if err != nil {
		return nil, fmt.Errorf("error querying pos_sale_items: %w", err)
	}
	defer rows.Close()

	var items []entity.PosSaleItem
	for rows.Next() {
		item := entity.PosSaleItem{}
		if err := rows.Scan(
			&item.ID,
			&item.PosSaleID,
			&item.SKU,
			&item.ProductName,
			&item.Quantity,
			&item.UnitPrice,
			&item.Subtotal,
			&item.StockEntryID,
		); err != nil {
			return nil, fmt.Errorf("error scanning pos_sale_item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pos_sale_items: %w", err)
	}
	sale.Items = items

	return sale, nil
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

// CountByTenant cuenta las ventas POS de un tenant
func (r *PosSalePostgresRepository) CountByTenant(ctx context.Context, tenantID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM pos_sales WHERE tenant_id = $1`
	var count int
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("error counting pos_sales: %w", err)
	}
	return count, nil
}

// CountSalesTodayByTenant cuenta ventas del tenant en el día actual (UTC)
// H8 DAT: Si count == 1 tras Create, es la primera venta del día
func (r *PosSalePostgresRepository) CountSalesTodayByTenant(ctx context.Context, tenantID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*) FROM pos_sales
		WHERE tenant_id = $1 AND created_at >= CURRENT_DATE
	`
	var count int
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("error counting pos_sales today: %w", err)
	}
	return count, nil
}

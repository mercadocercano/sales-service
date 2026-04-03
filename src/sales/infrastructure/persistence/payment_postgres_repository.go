package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"sales/src/sales/domain/entity"
	"sales/src/sales/domain/port"

	"github.com/google/uuid"
)

// PaymentPostgresRepository implementa port.PaymentRepository con PostgreSQL
type PaymentPostgresRepository struct {
	db *sql.DB
}

// NewPaymentPostgresRepository crea una nueva instancia
func NewPaymentPostgresRepository(db *sql.DB) *PaymentPostgresRepository {
	return &PaymentPostgresRepository{db: db}
}

// RegisterPayment registra un pago con protección transaccional contra race conditions
func (r *PaymentPostgresRepository) RegisterPayment(
	ctx context.Context,
	tenantID, orderID string,
	amount float64,
) (*port.PaymentRegistration, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. SELECT FOR UPDATE sobre la orden (row lock)
	var orderStatus string
	var orderTotal float64
	var orderCustomerID string

	err = tx.QueryRowContext(ctx, `
		SELECT status, total_amount, customer_id
		FROM sales_orders
		WHERE id = $1 AND tenant_id = $2
		FOR UPDATE
	`, orderID, tenantID).Scan(&orderStatus, &orderTotal, &orderCustomerID)

	if err == sql.ErrNoRows {
		return nil, entity.ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to lock order: %w", err)
	}

	// 2. Validar estado CONFIRMED
	if orderStatus != string(entity.OrderStatusConfirmed) {
		return nil, fmt.Errorf("order must be confirmed (current: %s)", orderStatus)
	}

	// 3. Calcular total pagado
	var totalPaid sql.NullFloat64
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount), 0)
		FROM sales_payments
		WHERE sales_order_id = $1 AND tenant_id = $2
	`, orderID, tenantID).Scan(&totalPaid)
	if err != nil {
		return nil, fmt.Errorf("failed to sum payments: %w", err)
	}

	currentPaid := totalPaid.Float64

	// 4. Validar que no exceda el total
	if currentPaid+amount > orderTotal {
		return nil, fmt.Errorf(
			"payment exceeds order total (paid: %.2f, new: %.2f, total: %.2f)",
			currentPaid, amount, orderTotal,
		)
	}

	// 5. Insertar pago
	paymentID := uuid.New().String()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO sales_payments (id, tenant_id, sales_order_id, amount, currency)
		VALUES ($1, $2, $3, $4, $5)
	`, paymentID, tenantID, orderID, amount, "ARS")
	if err != nil {
		return nil, fmt.Errorf("failed to insert payment: %w", err)
	}

	// 6. Transición automática a PAID cuando remaining == 0
	finalStatus := orderStatus
	newTotalPaid := currentPaid + amount
	if newTotalPaid >= orderTotal-1e-9 {
		_, err = tx.ExecContext(ctx,
			`UPDATE sales_orders SET status = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3`,
			string(entity.OrderStatusPaid), orderID, tenantID)
		if err != nil {
			return nil, fmt.Errorf("failed to update order status to PAID: %w", err)
		}
		finalStatus = string(entity.OrderStatusPaid)
	}

	// 7. COMMIT
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &port.PaymentRegistration{
		PaymentID:   paymentID,
		OrderStatus: finalStatus,
		CustomerID:  orderCustomerID,
	}, nil
}

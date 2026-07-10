package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/hornosg/go-shared/infrastructure/postgres"

	"sales/src/sales/domain/port"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CreditPostgresRepository implementa port.CreditRepository con PostgreSQL
type CreditPostgresRepository struct {
	db *sql.DB
}

// NewCreditPostgresRepository crea una nueva instancia
func NewCreditPostgresRepository(db *sql.DB) *CreditPostgresRepository {
	return &CreditPostgresRepository{db: db}
}

// CreateCredit crea un crédito a cuenta del cliente
// PLAT-E25 T8: corre dentro de WithRLSInTransaction (SET LOCAL app.tenant_id) en vez de
// BeginTx/Commit manual — mismo criterio que T4/T5/T6/T7.
func (r *CreditPostgresRepository) CreateCredit(
	ctx context.Context,
	tenantID, customerID string,
	amount decimal.Decimal,
) (string, error) {
	rc := postgres.RLSContext{TenantID: tenantID}

	creditID := uuid.New().String()
	amountStr := amount.String()

	err := postgres.WithRLSInTransaction(ctx, r.db, rc, func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO customer_credits (
				id, tenant_id, customer_id,
				total_amount, applied_amount, remaining_amount,
				status, created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, 0, $4, 'AVAILABLE', NOW(), NOW())
		`, creditID, tenantID, customerID, amountStr)
		if err != nil {
			return fmt.Errorf("failed to insert customer credit: %w", err)
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	log.Printf("Credit created: %s customer %s amount %s", creditID, customerID, amountStr)
	return creditID, nil
}

// ApplyCredit aplica un crédito (parcial o total) a una orden
// PLAT-E25 T8: corre dentro de WithRLSInTransaction (SET LOCAL app.tenant_id) en vez de
// BeginTx/Commit manual — mismo criterio que T4/T5/T6/T7. Los locks FOR UPDATE conservan
// su semántica: SetRLSContextTx solo agrega un SET LOCAL antes de fn, no cambia el nivel
// de aislamiento de la tx (mismo punto de riesgo verificado en T7).
func (r *CreditPostgresRepository) ApplyCredit(
	ctx context.Context,
	tenantID, creditID, orderID string,
	amount decimal.Decimal,
) (*port.CreditApplication, error) {
	rc := postgres.RLSContext{TenantID: tenantID}

	var result *port.CreditApplication
	err := postgres.WithRLSInTransaction(ctx, r.db, rc, func(ctx context.Context, tx *sql.Tx) error {
		// 1. Lock Order (siempre primero)
		var orderTotal float64
		var orderCustomerID string
		var orderStatus string
		err := tx.QueryRowContext(ctx, `
			SELECT customer_id, total_amount, status
			FROM sales_orders
			WHERE id = $1 AND tenant_id = $2
			FOR UPDATE
		`, orderID, tenantID).Scan(&orderCustomerID, &orderTotal, &orderStatus)
		if err == sql.ErrNoRows {
			return fmt.Errorf("order not found")
		}
		if err != nil {
			return fmt.Errorf("failed to lock order: %w", err)
		}
		if orderStatus != "CREATED" && orderStatus != "CONFIRMED" {
			return fmt.Errorf("order must be CREATED or CONFIRMED (current: %s)", orderStatus)
		}

		// 2. Lock Credit (siempre segundo)
		var creditCustomerID string
		var creditTotal, creditApplied, creditRemaining float64
		var creditStatus string
		err = tx.QueryRowContext(ctx, `
			SELECT customer_id, total_amount, applied_amount, remaining_amount, status
			FROM customer_credits
			WHERE id = $1 AND tenant_id = $2
			FOR UPDATE
		`, creditID, tenantID).Scan(&creditCustomerID, &creditTotal, &creditApplied, &creditRemaining, &creditStatus)
		if err == sql.ErrNoRows {
			return fmt.Errorf("credit not found")
		}
		if err != nil {
			return fmt.Errorf("failed to lock credit: %w", err)
		}
		if creditStatus == "FULLY_APPLIED" {
			return fmt.Errorf("credit is fully applied")
		}

		amountF, _ := amount.Float64()
		if creditRemaining < amountF {
			return fmt.Errorf("credit remaining (%.2f) is less than amount (%.2f)", creditRemaining, amountF)
		}

		// 3. Cross validations
		if creditCustomerID != orderCustomerID {
			return fmt.Errorf("credit does not belong to the same customer as the order")
		}

		// 4. Order remaining
		var sumPayments, sumApplications sql.NullFloat64
		err = tx.QueryRowContext(ctx, `
			SELECT
				(SELECT COALESCE(SUM(amount), 0) FROM sales_payments WHERE sales_order_id = $1 AND tenant_id = $2),
				(SELECT COALESCE(SUM(amount), 0) FROM customer_credit_applications WHERE sales_order_id = $1)
		`, orderID, tenantID).Scan(&sumPayments, &sumApplications)
		if err != nil {
			return fmt.Errorf("failed to sum order paid: %w", err)
		}
		paid := sumPayments.Float64 + sumApplications.Float64
		orderRemaining := orderTotal - paid
		if orderRemaining < amountF {
			return fmt.Errorf("order remaining (%.2f) is less than amount (%.2f)", orderRemaining, amountF)
		}

		// 5. Insert application
		applicationID := uuid.New().String()
		_, err = tx.ExecContext(ctx, `
			INSERT INTO customer_credit_applications (id, tenant_id, customer_credit_id, sales_order_id, amount, created_at)
			VALUES ($1, $2, $3, $4, $5, NOW())
		`, applicationID, tenantID, creditID, orderID, amount.String())
		if err != nil {
			return fmt.Errorf("failed to insert application: %w", err)
		}

		// 6. Update credit
		newApplied := creditApplied + amountF
		newRemaining := creditTotal - newApplied
		newStatus := "AVAILABLE"
		if newRemaining <= 0 {
			newStatus = "FULLY_APPLIED"
		}
		_, err = tx.ExecContext(ctx, `
			UPDATE customer_credits
			SET applied_amount = $1, remaining_amount = $2, status = $3, updated_at = NOW()
			WHERE id = $4 AND tenant_id = $5
		`, newApplied, newRemaining, newStatus, creditID, tenantID)
		if err != nil {
			return fmt.Errorf("failed to update credit: %w", err)
		}

		// 7. Update order to PAID if fully covered
		if orderRemaining-amountF <= 1e-9 {
			_, err = tx.ExecContext(ctx, `
				UPDATE sales_orders SET status = 'PAID', updated_at = NOW() WHERE id = $1 AND tenant_id = $2
			`, orderID, tenantID)
			if err != nil {
				return fmt.Errorf("failed to update order to PAID: %w", err)
			}
		}

		log.Printf("Credit applied: %s -> order %s amount %s", creditID, orderID, amount.String())
		result = &port.CreditApplication{
			ApplicationID: applicationID,
			CustomerID:    creditCustomerID,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

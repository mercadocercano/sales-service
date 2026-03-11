package usecase

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/mercadocercano/eventbus"
	"github.com/shopspring/decimal"
)

// ApplyCustomerCreditUseCase aplica un crédito (parcial o total) a una orden.
// Orden de locks: 1) Order, 2) Credit. Evento después del COMMIT.
type ApplyCustomerCreditUseCase struct {
	db             *sql.DB
	publishUseCase *eventbus.PublishEventUseCase
}

// NewApplyCustomerCreditUseCase crea el caso de uso.
func NewApplyCustomerCreditUseCase(db *sql.DB, publishUseCase *eventbus.PublishEventUseCase) *ApplyCustomerCreditUseCase {
	return &ApplyCustomerCreditUseCase{
		db:             db,
		publishUseCase: publishUseCase,
	}
}

// Execute aplica amount del crédito a la orden. amount debe ser positivo.
// Devuelve el ID de la aplicación creada.
func (uc *ApplyCustomerCreditUseCase) Execute(
	ctx context.Context,
	tenantID, creditID, orderID string,
	amount decimal.Decimal,
) (applicationID string, err error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return "", fmt.Errorf("amount must be positive")
	}

	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1️⃣ Lock Order (siempre primero)
	var orderTotal float64
	var orderCustomerID string
	var orderStatus string
	err = tx.QueryRowContext(ctx, `
		SELECT customer_id, total_amount, status
		FROM sales_orders
		WHERE id = $1 AND tenant_id = $2
		FOR UPDATE
	`, orderID, tenantID).Scan(&orderCustomerID, &orderTotal, &orderStatus)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("order not found")
	}
	if err != nil {
		return "", fmt.Errorf("failed to lock order: %w", err)
	}
	if orderStatus != "CREATED" && orderStatus != "CONFIRMED" {
		return "", fmt.Errorf("order must be CREATED or CONFIRMED (current: %s)", orderStatus)
	}

	// 2️⃣ Lock Credit (siempre segundo)
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
		return "", fmt.Errorf("credit not found")
	}
	if err != nil {
		return "", fmt.Errorf("failed to lock credit: %w", err)
	}
	if creditStatus == "FULLY_APPLIED" {
		return "", fmt.Errorf("credit is fully applied")
	}

	amountF, _ := amount.Float64()
	if creditRemaining < amountF {
		return "", fmt.Errorf("credit remaining (%.2f) is less than amount (%.2f)", creditRemaining, amountF)
	}

	// 3️⃣ Cross validations
	if creditCustomerID != orderCustomerID {
		return "", fmt.Errorf("credit does not belong to the same customer as the order")
	}

	// 4️⃣ Order remaining = total - payments - credit applications (híbrido Fase 1)
	var sumPayments, sumApplications sql.NullFloat64
	err = tx.QueryRowContext(ctx, `
		SELECT
			(SELECT COALESCE(SUM(amount), 0) FROM sales_payments WHERE sales_order_id = $1 AND tenant_id = $2),
			(SELECT COALESCE(SUM(amount), 0) FROM customer_credit_applications WHERE sales_order_id = $1)
	`, orderID, tenantID).Scan(&sumPayments, &sumApplications)
	if err != nil {
		return "", fmt.Errorf("failed to sum order paid: %w", err)
	}
	paid := sumPayments.Float64 + sumApplications.Float64
	orderRemaining := orderTotal - paid
	if orderRemaining < amountF {
		return "", fmt.Errorf("order remaining (%.2f) is less than amount (%.2f)", orderRemaining, amountF)
	}

	// 5️⃣ Insert application
	applicationID = uuid.New().String()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO customer_credit_applications (id, tenant_id, customer_credit_id, sales_order_id, amount, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, applicationID, tenantID, creditID, orderID, amount.String())
	if err != nil {
		return "", fmt.Errorf("failed to insert application: %w", err)
	}

	// 6️⃣ Update credit
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
		return "", fmt.Errorf("failed to update credit: %w", err)
	}

	// 7️⃣ Update order to PAID if fully covered
	if orderRemaining-amountF <= 1e-9 {
		_, err = tx.ExecContext(ctx, `
			UPDATE sales_orders SET status = 'PAID', updated_at = NOW() WHERE id = $1 AND tenant_id = $2
		`, orderID, tenantID)
		if err != nil {
			return "", fmt.Errorf("failed to update order to PAID: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("✅ Credit applied: %s -> order %s amount %s", creditID, orderID, amount.String())

	// 8️⃣ Publish event AFTER commit
	if uc.publishUseCase != nil {
		if err := uc.publishCreditAppliedEvent(ctx, applicationID, tenantID, creditCustomerID, creditID, orderID, amount); err != nil {
			log.Printf("WARNING: Failed to publish sales.credit.applied: %v", err)
		}
	}

	return applicationID, nil
}

func (uc *ApplyCustomerCreditUseCase) publishCreditAppliedEvent(
	ctx context.Context,
	applicationID, tenantID, customerID, creditID, orderID string,
	amount decimal.Decimal,
) error {
	amountF, _ := amount.Float64()
	payload := map[string]interface{}{
		"event_type":  "sales.credit.applied",
		"tenant_id":   tenantID,
		"customer_id": customerID,
		"credit_id":   creditID,
		"order_id":    orderID,
		"amount":      amountF,
	}
	envelope := map[string]interface{}{
		"event_id":       applicationID + "-evt",
		"event_type":     "sales.credit.applied",
		"event_version":  1,
		"aggregate_type": "customer_credit_application",
		"aggregate_id":   applicationID,
		"tenant_id":      tenantID,
		"occurred_at":    time.Now().UTC().Format(time.RFC3339),
		"payload":       payload,
	}
	envelopeBytes, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	return uc.publishUseCase.Execute(ctx, applicationID, "customer_credit_application", "sales.credit.applied", envelopeBytes, "sales-service")
}

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

// CreateCustomerCreditUseCase crea un crédito a cuenta del cliente (pago anticipado).
// No aplica a órdenes; no toca sales_orders ni sales_payments.
// Evento sales.customer.credit.created después del COMMIT.
type CreateCustomerCreditUseCase struct {
	db             *sql.DB
	publishUseCase *eventbus.PublishEventUseCase
}

// NewCreateCustomerCreditUseCase crea el caso de uso.
func NewCreateCustomerCreditUseCase(db *sql.DB, publishUseCase *eventbus.PublishEventUseCase) *CreateCustomerCreditUseCase {
	return &CreateCustomerCreditUseCase{
		db:             db,
		publishUseCase: publishUseCase,
	}
}

// Execute crea el crédito. amount debe ser > 0.
// Devuelve el ID del crédito creado.
func (uc *CreateCustomerCreditUseCase) Execute(
	ctx context.Context,
	tenantID, customerID string,
	amount decimal.Decimal,
) (creditID string, err error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return "", fmt.Errorf("amount must be greater than zero")
	}

	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	creditID = uuid.New().String()
	amountStr := amount.String()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO customer_credits (
			id, tenant_id, customer_id,
			total_amount, applied_amount, remaining_amount,
			status, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, 0, $4, 'AVAILABLE', NOW(), NOW())
	`, creditID, tenantID, customerID, amountStr)
	if err != nil {
		return "", fmt.Errorf("failed to insert customer credit: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("✅ Customer credit created: %s customer %s amount %s", creditID, customerID, amountStr)

	if uc.publishUseCase != nil {
		if err := uc.publishCustomerCreditCreatedEvent(ctx, creditID, tenantID, customerID, amountStr); err != nil {
			log.Printf("WARNING: Failed to publish sales.customer.credit.created: %v", err)
		}
	}

	return creditID, nil
}

func (uc *CreateCustomerCreditUseCase) publishCustomerCreditCreatedEvent(
	ctx context.Context,
	creditID, tenantID, customerID, amountStr string,
) error {
	payload := map[string]interface{}{
		"tenant_id":   tenantID,
		"customer_id": customerID,
		"credit_id":   creditID,
		"amount":      amountStr,
	}
	envelope := map[string]interface{}{
		"event_id":       creditID + "-evt",
		"event_type":     "sales.customer.credit.created",
		"event_version":  1,
		"aggregate_type": "customer_credit",
		"aggregate_id":   creditID,
		"tenant_id":      tenantID,
		"occurred_at":   time.Now().UTC().Format(time.RFC3339),
		"payload":       payload,
	}
	envelopeBytes, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	return uc.publishUseCase.Execute(ctx, creditID, "customer_credit", "sales.customer.credit.created", envelopeBytes, "sales-service")
}

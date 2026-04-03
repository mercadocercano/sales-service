package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"sales/src/sales/domain/port"

	"github.com/shopspring/decimal"
)

// ApplyCustomerCreditUseCase aplica un crédito (parcial o total) a una orden.
type ApplyCustomerCreditUseCase struct {
	creditRepo     port.CreditRepository
	eventPublisher port.EventPublisher
}

// NewApplyCustomerCreditUseCase crea el caso de uso.
func NewApplyCustomerCreditUseCase(creditRepo port.CreditRepository, eventPublisher port.EventPublisher) *ApplyCustomerCreditUseCase {
	return &ApplyCustomerCreditUseCase{
		creditRepo:     creditRepo,
		eventPublisher: eventPublisher,
	}
}

// Execute aplica amount del crédito a la orden. amount debe ser positivo.
func (uc *ApplyCustomerCreditUseCase) Execute(
	ctx context.Context,
	tenantID, creditID, orderID string,
	amount decimal.Decimal,
) (string, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return "", fmt.Errorf("amount must be positive")
	}

	result, err := uc.creditRepo.ApplyCredit(ctx, tenantID, creditID, orderID, amount)
	if err != nil {
		return "", err
	}

	log.Printf("Credit applied: %s -> order %s amount %s", creditID, orderID, amount.String())

	// Publish event AFTER commit
	if uc.eventPublisher != nil {
		if err := uc.publishCreditAppliedEvent(ctx, result.ApplicationID, tenantID, result.CustomerID, creditID, orderID, amount); err != nil {
			log.Printf("WARNING: Failed to publish sales.credit.applied: %v", err)
		}
	}

	return result.ApplicationID, nil
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
		"payload":        payload,
	}
	envelopeBytes, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	return uc.eventPublisher.Execute(ctx, applicationID, "customer_credit_application", "sales.credit.applied", envelopeBytes, "sales-service")
}

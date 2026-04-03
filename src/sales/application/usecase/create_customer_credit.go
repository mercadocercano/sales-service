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

// CreateCustomerCreditUseCase crea un crédito a cuenta del cliente (pago anticipado).
type CreateCustomerCreditUseCase struct {
	creditRepo     port.CreditRepository
	eventPublisher port.EventPublisher
}

// NewCreateCustomerCreditUseCase crea el caso de uso.
func NewCreateCustomerCreditUseCase(creditRepo port.CreditRepository, eventPublisher port.EventPublisher) *CreateCustomerCreditUseCase {
	return &CreateCustomerCreditUseCase{
		creditRepo:     creditRepo,
		eventPublisher: eventPublisher,
	}
}

// Execute crea el crédito. amount debe ser > 0.
func (uc *CreateCustomerCreditUseCase) Execute(
	ctx context.Context,
	tenantID, customerID string,
	amount decimal.Decimal,
) (string, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return "", fmt.Errorf("amount must be greater than zero")
	}

	creditID, err := uc.creditRepo.CreateCredit(ctx, tenantID, customerID, amount)
	if err != nil {
		return "", err
	}

	log.Printf("Customer credit created: %s customer %s amount %s", creditID, customerID, amount.String())

	if uc.eventPublisher != nil {
		if err := uc.publishCustomerCreditCreatedEvent(ctx, creditID, tenantID, customerID, amount.String()); err != nil {
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
		"occurred_at":    time.Now().UTC().Format(time.RFC3339),
		"payload":        payload,
	}
	envelopeBytes, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	return uc.eventPublisher.Execute(ctx, creditID, "customer_credit", "sales.customer.credit.created", envelopeBytes, "sales-service")
}

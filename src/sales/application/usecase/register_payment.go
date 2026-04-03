package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"sales/src/sales/domain/port"
)

// RegisterPaymentUseCase registra un pago contra una orden confirmada
type RegisterPaymentUseCase struct {
	paymentRepo    port.PaymentRepository
	eventPublisher port.EventPublisher
}

// NewRegisterPaymentUseCase crea una nueva instancia del caso de uso
func NewRegisterPaymentUseCase(
	paymentRepo port.PaymentRepository,
	eventPublisher port.EventPublisher,
) *RegisterPaymentUseCase {
	return &RegisterPaymentUseCase{
		paymentRepo:    paymentRepo,
		eventPublisher: eventPublisher,
	}
}

// Execute registra un pago
func (uc *RegisterPaymentUseCase) Execute(
	ctx context.Context,
	tenantID string,
	orderID string,
	amount float64,
) (string, error) {
	if amount <= 0 {
		return "", fmt.Errorf("amount must be positive")
	}

	result, err := uc.paymentRepo.RegisterPayment(ctx, tenantID, orderID, amount)
	if err != nil {
		return "", err
	}

	log.Printf("Payment registered: %s (%.2f ARS) for order %s", result.PaymentID, amount, orderID)

	// Publicar evento (fuera de transacción DB)
	if uc.eventPublisher != nil {
		if err := uc.publishPaymentRegisteredEvent(
			ctx,
			result.PaymentID,
			tenantID,
			orderID,
			result.CustomerID,
			amount,
		); err != nil {
			log.Printf("WARNING: Failed to publish sales.payment.registered: %v", err)
		}
	}

	return result.PaymentID, nil
}

// publishPaymentRegisteredEvent publica el evento según contrato v1
func (uc *RegisterPaymentUseCase) publishPaymentRegisteredEvent(
	ctx context.Context,
	paymentID string,
	tenantID string,
	orderID string,
	customerID string,
	amount float64,
) error {
	payload := map[string]interface{}{
		"customer": map[string]interface{}{
			"customer_id": customerID,
		},
		"sales_order_id": orderID,
		"amount":         amount,
		"currency":       "ARS",
		"exchange_rate":  1.0,
	}

	envelope := map[string]interface{}{
		"event_id":       paymentID + "-evt",
		"event_type":     "sales.payment.registered",
		"event_version":  1,
		"aggregate_type": "sales_payment",
		"aggregate_id":   paymentID,
		"tenant_id":      tenantID,
		"occurred_at":    time.Now().UTC().Format(time.RFC3339),
		"payload":        payload,
	}

	envelopeBytes, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("failed to marshal envelope: %w", err)
	}

	return uc.eventPublisher.Execute(
		ctx,
		paymentID,
		"sales_payment",
		"sales.payment.registered",
		envelopeBytes,
		"sales-service",
	)
}

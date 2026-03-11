package usecase

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"sales/src/sales/domain/entity"
	
	"github.com/google/uuid"
	"github.com/mercadocercano/eventbus"
)

// RegisterPaymentUseCase registra un pago contra una orden confirmada
// con protección transaccional contra race conditions
type RegisterPaymentUseCase struct {
	db             *sql.DB
	publishUseCase *eventbus.PublishEventUseCase
}

// NewRegisterPaymentUseCase crea una nueva instancia del caso de uso
func NewRegisterPaymentUseCase(
	db *sql.DB,
	publishUseCase *eventbus.PublishEventUseCase,
) *RegisterPaymentUseCase {
	return &RegisterPaymentUseCase{
		db:             db,
		publishUseCase: publishUseCase,
	}
}

// Execute registra un pago con protección contra race conditions
func (uc *RegisterPaymentUseCase) Execute(
	ctx context.Context,
	tenantID string,
	orderID string,
	amount float64,
) (string, error) {
	// Validación básica
	if amount <= 0 {
		return "", fmt.Errorf("amount must be positive")
	}

	// 🔒 INICIO TRANSACCIÓN
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback automático si no se hace commit

	// 1. 🔒 SELECT FOR UPDATE sobre la orden (row lock)
	var (
		orderStatus     string
		orderTotal      float64
		orderCustomerID string
		orderTenantID   string
	)
	
	queryOrder := `
		SELECT status, total_amount, customer_id, tenant_id
		FROM sales_orders
		WHERE id = $1 AND tenant_id = $2
		FOR UPDATE
	`
	
	err = tx.QueryRowContext(ctx, queryOrder, orderID, tenantID).Scan(
		&orderStatus,
		&orderTotal,
		&orderCustomerID,
		&orderTenantID,
	)
	
	if err == sql.ErrNoRows {
		return "", entity.ErrOrderNotFound
	}
	if err != nil {
		return "", fmt.Errorf("failed to lock order: %w", err)
	}
	
	// 2. Validar estado CONFIRMED
	if orderStatus != string(entity.OrderStatusConfirmed) {
		return "", fmt.Errorf("order must be confirmed (current: %s)", orderStatus)
	}
	
	// 3. Calcular total pagado (dentro de la transacción)
	var totalPaid sql.NullFloat64
	querySumPayments := `
		SELECT COALESCE(SUM(amount), 0)
		FROM sales_payments
		WHERE sales_order_id = $1 AND tenant_id = $2
	`
	
	err = tx.QueryRowContext(ctx, querySumPayments, orderID, tenantID).Scan(&totalPaid)
	if err != nil {
		return "", fmt.Errorf("failed to sum payments: %w", err)
	}
	
	currentPaid := totalPaid.Float64
	
	// 4. Validar que no exceda el total
	if currentPaid + amount > orderTotal {
		return "", fmt.Errorf(
			"payment exceeds order total (paid: %.2f, new: %.2f, total: %.2f)",
			currentPaid, amount, orderTotal,
		)
	}
	
	// 5. Insertar pago
	paymentID := uuid.New().String()
	insertPayment := `
		INSERT INTO sales_payments (id, tenant_id, sales_order_id, amount, currency)
		VALUES ($1, $2, $3, $4, $5)
	`
	
	_, err = tx.ExecContext(ctx, insertPayment, paymentID, tenantID, orderID, amount, "ARS")
	if err != nil {
		return "", fmt.Errorf("failed to insert payment: %w", err)
	}

	// 5b. Transición automática a PAID cuando remaining == 0 (misma tx)
	newTotalPaid := currentPaid + amount
	if newTotalPaid >= orderTotal-1e-9 && orderStatus == string(entity.OrderStatusConfirmed) {
		_, err = tx.ExecContext(ctx,
			`UPDATE sales_orders SET status = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3`,
			string(entity.OrderStatusPaid), orderID, tenantID)
		if err != nil {
			return "", fmt.Errorf("failed to update order status to PAID: %w", err)
		}
	}

	// 6. 🔒 COMMIT transacción ANTES de publicar evento
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}
	// 🔓 FIN TRANSACCIÓN
	
	log.Printf("✅ Payment registered: %s (%.2f ARS) for order %s", paymentID, amount, orderID)
	
	// 7. Publicar evento (fuera de transacción DB)
	if uc.publishUseCase != nil {
		if err := uc.publishPaymentRegisteredEvent(
			ctx,
			paymentID,
			tenantID,
			orderID,
			orderCustomerID,
			amount,
		); err != nil {
			// Log warning pero no fallar - pago ya registrado
			log.Printf("WARNING: Failed to publish sales.payment.registered: %v", err)
		}
	}
	
	return paymentID, nil
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
	// Construir payload de negocio según contrato v1
	payload := map[string]interface{}{
		"customer": map[string]interface{}{
			"customer_id": customerID,
		},
		"sales_order_id": orderID,
		"amount":         amount,
		"currency":       "ARS",
		"exchange_rate":  1.0, // Hardcoded para HITO
	}
	
	// Crear EventEnvelope completo (ledger espera este formato)
	envelope := map[string]interface{}{
		"event_id":       paymentID + "-evt", // ID único del evento
		"event_type":     "sales.payment.registered",
		"event_version":  1,
		"aggregate_type": "sales_payment",
		"aggregate_id":   paymentID,
		"tenant_id":      tenantID,
		"occurred_at":    time.Now().UTC().Format(time.RFC3339),
		"payload":        payload,
	}
	
	// Serializar envelope completo
	envelopeBytes, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("failed to marshal envelope: %w", err)
	}
	
	// Publicar usando eventbus
	return uc.publishUseCase.Execute(
		ctx,
		paymentID,                       // aggregateID
		"sales_payment",                 // aggregateType
		"sales.payment.registered",      // eventType
		envelopeBytes,                   // payload (envelope completo)
		"sales-service",                 // publishedBy
	)
}

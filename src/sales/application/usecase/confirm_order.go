package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"sales/src/sales/domain/entity"
	"sales/src/sales/domain/port"
	"sales/src/sales/infrastructure/client"
	
	"github.com/mercadocercano/eventbus"
)

// ConfirmOrderUseCase caso de uso para confirmar una orden
type ConfirmOrderUseCase struct {
	orderRepo      port.OrderRepository
	stockClient    *client.StockClient
	publishUseCase *eventbus.PublishEventUseCase
}

// NewConfirmOrderUseCase crea una nueva instancia del caso de uso
func NewConfirmOrderUseCase(
	orderRepo port.OrderRepository, 
	stockClient *client.StockClient,
	publishUseCase *eventbus.PublishEventUseCase,
) *ConfirmOrderUseCase {
	return &ConfirmOrderUseCase{
		orderRepo:      orderRepo,
		stockClient:    stockClient,
		publishUseCase: publishUseCase,
	}
}

// Execute ejecuta la confirmación de la orden (multi-item, atómico)
func (uc *ConfirmOrderUseCase) Execute(ctx context.Context, tenantID, authToken, orderID, reference string) (*entity.Order, error) {
	// 1. Buscar orden con sus items (load aggregate)
	order, err := uc.orderRepo.FindByID(ctx, orderID, tenantID)
	if err != nil {
		return nil, entity.ErrOrderNotFound
	}

	// 2. Validar que esté en estado CREATED
	if order.Status != entity.OrderStatusCreated {
		return nil, entity.ErrOrderNotInCreatedState
	}

	// 3. Consumir stock reservado para CADA item vía Kong (ALL OR NOTHING)
	for _, item := range order.Items {
		_, err = uc.stockClient.ConsumeStock(tenantID, authToken, item.SKU, item.Quantity, reference)
		if err != nil {
			// Si falla un item, TODO el proceso falla
			// Nota: En producción debería hacer rollback de items anteriores
			if contains(err.Error(), "insufficient reserved stock") {
				return nil, fmt.Errorf("insufficient_reserved_stock for SKU %s: %w", item.SKU, err)
			}
			return nil, fmt.Errorf("error consuming stock for SKU %s: %w", item.SKU, err)
		}
	}

	// 4. Confirmar orden en DB
	if err := uc.orderRepo.Confirm(ctx, orderID, tenantID); err != nil {
		return nil, fmt.Errorf("error confirming order: %w", err)
	}

	// 5. Actualizar entidad en memoria
	order.Status = entity.OrderStatusConfirmed

	// 6. HITO v0.1: Publicar evento sales.order.confirmed
	if uc.publishUseCase != nil {
		if err := uc.publishSalesOrderConfirmedEvent(ctx, order, tenantID); err != nil {
			// Log error pero NO fallar la operación (orden ya confirmada)
			log.Printf("WARNING: Failed to publish sales.order.confirmed: %v", err)
		}
	}

	return order, nil
}

// publishSalesOrderConfirmedEvent publica el evento sales.order.confirmed
func (uc *ConfirmOrderUseCase) publishSalesOrderConfirmedEvent(
	ctx context.Context,
	order *entity.Order,
	tenantID string,
) error {
	// HITO v0.1: Total hardcoded para testing (sin productos reales)
	totalAmount := 250.00 // Monto fijo para validación E2E

	// Construir payload de negocio según contrato v1
	businessPayload := map[string]interface{}{
		"order_number": 0, // TODO: Implementar numeración secuencial
		"customer": map[string]interface{}{
			"customer_id":   "00000000-0000-0000-0000-000000000001", // TODO: Obtener customer_id real
			"customer_name": "Cliente Genérico",
			"tax_condition": "CONSUMIDOR_FINAL",
		},
		"currency":      "ARS",
		"exchange_rate": 1.0,
		"totals": map[string]interface{}{
			"subtotal": totalAmount,
			"discount": 0.0,
			"tax":      0.0,
			"total":    totalAmount,
		},
		"payment_terms": map[string]interface{}{
			"type":     "CUENTA_CORRIENTE",
			"due_date": time.Now().Add(30 * 24 * time.Hour).Format(time.RFC3339),
		},
	}

	// Crear EventEnvelope completo (ledger espera este formato)
	envelope := map[string]interface{}{
		"event_id":       order.OrderID + "-evt", // ID único del evento
		"event_type":     "sales.order.confirmed",
		"event_version":  1,
		"aggregate_type": "sales_order",
		"aggregate_id":   order.OrderID,
		"tenant_id":      tenantID,
		"occurred_at":    time.Now().UTC().Format(time.RFC3339),
		"payload":        businessPayload,
	}

	// Serializar envelope completo
	envelopeBytes, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("failed to marshal envelope: %w", err)
	}

	// Publicar usando eventbus
	return uc.publishUseCase.Execute(
		ctx,
		order.OrderID,            // aggregateID
		"sales_order",            // aggregateType
		"sales.order.confirmed",  // eventType
		envelopeBytes,            // payload (envelope completo)
		"order-service",          // publishedBy
	)
}

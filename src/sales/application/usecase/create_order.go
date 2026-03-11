package usecase

import (
	"context"
	"fmt"
	"log"
	"sales/src/sales/application/request"
	"sales/src/sales/application/response"
	"sales/src/sales/domain/entity"
	"sales/src/sales/domain/port"
	"sales/src/sales/infrastructure/client"

	"github.com/google/uuid"
)

// CreateOrderUseCase caso de uso para crear una orden
// HITO v0.6: Ahora valida customer_id contra customer-service
type CreateOrderUseCase struct {
	orderRepo      port.OrderRepository
	pimClient      *client.PIMClient
	stockClient    *client.StockClient
	customerClient *client.CustomerClient
}

// NewCreateOrderUseCase crea una nueva instancia del caso de uso
func NewCreateOrderUseCase(
	orderRepo port.OrderRepository,
	pimClient *client.PIMClient,
	stockClient *client.StockClient,
	customerClient *client.CustomerClient,
) *CreateOrderUseCase {
	return &CreateOrderUseCase{
		orderRepo:      orderRepo,
		pimClient:      pimClient,
		stockClient:    stockClient,
		customerClient: customerClient,
	}
}

// Execute ejecuta la creación de la orden con operación atómica y compensación
// HITO v0.6: Ahora valida customer_id ANTES de procesar
// HITO D - Flujo transaccional robusto:
// 0. Validar customer_id contra customer-service (HITO v0.6)
// 1. Obtener snapshots de PIM para todos los items
// 2. Crear aggregate Order (en memoria)
// 3. Ejecutar ProcessSaleAtomic para cada item (validación + descuento atómico)
// 4. Si falla un item → compensar todos los anteriores
// 5. Persistir orden
// 6. Si falla persistencia → compensar todo el stock descontado
func (uc *CreateOrderUseCase) Execute(ctx context.Context, tenantID, authToken string, req *request.CreateOrderRequest) (*response.CreateOrderResponse, error) {
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("order must contain at least one item")
	}

	// ========================================================================
	// HITO v0.6: PASO 0 - Validar customer_id contra customer-service
	// ========================================================================
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant_id: %w", err)
	}

	exists, err := uc.customerClient.Exists(ctx, tenantUUID, req.CustomerID)
	if err != nil {
		// Error técnico comunicándose con customer-service
		return nil, fmt.Errorf("error validating customer: %w", err)
	}
	if !exists {
		// Cliente no existe (404) - Error de dominio
		return nil, fmt.Errorf("customer_not_found: %s", req.CustomerID.String())
	}

	// ========================================================================
	// PASO 1: Obtener snapshots inmutables de PIM para todos los items
	// ========================================================================
	var items []entity.OrderItem
	for _, itemReq := range req.Items {
		// Validar unit_price del request (fuente primaria)
		if itemReq.UnitPrice <= 0 {
			return nil, fmt.Errorf("invalid unit_price for SKU %s: must be > 0", itemReq.SKU)
		}

		// Intentar obtener snapshots de PIM para trazabilidad (best effort)
		productSnapshot, variantSnapshot, err := uc.pimClient.GetSnapshotForSKU(tenantID, authToken, itemReq.SKU)
		if err != nil {
			// PIM falló - loggear pero NO bloquear
			// Sales es autosuficiente con el unit_price recibido
			log.Printf("WARNING: PIM snapshot failed for SKU %s: %v. Using request unit_price.", itemReq.SKU, err)
			productSnapshot = nil
			variantSnapshot = nil
		}

		// Crear item con unit_price del request (fuente primaria)
		// Snapshots son enriquecimiento opcional para trazabilidad
		item, err := entity.NewOrderItemWithSnapshots("", itemReq.SKU, itemReq.Quantity, itemReq.UnitPrice, productSnapshot, variantSnapshot)
		if err != nil {
			return nil, fmt.Errorf("error creating order item for SKU %s: %w", itemReq.SKU, err)
		}
		items = append(items, *item)
	}

	// ========================================================================
	// PASO 2: Crear entidad Order (aggregate root) EN MEMORIA - AÚN NO persiste
	// HITO v0.6: Ahora incluye customer_id
	// ========================================================================
	order, err := entity.NewOrder(tenantID, req.CustomerID.String(), items)
	if err != nil {
		return nil, fmt.Errorf("error creating order entity: %w", err)
	}

	// ========================================================================
	// PASO 3: Ejecutar ProcessSaleAtomic para cada item
	// HITO D: Operación atómica (SELECT FOR UPDATE) elimina race condition
	// ========================================================================
	processedStockEntries := make([]string, 0, len(order.Items))

	for _, item := range order.Items {
		saleResp, err := uc.stockClient.ProcessSaleAtomic(
			tenantID,
			authToken,
			item.SKU,
			float64(item.Quantity),
			order.OrderID, // Reference para trazabilidad
		)

		if err != nil {
			// Error técnico (HTTP, network, etc.)
			uc.compensateProcessedStock(ctx, tenantID, authToken, processedStockEntries, "order_creation_failed")
			return nil, fmt.Errorf("error processing stock for SKU %s: %w", item.SKU, err)
		}

		if !saleResp.Success {
			// Error de negocio (stock insuficiente, no inicializado, etc.)
			uc.compensateProcessedStock(ctx, tenantID, authToken, processedStockEntries, "insufficient_stock")
			return nil, fmt.Errorf("stock rejected for SKU %s: %s", item.SKU, saleResp.Message)
		}

		// Guardar stock_entry_id para posible compensación
		processedStockEntries = append(processedStockEntries, saleResp.StockEntryID)
	}

	// ========================================================================
	// PASO 4: Persistir orden SOLO si todo el stock salió correctamente
	// ========================================================================
	if err := uc.orderRepo.Save(ctx, order); err != nil {
		// CRÍTICO: Stock ya fue descontado, debemos revertirlo
		uc.compensateProcessedStock(ctx, tenantID, authToken, processedStockEntries, "order_persistence_failed")
		return nil, fmt.Errorf("error saving order (stock compensated): %w", err)
	}

	// ========================================================================
	// PASO 5: Construir respuesta exitosa
	// ========================================================================
	var itemsResp []response.CreateOrderItemResponse
	for _, item := range order.Items {
		itemsResp = append(itemsResp, response.CreateOrderItemResponse{
			ItemID:   item.ItemID,
			SKU:      item.SKU,
			Quantity: item.Quantity,
		})
	}

	return &response.CreateOrderResponse{
		OrderID:    order.OrderID,
		Items:      itemsResp,
		TotalItems: len(order.Items),
		Status:     string(order.Status),
	}, nil
}

// compensateProcessedStock revierte todas las ventas procesadas
// HITO D: Función crítica para garantizar consistencia transaccional
func (uc *CreateOrderUseCase) compensateProcessedStock(
	ctx context.Context,
	tenantID, authToken string,
	stockEntryIDs []string,
	reason string,
) {
	for _, entryID := range stockEntryIDs {
		err := uc.stockClient.CompensateSale(tenantID, authToken, entryID, reason)
		if err != nil {
			// CRÍTICO: Si falla compensación, log para auditoría manual
			// No hacer panic ni detener el flujo
			fmt.Printf("CRITICAL ERROR: Failed to compensate stock entry %s: %v\n", entryID, err)
			// TODO: Enviar alerta a sistema de monitoreo
		}
	}
}

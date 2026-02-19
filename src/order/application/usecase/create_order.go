package usecase

import (
	"context"
	"fmt"
	"order/src/order/application/request"
	"order/src/order/application/response"
	"order/src/order/domain/entity"
	"order/src/order/domain/port"
	"order/src/order/infrastructure/client"
)

// CreateOrderUseCase caso de uso para crear una orden
type CreateOrderUseCase struct {
	orderRepo   port.OrderRepository
	pimClient   *client.PIMClient
	stockClient *client.StockClient
}

// NewCreateOrderUseCase crea una nueva instancia del caso de uso
func NewCreateOrderUseCase(orderRepo port.OrderRepository, pimClient *client.PIMClient, stockClient *client.StockClient) *CreateOrderUseCase {
	return &CreateOrderUseCase{
		orderRepo:   orderRepo,
		pimClient:   pimClient,
		stockClient: stockClient,
	}
}

// Execute ejecuta la creación de la orden con operación atómica y compensación
// HITO D - Flujo transaccional robusto:
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
	// PASO 1: Obtener snapshots inmutables de PIM para todos los items
	// ========================================================================
	var items []entity.OrderItem
	for _, itemReq := range req.Items {
		// Obtener snapshots inmutables de PIM al momento de crear la orden
		productSnapshot, variantSnapshot, err := uc.pimClient.GetSnapshotForSKU(tenantID, authToken, itemReq.SKU)
		if err != nil {
			return nil, fmt.Errorf("error fetching snapshot for SKU %s: %w", itemReq.SKU, err)
		}

		// Crear item con snapshots
		item, err := entity.NewOrderItemWithSnapshots("", itemReq.SKU, itemReq.Quantity, productSnapshot, variantSnapshot)
		if err != nil {
			return nil, fmt.Errorf("error creating order item: %w", itemReq.SKU, err)
		}
		items = append(items, *item)
	}

	// ========================================================================
	// PASO 2: Crear entidad Order (aggregate root) EN MEMORIA - AÚN NO persiste
	// ========================================================================
	order, err := entity.NewOrder(tenantID, items)
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

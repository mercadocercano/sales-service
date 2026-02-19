package usecase

import (
	"context"
	"fmt"
	"log"
	"time"

	"order/src/order/application/request"
	"order/src/order/application/response"
	"order/src/order/domain/entity"
	"order/src/order/domain/port"
	"order/src/order/infrastructure/cache"
	"order/src/order/infrastructure/client"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// POSSaleUseCase caso de uso para venta directa POS
// Hito: POS-SALE-02.BE - Paso 3
// HITO: POST /pos/sale devuelve DTO listo para imprimir
type POSSaleUseCase struct {
	stockClient      *client.StockClient
	posSaleRepo      port.PosSaleRepository
	paymentMethodCache *cache.PaymentMethodCache
}

// NewPOSSaleUseCase crea una nueva instancia del caso de uso
func NewPOSSaleUseCase(
	stockClient *client.StockClient,
	posSaleRepo port.PosSaleRepository,
	paymentMethodCache *cache.PaymentMethodCache,
) *POSSaleUseCase {
	return &POSSaleUseCase{
		stockClient:        stockClient,
		posSaleRepo:        posSaleRepo,
		paymentMethodCache: paymentMethodCache,
	}
}

// Execute ejecuta una venta directa POS multi-item con operación atómica y compensación
// HITO D - Flujo transaccional robusto:
// 1. Validar request
// 2. Ejecutar ProcessSaleAtomic para cada item (validación + descuento atómico)
// 3. Si falla un item → compensar todos los anteriores
// 4. Crear pos_sale aggregate
// 5. Persistir pos_sale
// 6. Si falla persistencia → compensar todo el stock descontado
func (uc *POSSaleUseCase) Execute(tenantID, authToken string, req *request.POSSaleRequest) (*response.POSSaleResponse, error) {
	log.Printf("🛒 POS Sale Multi-Item - Items: %d, Tenant: %s", len(req.Items), tenantID)

	// ========================================================================
	// PASO 1: VALIDACIONES TÉCNICAS
	// ========================================================================
	if req.PaymentMethodID == uuid.Nil {
		return nil, fmt.Errorf("payment_method_id is required")
	}
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("at least one item is required")
	}

	// Default discount
	discountAmount := req.DiscountAmount
	if discountAmount.IsZero() {
		discountAmount = decimal.Zero
	}

	// Default currency
	currency := req.Currency
	if currency == "" {
		currency = "ARS"
	}

	// HITO: Validar amount_paid
	if req.AmountPaid.IsZero() || req.AmountPaid.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("amount_paid must be greater than 0")
	}

	// Parsear tenant_id a UUID
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant_id format: %w", err)
	}

	// ========================================================================
	// PASO 2: EJECUTAR PROCESAMIENTO ATÓMICO DE STOCK PARA CADA ITEM
	// HITO D: ProcessSaleAtomic elimina race condition
	// ========================================================================
	
	// Generar reference única base para toda la venta POS
	tenantShort := tenantID
	if len(tenantID) > 8 {
		tenantShort = tenantID[:8]
	}
	baseReference := fmt.Sprintf("POS-%s-%d", tenantShort, time.Now().UnixNano())

	processedStockEntries := make([]string, 0, len(req.Items))
	var posSaleItems []entity.PosSaleItem

	for i, itemReq := range req.Items {
		// Generar reference por item
		itemReference := fmt.Sprintf("%s-ITEM%d", baseReference, i+1)

		// OPERACIÓN ATÓMICA: validar + descontar en una sola transacción
		log.Printf("📦 ProcessSaleAtomic for item %d: SKU=%s, Qty=%d", i+1, itemReq.SKU, itemReq.Quantity)
		
		saleResp, err := uc.stockClient.ProcessSaleAtomic(
			tenantID,
			authToken,
			itemReq.SKU,
			float64(itemReq.Quantity),
			itemReference,
		)

		if err != nil {
			// Error técnico (HTTP, network, etc.)
			log.Printf("❌ Stock service error for SKU %s: %v", itemReq.SKU, err)
			uc.compensateProcessedStock(tenantID, authToken, processedStockEntries, "pos_sale_creation_failed")
			return nil, fmt.Errorf("error processing stock for SKU %s: %w", itemReq.SKU, err)
		}

		if !saleResp.Success {
			// Error de negocio (stock insuficiente, no inicializado, etc.)
			log.Printf("❌ Stock rejected for SKU %s: %s", itemReq.SKU, saleResp.Message)
			uc.compensateProcessedStock(tenantID, authToken, processedStockEntries, "insufficient_stock")
			return nil, fmt.Errorf("stock rejected for SKU %s: %s", itemReq.SKU, saleResp.Message)
		}

		log.Printf("✅ Stock OK for item %d: EntryID=%s, QtySold=%.2f, Remaining=%.2f", 
			i+1, saleResp.StockEntryID, saleResp.QuantitySold, saleResp.RemainingStock)

		// Guardar stock_entry_id para posible compensación
		processedStockEntries = append(processedStockEntries, saleResp.StockEntryID)

		// Parsear stock_entry_id
		stockEntryUUID, err := uuid.Parse(saleResp.StockEntryID)
		if err != nil {
			uc.compensateProcessedStock(tenantID, authToken, processedStockEntries, "invalid_stock_entry_id")
			return nil, fmt.Errorf("invalid stock_entry_id from stock-service: %w", err)
		}

		// Crear item entity (subtotal se calcula en NewPosSaleItem)
		item, err := entity.NewPosSaleItem(
			uuid.Nil, // Se asignará en NewPosSale
			itemReq.SKU,
			saleResp.VariantSKU, // Usar SKU como product_name temporal (TODO: obtener de PIM)
			itemReq.Quantity,
			itemReq.UnitPrice,
			stockEntryUUID,
		)
		if err != nil {
			uc.compensateProcessedStock(tenantID, authToken, processedStockEntries, "item_creation_failed")
			return nil, fmt.Errorf("error creating pos_sale_item: %w", err)
		}

		posSaleItems = append(posSaleItems, *item)
	}

	// ========================================================================
	// PASO 3: CREAR AGGREGATE POS_SALE
	// ========================================================================
	var posSale *entity.PosSale
	if uc.posSaleRepo != nil {
		log.Printf("💾 Creating pos_sale with %d items...", len(posSaleItems))
		posSale, err = entity.NewPosSale(
			tenantUUID,
			req.CustomerID,
			req.PaymentMethodID,
			posSaleItems,
			discountAmount,
			req.AmountPaid,
			currency,
		)
		if err != nil {
			uc.compensateProcessedStock(tenantID, authToken, processedStockEntries, "aggregate_creation_failed")
			return nil, fmt.Errorf("error creating pos_sale entity: %w", err)
		}

		// ========================================================================
		// PASO 4: PERSISTIR ATOMICALLY
		// HITO D: Si falla persistencia → compensar todo el stock descontado
		// ========================================================================
		ctx := context.Background()
		err = uc.posSaleRepo.Create(ctx, posSale)
		if err != nil {
			// CRÍTICO: Stock ya fue descontado, debemos revertirlo
			log.Printf("⚠️ CRITICAL: Stock consumed but pos_sale persistence failed: %v", err)
			uc.compensateProcessedStock(tenantID, authToken, processedStockEntries, "pos_sale_persistence_failed")
			return nil, fmt.Errorf("error saving pos_sale (stock compensated): %w", err)
		}

		log.Printf("✅ PosSale created: ID=%s, Items=%d, FinalAmount=%s", posSale.ID, posSale.TotalItems(), posSale.FinalAmount)
	} else {
		uc.compensateProcessedStock(tenantID, authToken, processedStockEntries, "repository_not_available")
		return nil, fmt.Errorf("pos_sale repository not available")
	}

	// ========================================================================
	// PASO 5: ARMAR RESPONSE
	// ========================================================================
	var itemsResp []response.POSSaleItemResponse
	for _, item := range posSale.Items {
		itemsResp = append(itemsResp, response.POSSaleItemResponse{
			ItemID:       item.ID,
			SKU:          item.SKU,
			ProductName:  item.ProductName,
			Quantity:     item.Quantity,
			UnitPrice:    item.UnitPrice,
			Subtotal:     item.Subtotal,
			StockEntryID: item.StockEntryID,
		})
	}

	// HITO: Obtener nombre del método de pago desde cache
	paymentMethodName := "Unknown"
	if uc.paymentMethodCache != nil {
		paymentMethodName = uc.paymentMethodCache.GetName(posSale.PaymentMethodID)
	}

	// HITO: Usar UUID completo como sale_number
	saleNumber := posSale.ID.String()

	return &response.POSSaleResponse{
		PosSaleID:         posSale.ID,
		SaleNumber:        saleNumber,
		Items:             itemsResp,
		TotalItems:        posSale.TotalItems(),
		SubtotalAmount:    posSale.TotalAmount,
		DiscountAmount:    posSale.DiscountAmount,
		FinalAmount:       posSale.FinalAmount,
		PaymentMethodID:   posSale.PaymentMethodID,
		PaymentMethodName: paymentMethodName,
		AmountPaid:        posSale.AmountPaid,
		Change:            posSale.Change,
		Currency:          posSale.Currency,
		CustomerID:        posSale.CustomerID,
		CreatedAt:         posSale.CreatedAt,
	}, nil
}

// compensateProcessedStock revierte todas las ventas procesadas
// HITO D: Función crítica para garantizar consistencia transaccional en POS
func (uc *POSSaleUseCase) compensateProcessedStock(
	tenantID, authToken string,
	stockEntryIDs []string,
	reason string,
) {
	log.Printf("🔄 Compensating %d stock entries. Reason: %s", len(stockEntryIDs), reason)
	
	for _, entryID := range stockEntryIDs {
		err := uc.stockClient.CompensateSale(tenantID, authToken, entryID, reason)
		if err != nil {
			// CRÍTICO: Si falla compensación, log para auditoría manual
			// No hacer panic ni detener el flujo de compensación
			log.Printf("❌ CRITICAL ERROR: Failed to compensate stock entry %s: %v", entryID, err)
			// TODO: Enviar alerta a sistema de monitoreo (Prometheus/Grafana)
		} else {
			log.Printf("✅ Compensated stock entry: %s", entryID)
		}
	}
}

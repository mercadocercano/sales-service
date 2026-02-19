package usecase

import (
	"context"
	"fmt"
	"sales/src/sales/domain/entity"
	"sales/src/sales/domain/port"
	"sales/src/sales/infrastructure/client"
)

// CancelOrderUseCase caso de uso para cancelar una orden
type CancelOrderUseCase struct {
	orderRepo   port.OrderRepository
	stockClient *client.StockClient
}

// NewCancelOrderUseCase crea una nueva instancia del caso de uso
func NewCancelOrderUseCase(orderRepo port.OrderRepository, stockClient *client.StockClient) *CancelOrderUseCase {
	return &CancelOrderUseCase{
		orderRepo:   orderRepo,
		stockClient: stockClient,
	}
}

// Execute ejecuta la cancelación de la orden (multi-item, atómico)
func (uc *CancelOrderUseCase) Execute(ctx context.Context, tenantID, authToken, orderID string) (*entity.Order, error) {
	// 1. Buscar orden con sus items (load aggregate)
	order, err := uc.orderRepo.FindByID(ctx, orderID, tenantID)
	if err != nil {
		return nil, entity.ErrOrderNotFound
	}

	// 2. Validar que esté en estado CONFIRMED
	if order.Status != entity.OrderStatusConfirmed {
		return nil, entity.ErrOrderNotInConfirmedState
	}

	// 3. Revertir consumo de stock para CADA item vía Kong
	for _, item := range order.Items {
		_, err = uc.stockClient.RevertConsume(tenantID, authToken, item.SKU, item.Quantity, orderID)
		if err != nil {
			// Si falla un item, TODO el proceso falla
			return nil, fmt.Errorf("error reverting stock for SKU %s: %w", item.SKU, err)
		}
	}

	// 4. Cancelar orden en DB
	if err := uc.orderRepo.Cancel(ctx, orderID, tenantID); err != nil {
		return nil, err
	}

	// 5. Actualizar entidad en memoria
	order.Status = entity.OrderStatusCanceled

	return order, nil
}

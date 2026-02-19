package usecase

import (
	"context"
	"order/src/order/application/response"
	"order/src/order/domain/entity"
	"order/src/order/domain/port"
)

// GetOrderUseCase caso de uso para obtener una orden por ID
type GetOrderUseCase struct {
	orderRepo port.OrderRepository
}

// NewGetOrderUseCase crea una nueva instancia del caso de uso
func NewGetOrderUseCase(orderRepo port.OrderRepository) *GetOrderUseCase {
	return &GetOrderUseCase{
		orderRepo: orderRepo,
	}
}

// Execute ejecuta la obtención de la orden
func (uc *GetOrderUseCase) Execute(ctx context.Context, tenantID, orderID string) (*response.GetOrderResponse, error) {
	// Buscar orden
	order, err := uc.orderRepo.FindByID(ctx, orderID, tenantID)
	if err != nil {
		if err.Error() == "order not found" {
			return nil, entity.ErrOrderNotFound
		}
		return nil, err
	}

	// Convertir items con snapshots
	var items []response.OrderItemResponse
	for _, item := range order.Items {
		items = append(items, response.OrderItemResponse{
			ItemID:          item.ItemID,
			SKU:             item.SKU,
			Quantity:        item.Quantity,
			ProductSnapshot: item.ProductSnapshot,
			VariantSnapshot: item.VariantSnapshot,
		})
	}

	return &response.GetOrderResponse{
		OrderID:   order.OrderID,
		TenantID:  order.TenantID,
		Status:    string(order.Status),
		CreatedAt: order.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Items:     items,
	}, nil
}

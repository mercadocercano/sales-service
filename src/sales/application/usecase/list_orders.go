package usecase

import (
	"context"
	"math"
	"sales/src/sales/application/response"
	"sales/src/sales/domain/port"
)

// ListOrdersUseCase caso de uso para listar órdenes con paginación
type ListOrdersUseCase struct {
	orderRepo port.OrderRepository
}

// NewListOrdersUseCase crea una nueva instancia del caso de uso
func NewListOrdersUseCase(orderRepo port.OrderRepository) *ListOrdersUseCase {
	return &ListOrdersUseCase{
		orderRepo: orderRepo,
	}
}

// Execute ejecuta el listado de órdenes
func (uc *ListOrdersUseCase) Execute(ctx context.Context, tenantID string, page, pageSize int) (*response.ListOrdersResponse, error) {
	// Valores por defecto
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// Obtener órdenes del repositorio
	orders, totalCount, err := uc.orderRepo.List(ctx, tenantID, page, pageSize)
	if err != nil {
		return nil, err
	}

	// Convertir a respuesta con snapshots
	var items []response.OrderListItem
	for _, order := range orders {
		var orderItems []response.OrderItemResponse
		for _, item := range order.Items {
			orderItems = append(orderItems, response.OrderItemResponse{
				ItemID:          item.ItemID,
				SKU:             item.SKU,
				Quantity:        item.Quantity,
				ProductSnapshot: item.ProductSnapshot,
				VariantSnapshot: item.VariantSnapshot,
			})
		}

		items = append(items, response.OrderListItem{
			OrderID:   order.OrderID,
			TenantID:  order.TenantID,
			Status:    string(order.Status),
			CreatedAt: order.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Items:     orderItems,
		})
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(pageSize)))

	return &response.ListOrdersResponse{
		Items:      items,
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

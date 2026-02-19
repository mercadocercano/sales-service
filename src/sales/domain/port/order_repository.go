package port

import (
	"context"
	"sales/src/sales/domain/entity"
)

// OrderRepository define los métodos para persistir Orders
type OrderRepository interface {
	Save(ctx context.Context, order *entity.Order) error
	FindByID(ctx context.Context, orderID, tenantID string) (*entity.Order, error)
	List(ctx context.Context, tenantID string, page, pageSize int) ([]*entity.Order, int, error)
	Confirm(ctx context.Context, orderID, tenantID string) error
	Cancel(ctx context.Context, orderID, tenantID string) error
}

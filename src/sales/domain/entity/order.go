package entity

import (
	"time"

	"github.com/google/uuid"
)

// OrderStatus representa el estado de una orden
type OrderStatus string

const (
	OrderStatusCreated   OrderStatus = "CREATED"
	OrderStatusConfirmed OrderStatus = "CONFIRMED"
	OrderStatusCanceled  OrderStatus = "CANCELED"
)

// Order representa una orden (Aggregate Root)
// Una orden contiene uno o más OrderItems
type Order struct {
	OrderID   string      `json:"order_id"`
	TenantID  string      `json:"tenant_id"`
	Status    OrderStatus `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	Items     []OrderItem `json:"items"` // DDD: Collection of entities

	// Campos legacy (deprecated, usar Items)
	SKU      string `json:"sku,omitempty"`
	Quantity int    `json:"quantity,omitempty"`
}

// NewOrder crea una nueva orden con múltiples items (DDD Aggregate Root)
func NewOrder(tenantID string, items []OrderItem) (*Order, error) {
	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}
	if len(items) == 0 {
		return nil, ErrOrderMustHaveItems
	}

	orderID := uuid.New().String()
	now := time.Now()

	// Asignar order_id a todos los items
	for i := range items {
		items[i].OrderID = orderID
	}

	return &Order{
		OrderID:   orderID,
		TenantID:  tenantID,
		Status:    OrderStatusCreated,
		CreatedAt: now,
		Items:     items,
	}, nil
}

// NewOrderSingleItem crea una orden con un solo item (backward compatibility)
func NewOrderSingleItem(tenantID, sku string, quantity int) (*Order, error) {
	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}
	if sku == "" {
		return nil, ErrSKURequired
	}
	if quantity <= 0 {
		return nil, ErrInvalidQuantity
	}

	item, err := NewOrderItem("", sku, quantity)
	if err != nil {
		return nil, err
	}

	return NewOrder(tenantID, []OrderItem{*item})
}

// AddItem agrega un item a la orden (DDD: modificar aggregate)
func (o *Order) AddItem(sku string, quantity int) error {
	item, err := NewOrderItem(o.OrderID, sku, quantity)
	if err != nil {
		return err
	}
	o.Items = append(o.Items, *item)
	return nil
}

// TotalItems retorna el número total de items
func (o *Order) TotalItems() int {
	return len(o.Items)
}

// Confirm confirma una orden
func (o *Order) Confirm() error {
	if o.Status != OrderStatusCreated {
		return ErrOrderNotInCreatedState
	}
	o.Status = OrderStatusConfirmed
	return nil
}

// Cancel cancela una orden
func (o *Order) Cancel() error {
	if o.Status != OrderStatusConfirmed {
		return ErrOrderNotInConfirmedState
	}
	o.Status = OrderStatusCanceled
	return nil
}

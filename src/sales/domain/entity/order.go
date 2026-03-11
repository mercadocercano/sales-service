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
	OrderStatusPaid      OrderStatus = "PAID" // Deuda saldada (remaining == 0)
	OrderStatusCanceled  OrderStatus = "CANCELED"
)

// Order representa una orden (Aggregate Root)
// Una orden contiene uno o más OrderItems
// HITO v0.6: Ahora incluye customer_id obligatorio
type Order struct {
	OrderID     string      `json:"order_id"`
	TenantID    string      `json:"tenant_id"`
	CustomerID  string      `json:"customer_id"`             // HITO v0.6: Obligatorio
	OrderNumber *int        `json:"order_number,omitempty"` // HITO v0.4: Numeración secuencial
	Status      OrderStatus `json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
	Items       []OrderItem `json:"items"` // DDD: Collection of entities

	// Campos legacy (deprecated, usar Items)
	SKU      string `json:"sku,omitempty"`
	Quantity int    `json:"quantity,omitempty"`
}

// NewOrder crea una nueva orden con múltiples items (DDD Aggregate Root)
// HITO v0.6: Ahora requiere customer_id
func NewOrder(tenantID string, customerID string, items []OrderItem) (*Order, error) {
	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}
	if customerID == "" {
		return nil, ErrCustomerIDRequired
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
		OrderID:    orderID,
		TenantID:   tenantID,
		CustomerID: customerID,
		Status:     OrderStatusCreated,
		CreatedAt:  now,
		Items:      items,
	}, nil
}

// NewOrderSingleItem crea una orden con un solo item (backward compatibility)
// HITO v0.6: Ahora requiere customer_id
func NewOrderSingleItem(tenantID, customerID, sku string, quantity int) (*Order, error) {
	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}
	if customerID == "" {
		return nil, ErrCustomerIDRequired
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

	return NewOrder(tenantID, customerID, []OrderItem{*item})
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

// CalculateTotal calcula el total de la orden desde los items con snapshots
// Retorna el total calculado a partir de los precios en los snapshots
func (o *Order) CalculateTotal() float64 {
	var total float64
	for _, item := range o.Items {
		itemTotal := item.CalculateTotal()
		total += itemTotal
	}
	return total
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

// AssignOrderNumber asigna el número de orden (HITO v0.4)
func (o *Order) AssignOrderNumber(number int) {
	o.OrderNumber = &number
}

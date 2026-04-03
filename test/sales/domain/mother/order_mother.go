package mother

import (
	"sales/src/sales/domain/entity"

	"github.com/google/uuid"
)

// OrderMother genera instancias de Order para tests
type OrderMother struct {
	tenantID   string
	customerID string
	items      []entity.OrderItem
}

// RandomOrder crea un OrderMother con valores aleatorios válidos
func RandomOrder() *OrderMother {
	item := RandomOrderItem().MustBuild()
	return &OrderMother{
		tenantID:   uuid.New().String(),
		customerID: uuid.New().String(),
		items:      []entity.OrderItem{*item},
	}
}

// WithTenantID establece el tenantID
func (m *OrderMother) WithTenantID(tenantID string) *OrderMother {
	m.tenantID = tenantID
	return m
}

// WithCustomerID establece el customerID
func (m *OrderMother) WithCustomerID(customerID string) *OrderMother {
	m.customerID = customerID
	return m
}

// WithItems establece los items
func (m *OrderMother) WithItems(items []entity.OrderItem) *OrderMother {
	m.items = items
	return m
}

// Build construye la Order
func (m *OrderMother) Build() (*entity.Order, error) {
	return entity.NewOrder(m.tenantID, m.customerID, m.items)
}

// MustBuild construye la Order o hace panic
func (m *OrderMother) MustBuild() *entity.Order {
	order, err := m.Build()
	if err != nil {
		panic("OrderMother.MustBuild failed: " + err.Error())
	}
	return order
}

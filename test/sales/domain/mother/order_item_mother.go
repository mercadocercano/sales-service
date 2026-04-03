package mother

import (
	"encoding/json"
	"sales/src/sales/domain/entity"

	"github.com/google/uuid"
)

// OrderItemMother genera instancias de OrderItem para tests
type OrderItemMother struct {
	orderID         string
	sku             string
	quantity        int
	unitPrice       float64
	productSnapshot json.RawMessage
	variantSnapshot json.RawMessage
	withSnapshots   bool
}

// RandomOrderItem crea un OrderItemMother con valores aleatorios válidos
func RandomOrderItem() *OrderItemMother {
	return &OrderItemMother{
		orderID:  uuid.New().String(),
		sku:      "SKU-" + uuid.New().String()[:8],
		quantity: 2,
	}
}

// WithOrderID establece el orderID
func (m *OrderItemMother) WithOrderID(orderID string) *OrderItemMother {
	m.orderID = orderID
	return m
}

// WithSKU establece el SKU
func (m *OrderItemMother) WithSKU(sku string) *OrderItemMother {
	m.sku = sku
	return m
}

// WithQuantity establece la cantidad
func (m *OrderItemMother) WithQuantity(qty int) *OrderItemMother {
	m.quantity = qty
	return m
}

// WithUnitPrice establece el precio unitario (activa snapshots)
func (m *OrderItemMother) WithUnitPrice(price float64) *OrderItemMother {
	m.unitPrice = price
	m.withSnapshots = true
	return m
}

// WithSnapshots activa la creación con snapshots
func (m *OrderItemMother) WithSnapshots(productSnap, variantSnap json.RawMessage) *OrderItemMother {
	m.productSnapshot = productSnap
	m.variantSnapshot = variantSnap
	m.withSnapshots = true
	return m
}

// Build construye el OrderItem
func (m *OrderItemMother) Build() (*entity.OrderItem, error) {
	if m.withSnapshots {
		price := m.unitPrice
		if price <= 0 {
			price = 100.0
		}
		if m.productSnapshot == nil {
			m.productSnapshot = json.RawMessage(`{"name":"Test Product"}`)
		}
		if m.variantSnapshot == nil {
			m.variantSnapshot = json.RawMessage(`{"color":"red"}`)
		}
		return entity.NewOrderItemWithSnapshots(
			m.orderID, m.sku, m.quantity,
			price, m.productSnapshot, m.variantSnapshot,
		)
	}
	return entity.NewOrderItem(m.orderID, m.sku, m.quantity)
}

// MustBuild construye el OrderItem o hace panic
func (m *OrderItemMother) MustBuild() *entity.OrderItem {
	item, err := m.Build()
	if err != nil {
		panic("OrderItemMother.MustBuild failed: " + err.Error())
	}
	return item
}

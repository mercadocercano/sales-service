package mother

import (
	"sales/src/sales/domain/entity"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PosSaleItemMother genera instancias de PosSaleItem para tests
type PosSaleItemMother struct {
	posSaleID    uuid.UUID
	sku          string
	productName  string
	quantity     int
	unitPrice    decimal.Decimal
	stockEntryID uuid.UUID
}

// RandomPosSaleItem crea un PosSaleItemMother con valores aleatorios válidos
func RandomPosSaleItem() *PosSaleItemMother {
	return &PosSaleItemMother{
		posSaleID:    uuid.New(),
		sku:          "SKU-" + uuid.New().String()[:8],
		productName:  "Producto Test",
		quantity:     3,
		unitPrice:    decimal.NewFromFloat(150.50),
		stockEntryID: uuid.New(),
	}
}

// WithPosSaleID establece el posSaleID
func (m *PosSaleItemMother) WithPosSaleID(id uuid.UUID) *PosSaleItemMother {
	m.posSaleID = id
	return m
}

// WithSKU establece el SKU
func (m *PosSaleItemMother) WithSKU(sku string) *PosSaleItemMother {
	m.sku = sku
	return m
}

// WithProductName establece el nombre del producto
func (m *PosSaleItemMother) WithProductName(name string) *PosSaleItemMother {
	m.productName = name
	return m
}

// WithQuantity establece la cantidad
func (m *PosSaleItemMother) WithQuantity(qty int) *PosSaleItemMother {
	m.quantity = qty
	return m
}

// WithUnitPrice establece el precio unitario
func (m *PosSaleItemMother) WithUnitPrice(price decimal.Decimal) *PosSaleItemMother {
	m.unitPrice = price
	return m
}

// WithStockEntryID establece el stockEntryID
func (m *PosSaleItemMother) WithStockEntryID(id uuid.UUID) *PosSaleItemMother {
	m.stockEntryID = id
	return m
}

// Build construye el PosSaleItem
func (m *PosSaleItemMother) Build() (*entity.PosSaleItem, error) {
	return entity.NewPosSaleItem(
		m.posSaleID,
		m.sku,
		m.productName,
		m.quantity,
		m.unitPrice,
		m.stockEntryID,
	)
}

// MustBuild construye el PosSaleItem o hace panic
func (m *PosSaleItemMother) MustBuild() *entity.PosSaleItem {
	item, err := m.Build()
	if err != nil {
		panic("PosSaleItemMother.MustBuild failed: " + err.Error())
	}
	return item
}

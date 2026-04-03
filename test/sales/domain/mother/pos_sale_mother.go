package mother

import (
	"sales/src/sales/domain/entity"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PosSaleMother genera instancias de PosSale para tests
type PosSaleMother struct {
	tenantID        uuid.UUID
	customerID      uuid.UUID
	paymentMethodID uuid.UUID
	items           []entity.PosSaleItem
	discountAmount  decimal.Decimal
	amountPaid      decimal.Decimal
	currency        string
}

// RandomPosSale crea un PosSaleMother con valores aleatorios válidos
func RandomPosSale() *PosSaleMother {
	item := RandomPosSaleItem().MustBuild()
	// amountPaid debe ser >= subtotal del item
	amountPaid := item.Subtotal.Add(decimal.NewFromInt(100))

	return &PosSaleMother{
		tenantID:        uuid.New(),
		customerID:      uuid.New(),
		paymentMethodID: uuid.New(),
		items:           []entity.PosSaleItem{*item},
		discountAmount:  decimal.Zero,
		amountPaid:      amountPaid,
		currency:        "ARS",
	}
}

// WithTenantID establece el tenantID
func (m *PosSaleMother) WithTenantID(id uuid.UUID) *PosSaleMother {
	m.tenantID = id
	return m
}

// WithCustomerID establece el customerID
func (m *PosSaleMother) WithCustomerID(id uuid.UUID) *PosSaleMother {
	m.customerID = id
	return m
}

// WithPaymentMethodID establece el paymentMethodID
func (m *PosSaleMother) WithPaymentMethodID(id uuid.UUID) *PosSaleMother {
	m.paymentMethodID = id
	return m
}

// WithItems establece los items
func (m *PosSaleMother) WithItems(items []entity.PosSaleItem) *PosSaleMother {
	m.items = items
	return m
}

// WithDiscountAmount establece el descuento
func (m *PosSaleMother) WithDiscountAmount(amount decimal.Decimal) *PosSaleMother {
	m.discountAmount = amount
	return m
}

// WithAmountPaid establece el monto pagado
func (m *PosSaleMother) WithAmountPaid(amount decimal.Decimal) *PosSaleMother {
	m.amountPaid = amount
	return m
}

// WithCurrency establece la moneda
func (m *PosSaleMother) WithCurrency(currency string) *PosSaleMother {
	m.currency = currency
	return m
}

// Build construye la PosSale
func (m *PosSaleMother) Build() (*entity.PosSale, error) {
	return entity.NewPosSale(
		m.tenantID,
		m.customerID,
		m.paymentMethodID,
		m.items,
		m.discountAmount,
		m.amountPaid,
		m.currency,
	)
}

// MustBuild construye la PosSale o hace panic
func (m *PosSaleMother) MustBuild() *entity.PosSale {
	sale, err := m.Build()
	if err != nil {
		panic("PosSaleMother.MustBuild failed: " + err.Error())
	}
	return sale
}

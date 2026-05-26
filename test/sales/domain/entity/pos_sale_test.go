package entity_test

import (
	"testing"

	"sales/src/sales/domain/entity"
	"sales/test/sales/domain/mother"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- NewPosSale ---

func TestNewPosSale_WithValidData_ReturnsSale(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	customerID := uuid.New()
	paymentMethodID := uuid.New()
	item := mother.RandomPosSaleItem().WithUnitPrice(decimal.NewFromInt(100)).WithQuantity(2).MustBuild()
	items := []entity.PosSaleItem{*item}
	discount := decimal.Zero
	amountPaid := decimal.NewFromInt(200)

	// Act
	sale, err := entity.NewPosSale(tenantID, customerID, paymentMethodID, items, discount, amountPaid, "ARS")

	// Assert
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, sale.ID)
	assert.Equal(t, tenantID, sale.TenantID)
	assert.Equal(t, customerID, sale.CustomerID)
	assert.Equal(t, paymentMethodID, sale.PaymentMethodID)
	assert.Equal(t, "ARS", sale.Currency)
	assert.NotZero(t, sale.CreatedAt)
	assert.Len(t, sale.Items, 1)
}

func TestNewPosSale_CalculatesTotalAmountFromItems(t *testing.T) {
	// Arrange
	item1 := mother.RandomPosSaleItem().WithUnitPrice(decimal.NewFromInt(100)).WithQuantity(2).MustBuild() // subtotal=200
	item2 := mother.RandomPosSaleItem().WithUnitPrice(decimal.NewFromInt(50)).WithQuantity(3).MustBuild()  // subtotal=150
	items := []entity.PosSaleItem{*item1, *item2}
	amountPaid := decimal.NewFromInt(350)

	// Act
	sale, err := entity.NewPosSale(uuid.New(), uuid.New(), uuid.New(), items, decimal.Zero, amountPaid, "ARS")

	// Assert
	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(350).Equal(sale.TotalAmount), "TotalAmount should be 200+150=350")
	assert.True(t, decimal.NewFromInt(350).Equal(sale.FinalAmount), "FinalAmount with no discount = TotalAmount")
}

func TestNewPosSale_AppliesDiscount(t *testing.T) {
	// Arrange
	item := mother.RandomPosSaleItem().WithUnitPrice(decimal.NewFromInt(100)).WithQuantity(1).MustBuild()
	items := []entity.PosSaleItem{*item}
	discount := decimal.NewFromInt(20)
	amountPaid := decimal.NewFromInt(80)

	// Act
	sale, err := entity.NewPosSale(uuid.New(), uuid.New(), uuid.New(), items, discount, amountPaid, "ARS")

	// Assert
	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(100).Equal(sale.TotalAmount))
	assert.True(t, decimal.NewFromInt(20).Equal(sale.DiscountAmount))
	assert.True(t, decimal.NewFromInt(80).Equal(sale.FinalAmount))
}

func TestNewPosSale_CalculatesChange(t *testing.T) {
	// Arrange
	item := mother.RandomPosSaleItem().WithUnitPrice(decimal.NewFromInt(100)).WithQuantity(1).MustBuild()
	items := []entity.PosSaleItem{*item}
	amountPaid := decimal.NewFromInt(150) // paga 150 por 100

	// Act
	sale, err := entity.NewPosSale(uuid.New(), uuid.New(), uuid.New(), items, decimal.Zero, amountPaid, "ARS")

	// Assert
	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(50).Equal(sale.Change), "Change should be 150-100=50")
}

func TestNewPosSale_DiscountGreaterThanTotal_FinalAmountIsZero(t *testing.T) {
	// Arrange
	item := mother.RandomPosSaleItem().WithUnitPrice(decimal.NewFromInt(50)).WithQuantity(1).MustBuild()
	items := []entity.PosSaleItem{*item}
	discount := decimal.NewFromInt(100) // descuento mayor que el total
	amountPaid := decimal.Zero          // final es 0

	// Act
	sale, err := entity.NewPosSale(uuid.New(), uuid.New(), uuid.New(), items, discount, amountPaid, "ARS")

	// Assert
	require.NoError(t, err)
	assert.True(t, decimal.Zero.Equal(sale.FinalAmount), "FinalAmount should not go negative")
}

func TestNewPosSale_DefaultCurrency_IsARS(t *testing.T) {
	// Arrange
	item := mother.RandomPosSaleItem().MustBuild()
	amountPaid := item.Subtotal

	// Act
	sale, err := entity.NewPosSale(uuid.New(), uuid.New(), uuid.New(), []entity.PosSaleItem{*item}, decimal.Zero, amountPaid, "")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "ARS", sale.Currency)
}

func TestNewPosSale_AssignsPosSaleIDToItems(t *testing.T) {
	// Arrange
	item := mother.RandomPosSaleItem().MustBuild()
	amountPaid := item.Subtotal

	// Act
	sale, err := entity.NewPosSale(uuid.New(), uuid.New(), uuid.New(), []entity.PosSaleItem{*item}, decimal.Zero, amountPaid, "ARS")

	// Assert
	require.NoError(t, err)
	for _, i := range sale.Items {
		assert.Equal(t, sale.ID, i.PosSaleID)
	}
}

// --- Validaciones de error ---

func TestNewPosSale_WithNilTenantID_ReturnsError(t *testing.T) {
	// Arrange
	item := mother.RandomPosSaleItem().MustBuild()

	// Act
	sale, err := entity.NewPosSale(uuid.Nil, uuid.New(), uuid.New(), []entity.PosSaleItem{*item}, decimal.Zero, item.Subtotal, "ARS")

	// Assert
	assert.Nil(t, sale)
	assert.ErrorIs(t, err, entity.ErrTenantIDRequired)
}

func TestNewPosSale_WithNilCustomerID_AllowsAnonymousSale(t *testing.T) {
	// uuid.Nil = Consumidor Final — venta anónima permitida
	item := mother.RandomPosSaleItem().MustBuild()

	sale, err := entity.NewPosSale(uuid.New(), uuid.Nil, uuid.New(), []entity.PosSaleItem{*item}, decimal.Zero, item.Subtotal, "ARS")

	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, sale.CustomerID)
}

func TestNewPosSale_WithNilPaymentMethodID_ReturnsError(t *testing.T) {
	// Arrange
	item := mother.RandomPosSaleItem().MustBuild()

	// Act
	sale, err := entity.NewPosSale(uuid.New(), uuid.New(), uuid.Nil, []entity.PosSaleItem{*item}, decimal.Zero, item.Subtotal, "ARS")

	// Assert
	assert.Nil(t, sale)
	assert.Error(t, err) // Retorna ErrTenantIDRequired (TODO en código fuente)
}

func TestNewPosSale_WithEmptyItems_ReturnsError(t *testing.T) {
	// Arrange & Act
	sale, err := entity.NewPosSale(uuid.New(), uuid.New(), uuid.New(), []entity.PosSaleItem{}, decimal.Zero, decimal.NewFromInt(100), "ARS")

	// Assert
	assert.Nil(t, sale)
	assert.ErrorIs(t, err, entity.ErrPosSaleMustHaveItems)
}

func TestNewPosSale_WithNilItems_ReturnsError(t *testing.T) {
	// Arrange & Act
	sale, err := entity.NewPosSale(uuid.New(), uuid.New(), uuid.New(), nil, decimal.Zero, decimal.NewFromInt(100), "ARS")

	// Assert
	assert.Nil(t, sale)
	assert.ErrorIs(t, err, entity.ErrPosSaleMustHaveItems)
}

func TestNewPosSale_WithNegativeDiscount_ReturnsError(t *testing.T) {
	// Arrange
	item := mother.RandomPosSaleItem().MustBuild()

	// Act
	sale, err := entity.NewPosSale(uuid.New(), uuid.New(), uuid.New(), []entity.PosSaleItem{*item}, decimal.NewFromInt(-10), item.Subtotal, "ARS")

	// Assert
	assert.Nil(t, sale)
	assert.ErrorIs(t, err, entity.ErrInvalidDiscount)
}

func TestNewPosSale_WithInsufficientPayment_ReturnsError(t *testing.T) {
	// Arrange
	item := mother.RandomPosSaleItem().WithUnitPrice(decimal.NewFromInt(100)).WithQuantity(1).MustBuild()
	amountPaid := decimal.NewFromInt(50) // paga menos del total

	// Act
	sale, err := entity.NewPosSale(uuid.New(), uuid.New(), uuid.New(), []entity.PosSaleItem{*item}, decimal.Zero, amountPaid, "ARS")

	// Assert
	assert.Nil(t, sale)
	assert.ErrorIs(t, err, entity.ErrInsufficientPayment)
}

// --- TotalItems ---

func TestPosSale_TotalItems_ReturnsCorrectCount(t *testing.T) {
	// Arrange
	item1 := mother.RandomPosSaleItem().WithUnitPrice(decimal.NewFromInt(100)).WithQuantity(1).MustBuild()
	item2 := mother.RandomPosSaleItem().WithUnitPrice(decimal.NewFromInt(200)).WithQuantity(1).MustBuild()
	items := []entity.PosSaleItem{*item1, *item2}
	amountPaid := decimal.NewFromInt(300)

	sale, err := entity.NewPosSale(uuid.New(), uuid.New(), uuid.New(), items, decimal.Zero, amountPaid, "ARS")
	require.NoError(t, err)

	// Act & Assert
	assert.Equal(t, 2, sale.TotalItems())
}

// --- ObjectMother integration ---

func TestPosSale_UsingMother_ReturnsValidSale(t *testing.T) {
	// Arrange & Act
	sale := mother.RandomPosSale().MustBuild()

	// Assert
	assert.NotEqual(t, uuid.Nil, sale.ID)
	assert.NotEqual(t, uuid.Nil, sale.TenantID)
	assert.NotEqual(t, uuid.Nil, sale.CustomerID)
	assert.True(t, sale.TotalItems() >= 1)
	assert.True(t, sale.FinalAmount.GreaterThanOrEqual(decimal.Zero))
}

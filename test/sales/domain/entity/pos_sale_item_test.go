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

func TestNewPosSaleItem_WithValidData_ReturnsItem(t *testing.T) {
	// Arrange
	posSaleID := uuid.New()
	sku := "SKU-001"
	productName := "Producto A"
	quantity := 3
	unitPrice := decimal.NewFromFloat(100.50)
	stockEntryID := uuid.New()

	// Act
	item, err := entity.NewPosSaleItem(posSaleID, sku, productName, quantity, unitPrice, stockEntryID)

	// Assert
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, item.ID)
	assert.Equal(t, posSaleID, item.PosSaleID)
	assert.Equal(t, sku, item.SKU)
	assert.Equal(t, productName, item.ProductName)
	assert.Equal(t, quantity, item.Quantity)
	assert.True(t, unitPrice.Equal(item.UnitPrice))
	assert.Equal(t, stockEntryID, item.StockEntryID)
}

func TestNewPosSaleItem_CalculatesSubtotalCorrectly(t *testing.T) {
	// Arrange
	unitPrice := decimal.NewFromFloat(150.00)
	quantity := 4

	// Act
	item, err := entity.NewPosSaleItem(uuid.New(), "SKU-001", "Producto", quantity, unitPrice, uuid.New())

	// Assert
	require.NoError(t, err)
	expectedSubtotal := decimal.NewFromFloat(600.00) // 150 * 4
	assert.True(t, expectedSubtotal.Equal(item.Subtotal))
}

func TestNewPosSaleItem_WithEmptySKU_ReturnsError(t *testing.T) {
	// Arrange & Act
	item, err := entity.NewPosSaleItem(uuid.New(), "", "Producto", 1, decimal.NewFromInt(100), uuid.New())

	// Assert
	assert.Nil(t, item)
	assert.ErrorIs(t, err, entity.ErrSKURequired)
}

func TestNewPosSaleItem_WithEmptyProductName_ReturnsError(t *testing.T) {
	// Arrange & Act
	item, err := entity.NewPosSaleItem(uuid.New(), "SKU-001", "", 1, decimal.NewFromInt(100), uuid.New())

	// Assert
	assert.Nil(t, item)
	assert.ErrorIs(t, err, entity.ErrProductNameRequired)
}

func TestNewPosSaleItem_WithZeroQuantity_ReturnsError(t *testing.T) {
	// Arrange & Act
	item, err := entity.NewPosSaleItem(uuid.New(), "SKU-001", "Producto", 0, decimal.NewFromInt(100), uuid.New())

	// Assert
	assert.Nil(t, item)
	assert.ErrorIs(t, err, entity.ErrInvalidQuantity)
}

func TestNewPosSaleItem_WithNegativeQuantity_ReturnsError(t *testing.T) {
	// Arrange & Act
	item, err := entity.NewPosSaleItem(uuid.New(), "SKU-001", "Producto", -1, decimal.NewFromInt(100), uuid.New())

	// Assert
	assert.Nil(t, item)
	assert.ErrorIs(t, err, entity.ErrInvalidQuantity)
}

func TestNewPosSaleItem_WithNegativePrice_ReturnsError(t *testing.T) {
	// Arrange & Act
	item, err := entity.NewPosSaleItem(uuid.New(), "SKU-001", "Producto", 1, decimal.NewFromFloat(-10.0), uuid.New())

	// Assert
	assert.Nil(t, item)
	assert.ErrorIs(t, err, entity.ErrInvalidPrice)
}

func TestNewPosSaleItem_WithZeroPrice_Succeeds(t *testing.T) {
	// Arrange & Act — precio 0 es válido (regalo/bonificación)
	item, err := entity.NewPosSaleItem(uuid.New(), "SKU-001", "Producto", 1, decimal.Zero, uuid.New())

	// Assert
	require.NoError(t, err)
	assert.True(t, decimal.Zero.Equal(item.Subtotal))
}

func TestNewPosSaleItem_WithNilStockEntryID_ReturnsError(t *testing.T) {
	// Arrange & Act
	item, err := entity.NewPosSaleItem(uuid.New(), "SKU-001", "Producto", 1, decimal.NewFromInt(100), uuid.Nil)

	// Assert
	assert.Nil(t, item)
	assert.ErrorIs(t, err, entity.ErrStockEntryIDRequired)
}

func TestNewPosSaleItem_UsingMother_ReturnsValidItem(t *testing.T) {
	// Arrange & Act
	item := mother.RandomPosSaleItem().
		WithSKU("SKU-CUSTOM").
		WithQuantity(5).
		WithUnitPrice(decimal.NewFromFloat(200.0)).
		MustBuild()

	// Assert
	assert.Equal(t, "SKU-CUSTOM", item.SKU)
	assert.Equal(t, 5, item.Quantity)
	expectedSubtotal := decimal.NewFromFloat(1000.0)
	assert.True(t, expectedSubtotal.Equal(item.Subtotal))
}

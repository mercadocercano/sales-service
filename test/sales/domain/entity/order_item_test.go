package entity_test

import (
	"encoding/json"
	"testing"

	"sales/src/sales/domain/entity"
	"sales/test/sales/domain/mother"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOrderItem_WithValidData_ReturnsItem(t *testing.T) {
	// Arrange
	orderID := "order-123"
	sku := "SKU-001"
	quantity := 5

	// Act
	item, err := entity.NewOrderItem(orderID, sku, quantity)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, item.ItemID)
	assert.Equal(t, orderID, item.OrderID)
	assert.Equal(t, sku, item.SKU)
	assert.Equal(t, quantity, item.Quantity)
	assert.Equal(t, float64(0), item.UnitPrice)
}

func TestNewOrderItem_WithEmptySKU_ReturnsError(t *testing.T) {
	// Arrange & Act
	item, err := entity.NewOrderItem("order-123", "", 5)

	// Assert
	assert.Nil(t, item)
	assert.ErrorIs(t, err, entity.ErrSKURequired)
}

func TestNewOrderItem_WithZeroQuantity_ReturnsError(t *testing.T) {
	// Arrange & Act
	item, err := entity.NewOrderItem("order-123", "SKU-001", 0)

	// Assert
	assert.Nil(t, item)
	assert.ErrorIs(t, err, entity.ErrInvalidQuantity)
}

func TestNewOrderItem_WithNegativeQuantity_ReturnsError(t *testing.T) {
	// Arrange & Act
	item, err := entity.NewOrderItem("order-123", "SKU-001", -1)

	// Assert
	assert.Nil(t, item)
	assert.ErrorIs(t, err, entity.ErrInvalidQuantity)
}

func TestNewOrderItemWithSnapshots_WithValidData_ReturnsItem(t *testing.T) {
	// Arrange
	orderID := "order-123"
	sku := "SKU-001"
	quantity := 3
	unitPrice := 250.50
	productSnap := json.RawMessage(`{"name":"Producto A"}`)
	variantSnap := json.RawMessage(`{"color":"azul"}`)

	// Act
	item, err := entity.NewOrderItemWithSnapshots(orderID, sku, quantity, unitPrice, productSnap, variantSnap)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, item.ItemID)
	assert.Equal(t, orderID, item.OrderID)
	assert.Equal(t, sku, item.SKU)
	assert.Equal(t, quantity, item.Quantity)
	assert.Equal(t, unitPrice, item.UnitPrice)
	assert.JSONEq(t, `{"name":"Producto A"}`, string(item.ProductSnapshot))
	assert.JSONEq(t, `{"color":"azul"}`, string(item.VariantSnapshot))
}

func TestNewOrderItemWithSnapshots_WithEmptySKU_ReturnsError(t *testing.T) {
	// Arrange & Act
	item, err := entity.NewOrderItemWithSnapshots("order-123", "", 3, 100.0, nil, nil)

	// Assert
	assert.Nil(t, item)
	assert.ErrorIs(t, err, entity.ErrSKURequired)
}

func TestNewOrderItemWithSnapshots_WithZeroQuantity_ReturnsError(t *testing.T) {
	// Arrange & Act
	item, err := entity.NewOrderItemWithSnapshots("order-123", "SKU-001", 0, 100.0, nil, nil)

	// Assert
	assert.Nil(t, item)
	assert.ErrorIs(t, err, entity.ErrInvalidQuantity)
}

func TestNewOrderItemWithSnapshots_WithZeroPrice_ReturnsError(t *testing.T) {
	// Arrange & Act
	item, err := entity.NewOrderItemWithSnapshots("order-123", "SKU-001", 3, 0, nil, nil)

	// Assert
	assert.Nil(t, item)
	assert.ErrorIs(t, err, entity.ErrInvalidPrice)
}

func TestNewOrderItemWithSnapshots_WithNegativePrice_ReturnsError(t *testing.T) {
	// Arrange & Act
	item, err := entity.NewOrderItemWithSnapshots("order-123", "SKU-001", 3, -10.0, nil, nil)

	// Assert
	assert.Nil(t, item)
	assert.ErrorIs(t, err, entity.ErrInvalidPrice)
}

func TestOrderItem_CalculateTotal_ReturnsUnitPriceTimesQuantity(t *testing.T) {
	// Arrange
	item := mother.RandomOrderItem().
		WithUnitPrice(100.0).
		WithQuantity(3).
		MustBuild()

	// Act
	total := item.CalculateTotal()

	// Assert
	assert.Equal(t, 300.0, total)
}

func TestOrderItem_CalculateTotal_WithZeroPrice_ReturnsZero(t *testing.T) {
	// Arrange — item sin snapshots tiene UnitPrice = 0
	item := mother.RandomOrderItem().MustBuild()

	// Act
	total := item.CalculateTotal()

	// Assert
	assert.Equal(t, 0.0, total)
}

func TestOrderItem_CalculateTotal_WithDecimalPrice_ReturnsCorrectTotal(t *testing.T) {
	// Arrange
	item := mother.RandomOrderItem().
		WithUnitPrice(33.33).
		WithQuantity(3).
		MustBuild()

	// Act
	total := item.CalculateTotal()

	// Assert
	assert.InDelta(t, 99.99, total, 0.001)
}

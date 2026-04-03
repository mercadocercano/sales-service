package entity_test

import (
	"testing"

	"sales/src/sales/domain/entity"
	"sales/test/sales/domain/mother"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- NewOrder ---

func TestNewOrder_WithValidData_ReturnsOrder(t *testing.T) {
	// Arrange
	item := mother.RandomOrderItem().MustBuild()

	// Act
	order, err := entity.NewOrder("tenant-1", "customer-1", []entity.OrderItem{*item})

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, order.OrderID)
	assert.Equal(t, "tenant-1", order.TenantID)
	assert.Equal(t, "customer-1", order.CustomerID)
	assert.Equal(t, entity.OrderStatusCreated, order.Status)
	assert.Len(t, order.Items, 1)
	assert.NotZero(t, order.CreatedAt)
}

func TestNewOrder_AssignsOrderIDToItems(t *testing.T) {
	// Arrange
	item := mother.RandomOrderItem().WithOrderID("").MustBuild()

	// Act
	order, err := entity.NewOrder("tenant-1", "customer-1", []entity.OrderItem{*item})

	// Assert
	require.NoError(t, err)
	for _, i := range order.Items {
		assert.Equal(t, order.OrderID, i.OrderID)
	}
}

func TestNewOrder_WithEmptyTenantID_ReturnsError(t *testing.T) {
	// Arrange
	item := mother.RandomOrderItem().MustBuild()

	// Act
	order, err := entity.NewOrder("", "customer-1", []entity.OrderItem{*item})

	// Assert
	assert.Nil(t, order)
	assert.ErrorIs(t, err, entity.ErrTenantIDRequired)
}

func TestNewOrder_WithEmptyCustomerID_ReturnsError(t *testing.T) {
	// Arrange
	item := mother.RandomOrderItem().MustBuild()

	// Act
	order, err := entity.NewOrder("tenant-1", "", []entity.OrderItem{*item})

	// Assert
	assert.Nil(t, order)
	assert.ErrorIs(t, err, entity.ErrCustomerIDRequired)
}

func TestNewOrder_WithEmptyItems_ReturnsError(t *testing.T) {
	// Arrange & Act
	order, err := entity.NewOrder("tenant-1", "customer-1", []entity.OrderItem{})

	// Assert
	assert.Nil(t, order)
	assert.ErrorIs(t, err, entity.ErrOrderMustHaveItems)
}

func TestNewOrder_WithNilItems_ReturnsError(t *testing.T) {
	// Arrange & Act
	order, err := entity.NewOrder("tenant-1", "customer-1", nil)

	// Assert
	assert.Nil(t, order)
	assert.ErrorIs(t, err, entity.ErrOrderMustHaveItems)
}

// --- NewOrderSingleItem ---

func TestNewOrderSingleItem_WithValidData_ReturnsOrder(t *testing.T) {
	// Arrange & Act
	order, err := entity.NewOrderSingleItem("tenant-1", "customer-1", "SKU-001", 5)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, order.OrderID)
	assert.Equal(t, "tenant-1", order.TenantID)
	assert.Equal(t, "customer-1", order.CustomerID)
	assert.Equal(t, entity.OrderStatusCreated, order.Status)
	assert.Len(t, order.Items, 1)
	assert.Equal(t, "SKU-001", order.Items[0].SKU)
	assert.Equal(t, 5, order.Items[0].Quantity)
}

func TestNewOrderSingleItem_WithEmptyTenantID_ReturnsError(t *testing.T) {
	// Arrange & Act
	order, err := entity.NewOrderSingleItem("", "customer-1", "SKU-001", 5)

	// Assert
	assert.Nil(t, order)
	assert.ErrorIs(t, err, entity.ErrTenantIDRequired)
}

func TestNewOrderSingleItem_WithEmptyCustomerID_ReturnsError(t *testing.T) {
	// Arrange & Act
	order, err := entity.NewOrderSingleItem("tenant-1", "", "SKU-001", 5)

	// Assert
	assert.Nil(t, order)
	assert.ErrorIs(t, err, entity.ErrCustomerIDRequired)
}

func TestNewOrderSingleItem_WithEmptySKU_ReturnsError(t *testing.T) {
	// Arrange & Act
	order, err := entity.NewOrderSingleItem("tenant-1", "customer-1", "", 5)

	// Assert
	assert.Nil(t, order)
	assert.ErrorIs(t, err, entity.ErrSKURequired)
}

func TestNewOrderSingleItem_WithInvalidQuantity_ReturnsError(t *testing.T) {
	// Arrange & Act
	order, err := entity.NewOrderSingleItem("tenant-1", "customer-1", "SKU-001", 0)

	// Assert
	assert.Nil(t, order)
	assert.ErrorIs(t, err, entity.ErrInvalidQuantity)
}

// --- AddItem ---

func TestOrder_AddItem_WithValidData_AddsItem(t *testing.T) {
	// Arrange
	order := mother.RandomOrder().MustBuild()
	initialCount := order.TotalItems()

	// Act
	err := order.AddItem("SKU-NEW", 3)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, initialCount+1, order.TotalItems())
	lastItem := order.Items[len(order.Items)-1]
	assert.Equal(t, "SKU-NEW", lastItem.SKU)
	assert.Equal(t, 3, lastItem.Quantity)
	assert.Equal(t, order.OrderID, lastItem.OrderID)
}

func TestOrder_AddItem_WithEmptySKU_ReturnsError(t *testing.T) {
	// Arrange
	order := mother.RandomOrder().MustBuild()

	// Act
	err := order.AddItem("", 3)

	// Assert
	assert.ErrorIs(t, err, entity.ErrSKURequired)
}

func TestOrder_AddItem_WithInvalidQuantity_ReturnsError(t *testing.T) {
	// Arrange
	order := mother.RandomOrder().MustBuild()

	// Act
	err := order.AddItem("SKU-NEW", 0)

	// Assert
	assert.ErrorIs(t, err, entity.ErrInvalidQuantity)
}

// --- TotalItems ---

func TestOrder_TotalItems_ReturnsCorrectCount(t *testing.T) {
	// Arrange
	item1 := mother.RandomOrderItem().WithSKU("SKU-1").MustBuild()
	item2 := mother.RandomOrderItem().WithSKU("SKU-2").MustBuild()
	order, err := entity.NewOrder("tenant-1", "customer-1", []entity.OrderItem{*item1, *item2})
	require.NoError(t, err)

	// Act & Assert
	assert.Equal(t, 2, order.TotalItems())
}

// --- CalculateTotal ---

func TestOrder_CalculateTotal_SumsAllItemTotals(t *testing.T) {
	// Arrange
	item1 := mother.RandomOrderItem().WithUnitPrice(100.0).WithQuantity(2).MustBuild()
	item2 := mother.RandomOrderItem().WithUnitPrice(50.0).WithQuantity(3).MustBuild()
	order, err := entity.NewOrder("tenant-1", "customer-1", []entity.OrderItem{*item1, *item2})
	require.NoError(t, err)

	// Act
	total := order.CalculateTotal()

	// Assert — (100*2) + (50*3) = 200 + 150 = 350
	assert.Equal(t, 350.0, total)
}

func TestOrder_CalculateTotal_WithNoPrice_ReturnsZero(t *testing.T) {
	// Arrange — items sin snapshots tienen UnitPrice=0
	order := mother.RandomOrder().MustBuild()

	// Act
	total := order.CalculateTotal()

	// Assert
	assert.Equal(t, 0.0, total)
}

// --- Confirm ---

func TestOrder_Confirm_FromCreatedState_Succeeds(t *testing.T) {
	// Arrange
	order := mother.RandomOrder().MustBuild()
	assert.Equal(t, entity.OrderStatusCreated, order.Status)

	// Act
	err := order.Confirm()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, entity.OrderStatusConfirmed, order.Status)
}

func TestOrder_Confirm_FromConfirmedState_ReturnsError(t *testing.T) {
	// Arrange
	order := mother.RandomOrder().MustBuild()
	_ = order.Confirm()

	// Act
	err := order.Confirm()

	// Assert
	assert.ErrorIs(t, err, entity.ErrOrderNotInCreatedState)
}

func TestOrder_Confirm_FromCanceledState_ReturnsError(t *testing.T) {
	// Arrange
	order := mother.RandomOrder().MustBuild()
	_ = order.Confirm()
	_ = order.Cancel()

	// Act
	err := order.Confirm()

	// Assert
	assert.ErrorIs(t, err, entity.ErrOrderNotInCreatedState)
}

// --- Cancel ---

func TestOrder_Cancel_FromConfirmedState_Succeeds(t *testing.T) {
	// Arrange
	order := mother.RandomOrder().MustBuild()
	_ = order.Confirm()
	assert.Equal(t, entity.OrderStatusConfirmed, order.Status)

	// Act
	err := order.Cancel()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, entity.OrderStatusCanceled, order.Status)
}

func TestOrder_Cancel_FromCreatedState_ReturnsError(t *testing.T) {
	// Arrange
	order := mother.RandomOrder().MustBuild()

	// Act
	err := order.Cancel()

	// Assert
	assert.ErrorIs(t, err, entity.ErrOrderNotInConfirmedState)
}

// --- AssignOrderNumber ---

func TestOrder_AssignOrderNumber_SetsNumber(t *testing.T) {
	// Arrange
	order := mother.RandomOrder().MustBuild()
	assert.Nil(t, order.OrderNumber)

	// Act
	order.AssignOrderNumber(42)

	// Assert
	require.NotNil(t, order.OrderNumber)
	assert.Equal(t, 42, *order.OrderNumber)
}

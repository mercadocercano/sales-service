package entity

import "errors"

var (
	ErrTenantIDRequired         = errors.New("tenant_id is required")
	ErrSKURequired              = errors.New("sku is required")
	ErrInvalidQuantity          = errors.New("quantity must be greater than 0")
	ErrOrderNotFound            = errors.New("order not found")
	ErrOrderNotInCreatedState   = errors.New("order is not in CREATED state")
	ErrOrderNotInConfirmedState = errors.New("order is not in CONFIRMED state")
	ErrOrderMustHaveItems       = errors.New("order must have at least one item")
	
	// HITO B - POS Multi-Item
	ErrProductNameRequired  = errors.New("product_name is required")
	ErrInvalidPrice         = errors.New("price must be greater than or equal to 0")
	ErrStockEntryIDRequired = errors.New("stock_entry_id is required")
	ErrInvalidDiscount      = errors.New("discount_amount must be greater than or equal to 0")
	ErrPosSaleMustHaveItems = errors.New("pos_sale must have at least one item")
	
	// HITO: POST /pos/sale devuelve DTO listo para imprimir
	ErrInsufficientPayment = errors.New("amount_paid must be greater than or equal to final_amount")
)

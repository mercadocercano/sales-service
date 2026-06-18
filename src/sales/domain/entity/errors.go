package entity

import "errors"

var (
	ErrTenantIDRequired         = errors.New("tenant_id is required")
	ErrCustomerIDRequired       = errors.New("customer_id is required") // HITO v0.6
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

	ErrPaymentMethodRequired = errors.New("payment_method_id is required")

	// E18 Tramo B - Detalle de comprobante POS
	ErrPosSaleNotFound = errors.New("pos_sale not found")

	// E18 Tramo A - Caja / sesión de caja (ADR-003)
	ErrPointOfSaleRequired    = errors.New("point_of_sale_id is required")
	ErrOpenedByRequired       = errors.New("opened_by (user) is required")
	ErrOpeningAmountNegative  = errors.New("opening_amount must be greater than or equal to 0")
	ErrCashSessionNotFound    = errors.New("cash register session not found")
	ErrCashSessionAlreadyOpen = errors.New("a cash register session is already open for this terminal")
	ErrCashSessionNotOpen     = errors.New("cash register session is not open")
	ErrInvalidCashStatus      = errors.New("invalid cash session status transition")
	ErrInvalidMovementType    = errors.New("invalid cash movement type")
	ErrMovementAmountInvalid  = errors.New("movement amount must be greater than 0")
	ErrMovementReasonRequired = errors.New("movement reason is required")
	ErrCountedAmountNegative  = errors.New("counted_amount must be greater than or equal to 0")
	ErrCashSessionNotInReview = errors.New("cash register session is not pending review")
	// Separación de funciones (ADR-003 gate #2): el aprobador no puede ser el operador.
	ErrApproverIsOperator = errors.New("approver must be different from the cashier who operated the session")
	// Concurrencia (optimistic lock): otra transacción modificó la sesión.
	ErrCashSessionConflict = errors.New("cash register session was modified concurrently, retry")
	// Fail-closed (sign-off seguridad C1): no se pudo resolver el método de pago "cash".
	// Cerrar la caja sin poder identificar el efectivo subvaluaría el expected en
	// silencio; en una operación de dinero se falla en vez de degradar.
	ErrCashMethodUnresolved = errors.New("cash payment method could not be resolved; cannot compute arqueo")
)

package response

// OrderFinancialResponse es la respuesta de GET /orders/:id/financial
// Solo datos de sales_orders + sales_payments. No consulta ledger.
type OrderFinancialResponse struct {
	OrderID        string   `json:"order_id"`
	TenantID       string   `json:"tenant_id"`
	TotalAmount    float64  `json:"total_amount"`
	TotalPaid      float64  `json:"total_paid"`
	Remaining      float64  `json:"remaining"`
	Status         string   `json:"status"` // CREATED | CONFIRMED | PAID | CANCELED
	PaymentCount   int      `json:"payment_count"`
	LastPaymentAt  *string  `json:"last_payment_at,omitempty"` // RFC3339
}

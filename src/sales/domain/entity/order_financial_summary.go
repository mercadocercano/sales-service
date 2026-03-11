package entity

import "time"

// OrderFinancialSummary es el resumen financiero de una orden (solo sales_orders + sales_payments)
// Usado por GET /orders/:id/financial. No consulta ledger.
type OrderFinancialSummary struct {
	OrderID        string
	TenantID      string
	TotalAmount   float64
	TotalPaid     float64
	Remaining     float64
	Status        string
	PaymentCount  int
	LastPaymentAt *time.Time
}

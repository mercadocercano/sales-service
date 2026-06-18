package request

import "github.com/shopspring/decimal"

// OpenCashSessionRequest abre una caja para una terminal con su monto inicial.
type OpenCashSessionRequest struct {
	PointOfSaleID string          `json:"point_of_sale_id" binding:"required"`
	OpeningAmount decimal.Decimal `json:"opening_amount"`
}

// CashMovementRequest registra un movimiento manual de efectivo.
// type: income | expense | withdrawal | correction
type CashMovementRequest struct {
	Type   string          `json:"type" binding:"required"`
	Amount decimal.Decimal `json:"amount" binding:"required"`
	Reason string          `json:"reason" binding:"required"`
}

// CloseCashSessionRequest cierra la caja con el efectivo contado por el cajero.
// El expected y la difference se calculan server-side (el cliente nunca los manda).
type CloseCashSessionRequest struct {
	CountedAmount decimal.Decimal `json:"counted_amount"`
}

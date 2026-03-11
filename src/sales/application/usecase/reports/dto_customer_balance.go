package reports

import "sales/src/sales/domain/port"

// CustomerBalanceResponse es la respuesta de GET /reports/customer-balance.
// Resumen y aging siempre completos; open_orders paginado y opcionalmente filtrado por aging_bucket.
type CustomerBalanceResponse struct {
	CustomerID   string                      `json:"customer_id"`
	Summary     port.CustomerBalanceSummary `json:"summary"`
	Aging       port.AgingRow               `json:"aging"`
	BucketCounts port.BucketCounts           `json:"bucket_counts"`
	OpenOrders  []port.OpenOrderRow         `json:"open_orders"`
}

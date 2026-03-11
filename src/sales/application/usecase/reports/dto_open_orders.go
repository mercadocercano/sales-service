package reports

import "sales/src/sales/domain/port"

// OpenOrdersReportResponse es la respuesta del reporte de órdenes abiertas.
// Incluye resumen (filtrado), conteos por bucket (para tabs) y listado paginado.
type OpenOrdersReportResponse struct {
	Summary      port.OpenOrdersSummary `json:"summary"`
	BucketCounts port.BucketCounts      `json:"bucket_counts"`
	Orders       []port.OpenOrderRow    `json:"orders"`
}

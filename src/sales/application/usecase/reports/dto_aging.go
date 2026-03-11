package reports

import "sales/src/sales/domain/port"

// AgingReportResponse es la respuesta del reporte de aging por cliente.
type AgingReportResponse struct {
	TotalOutstanding float64        `json:"total_outstanding"`
	Customers        []port.AgingRow `json:"customers"`
}

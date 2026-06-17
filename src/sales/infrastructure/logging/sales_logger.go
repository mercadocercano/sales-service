package logging

import (
	"io"

	"sales/src/sales/domain/port"

	sharedlog "github.com/hornosg/go-shared/infrastructure/logging"
)

// SalesLogger implementa port.SalesEventLogger emitiendo una línea JSON canónica
// (ADR-001) por evento, delegando el envelope (ts/level/service/event + campos flat
// omitempty) en go-shared CanonicalLogger (>= v0.8.0). El mapeo struct→fields y las
// reglas de nivel por evento viven acá; el formato canónico es compartido por la flota.
type SalesLogger struct {
	canonical *sharedlog.CanonicalLogger
}

// NewSalesLogger crea el adapter escribiendo a stdout. El service se fija acá, nunca por-call.
func NewSalesLogger(service string) *SalesLogger {
	return &SalesLogger{canonical: sharedlog.NewCanonicalLogger(service)}
}

// NewSalesLoggerWithWriter permite inyectar un io.Writer (tests).
func NewSalesLoggerWithWriter(service string, w io.Writer) *SalesLogger {
	return &SalesLogger{canonical: sharedlog.NewCanonicalLoggerWithWriter(service, w)}
}

// levelFor aplica las reglas de nivel del ADR-001 por tipo de evento.
func levelFor(event string) string {
	switch event {
	case "orders.pos_sale_completed":
		return "info"
	case "orders.pos_sale_failed":
		return "warn"
	case "orders.pos_sale_persist_failed":
		return "error"
	default:
		return "info"
	}
}

func (l *SalesLogger) Log(e port.SalesEvent) {
	fields := map[string]any{
		"tenant_id":      e.TenantID,
		"user_id":        e.UserID,
		"sale_id":        e.SaleID,
		"sku":            e.SKU,
		"payment_method": e.PaymentMethod,
		"final_amount":   e.FinalAmount,
		"reason":         e.Reason,
	}
	if e.Items > 0 {
		fields["items"] = e.Items
	}
	l.canonical.Emit(levelFor(e.Event), e.Event, fields)
}

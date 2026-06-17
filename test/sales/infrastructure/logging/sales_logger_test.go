package logging_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"sales/src/sales/domain/port"
	saleslog "sales/src/sales/infrastructure/logging"

	"github.com/stretchr/testify/assert"
)

// ADR-001: cada evento produce UNA línea JSON canónica con envelope ts/level/service/event.
func parseLine(t *testing.T, b []byte) map[string]any {
	t.Helper()
	lines := bytes.Split(bytes.TrimSpace(b), []byte("\n"))
	assert.Len(t, lines, 1, "debe ser exactamente una línea por evento")
	var m map[string]any
	assert.NoError(t, json.Unmarshal(lines[0], &m))
	return m
}

func TestSalesLogger_Completed_EnvelopeAndInfoLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := saleslog.NewSalesLoggerWithWriter("sales-test", &buf)

	logger.Log(port.SalesEvent{
		Event:       "orders.pos_sale_completed",
		TenantID:    "t-123",
		SaleID:      "s-456",
		Items:       3,
		FinalAmount: "1500.50",
	})

	line := parseLine(t, buf.Bytes())
	assert.Equal(t, "orders.pos_sale_completed", line["event"])
	assert.Equal(t, "info", line["level"])
	assert.Equal(t, "sales-test", line["service"])
	assert.NotEmpty(t, line["ts"], "ts (RFC3339 UTC) siempre presente")
	assert.Equal(t, "t-123", line["tenant_id"])
	assert.Equal(t, "s-456", line["sale_id"])
	assert.Equal(t, float64(3), line["items"])
	assert.Equal(t, "1500.50", line["final_amount"])
}

func TestSalesLogger_Failed_WarnLevel_OmitsEmptyFields(t *testing.T) {
	var buf bytes.Buffer
	logger := saleslog.NewSalesLoggerWithWriter("sales-test", &buf)

	logger.Log(port.SalesEvent{
		Event:    "orders.pos_sale_failed",
		TenantID: "t-123",
		Reason:   "payment_method_not_found: efectivo",
	})

	line := parseLine(t, buf.Bytes())
	assert.Equal(t, "warn", line["level"])
	assert.Equal(t, "payment_method_not_found: efectivo", line["reason"])
	// omitempty: los campos vacíos no aparecen
	_, hasSKU := line["sku"]
	assert.False(t, hasSKU, "sku vacío debe omitirse")
	_, hasItems := line["items"]
	assert.False(t, hasItems, "items=0 debe omitirse")
}

func TestSalesLogger_PersistFailed_ErrorLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := saleslog.NewSalesLoggerWithWriter("sales-test", &buf)

	logger.Log(port.SalesEvent{Event: "orders.pos_sale_persist_failed", TenantID: "t-1", Reason: "db down"})

	line := parseLine(t, buf.Bytes())
	assert.Equal(t, "error", line["level"])
	assert.Equal(t, "orders.pos_sale_persist_failed", line["event"])
}

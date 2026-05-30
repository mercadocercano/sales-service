package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// POSSalesTotal total de intentos de venta POS
	POSSalesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "pos_sales_total",
			Help: "Total POS sales attempts",
		},
	)

	// POSSalesSuccess ventas POS exitosas
	POSSalesSuccess = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "pos_sales_success",
			Help: "Successful POS sales",
		},
	)

	// POSSalesFailed ventas POS fallidas
	POSSalesFailed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "pos_sales_failed",
			Help: "Failed POS sales",
		},
	)

	// POSSaleLatency latencia de ventas POS (ms)
	POSSaleLatency = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "pos_sale_latency_ms",
			Help:    "Latency of POS sales in milliseconds",
			Buckets: prometheus.ExponentialBuckets(10, 2, 10),
		},
	)

	// TTFS time to first sale (segundos desde registro del tenant)
	TTFS = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ttfs_seconds",
			Help:    "Time to first sale (seconds since tenant registration)",
			Buckets: prometheus.ExponentialBuckets(60, 2, 10),
		},
		[]string{"tenant_id"},
	)

	// QuickstartStartedTotal H8: tenants que iniciaron quickstart
	QuickstartStartedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "quickstart_started_total",
			Help: "Total quickstart wizard starts",
		},
	)

	// QuickstartCompletedTotal H8: tenants que completaron quickstart
	QuickstartCompletedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "quickstart_completed_total",
			Help: "Total quickstart wizard completions",
		},
	)

	// DailyActiveTenantsTotal H8: incremento por cada tenant con primera venta del día
	// PromQL: increase(daily_active_tenants_total[1d])
	DailyActiveTenantsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "daily_active_tenants_total",
			Help: "Incremented when a tenant makes their first sale of the day",
		},
	)

	// MCSalesTotal cuenta ventas POS exitosas por tenant y método de pago.
	// Usado por metrics-gateway para KPIs: top-products, sales-by-hour, payment-methods.
	MCSalesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mc_sales_total",
			Help: "Total successful POS sales by tenant",
		},
		[]string{"tenant_id", "payment_method", "sku"},
	)

	// MCSalesAmountTotal acumula el monto vendido (ARS) por tenant.
	// Usado por metrics-gateway para KPIs: daily-revenue, weekly-revenue.
	MCSalesAmountTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mc_sales_amount_total",
			Help: "Total sales amount in ARS by tenant",
		},
		[]string{"tenant_id"},
	)
)

func init() {
	prometheus.MustRegister(
		POSSalesTotal,
		POSSalesSuccess,
		POSSalesFailed,
		POSSaleLatency,
		TTFS,
		QuickstartStartedTotal,
		QuickstartCompletedTotal,
		DailyActiveTenantsTotal,
		MCSalesTotal,
		MCSalesAmountTotal,
	)
}

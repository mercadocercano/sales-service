package controller

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	reportOpenOrdersDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "report_open_orders_duration_seconds",
			Help:    "Duration of open orders report execution",
			Buckets: prometheus.DefBuckets,
		},
	)
	reportAgingDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "report_aging_duration_seconds",
			Help:    "Duration of aging report execution",
			Buckets: prometheus.DefBuckets,
		},
	)
)

func init() {
	prometheus.MustRegister(reportOpenOrdersDuration, reportAgingDuration)
}

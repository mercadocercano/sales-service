package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// EventsProcessedTotal cuenta eventos procesados exitosamente por consumer
	EventsProcessedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "eventbus_events_processed_total",
			Help: "Total number of events successfully processed",
		},
		[]string{"consumer", "event_type"},
	)

	// EventsFailedTotal cuenta eventos que fallaron definitivamente
	EventsFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "eventbus_events_failed_total",
			Help: "Total number of events that failed permanently",
		},
		[]string{"consumer", "event_type"},
	)

	// EventsRetryTotal cuenta reintentos de eventos
	EventsRetryTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "eventbus_events_retry_total",
			Help: "Total number of event retries",
		},
		[]string{"consumer", "event_type"},
	)

	// ProcessingDurationSeconds mide latencia de procesamiento
	ProcessingDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "eventbus_processing_duration_seconds",
			Help:    "Time spent processing events",
			Buckets: prometheus.DefBuckets, // [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]
		},
		[]string{"consumer", "event_type"},
	)

	// EventsUnprocessedGauge mide eventos pendientes
	EventsUnprocessedGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "eventbus_events_unprocessed",
			Help: "Current number of unprocessed events",
		},
		[]string{"consumer"},
	)

	// RetryCountHistogram distribución de retry counts
	RetryCountHistogram = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "eventbus_retry_count_distribution",
			Help:    "Distribution of retry counts before success or failure",
			Buckets: []float64{0, 1, 2, 3, 4, 5, 10, 20},
		},
	)
)

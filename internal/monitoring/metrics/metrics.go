package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP Metrics
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	HttpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	HttpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Current number of HTTP requests being processed",
		},
	)

	// Telemetry Service Metrics
	TelemetryIngestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "telemetry_ingest_total",
			Help: "Total number of telemetry data points ingested",
		},
		[]string{"switch_id", "metric_type", "status"},
	)

	TelemetryIngestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "telemetry_ingest_duration_seconds",
			Help:    "Telemetry ingestion duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"switch_id", "metric_type"},
	)

	TelemetryQueryTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "telemetry_query_total",
			Help: "Total number of telemetry queries",
		},
		[]string{"switch_id", "metric_type", "status"},
	)

	TelemetryQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "telemetry_query_duration_seconds",
			Help:    "Telemetry query duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"switch_id", "metric_type"},
	)

	// Database Metrics
	DatabaseOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "database_operations_total",
			Help: "Total number of database operations",
		},
		[]string{"operation", "table", "status"},
	)

	DatabaseOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "database_operation_duration_seconds",
			Help:    "Database operation duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "table"},
	)

	DatabaseConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "database_connections_active",
			Help: "Number of active database connections",
		},
	)

	// Queue Metrics
	QueueDepth = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "queue_depth",
			Help: "Current number of items in the processing queue",
		},
	)

	QueueOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "queue_operations_total",
			Help: "Total number of queue operations",
		},
		[]string{"operation", "status"},
	)

	QueueWorkerUtilization = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "queue_worker_utilization_percent",
			Help: "Queue worker utilization percentage",
		},
	)

	// System Metrics
	SystemUptime = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "system_uptime_seconds",
			Help: "System uptime in seconds",
		},
	)

	SystemMemoryUsage = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "system_memory_usage_bytes",
			Help: "Current memory usage in bytes",
		},
	)

	SystemGoroutines = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "system_goroutines",
			Help: "Current number of goroutines",
		},
	)

	// Generator Metrics
	GeneratorDataPointsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "generator_data_points_total",
			Help: "Total number of data points generated",
		},
	)

	GeneratorSwitchesActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "generator_switches_active",
			Help: "Number of active switches in generator",
		},
	)

	GeneratorUpdateDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "generator_update_duration_seconds",
			Help:    "Generator update cycle duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	// Cache Metrics
	CacheHitsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total number of cache hits",
		},
	)

	CacheMissesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total number of cache misses",
		},
	)

	CacheSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "cache_size",
			Help: "Current number of items in cache",
		},
	)

	// Business Metrics
	ActiveSwitches = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_switches",
			Help: "Number of active switches",
		},
	)

	MetricsPerSwitch = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "metrics_per_switch",
			Help: "Number of metrics per switch",
		},
		[]string{"switch_id"},
	)

	// Error Metrics
	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "errors_total",
			Help: "Total number of errors",
		},
		[]string{"component", "error_type"},
	)
)

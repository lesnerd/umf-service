package storage

import (
	"context"
	"time"

	"github.com/ufm/internal/monitoring/metrics"
	"github.com/ufm/internal/telemetry/models"
)

// MetricsRepository wraps a TelemetryRepository to add metrics
type MetricsRepository struct {
	repo TelemetryRepository
}

// NewMetricsRepository creates a new metrics repository wrapper
func NewMetricsRepository(repo TelemetryRepository) *MetricsRepository {
	return &MetricsRepository{
		repo: repo,
	}
}

// CreateSwitch creates a new switch with metrics
func (r *MetricsRepository) CreateSwitch(ctx context.Context, sw models.Switch) error {
	start := time.Now()

	err := r.repo.CreateSwitch(ctx, sw)

	duration := time.Since(start).Seconds()
	if err != nil {
		metrics.DatabaseOperationsTotal.WithLabelValues("create", "switches", "error").Inc()
		metrics.ErrorsTotal.WithLabelValues("database", "create_switch").Inc()
	} else {
		metrics.DatabaseOperationsTotal.WithLabelValues("create", "switches", "success").Inc()
	}
	metrics.DatabaseOperationDuration.WithLabelValues("create", "switches").Observe(duration)

	return err
}

// GetSwitch retrieves a switch with metrics
func (r *MetricsRepository) GetSwitch(ctx context.Context, switchID string) (*models.Switch, error) {
	start := time.Now()

	switchData, err := r.repo.GetSwitch(ctx, switchID)

	duration := time.Since(start).Seconds()
	if err != nil {
		metrics.DatabaseOperationsTotal.WithLabelValues("get", "switches", "error").Inc()
		metrics.ErrorsTotal.WithLabelValues("database", "get_switch").Inc()
	} else {
		metrics.DatabaseOperationsTotal.WithLabelValues("get", "switches", "success").Inc()
	}
	metrics.DatabaseOperationDuration.WithLabelValues("get", "switches").Observe(duration)

	return switchData, err
}

// ListSwitches retrieves all switches with metrics
func (r *MetricsRepository) ListSwitches(ctx context.Context) ([]models.Switch, error) {
	start := time.Now()

	switches, err := r.repo.ListSwitches(ctx)

	duration := time.Since(start).Seconds()
	if err != nil {
		metrics.DatabaseOperationsTotal.WithLabelValues("list", "switches", "error").Inc()
		metrics.ErrorsTotal.WithLabelValues("database", "list_switches").Inc()
	} else {
		metrics.DatabaseOperationsTotal.WithLabelValues("list", "switches", "success").Inc()
	}
	metrics.DatabaseOperationDuration.WithLabelValues("list", "switches").Observe(duration)

	return switches, err
}

// StoreMetrics stores metrics with metrics
func (r *MetricsRepository) StoreMetrics(ctx context.Context, metricsData []models.TelemetryData) error {
	start := time.Now()

	err := r.repo.StoreMetrics(ctx, metricsData)

	duration := time.Since(start).Seconds()
	if err != nil {
		metrics.DatabaseOperationsTotal.WithLabelValues("store", "telemetry", "error").Inc()
		metrics.ErrorsTotal.WithLabelValues("database", "store_metrics").Inc()
	} else {
		metrics.DatabaseOperationsTotal.WithLabelValues("store", "telemetry", "success").Inc()
	}
	metrics.DatabaseOperationDuration.WithLabelValues("store", "telemetry").Observe(duration)

	return err
}

// GetLatestMetrics gets latest metrics with metrics
func (r *MetricsRepository) GetLatestMetrics(ctx context.Context, switchID string) (*models.TelemetryData, error) {
	start := time.Now()

	data, err := r.repo.GetLatestMetrics(ctx, switchID)

	duration := time.Since(start).Seconds()
	if err != nil {
		metrics.DatabaseOperationsTotal.WithLabelValues("get_latest", "telemetry", "error").Inc()
		metrics.ErrorsTotal.WithLabelValues("database", "get_latest_metrics").Inc()
	} else {
		metrics.DatabaseOperationsTotal.WithLabelValues("get_latest", "telemetry", "success").Inc()
	}
	metrics.DatabaseOperationDuration.WithLabelValues("get_latest", "telemetry").Observe(duration)

	return data, err
}

// GetHistoricalMetrics gets historical metrics with metrics
func (r *MetricsRepository) GetHistoricalMetrics(ctx context.Context, switchID string, from, to time.Time) ([]models.TelemetryData, error) {
	start := time.Now()

	data, err := r.repo.GetHistoricalMetrics(ctx, switchID, from, to)

	duration := time.Since(start).Seconds()
	if err != nil {
		metrics.DatabaseOperationsTotal.WithLabelValues("get_historical", "telemetry", "error").Inc()
		metrics.ErrorsTotal.WithLabelValues("database", "get_historical_metrics").Inc()
	} else {
		metrics.DatabaseOperationsTotal.WithLabelValues("get_historical", "telemetry", "success").Inc()
	}
	metrics.DatabaseOperationDuration.WithLabelValues("get_historical", "telemetry").Observe(duration)

	return data, err
}

// DeleteOldMetrics deletes old metrics with metrics
func (r *MetricsRepository) DeleteOldMetrics(ctx context.Context, olderThan time.Time) error {
	start := time.Now()

	err := r.repo.DeleteOldMetrics(ctx, olderThan)

	duration := time.Since(start).Seconds()
	if err != nil {
		metrics.DatabaseOperationsTotal.WithLabelValues("delete", "telemetry", "error").Inc()
		metrics.ErrorsTotal.WithLabelValues("database", "delete_old_metrics").Inc()
	} else {
		metrics.DatabaseOperationsTotal.WithLabelValues("delete", "telemetry", "success").Inc()
	}
	metrics.DatabaseOperationDuration.WithLabelValues("delete", "telemetry").Observe(duration)

	return err
}

// GetMetricsCount gets metrics count with metrics
func (r *MetricsRepository) GetMetricsCount(ctx context.Context) (int64, error) {
	start := time.Now()

	count, err := r.repo.GetMetricsCount(ctx)

	duration := time.Since(start).Seconds()
	if err != nil {
		metrics.DatabaseOperationsTotal.WithLabelValues("count", "telemetry", "error").Inc()
		metrics.ErrorsTotal.WithLabelValues("database", "get_metrics_count").Inc()
	} else {
		metrics.DatabaseOperationsTotal.WithLabelValues("count", "telemetry", "success").Inc()
	}
	metrics.DatabaseOperationDuration.WithLabelValues("count", "telemetry").Observe(duration)

	return count, err
}

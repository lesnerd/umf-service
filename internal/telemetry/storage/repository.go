package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/ufm/internal/telemetry/models"
)

// PostgreSQLRepository implements the TelemetryRepository interface
type PostgreSQLRepository struct {
	db *sql.DB
}

// NewPostgreSQLRepository creates a new PostgreSQL repository instance
func NewPostgreSQLRepository(db *sql.DB) *PostgreSQLRepository {
	return &PostgreSQLRepository{
		db: db,
	}
}

// Close closes the database connection
func (r *PostgreSQLRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// CreateSwitch creates a new switch in the database
func (r *PostgreSQLRepository) CreateSwitch(ctx context.Context, sw models.Switch) error {
	query := `
		INSERT INTO switches (id, name, location, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			location = EXCLUDED.location
	`

	createdAt := sw.Created
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	_, err := r.db.ExecContext(ctx, query, sw.ID, sw.Name, sw.Location, createdAt)
	if err != nil {
		return fmt.Errorf("failed to create switch %s: %w", sw.ID, err)
	}

	return nil
}

// GetSwitch retrieves a switch by ID
func (r *PostgreSQLRepository) GetSwitch(ctx context.Context, switchID string) (*models.Switch, error) {
	query := `
		SELECT id, name, location, created_at
		FROM switches
		WHERE id = $1
	`

	var sw models.Switch
	err := r.db.QueryRowContext(ctx, query, switchID).Scan(
		&sw.ID, &sw.Name, &sw.Location, &sw.Created,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("switch %s not found", switchID)
		}
		return nil, fmt.Errorf("failed to get switch %s: %w", switchID, err)
	}

	return &sw, nil
}

// ListSwitches retrieves all switches
func (r *PostgreSQLRepository) ListSwitches(ctx context.Context) ([]models.Switch, error) {
	query := `
		SELECT id, name, location, created_at
		FROM switches
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list switches: %w", err)
	}
	defer rows.Close()

	var switches []models.Switch
	for rows.Next() {
		var sw models.Switch
		err := rows.Scan(&sw.ID, &sw.Name, &sw.Location, &sw.Created)
		if err != nil {
			return nil, fmt.Errorf("failed to scan switch row: %w", err)
		}
		switches = append(switches, sw)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating switch rows: %w", err)
	}

	return switches, nil
}

// StoreMetrics stores multiple telemetry metrics in batch
func (r *PostgreSQLRepository) StoreMetrics(ctx context.Context, metrics []models.TelemetryData) error {
	if len(metrics) == 0 {
		return nil
	}

	// Use bulk insert for better performance
	query := `
		INSERT INTO telemetry_metrics 
		(switch_id, timestamp, bandwidth_mbps, latency_ms, packet_errors, utilization_pct, temperature_c, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, metric := range metrics {
		timestamp := metric.Timestamp
		if timestamp.IsZero() {
			timestamp = now
		}

		createdAt := metric.CreatedAt
		if createdAt.IsZero() {
			createdAt = now
		}

		_, err = stmt.ExecContext(ctx,
			metric.SwitchID,
			timestamp,
			metric.BandwidthMbps,
			metric.LatencyMs,
			metric.PacketErrors,
			metric.UtilizationPct,
			metric.TemperatureC,
			createdAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert metric for switch %s: %w", metric.SwitchID, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetLatestMetrics retrieves the most recent metrics for a switch
func (r *PostgreSQLRepository) GetLatestMetrics(ctx context.Context, switchID string) (*models.TelemetryData, error) {
	query := `
		SELECT id, switch_id, timestamp, bandwidth_mbps, latency_ms, 
		       packet_errors, utilization_pct, temperature_c, created_at
		FROM telemetry_metrics
		WHERE switch_id = $1
		ORDER BY timestamp DESC
		LIMIT 1
	`

	var metric models.TelemetryData
	err := r.db.QueryRowContext(ctx, query, switchID).Scan(
		&metric.ID,
		&metric.SwitchID,
		&metric.Timestamp,
		&metric.BandwidthMbps,
		&metric.LatencyMs,
		&metric.PacketErrors,
		&metric.UtilizationPct,
		&metric.TemperatureC,
		&metric.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no metrics found for switch %s", switchID)
		}
		return nil, fmt.Errorf("failed to get latest metrics for switch %s: %w", switchID, err)
	}

	return &metric, nil
}

// GetHistoricalMetrics retrieves metrics for a switch within a time range
func (r *PostgreSQLRepository) GetHistoricalMetrics(ctx context.Context, switchID string, from, to time.Time) ([]models.TelemetryData, error) {
	query := `
		SELECT id, switch_id, timestamp, bandwidth_mbps, latency_ms,
		       packet_errors, utilization_pct, temperature_c, created_at
		FROM telemetry_metrics
		WHERE switch_id = $1 AND timestamp >= $2 AND timestamp <= $3
		ORDER BY timestamp DESC
		LIMIT 1000
	`

	rows, err := r.db.QueryContext(ctx, query, switchID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to query historical metrics: %w", err)
	}
	defer rows.Close()

	var metrics []models.TelemetryData
	for rows.Next() {
		var metric models.TelemetryData
		err := rows.Scan(
			&metric.ID,
			&metric.SwitchID,
			&metric.Timestamp,
			&metric.BandwidthMbps,
			&metric.LatencyMs,
			&metric.PacketErrors,
			&metric.UtilizationPct,
			&metric.TemperatureC,
			&metric.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metric row: %w", err)
		}
		metrics = append(metrics, metric)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating metric rows: %w", err)
	}

	return metrics, nil
}

// DeleteOldMetrics removes metrics older than the specified time
func (r *PostgreSQLRepository) DeleteOldMetrics(ctx context.Context, olderThan time.Time) error {
	query := `DELETE FROM telemetry_metrics WHERE created_at < $1`

	result, err := r.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return fmt.Errorf("failed to delete old metrics: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		// Log the cleanup operation (would normally use proper logger)
		fmt.Printf("Deleted %d old telemetry records\n", rowsAffected)
	}

	return nil
}

// GetMetricsCount returns the total number of metrics in the database
func (r *PostgreSQLRepository) GetMetricsCount(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM telemetry_metrics`

	var count int64
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get metrics count: %w", err)
	}

	return count, nil
}

// BulkStoreMetrics uses PostgreSQL's COPY command for high-performance bulk inserts
func (r *PostgreSQLRepository) BulkStoreMetrics(ctx context.Context, metrics []models.TelemetryData) error {
	if len(metrics) == 0 {
		return nil
	}

	// Begin transaction for COPY command
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare data for COPY
	var values [][]interface{}
	now := time.Now()

	for _, metric := range metrics {
		timestamp := metric.Timestamp
		if timestamp.IsZero() {
			timestamp = now
		}

		createdAt := metric.CreatedAt
		if createdAt.IsZero() {
			createdAt = now
		}

		values = append(values, []interface{}{
			metric.SwitchID,
			timestamp,
			metric.BandwidthMbps,
			metric.LatencyMs,
			metric.PacketErrors,
			metric.UtilizationPct,
			metric.TemperatureC,
			createdAt,
		})
	}

	// Use pq.CopyIn for bulk insert within transaction
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn("telemetry_metrics",
		"switch_id", "timestamp", "bandwidth_mbps", "latency_ms",
		"packet_errors", "utilization_pct", "temperature_c", "created_at"))
	if err != nil {
		return fmt.Errorf("failed to prepare copy statement: %w", err)
	}
	defer stmt.Close()

	for _, row := range values {
		_, err = stmt.ExecContext(ctx, row...)
		if err != nil {
			return fmt.Errorf("failed to exec copy row: %w", err)
		}
	}

	_, err = stmt.ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to flush copy: %w", err)
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetSwitchMetricsSummary returns aggregated metrics for a switch over a time period
func (r *PostgreSQLRepository) GetSwitchMetricsSummary(ctx context.Context, switchID string, hours int) (map[string]interface{}, error) {
	query := `
		SELECT 
			COUNT(*) as sample_count,
			AVG(bandwidth_mbps) as avg_bandwidth,
			MAX(bandwidth_mbps) as max_bandwidth,
			AVG(latency_ms) as avg_latency,
			MAX(latency_ms) as max_latency,
			SUM(packet_errors) as total_errors,
			AVG(utilization_pct) as avg_utilization,
			MAX(utilization_pct) as max_utilization,
			AVG(temperature_c) as avg_temperature,
			MAX(temperature_c) as max_temperature
		FROM telemetry_metrics
		WHERE switch_id = $1 AND timestamp > NOW() - INTERVAL '%d hours'
	`

	query = fmt.Sprintf(query, hours)

	var summary struct {
		SampleCount    int64
		AvgBandwidth   sql.NullFloat64
		MaxBandwidth   sql.NullFloat64
		AvgLatency     sql.NullFloat64
		MaxLatency     sql.NullFloat64
		TotalErrors    sql.NullInt64
		AvgUtilization sql.NullFloat64
		MaxUtilization sql.NullFloat64
		AvgTemperature sql.NullFloat64
		MaxTemperature sql.NullFloat64
	}

	err := r.db.QueryRowContext(ctx, query, switchID).Scan(
		&summary.SampleCount,
		&summary.AvgBandwidth,
		&summary.MaxBandwidth,
		&summary.AvgLatency,
		&summary.MaxLatency,
		&summary.TotalErrors,
		&summary.AvgUtilization,
		&summary.MaxUtilization,
		&summary.AvgTemperature,
		&summary.MaxTemperature,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get switch metrics summary: %w", err)
	}

	result := map[string]interface{}{
		"switch_id":       switchID,
		"time_period":     fmt.Sprintf("%d hours", hours),
		"sample_count":    summary.SampleCount,
		"avg_bandwidth":   nullFloat64ToInterface(summary.AvgBandwidth),
		"max_bandwidth":   nullFloat64ToInterface(summary.MaxBandwidth),
		"avg_latency":     nullFloat64ToInterface(summary.AvgLatency),
		"max_latency":     nullFloat64ToInterface(summary.MaxLatency),
		"total_errors":    nullInt64ToInterface(summary.TotalErrors),
		"avg_utilization": nullFloat64ToInterface(summary.AvgUtilization),
		"max_utilization": nullFloat64ToInterface(summary.MaxUtilization),
		"avg_temperature": nullFloat64ToInterface(summary.AvgTemperature),
		"max_temperature": nullFloat64ToInterface(summary.MaxTemperature),
	}

	return result, nil
}

// Health check for database connectivity
func (r *PostgreSQLRepository) HealthCheck(ctx context.Context) error {
	query := `SELECT 1`

	var result int
	err := r.db.QueryRowContext(ctx, query).Scan(&result)
	if err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}

// Helper functions for handling SQL null values
func nullFloat64ToInterface(nf sql.NullFloat64) interface{} {
	if nf.Valid {
		return nf.Float64
	}
	return nil
}

func nullInt64ToInterface(ni sql.NullInt64) interface{} {
	if ni.Valid {
		return ni.Int64
	}
	return nil
}

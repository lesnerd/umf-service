package telemetry

import (
	"context"
	"fmt"
	"time"

	"github.com/ufm/internal/log"
	"github.com/ufm/internal/monitoring/metrics"
	"github.com/ufm/internal/telemetry/models"
	"github.com/ufm/internal/telemetry/storage"
)

// TelemetryService defines the main business logic interface
type TelemetryService interface {
	// Core operations
	IngestMetrics(data models.TelemetryData) error
	IngestBatch(data []models.TelemetryData) error

	// Query operations
	GetMetric(switchID string, metricType models.MetricType) (*models.MetricResponse, error)
	GetSwitchMetrics(switchID string) (*models.MetricsListResponse, error)
	GetAllMetrics() (*models.AllMetricsResponse, error)

	// Management operations
	RegisterSwitch(sw models.Switch) error
	GetSwitches() ([]models.Switch, error)

	// Health and observability
	GetPerformanceMetrics() *models.PerformanceMetrics
	GetHealthStatus() map[string]interface{}

	// Lifecycle
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// telemetryService implements the TelemetryService interface
type telemetryService struct {
	store     storage.TelemetryStore
	logger    log.Logger
	startTime time.Time
}

// NewTelemetryService creates a new telemetry service instance
func NewTelemetryService(store storage.TelemetryStore, logger log.Logger) TelemetryService {
	if logger == nil {
		logger = log.DefaultLogger
	}

	return &telemetryService{
		store:     store,
		logger:    logger,
		startTime: time.Now(),
	}
}

// IngestMetrics ingests a single telemetry data point
func (s *telemetryService) IngestMetrics(data models.TelemetryData) error {
	start := time.Now()

	if data.SwitchID == "" {
		metrics.ErrorsTotal.WithLabelValues("telemetry_service", "empty_switch_id").Inc()
		return fmt.Errorf("switchID cannot be empty")
	}

	err := s.store.UpdateMetrics(data.SwitchID, data)
	if err != nil {
		metrics.ErrorsTotal.WithLabelValues("telemetry_service", "store_error").Inc()
		s.logger.Errorf("Failed to ingest metrics for switch %s: %v", data.SwitchID, err)
		return fmt.Errorf("failed to ingest metrics: %w", err)
	}

	duration := time.Since(start).Seconds()

	// Record metrics for each metric type in the telemetry data
	metrics.TelemetryIngestTotal.WithLabelValues(data.SwitchID, string(models.MetricBandwidth), "success").Inc()
	metrics.TelemetryIngestTotal.WithLabelValues(data.SwitchID, string(models.MetricLatency), "success").Inc()
	metrics.TelemetryIngestTotal.WithLabelValues(data.SwitchID, string(models.MetricPacketErrors), "success").Inc()
	metrics.TelemetryIngestTotal.WithLabelValues(data.SwitchID, string(models.MetricUtilization), "success").Inc()
	metrics.TelemetryIngestTotal.WithLabelValues(data.SwitchID, string(models.MetricTemperature), "success").Inc()

	metrics.TelemetryIngestDuration.WithLabelValues(data.SwitchID, "all_metrics").Observe(duration)

	s.logger.Debugf("Ingested metrics for switch %s", data.SwitchID)
	return nil
}

// IngestBatch ingests multiple telemetry data points
func (s *telemetryService) IngestBatch(data []models.TelemetryData) error {
	if len(data) == 0 {
		return nil
	}

	// Filter out records with empty switchID
	var validData []models.TelemetryData
	for _, telemetryData := range data {
		if telemetryData.SwitchID == "" {
			s.logger.Warnf("Skipping telemetry data with empty switchID")
			continue
		}
		validData = append(validData, telemetryData)
	}

	if len(validData) == 0 {
		return fmt.Errorf("no valid telemetry data to ingest")
	}

	// Store all valid records directly to the database
	err := s.store.StoreMetricsBulk(context.Background(), validData)
	if err != nil {
		s.logger.Errorf("Failed to ingest batch of %d metrics: %v", len(validData), err)
		return fmt.Errorf("failed to ingest batch metrics: %w", err)
	}

	s.logger.Debugf("Ingested batch of %d metrics", len(validData))
	return nil
}

// GetMetric retrieves a specific metric for a switch
func (s *telemetryService) GetMetric(switchID string, metricType models.MetricType) (*models.MetricResponse, error) {
	start := time.Now()

	if switchID == "" {
		metrics.TelemetryQueryTotal.WithLabelValues("", string(metricType), "error").Inc()
		metrics.ErrorsTotal.WithLabelValues("telemetry_service", "empty_switch_id").Inc()
		return nil, fmt.Errorf("switchID cannot be empty")
	}

	value, err := s.store.GetMetric(switchID, metricType)
	if err != nil {
		metrics.TelemetryQueryTotal.WithLabelValues(switchID, string(metricType), "error").Inc()
		metrics.ErrorsTotal.WithLabelValues("telemetry_service", "store_error").Inc()
		return nil, fmt.Errorf("failed to get metric %s for switch %s: %w", metricType, switchID, err)
	}

	// Get the timestamp of the data
	lastUpdate := s.store.GetLastUpdate(switchID)
	if lastUpdate.IsZero() {
		lastUpdate = time.Now()
	}

	duration := time.Since(start).Seconds()
	metrics.TelemetryQueryTotal.WithLabelValues(switchID, string(metricType), "success").Inc()
	metrics.TelemetryQueryDuration.WithLabelValues(switchID, string(metricType)).Observe(duration)

	response := &models.MetricResponse{
		SwitchID:   switchID,
		MetricType: string(metricType),
		Value:      value,
		Timestamp:  lastUpdate,
	}

	return response, nil
}

// GetSwitchMetrics retrieves all metrics for a specific switch
func (s *telemetryService) GetSwitchMetrics(switchID string) (*models.MetricsListResponse, error) {
	if switchID == "" {
		return nil, fmt.Errorf("switchID cannot be empty")
	}

	data, err := s.store.GetAllMetrics(switchID)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics for switch %s: %w", switchID, err)
	}

	response := &models.MetricsListResponse{
		SwitchID:  switchID,
		Metrics:   data.ToMap(),
		Timestamp: data.Timestamp,
	}

	return response, nil
}

// GetAllMetrics retrieves metrics for all switches
func (s *telemetryService) GetAllMetrics() (*models.AllMetricsResponse, error) {
	allData := s.store.ListAllSwitches()

	var switchMetrics []models.MetricsListResponse
	for switchID, data := range allData {
		if data == nil {
			continue
		}

		switchMetric := models.MetricsListResponse{
			SwitchID:  switchID,
			Metrics:   data.ToMap(),
			Timestamp: data.Timestamp,
		}
		switchMetrics = append(switchMetrics, switchMetric)
	}

	response := &models.AllMetricsResponse{
		Switches:  switchMetrics,
		Count:     len(switchMetrics),
		Timestamp: time.Now(),
	}

	return response, nil
}

// RegisterSwitch registers a new switch in the system
func (s *telemetryService) RegisterSwitch(sw models.Switch) error {
	if sw.ID == "" {
		return fmt.Errorf("switch ID cannot be empty")
	}

	// Create the switch in the database
	ctx := context.Background()
	if err := s.store.CreateSwitch(ctx, sw); err != nil {
		s.logger.Errorf("Failed to register switch %s: %v", sw.ID, err)
		return fmt.Errorf("failed to register switch: %w", err)
	}

	s.logger.Infof("Registered switch: %s (%s) at %s", sw.ID, sw.Name, sw.Location)
	return nil
}

// GetSwitches retrieves all registered switches
func (s *telemetryService) GetSwitches() ([]models.Switch, error) {
	ctx := context.Background()
	switches, err := s.store.ListSwitches(ctx)
	if err != nil {
		s.logger.Errorf("Failed to get switches: %v", err)
		return nil, fmt.Errorf("failed to get switches: %w", err)
	}

	return switches, nil
}

// GetPerformanceMetrics returns comprehensive performance metrics
func (s *telemetryService) GetPerformanceMetrics() *models.PerformanceMetrics {
	return s.store.GetPerformanceMetrics()
}

// GetHealthStatus returns the health status of the telemetry service
func (s *telemetryService) GetHealthStatus() map[string]interface{} {
	switchCount := s.store.GetSwitchCount()
	performance := s.store.GetPerformanceMetrics()

	// Determine health status
	status := "healthy"
	checks := map[string]string{
		"storage":  "ok",
		"cache":    "ok",
		"switches": "ok",
	}

	if performance != nil && performance.DataAgeSeconds > 300 { // 5 minutes
		status = "degraded"
		checks["data_freshness"] = "warning"
	}

	if switchCount == 0 {
		status = "degraded"
		checks["switches"] = "no_data"
	}

	uptime := time.Since(s.startTime)

	return map[string]interface{}{
		"service":      "telemetry",
		"status":       status,
		"uptime":       uptime.String(),
		"switch_count": switchCount,
		"checks":       checks,
		"performance":  performance,
		"timestamp":    time.Now().Format(time.RFC3339),
		"version":      "1.0.0",
	}
}

// Start initializes the telemetry service
func (s *telemetryService) Start(ctx context.Context) error {
	s.logger.Infof("Starting telemetry service...")

	// Start the underlying store
	if err := s.store.Start(ctx); err != nil {
		return fmt.Errorf("failed to start telemetry store: %w", err)
	}

	s.logger.Infof("Telemetry service started successfully")
	return nil
}

// Stop gracefully shuts down the telemetry service
func (s *telemetryService) Stop(ctx context.Context) error {
	s.logger.Infof("Stopping telemetry service...")

	// Stop the underlying store
	if err := s.store.Stop(ctx); err != nil {
		s.logger.Errorf("Error stopping telemetry store: %v", err)
		return err
	}

	s.logger.Infof("Telemetry service stopped successfully")
	return nil
}

// Additional utility methods

// ValidateMetricData validates telemetry data before ingestion
func (s *telemetryService) ValidateMetricData(data models.TelemetryData) error {
	if data.SwitchID == "" {
		return fmt.Errorf("switchID cannot be empty")
	}

	if data.BandwidthMbps < 0 {
		return fmt.Errorf("bandwidth cannot be negative")
	}

	if data.LatencyMs < 0 {
		return fmt.Errorf("latency cannot be negative")
	}

	if data.PacketErrors < 0 {
		return fmt.Errorf("packet errors cannot be negative")
	}

	if data.UtilizationPct < 0 || data.UtilizationPct > 100 {
		return fmt.Errorf("utilization must be between 0 and 100")
	}

	return nil
}

// GetMetricStatistics returns statistical information about metrics
func (s *telemetryService) GetMetricStatistics() map[string]interface{} {
	allData := s.store.ListAllSwitches()

	if len(allData) == 0 {
		return map[string]interface{}{
			"switch_count": 0,
			"timestamp":    time.Now().Format(time.RFC3339),
		}
	}

	// Calculate basic statistics
	var totalBandwidth, totalLatency, totalUtilization, totalTemperature float64
	var totalErrors int64
	count := len(allData)

	for _, data := range allData {
		if data != nil {
			totalBandwidth += data.BandwidthMbps
			totalLatency += data.LatencyMs
			totalUtilization += data.UtilizationPct
			totalTemperature += data.TemperatureC
			totalErrors += data.PacketErrors
		}
	}

	return map[string]interface{}{
		"switch_count":       count,
		"avg_bandwidth_mbps": totalBandwidth / float64(count),
		"avg_latency_ms":     totalLatency / float64(count),
		"avg_utilization":    totalUtilization / float64(count),
		"avg_temperature":    totalTemperature / float64(count),
		"total_errors":       totalErrors,
		"timestamp":          time.Now().Format(time.RFC3339),
	}
}

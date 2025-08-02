package telemetry

import (
	"context"
	"fmt"

	"github.com/ufm/internal/log"
	"github.com/ufm/internal/telemetry/models"
	"github.com/ufm/internal/telemetry/queue"
)

// QueuedTelemetryService wraps the regular service with optional request queuing
type QueuedTelemetryService struct {
	baseService  TelemetryService
	requestQueue *queue.RequestQueue
	queueEnabled bool
	logger       log.Logger
}

// NewQueuedTelemetryService creates a service with optional queuing
func NewQueuedTelemetryService(
	baseService TelemetryService,
	queueConfig queue.QueueConfig,
	logger log.Logger,
) TelemetryService {
	service := &QueuedTelemetryService{
		baseService:  baseService,
		queueEnabled: queueConfig.EnableQueue,
		logger:       logger,
	}

	// Only create queue if enabled
	if queueConfig.EnableQueue {
		service.requestQueue = queue.NewRequestQueue(queueConfig, service, logger)
		logger.Infof("Request queuing enabled for high-load scenarios")
	} else {
		logger.Infof("Request queuing disabled - using direct service calls")
	}

	return service
}

// Start initializes the service and queue
func (s *QueuedTelemetryService) Start(ctx context.Context) error {
	// Start base service
	if err := s.baseService.Start(ctx); err != nil {
		return err
	}

	// Start queue if enabled
	if s.queueEnabled && s.requestQueue != nil {
		if err := s.requestQueue.Start(); err != nil {
			return fmt.Errorf("failed to start request queue: %w", err)
		}
	}

	return nil
}

// Stop gracefully shuts down the service and queue
func (s *QueuedTelemetryService) Stop(ctx context.Context) error {
	var lastErr error

	// Stop queue first if enabled
	if s.queueEnabled && s.requestQueue != nil {
		if err := s.requestQueue.Stop(); err != nil {
			s.logger.Errorf("Error stopping request queue: %v", err)
			lastErr = err
		}
	}

	// Stop base service
	if err := s.baseService.Stop(ctx); err != nil {
		s.logger.Errorf("Error stopping base service: %v", err)
		lastErr = err
	}

	return lastErr
}

// GetMetric retrieves a specific metric, optionally using the queue
func (s *QueuedTelemetryService) GetMetric(switchID string, metricType models.MetricType) (*models.MetricResponse, error) {
	if s.queueEnabled && s.requestQueue != nil {
		// Use queue for high-load scenarios
		requestID := fmt.Sprintf("metric-%s-%s", switchID, metricType)
		value, err := s.requestQueue.QueueGetMetric(requestID, switchID, metricType)
		if err != nil {
			return nil, err
		}

		// Convert to response format
		return &models.MetricResponse{
			SwitchID:   switchID,
			MetricType: string(metricType),
			Value:      value,
			Timestamp:  s.baseService.(*telemetryService).store.GetLastUpdate(switchID),
		}, nil
	}

	// Use direct service call for normal load
	return s.baseService.GetMetric(switchID, metricType)
}

// GetSwitchMetrics retrieves all metrics for a switch, optionally using the queue
func (s *QueuedTelemetryService) GetSwitchMetrics(switchID string) (*models.MetricsListResponse, error) {
	if s.queueEnabled && s.requestQueue != nil {
		// Use queue for high-load scenarios
		requestID := fmt.Sprintf("switch-%s", switchID)
		data, err := s.requestQueue.QueueGetAllMetrics(requestID, switchID)
		if err != nil {
			return nil, err
		}

		// Convert to response format
		return &models.MetricsListResponse{
			SwitchID:  switchID,
			Metrics:   data.ToMap(),
			Timestamp: data.Timestamp,
		}, nil
	}

	// Use direct service call for normal load
	return s.baseService.GetSwitchMetrics(switchID)
}

// Implement RequestHandler interface for the queue
func (s *QueuedTelemetryService) HandleGetMetric(switchID string, metricType models.MetricType) (interface{}, error) {
	response, err := s.baseService.GetMetric(switchID, metricType)
	if err != nil {
		return nil, err
	}
	return response.Value, nil
}

func (s *QueuedTelemetryService) HandleGetAllMetrics(switchID string) (*models.TelemetryData, error) {
	response, err := s.baseService.GetSwitchMetrics(switchID)
	if err != nil {
		return nil, err
	}

	// Convert back to TelemetryData format
	data := &models.TelemetryData{
		SwitchID:  switchID,
		Timestamp: response.Timestamp,
	}

	// Extract metrics from the map
	if bw, ok := response.Metrics["bandwidth_mbps"].(float64); ok {
		data.BandwidthMbps = bw
	}
	if lat, ok := response.Metrics["latency_ms"].(float64); ok {
		data.LatencyMs = lat
	}
	if errs, ok := response.Metrics["packet_errors"].(int64); ok {
		data.PacketErrors = errs
	}
	if util, ok := response.Metrics["utilization_pct"].(float64); ok {
		data.UtilizationPct = util
	}
	if temp, ok := response.Metrics["temperature_c"].(float64); ok {
		data.TemperatureC = temp
	}

	return data, nil
}

func (s *QueuedTelemetryService) HandleListAllSwitches() map[string]*models.TelemetryData {
	response, err := s.baseService.GetAllMetrics()
	if err != nil {
		return make(map[string]*models.TelemetryData)
	}

	result := make(map[string]*models.TelemetryData)
	for _, switchMetrics := range response.Switches {
		data := &models.TelemetryData{
			SwitchID:  switchMetrics.SwitchID,
			Timestamp: switchMetrics.Timestamp,
		}

		// Extract metrics from the map
		if bw, ok := switchMetrics.Metrics["bandwidth_mbps"].(float64); ok {
			data.BandwidthMbps = bw
		}
		if lat, ok := switchMetrics.Metrics["latency_ms"].(float64); ok {
			data.LatencyMs = lat
		}
		if errs, ok := switchMetrics.Metrics["packet_errors"].(int64); ok {
			data.PacketErrors = errs
		}
		if util, ok := switchMetrics.Metrics["utilization_pct"].(float64); ok {
			data.UtilizationPct = util
		}
		if temp, ok := switchMetrics.Metrics["temperature_c"].(float64); ok {
			data.TemperatureC = temp
		}

		result[switchMetrics.SwitchID] = data
	}

	return result
}

// Delegate all other methods to the base service
func (s *QueuedTelemetryService) IngestMetrics(data models.TelemetryData) error {
	return s.baseService.IngestMetrics(data)
}

func (s *QueuedTelemetryService) IngestBatch(data []models.TelemetryData) error {
	return s.baseService.IngestBatch(data)
}

func (s *QueuedTelemetryService) GetAllMetrics() (*models.AllMetricsResponse, error) {
	return s.baseService.GetAllMetrics()
}

func (s *QueuedTelemetryService) RegisterSwitch(sw models.Switch) error {
	return s.baseService.RegisterSwitch(sw)
}

func (s *QueuedTelemetryService) GetSwitches() ([]models.Switch, error) {
	return s.baseService.GetSwitches()
}

func (s *QueuedTelemetryService) GetPerformanceMetrics() *models.PerformanceMetrics {
	baseMetrics := s.baseService.GetPerformanceMetrics()

	// Add queue metrics if available
	if s.queueEnabled && s.requestQueue != nil {
		queueMetrics := s.requestQueue.GetMetrics()
		s.logger.Debugf("Queue metrics: queued=%d, processed=%d, dropped=%d, depth=%d",
			queueMetrics.QueuedRequests,
			queueMetrics.ProcessedRequests,
			queueMetrics.DroppedRequests,
			queueMetrics.QueueDepth)
	}

	return baseMetrics
}

func (s *QueuedTelemetryService) GetHealthStatus() map[string]interface{} {
	health := s.baseService.GetHealthStatus()

	// Add queue health if enabled
	if s.queueEnabled && s.requestQueue != nil {
		queueMetrics := s.requestQueue.GetMetrics()
		health["queue"] = map[string]interface{}{
			"enabled":            true,
			"queue_depth":        queueMetrics.QueueDepth,
			"processed_requests": queueMetrics.ProcessedRequests,
			"dropped_requests":   queueMetrics.DroppedRequests,
			"worker_utilization": fmt.Sprintf("%.1f%%", queueMetrics.WorkerUtilization),
		}
	} else {
		health["queue"] = map[string]interface{}{
			"enabled": false,
			"reason":  "disabled for normal load levels",
		}
	}

	return health
}

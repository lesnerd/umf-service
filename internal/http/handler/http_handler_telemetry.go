package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ufm/internal/http/utils"
	"github.com/ufm/internal/log"
	"github.com/ufm/internal/service"
	"github.com/ufm/internal/telemetry"
	"github.com/ufm/internal/telemetry/models"
)

// EXPORTED TYPES AND FUNCTIONS

type TelemetryHandler interface {
	GetMetric(c *gin.Context)   // GET /telemetry/metrics/:switchId/:metricType
	ListMetrics(c *gin.Context) // GET /telemetry/metrics/:switchId or /telemetry/metrics
	// Observability
	GetPerformanceMetrics(c *gin.Context) // GET /telemetry/performance
	GetHealthStatus(c *gin.Context)       // GET /telemetry/health
	GetSwitchList(c *gin.Context)         // GET /telemetry/switches
	GetMetricTypes(c *gin.Context)        // GET /telemetry/metric-types
}

type telemetryHandler struct {
	logger    log.Logger
	ctx       service.Context
	service   telemetry.TelemetryService
	startTime time.Time
}

func NewTelemetryHandler(ctx service.Context, telemetryService telemetry.TelemetryService) TelemetryHandler {
	return &telemetryHandler{
		logger:    ctx.LoggerFactory().(log.LoggerFactory).GetLogger("telemetry-handler"),
		ctx:       ctx,
		service:   telemetryService,
		startTime: time.Now(),
	}
}

// GetMetric handles GET /telemetry/metrics/:switchId/:metricType
func (h *telemetryHandler) GetMetric(c *gin.Context) {
	startTime := time.Now()

	switchID := c.Param("switchId")
	metricTypeStr := c.Param("metricType")

	// Validate parameters
	if switchID == "" {
		utils.RespondWithError(c, http.StatusBadRequest, "switchId is required")
		return
	}

	if metricTypeStr == "" {
		utils.RespondWithError(c, http.StatusBadRequest, "metricType is required")
		return
	}

	// Parse metric type
	metricType := models.MetricType(metricTypeStr)

	// Validate metric type
	if !h.isValidMetricType(metricType) {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid metric type: "+metricTypeStr)
		return
	}

	// Get the metric from service
	response, err := h.service.GetMetric(switchID, metricType)
	if err != nil {
		h.logger.Errorf("Failed to get metric %s for switch %s: %v", metricType, switchID, err)
		utils.RespondWithError(c, http.StatusNotFound, "metric not found: "+err.Error())
		return
	}

	// Add performance headers
	duration := time.Since(startTime)
	c.Header("X-Response-Time", duration.String())
	c.Header("X-Switch-ID", switchID)
	c.Header("X-Metric-Type", metricTypeStr)

	h.logger.Debugf("GetMetric: switch=%s, metric=%s, duration=%v", switchID, metricTypeStr, duration)

	utils.RespondWithSuccess(c, response)
}

// ListMetrics handles GET /telemetry/metrics/:switchId and GET /telemetry/metrics
func (h *telemetryHandler) ListMetrics(c *gin.Context) {
	startTime := time.Now()

	switchID := c.Param("switchId")
	h.logger.Infof("ListMetrics called with switchID: '%s', path: '%s', query: '%s'", switchID, c.Request.URL.Path, c.Request.URL.RawQuery)

	if switchID != "" {
		// Get metrics for specific switch
		response, err := h.service.GetSwitchMetrics(switchID)
		if err != nil {
			h.logger.Errorf("Failed to get metrics for switch %s: %v", switchID, err)
			utils.RespondWithError(c, http.StatusNotFound, "switch metrics not found: "+err.Error())
			return
		}

		duration := time.Since(startTime)
		c.Header("X-Response-Time", duration.String())
		c.Header("X-Switch-ID", switchID)
		c.Header("X-Metric-Count", "5")

		h.logger.Debugf("ListMetrics: switch=%s, duration=%v", switchID, duration)
		utils.RespondWithSuccess(c, response)
	} else {
		// Check for metric type filtering via query parameter
		metricTypesStr := c.Query("metrics")
		h.logger.Infof("ListMetrics: query parameter 'metrics' = '%s'", metricTypesStr)

		if metricTypesStr != "" {
			// Filter by specific metric types
			h.logger.Infof("ListMetrics: filtering by metrics: %s", metricTypesStr)
			h.handleMetricsByType(c, metricTypesStr, startTime)
			return
		}

		// Get metrics for all switches (no filtering)
		response, err := h.service.GetAllMetrics()
		if err != nil {
			h.logger.Errorf("Failed to get all metrics: %v", err)
			utils.RespondWithError(c, http.StatusInternalServerError, "failed to retrieve metrics: "+err.Error())
			return
		}

		duration := time.Since(startTime)
		c.Header("X-Response-Time", duration.String())
		c.Header("X-Switch-Count", strconv.Itoa(response.Count))
		c.Header("X-Total-Metrics", strconv.Itoa(response.Count*5)) // 5 metrics per switch

		h.logger.Debugf("ListMetrics: all switches, count=%d, duration=%v", response.Count, duration)
		utils.RespondWithSuccess(c, response)
	}
}

// ListMetricsByType handles GET /telemetry/metrics/:metricTypes
func (h *telemetryHandler) ListMetricsByType(c *gin.Context) {
	startTime := time.Now()

	metricTypesStr := c.Param("metricTypes")
	if metricTypesStr == "" {
		utils.RespondWithError(c, http.StatusBadRequest, "metricTypes parameter is required")
		return
	}

	// Parse comma-separated metric types
	metricTypeStrs := strings.Split(metricTypesStr, ",")
	var metricTypes []models.MetricType

	for _, metricTypeStr := range metricTypeStrs {
		metricTypeStr = strings.TrimSpace(metricTypeStr)
		if metricTypeStr == "" {
			continue
		}

		metricType := models.MetricType(metricTypeStr)
		if !h.isValidMetricType(metricType) {
			utils.RespondWithError(c, http.StatusBadRequest, "invalid metric type: "+metricTypeStr)
			return
		}
		metricTypes = append(metricTypes, metricType)
	}

	if len(metricTypes) == 0 {
		utils.RespondWithError(c, http.StatusBadRequest, "no valid metric types provided")
		return
	}

	// Get all switches data
	allMetricsResponse, err := h.service.GetAllMetrics()
	if err != nil {
		h.logger.Errorf("Failed to get all metrics: %v", err)
		utils.RespondWithError(c, http.StatusInternalServerError, "failed to retrieve metrics: "+err.Error())
		return
	}

	// Filter and transform the response to include only requested metric types
	var filteredSwitches []map[string]interface{}

	for _, switchData := range allMetricsResponse.Switches {
		filteredSwitch := map[string]interface{}{
			"switch_id": switchData.SwitchID,
			"timestamp": switchData.Timestamp,
		}

		// Add only the requested metrics
		for _, metricType := range metricTypes {
			switch metricType {
			case models.MetricBandwidth:
				if value, exists := switchData.Metrics["bandwidth_mbps"]; exists {
					filteredSwitch["bandwidth_mbps"] = value
				}
			case models.MetricLatency:
				if value, exists := switchData.Metrics["latency_ms"]; exists {
					filteredSwitch["latency_ms"] = value
				}
			case models.MetricPacketErrors:
				if value, exists := switchData.Metrics["packet_errors"]; exists {
					filteredSwitch["packet_errors"] = value
				}
			case models.MetricUtilization:
				if value, exists := switchData.Metrics["utilization_pct"]; exists {
					filteredSwitch["utilization_pct"] = value
				}
			case models.MetricTemperature:
				if value, exists := switchData.Metrics["temperature_c"]; exists {
					filteredSwitch["temperature_c"] = value
				}
			}
		}

		filteredSwitches = append(filteredSwitches, filteredSwitch)
	}

	response := map[string]interface{}{
		"metric_types": metricTypes,
		"switches":     filteredSwitches,
		"count":        len(filteredSwitches),
		"timestamp":    time.Now(),
	}

	duration := time.Since(startTime)
	c.Header("X-Response-Time", duration.String())
	c.Header("X-Metric-Types", metricTypesStr)
	c.Header("X-Switch-Count", strconv.Itoa(len(filteredSwitches)))

	h.logger.Debugf("ListMetricsByType: metricTypes=%s, count=%d, duration=%v", metricTypesStr, len(filteredSwitches), duration)
	utils.RespondWithSuccess(c, response)
}

// GetPerformanceMetrics handles GET /telemetry/performance
func (h *telemetryHandler) GetPerformanceMetrics(c *gin.Context) {
	startTime := time.Now()

	metrics := h.service.GetPerformanceMetrics()
	if metrics == nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "failed to retrieve performance metrics")
		return
	}

	// Add request duration to metrics
	duration := time.Since(startTime)
	c.Header("X-Response-Time", duration.String())

	utils.RespondWithSuccess(c, metrics)
}

// GetHealthStatus handles GET /telemetry/health
func (h *telemetryHandler) GetHealthStatus(c *gin.Context) {
	startTime := time.Now()

	healthStatus := h.service.GetHealthStatus()
	if healthStatus == nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "failed to retrieve health status")
		return
	}

	// Add uptime information
	healthStatus["uptime"] = time.Since(h.startTime).String()
	healthStatus["response_time"] = time.Since(startTime).String()

	// Determine overall health status
	status := http.StatusOK
	if healthStatus["status"] != "healthy" {
		status = http.StatusServiceUnavailable
	}

	c.Header("X-Response-Time", time.Since(startTime).String())
	c.JSON(status, healthStatus)
}

// GetSwitchList handles GET /telemetry/switches
func (h *telemetryHandler) GetSwitchList(c *gin.Context) {
	startTime := time.Now()

	switches, err := h.service.GetSwitches()
	if err != nil {
		h.logger.Errorf("Failed to get switches: %v", err)
		utils.RespondWithError(c, http.StatusInternalServerError, "failed to retrieve switches")
		return
	}

	response := map[string]interface{}{
		"switches":  switches,
		"count":     len(switches),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	duration := time.Since(startTime)
	c.Header("X-Response-Time", duration.String())
	c.Header("X-Switch-Count", strconv.Itoa(len(switches)))

	utils.RespondWithSuccess(c, response)
}

// GetMetricTypes handles GET /telemetry/metric-types
func (h *telemetryHandler) GetMetricTypes(c *gin.Context) {
	metricTypes := []string{
		string(models.MetricBandwidth),
		string(models.MetricLatency),
		string(models.MetricPacketErrors),
		string(models.MetricUtilization),
		string(models.MetricTemperature),
	}

	response := map[string]interface{}{
		"metric_types": metricTypes,
		"count":        len(metricTypes),
		"timestamp":    time.Now().Format(time.RFC3339),
	}

	utils.RespondWithSuccess(c, response)
}

// handleMetricsByType is a helper method for filtering metrics by type
func (h *telemetryHandler) handleMetricsByType(c *gin.Context, metricTypesStr string, startTime time.Time) {
	// Parse comma-separated metric types
	metricTypeStrs := strings.Split(metricTypesStr, ",")
	var metricTypes []models.MetricType

	for _, metricTypeStr := range metricTypeStrs {
		metricTypeStr = strings.TrimSpace(metricTypeStr)
		if metricTypeStr == "" {
			continue
		}

		metricType := models.MetricType(metricTypeStr)
		if !h.isValidMetricType(metricType) {
			utils.RespondWithError(c, http.StatusBadRequest, "invalid metric type: "+metricTypeStr)
			return
		}
		metricTypes = append(metricTypes, metricType)
	}

	if len(metricTypes) == 0 {
		utils.RespondWithError(c, http.StatusBadRequest, "no valid metric types provided")
		return
	}

	// Get all switches data
	allMetricsResponse, err := h.service.GetAllMetrics()
	if err != nil {
		h.logger.Errorf("Failed to get all metrics: %v", err)
		utils.RespondWithError(c, http.StatusInternalServerError, "failed to retrieve metrics: "+err.Error())
		return
	}

	// Filter and transform the response to include only requested metric types
	var filteredSwitches []map[string]interface{}

	for _, switchData := range allMetricsResponse.Switches {
		filteredSwitch := map[string]interface{}{
			"switch_id": switchData.SwitchID,
			"timestamp": switchData.Timestamp,
		}

		// Add only the requested metrics
		for _, metricType := range metricTypes {
			switch metricType {
			case models.MetricBandwidth:
				if value, exists := switchData.Metrics["bandwidth_mbps"]; exists {
					filteredSwitch["bandwidth_mbps"] = value
				}
			case models.MetricLatency:
				if value, exists := switchData.Metrics["latency_ms"]; exists {
					filteredSwitch["latency_ms"] = value
				}
			case models.MetricPacketErrors:
				if value, exists := switchData.Metrics["packet_errors"]; exists {
					filteredSwitch["packet_errors"] = value
				}
			case models.MetricUtilization:
				if value, exists := switchData.Metrics["utilization_pct"]; exists {
					filteredSwitch["utilization_pct"] = value
				}
			case models.MetricTemperature:
				if value, exists := switchData.Metrics["temperature_c"]; exists {
					filteredSwitch["temperature_c"] = value
				}
			}
		}

		filteredSwitches = append(filteredSwitches, filteredSwitch)
	}

	response := map[string]interface{}{
		"metric_types": metricTypes,
		"switches":     filteredSwitches,
		"count":        len(filteredSwitches),
		"timestamp":    time.Now(),
	}

	duration := time.Since(startTime)
	c.Header("X-Response-Time", duration.String())
	c.Header("X-Metric-Types", metricTypesStr)
	c.Header("X-Switch-Count", strconv.Itoa(len(filteredSwitches)))

	h.logger.Debugf("handleMetricsByType: metricTypes=%s, count=%d, duration=%v", metricTypesStr, len(filteredSwitches), duration)
	utils.RespondWithSuccess(c, response)
}

// isValidMetricType validates if the metric type is supported
func (h *telemetryHandler) isValidMetricType(metricType models.MetricType) bool {
	validTypes := []models.MetricType{
		models.MetricBandwidth,
		models.MetricLatency,
		models.MetricPacketErrors,
		models.MetricUtilization,
		models.MetricTemperature,
	}

	for _, validType := range validTypes {
		if metricType == validType {
			return true
		}
	}

	return false
}

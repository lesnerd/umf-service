package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// Switch represents a network fabric switch
type Switch struct {
	ID       string    `json:"id" db:"id"`
	Name     string    `json:"name" db:"name"`
	Location string    `json:"location" db:"location"`
	Created  time.Time `json:"created" db:"created_at"`
}

// MetricType represents the type of telemetry metric
type MetricType string

const (
	MetricBandwidth     MetricType = "bandwidth_mbps"
	MetricLatency       MetricType = "latency_ms"
	MetricPacketErrors  MetricType = "packet_errors"
	MetricUtilization   MetricType = "utilization_pct"
	MetricTemperature   MetricType = "temperature_c"
)

// TelemetryData represents a complete set of metrics for a switch at a point in time
type TelemetryData struct {
	ID           int64     `json:"id,omitempty" db:"id"`
	SwitchID     string    `json:"switch_id" db:"switch_id"`
	Timestamp    time.Time `json:"timestamp" db:"timestamp"`
	BandwidthMbps float64  `json:"bandwidth_mbps" db:"bandwidth_mbps"`
	LatencyMs    float64   `json:"latency_ms" db:"latency_ms"`
	PacketErrors int64     `json:"packet_errors" db:"packet_errors"`
	UtilizationPct float64 `json:"utilization_pct" db:"utilization_pct"`
	TemperatureC float64   `json:"temperature_c" db:"temperature_c"`
	CreatedAt    time.Time `json:"created_at,omitempty" db:"created_at"`
}

// GetMetricValue returns the value of a specific metric type
func (td *TelemetryData) GetMetricValue(metricType MetricType) (interface{}, error) {
	switch metricType {
	case MetricBandwidth:
		return td.BandwidthMbps, nil
	case MetricLatency:
		return td.LatencyMs, nil
	case MetricPacketErrors:
		return td.PacketErrors, nil
	case MetricUtilization:
		return td.UtilizationPct, nil
	case MetricTemperature:
		return td.TemperatureC, nil
	default:
		return nil, fmt.Errorf("unknown metric type: %s", metricType)
	}
}

// ToMap converts TelemetryData to a map for easy JSON serialization
func (td *TelemetryData) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"switch_id":        td.SwitchID,
		"timestamp":        td.Timestamp.Format(time.RFC3339),
		"bandwidth_mbps":   td.BandwidthMbps,
		"latency_ms":       td.LatencyMs,
		"packet_errors":    td.PacketErrors,
		"utilization_pct":  td.UtilizationPct,
		"temperature_c":    td.TemperatureC,
	}
}


// TelemetrySnapshot represents all metrics for all switches at a point in time
type TelemetrySnapshot struct {
	Timestamp    time.Time                  `json:"timestamp"`
	GenerationID string                     `json:"generation_id"`
	Switches     map[string]*TelemetryData `json:"switches"`
}


// APIResponse represents the standard API response format
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// MetricResponse represents a single metric response
type MetricResponse struct {
	SwitchID   string      `json:"switch_id"`
	MetricType string      `json:"metric_type"`
	Value      interface{} `json:"value"`
	Timestamp  time.Time   `json:"timestamp"`
}

// MetricsListResponse represents a list of metrics for a switch
type MetricsListResponse struct {
	SwitchID  string                 `json:"switch_id"`
	Metrics   map[string]interface{} `json:"metrics"`
	Timestamp time.Time              `json:"timestamp"`
}

// AllMetricsResponse represents metrics for all switches
type AllMetricsResponse struct {
	Switches  []MetricsListResponse `json:"switches"`
	Count     int                   `json:"count"`
	Timestamp time.Time             `json:"timestamp"`
}

// Performance metrics for observability
type PerformanceMetrics struct {
	APILatencyMs    float64 `json:"api_latency_ms"`
	ActiveSwitches  int     `json:"active_switches"`
	TotalRequests   int64   `json:"total_requests"`
	MemoryUsageMB   float64 `json:"memory_usage_mb"`
	DataAgeSeconds  float64 `json:"data_age_seconds"`
	LastUpdate      time.Time `json:"last_update"`
}

// String implements the Stringer interface for better logging
func (pm *PerformanceMetrics) String() string {
	data, _ := json.Marshal(pm)
	return string(data)
}
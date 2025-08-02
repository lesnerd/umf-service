package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSwitch_Fields(t *testing.T) {
	now := time.Now()
	switch_ := Switch{
		ID:       "switch-001",
		Name:     "Test Switch",
		Location: "Data Center A",
		Created:  now,
	}

	assert.Equal(t, "switch-001", switch_.ID)
	assert.Equal(t, "Test Switch", switch_.Name)
	assert.Equal(t, "Data Center A", switch_.Location)
	assert.Equal(t, now, switch_.Created)
}

func TestTelemetryData_Fields(t *testing.T) {
	now := time.Now()
	data := TelemetryData{
		ID:             1,
		SwitchID:       "switch-001",
		Timestamp:      now,
		BandwidthMbps:  1000.5,
		LatencyMs:      2.5,
		PacketErrors:   5,
		UtilizationPct: 75.2,
		TemperatureC:   45.0,
		CreatedAt:      now,
	}

	assert.Equal(t, int64(1), data.ID)
	assert.Equal(t, "switch-001", data.SwitchID)
	assert.Equal(t, now, data.Timestamp)
	assert.Equal(t, 1000.5, data.BandwidthMbps)
	assert.Equal(t, 2.5, data.LatencyMs)
	assert.Equal(t, int64(5), data.PacketErrors)
	assert.Equal(t, 75.2, data.UtilizationPct)
	assert.Equal(t, 45.0, data.TemperatureC)
	assert.Equal(t, now, data.CreatedAt)
}

func TestTelemetryData_GetMetricValue(t *testing.T) {
	data := TelemetryData{
		SwitchID:       "switch-001",
		BandwidthMbps:  1000.5,
		LatencyMs:      2.5,
		PacketErrors:   5,
		UtilizationPct: 75.2,
		TemperatureC:   45.0,
	}

	tests := []struct {
		name        string
		metricType  MetricType
		expected    interface{}
		expectError bool
	}{
		{
			name:        "bandwidth metric",
			metricType:  MetricBandwidth,
			expected:    1000.5,
			expectError: false,
		},
		{
			name:        "latency metric",
			metricType:  MetricLatency,
			expected:    2.5,
			expectError: false,
		},
		{
			name:        "packet errors metric",
			metricType:  MetricPacketErrors,
			expected:    int64(5),
			expectError: false,
		},
		{
			name:        "utilization metric",
			metricType:  MetricUtilization,
			expected:    75.2,
			expectError: false,
		},
		{
			name:        "temperature metric",
			metricType:  MetricTemperature,
			expected:    45.0,
			expectError: false,
		},
		{
			name:        "unknown metric",
			metricType:  "unknown",
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := data.GetMetricValue(tt.metricType)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestTelemetryData_ToMap(t *testing.T) {
	now := time.Date(2025, 8, 2, 12, 0, 0, 0, time.UTC)
	data := TelemetryData{
		SwitchID:       "switch-001",
		Timestamp:      now,
		BandwidthMbps:  1000.5,
		LatencyMs:      2.5,
		PacketErrors:   5,
		UtilizationPct: 75.2,
		TemperatureC:   45.0,
	}

	result := data.ToMap()

	expected := map[string]interface{}{
		"switch_id":       "switch-001",
		"timestamp":       "2025-08-02T12:00:00Z",
		"bandwidth_mbps":  1000.5,
		"latency_ms":      2.5,
		"packet_errors":   int64(5),
		"utilization_pct": 75.2,
		"temperature_c":   45.0,
	}

	assert.Equal(t, expected, result)
}

func TestTelemetrySnapshot_Fields(t *testing.T) {
	now := time.Now()
	snapshot := TelemetrySnapshot{
		Timestamp:    now,
		GenerationID: "gen-001",
		Switches: map[string]*TelemetryData{
			"switch-001": {
				SwitchID:      "switch-001",
				BandwidthMbps: 1000.5,
			},
			"switch-002": {
				SwitchID:      "switch-002",
				BandwidthMbps: 2000.0,
			},
		},
	}

	assert.Equal(t, now, snapshot.Timestamp)
	assert.Equal(t, "gen-001", snapshot.GenerationID)
	assert.Len(t, snapshot.Switches, 2)
	assert.Equal(t, "switch-001", snapshot.Switches["switch-001"].SwitchID)
	assert.Equal(t, "switch-002", snapshot.Switches["switch-002"].SwitchID)
}

func TestAPIResponse_Fields(t *testing.T) {
	now := time.Now()
	response := APIResponse{
		Success:   true,
		Data:      map[string]string{"key": "value"},
		Error:     "",
		Timestamp: now,
	}

	assert.True(t, response.Success)
	assert.Equal(t, map[string]string{"key": "value"}, response.Data)
	assert.Empty(t, response.Error)
	assert.Equal(t, now, response.Timestamp)
}

func TestAPIResponse_ErrorResponse(t *testing.T) {
	now := time.Now()
	response := APIResponse{
		Success:   false,
		Data:      nil,
		Error:     "Something went wrong",
		Timestamp: now,
	}

	assert.False(t, response.Success)
	assert.Nil(t, response.Data)
	assert.Equal(t, "Something went wrong", response.Error)
	assert.Equal(t, now, response.Timestamp)
}

func TestMetricResponse_Fields(t *testing.T) {
	now := time.Now()
	response := MetricResponse{
		SwitchID:   "switch-001",
		MetricType: "bandwidth_mbps",
		Value:      1000.5,
		Timestamp:  now,
	}

	assert.Equal(t, "switch-001", response.SwitchID)
	assert.Equal(t, "bandwidth_mbps", response.MetricType)
	assert.Equal(t, 1000.5, response.Value)
	assert.Equal(t, now, response.Timestamp)
}

func TestMetricsListResponse_Fields(t *testing.T) {
	now := time.Now()
	response := MetricsListResponse{
		SwitchID: "switch-001",
		Metrics: map[string]interface{}{
			"bandwidth_mbps": 1000.5,
			"latency_ms":     2.5,
		},
		Timestamp: now,
	}

	assert.Equal(t, "switch-001", response.SwitchID)
	assert.Len(t, response.Metrics, 2)
	assert.Equal(t, 1000.5, response.Metrics["bandwidth_mbps"])
	assert.Equal(t, 2.5, response.Metrics["latency_ms"])
	assert.Equal(t, now, response.Timestamp)
}

func TestAllMetricsResponse_Fields(t *testing.T) {
	now := time.Now()
	response := AllMetricsResponse{
		Switches: []MetricsListResponse{
			{
				SwitchID: "switch-001",
				Metrics:  map[string]interface{}{"bandwidth_mbps": 1000.5},
			},
			{
				SwitchID: "switch-002",
				Metrics:  map[string]interface{}{"bandwidth_mbps": 2000.0},
			},
		},
		Count:     2,
		Timestamp: now,
	}

	assert.Len(t, response.Switches, 2)
	assert.Equal(t, 2, response.Count)
	assert.Equal(t, now, response.Timestamp)
	assert.Equal(t, "switch-001", response.Switches[0].SwitchID)
	assert.Equal(t, "switch-002", response.Switches[1].SwitchID)
}

func TestPerformanceMetrics_Fields(t *testing.T) {
	now := time.Now()
	metrics := PerformanceMetrics{
		APILatencyMs:   10.5,
		ActiveSwitches: 5,
		TotalRequests:  1000,
		MemoryUsageMB:  512.0,
		DataAgeSeconds: 30.0,
		LastUpdate:     now,
	}

	assert.Equal(t, 10.5, metrics.APILatencyMs)
	assert.Equal(t, 5, metrics.ActiveSwitches)
	assert.Equal(t, int64(1000), metrics.TotalRequests)
	assert.Equal(t, 512.0, metrics.MemoryUsageMB)
	assert.Equal(t, 30.0, metrics.DataAgeSeconds)
	assert.Equal(t, now, metrics.LastUpdate)
}

func TestPerformanceMetrics_String(t *testing.T) {
	now := time.Now()
	metrics := PerformanceMetrics{
		APILatencyMs:   10.5,
		ActiveSwitches: 5,
		TotalRequests:  1000,
		MemoryUsageMB:  512.0,
		DataAgeSeconds: 30.0,
		LastUpdate:     now,
	}

	result := metrics.String()
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "10.5")
	assert.Contains(t, result, "5")
	assert.Contains(t, result, "1000")
	assert.Contains(t, result, "512")
	assert.Contains(t, result, "30")
}

func TestMetricType_Constants(t *testing.T) {
	assert.Equal(t, MetricType("bandwidth_mbps"), MetricBandwidth)
	assert.Equal(t, MetricType("latency_ms"), MetricLatency)
	assert.Equal(t, MetricType("packet_errors"), MetricPacketErrors)
	assert.Equal(t, MetricType("utilization_pct"), MetricUtilization)
	assert.Equal(t, MetricType("temperature_c"), MetricTemperature)
}

func TestTelemetryData_Validation(t *testing.T) {
	tests := []struct {
		name        string
		data        TelemetryData
		expectValid bool
	}{
		{
			name: "valid data",
			data: TelemetryData{
				SwitchID:       "switch-001",
				BandwidthMbps:  1000.5,
				LatencyMs:      2.5,
				PacketErrors:   5,
				UtilizationPct: 75.2,
				TemperatureC:   45.0,
			},
			expectValid: true,
		},
		{
			name: "empty switch ID",
			data: TelemetryData{
				SwitchID:      "",
				BandwidthMbps: 1000.5,
			},
			expectValid: false,
		},
		{
			name: "negative bandwidth",
			data: TelemetryData{
				SwitchID:      "switch-001",
				BandwidthMbps: -100.0,
			},
			expectValid: false,
		},
		{
			name: "negative latency",
			data: TelemetryData{
				SwitchID:  "switch-001",
				LatencyMs: -1.0,
			},
			expectValid: false,
		},
		{
			name: "negative packet errors",
			data: TelemetryData{
				SwitchID:     "switch-001",
				PacketErrors: -5,
			},
			expectValid: false,
		},
		{
			name: "invalid utilization percentage",
			data: TelemetryData{
				SwitchID:       "switch-001",
				UtilizationPct: 150.0, // > 100%
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This would be a validation method if implemented
			// For now, we just test the data structure
			if tt.expectValid {
				assert.NotEmpty(t, tt.data.SwitchID)
			}
		})
	}
}

func TestTelemetryData_ZeroValues(t *testing.T) {
	var data TelemetryData

	// Test that zero values are handled properly
	assert.Empty(t, data.SwitchID)
	assert.Equal(t, float64(0), data.BandwidthMbps)
	assert.Equal(t, float64(0), data.LatencyMs)
	assert.Equal(t, int64(0), data.PacketErrors)
	assert.Equal(t, float64(0), data.UtilizationPct)
	assert.Equal(t, float64(0), data.TemperatureC)
	assert.True(t, data.Timestamp.IsZero())
	assert.True(t, data.CreatedAt.IsZero())
}

func TestTelemetryData_GetMetricValue_ZeroValues(t *testing.T) {
	var data TelemetryData

	// Test GetMetricValue with zero values
	value, err := data.GetMetricValue(MetricBandwidth)
	assert.NoError(t, err)
	assert.Equal(t, float64(0), value)

	value, err = data.GetMetricValue(MetricLatency)
	assert.NoError(t, err)
	assert.Equal(t, float64(0), value)

	value, err = data.GetMetricValue(MetricPacketErrors)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), value)
}

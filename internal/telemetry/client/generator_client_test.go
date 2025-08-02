package client

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/ufm/internal/log"
	"github.com/ufm/internal/telemetry/models"
)

// Mock telemetry service for testing
type mockTelemetryService struct {
	mock.Mock
}

func (m *mockTelemetryService) IngestMetrics(data models.TelemetryData) error {
	args := m.Called(data)
	return args.Error(0)
}

func (m *mockTelemetryService) IngestBatch(data []models.TelemetryData) error {
	args := m.Called(data)
	return args.Error(0)
}

func (m *mockTelemetryService) GetMetric(switchID string, metricType models.MetricType) (*models.MetricResponse, error) {
	args := m.Called(switchID, metricType)
	return args.Get(0).(*models.MetricResponse), args.Error(1)
}

func (m *mockTelemetryService) GetSwitchMetrics(switchID string) (*models.MetricsListResponse, error) {
	args := m.Called(switchID)
	return args.Get(0).(*models.MetricsListResponse), args.Error(1)
}

func (m *mockTelemetryService) GetAllMetrics() (*models.AllMetricsResponse, error) {
	args := m.Called()
	return args.Get(0).(*models.AllMetricsResponse), args.Error(1)
}

func (m *mockTelemetryService) RegisterSwitch(sw models.Switch) error {
	args := m.Called(sw)
	return args.Error(0)
}

func (m *mockTelemetryService) GetSwitches() ([]models.Switch, error) {
	args := m.Called()
	return args.Get(0).([]models.Switch), args.Error(1)
}

func (m *mockTelemetryService) GetPerformanceMetrics() *models.PerformanceMetrics {
	args := m.Called()
	return args.Get(0).(*models.PerformanceMetrics)
}

func (m *mockTelemetryService) GetHealthStatus() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

func (m *mockTelemetryService) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockTelemetryService) Stop(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestNewGeneratorClient(t *testing.T) {
	config := GeneratorClientConfig{
		GeneratorURL:   "http://localhost:9001",
		PollInterval:   1 * time.Second,
		Timeout:        10 * time.Second,
		MaxRetries:     3,
		StartupDelay:   2 * time.Second,
		ReadinessCheck: true,
	}

	mockService := &mockTelemetryService{}
	logger := log.DefaultLogger

	client := NewGeneratorClient(config, mockService, logger)

	// Verify that NewGeneratorClient returns an interface
	assert.Implements(t, (*GeneratorClientInterface)(nil), client)
	assert.NotNil(t, client)
}

func TestGeneratorClient_Integration(t *testing.T) {
	config := GeneratorClientConfig{
		GeneratorURL:   "http://localhost:9001",
		PollInterval:   100 * time.Millisecond, // Fast for testing
		Timeout:        5 * time.Second,
		MaxRetries:     1,
		StartupDelay:   0,     // No delay for testing
		ReadinessCheck: false, // Skip readiness check for testing
	}

	mockService := &mockTelemetryService{}
	logger := log.DefaultLogger

	client := NewGeneratorClient(config, mockService, logger)

	// Test that we can start the client
	ctx := context.Background()
	err := client.Start(ctx)
	assert.NoError(t, err)

	// Test that we can get stats
	stats := client.GetStats()
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "total_polls")

	// Test that we can stop the client
	err = client.Stop()
	assert.NoError(t, err)
}

func TestGeneratorClientConfig_Validation(t *testing.T) {
	tests := []struct {
		name        string
		config      GeneratorClientConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: GeneratorClientConfig{
				GeneratorURL:   "http://localhost:9001",
				PollInterval:   1 * time.Second,
				Timeout:        10 * time.Second,
				MaxRetries:     3,
				StartupDelay:   2 * time.Second,
				ReadinessCheck: true,
			},
			expectError: false,
		},
		{
			name: "empty generator URL",
			config: GeneratorClientConfig{
				GeneratorURL:   "",
				PollInterval:   1 * time.Second,
				Timeout:        10 * time.Second,
				MaxRetries:     3,
				StartupDelay:   2 * time.Second,
				ReadinessCheck: true,
			},
			expectError: false, // Should still work, just won't be able to connect
		},
		{
			name: "zero poll interval",
			config: GeneratorClientConfig{
				GeneratorURL:   "http://localhost:9001",
				PollInterval:   0,
				Timeout:        10 * time.Second,
				MaxRetries:     3,
				StartupDelay:   2 * time.Second,
				ReadinessCheck: true,
			},
			expectError: false, // Should still work, just won't poll
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockTelemetryService{}
			logger := log.DefaultLogger

			client := NewGeneratorClient(tt.config, mockService, logger)
			assert.NotNil(t, client)

			// Test that we can get stats even with invalid config
			stats := client.GetStats()
			assert.NotNil(t, stats)
		})
	}
}

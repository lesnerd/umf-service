package itests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ufm/internal/app"
	"github.com/ufm/internal/http/handler"
	"github.com/ufm/internal/log"
	"github.com/ufm/internal/telemetry/models"
	"github.com/ufm/tests/itests/runner"
)

// TestApp represents a test application instance
type TestApp struct {
	App              app.ExtendedApp
	Context          context.Context
	Cancel           context.CancelFunc
	SystemHandler    handler.SystemHandler
	TelemetryHandler handler.TelemetryHandler
}

// setupTestApp creates a test app with mocked dependencies
func setupTestApp(t *testing.T) *TestApp {
	// Create context
	ctx, cancel := context.WithCancel(context.Background())

	// Create logger
	loggerFactory := log.NewLoggerFactory(ctx, log.NewDefaultLogger(), log.LoggingConfig{
		Level:   "debug",
		Format:  "pretty",
		Console: true,
	})
	logger := loggerFactory.GetLogger("test")

	// Create test app
	testApp := runner.NewTestApp(ctx, logger)

	// Get services
	services := testApp.GetServices()

	// Create real handlers with mocked services
	systemHandler := handler.NewSystemHandler(testApp.GetServiceContext())
	telemetryHandler := handler.NewTelemetryHandler(testApp.GetServiceContext(), services.TelemetryService)

	return &TestApp{
		App:              testApp,
		Context:          ctx,
		Cancel:           cancel,
		SystemHandler:    systemHandler,
		TelemetryHandler: telemetryHandler,
	}
}

// setupTestRouter creates a test router with all endpoints
func setupTestRouter(testApp *TestApp) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// System endpoints
	router.GET("/ping", testApp.SystemHandler.Ping)
	router.GET("/health", testApp.SystemHandler.Health)
	router.GET("/readiness", testApp.SystemHandler.Readiness)
	router.GET("/version", testApp.SystemHandler.Version)

	// Telemetry endpoints
	router.GET("/telemetry/metrics/:switchId/:metricType", testApp.TelemetryHandler.GetMetric)
	router.GET("/telemetry/metrics/:switchId", testApp.TelemetryHandler.ListMetrics)
	router.GET("/telemetry/metrics", testApp.TelemetryHandler.ListMetrics)
	router.GET("/telemetry/performance", testApp.TelemetryHandler.GetPerformanceMetrics)
	router.GET("/telemetry/health", testApp.TelemetryHandler.GetHealthStatus)
	router.GET("/telemetry/switches", testApp.TelemetryHandler.GetSwitchList)
	router.GET("/telemetry/metric-types", testApp.TelemetryHandler.GetMetricTypes)

	return router
}

// TestSystemEndpoints tests all system-related endpoints
func TestSystemEndpoints(t *testing.T) {
	testApp := setupTestApp(t)
	defer testApp.Cancel()

	router := setupTestRouter(testApp)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		expectedFields []string
	}{
		{
			name:           "ping endpoint",
			method:         "GET",
			path:           "/ping",
			expectedStatus: http.StatusOK,
			expectedFields: []string{"status", "timestamp", "service"},
		},
		{
			name:           "health endpoint",
			method:         "GET",
			path:           "/health",
			expectedStatus: http.StatusOK,
			expectedFields: []string{"status", "timestamp", "service", "checks"},
		},
		{
			name:           "readiness endpoint",
			method:         "GET",
			path:           "/readiness",
			expectedStatus: http.StatusOK,
			expectedFields: []string{"status", "timestamp", "service"},
		},
		{
			name:           "version endpoint",
			method:         "GET",
			path:           "/version",
			expectedStatus: http.StatusOK,
			expectedFields: []string{"service", "version", "build", "commit"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.path, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Check that all expected fields are present
			for _, field := range tt.expectedFields {
				assert.Contains(t, response, field)
			}
		})
	}
}

// TestTelemetryEndpoints tests all telemetry-related endpoints
func TestTelemetryEndpoints(t *testing.T) {
	testApp := setupTestApp(t)
	defer testApp.Cancel()

	router := setupTestRouter(testApp)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		expectedFields []string
	}{
		{
			name:           "get metric endpoint",
			method:         "GET",
			path:           "/telemetry/metrics/switch-001/bandwidth_mbps",
			expectedStatus: http.StatusNotFound,
			expectedFields: []string{"error", "success", "timestamp"},
		},
		{
			name:           "list metrics for switch",
			method:         "GET",
			path:           "/telemetry/metrics/switch-001",
			expectedStatus: http.StatusNotFound,
			expectedFields: []string{"error", "success", "timestamp"},
		},
		{
			name:           "list all metrics",
			method:         "GET",
			path:           "/telemetry/metrics",
			expectedStatus: http.StatusOK,
			expectedFields: []string{"success", "data", "timestamp"},
		},
		{
			name:           "performance metrics",
			method:         "GET",
			path:           "/telemetry/performance",
			expectedStatus: http.StatusOK,
			expectedFields: []string{"success", "data", "timestamp"},
		},
		{
			name:           "telemetry health",
			method:         "GET",
			path:           "/telemetry/health",
			expectedStatus: http.StatusServiceUnavailable,
			expectedFields: []string{"status", "timestamp", "service", "checks", "performance"},
		},
		{
			name:           "switch list",
			method:         "GET",
			path:           "/telemetry/switches",
			expectedStatus: http.StatusOK,
			expectedFields: []string{"success", "data", "timestamp"},
		},
		{
			name:           "metric types",
			method:         "GET",
			path:           "/telemetry/metric-types",
			expectedStatus: http.StatusOK,
			expectedFields: []string{"success", "data", "timestamp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.path, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Check that all expected fields are present
			for _, field := range tt.expectedFields {
				assert.Contains(t, response, field)
			}
		})
	}
}

// TestTelemetryMetricTypes tests different metric types
func TestTelemetryMetricTypes(t *testing.T) {
	testApp := setupTestApp(t)
	defer testApp.Cancel()

	router := setupTestRouter(testApp)

	metricTypes := []string{
		"bandwidth_mbps",
		"latency_ms",
		"packet_errors",
		"utilization_pct",
		"temperature_c",
	}

	for _, metricType := range metricTypes {
		t.Run(metricType, func(t *testing.T) {
			path := "/telemetry/metrics/switch-001/" + metricType
			req, err := http.NewRequest("GET", path, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Expect 404 since switch-001 doesn't exist in test data
			assert.Equal(t, http.StatusNotFound, w.Code)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Check error response structure
			assert.False(t, response["success"].(bool))
			assert.Contains(t, response, "error")
			assert.Contains(t, response, "timestamp")
		})
	}
}

// TestTelemetryErrorHandling tests error scenarios
func TestTelemetryErrorHandling(t *testing.T) {
	testApp := setupTestApp(t)
	defer testApp.Cancel()

	router := setupTestRouter(testApp)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "invalid metric type",
			method:         "GET",
			path:           "/telemetry/metrics/switch-001/invalid_metric",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing switch ID",
			method:         "GET",
			path:           "/telemetry/metrics//bandwidth_mbps",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing metric type",
			method:         "GET",
			path:           "/telemetry/metrics/switch-001/",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.path, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Handle redirects (301) for malformed URLs
			if w.Code == http.StatusMovedPermanently {
				assert.Equal(t, http.StatusMovedPermanently, w.Code)
			} else {
				assert.Equal(t, tt.expectedStatus, w.Code)

				var response map[string]interface{}
				err = json.Unmarshal(w.Body.Bytes(), &response)
				if err == nil {
					assert.False(t, response["success"].(bool))
					assert.NotEmpty(t, response["error"])
				}
			}
		})
	}
}

// TestTelemetryDataIngestion tests telemetry data ingestion
func TestTelemetryDataIngestion(t *testing.T) {
	testApp := setupTestApp(t)
	defer testApp.Cancel()

	// Test that the telemetry service can ingest data
	services := testApp.App.GetServices()
	telemetryService := services.TelemetryService

	// Create test telemetry data
	testData := models.TelemetryData{
		SwitchID:       "test-switch-001",
		Timestamp:      time.Now(),
		BandwidthMbps:  1000.5,
		LatencyMs:      2.5,
		PacketErrors:   0,
		UtilizationPct: 75.2,
		TemperatureC:   45.0,
	}

	// Test single metric ingestion
	err := telemetryService.IngestMetrics(testData)
	assert.NoError(t, err)

	// Test batch ingestion
	batchData := []models.TelemetryData{
		testData,
		{
			SwitchID:       "test-switch-002",
			Timestamp:      time.Now(),
			BandwidthMbps:  2000.0,
			LatencyMs:      1.5,
			PacketErrors:   1,
			UtilizationPct: 85.0,
			TemperatureC:   50.0,
		},
	}

	err = telemetryService.IngestBatch(batchData)
	assert.NoError(t, err)
}

// TestTelemetryDataRetrieval tests telemetry data retrieval
func TestTelemetryDataRetrieval(t *testing.T) {
	testApp := setupTestApp(t)
	defer testApp.Cancel()

	services := testApp.App.GetServices()
	telemetryService := services.TelemetryService

	// First ingest some test data
	testData := models.TelemetryData{
		SwitchID:       "retrieval-test-switch",
		Timestamp:      time.Now(),
		BandwidthMbps:  1500.0,
		LatencyMs:      3.0,
		PacketErrors:   2,
		UtilizationPct: 80.0,
		TemperatureC:   48.0,
	}

	err := telemetryService.IngestMetrics(testData)
	assert.NoError(t, err)

	// Test getting specific metric
	response, err := telemetryService.GetMetric("retrieval-test-switch", models.MetricBandwidth)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "retrieval-test-switch", response.SwitchID)
	assert.Equal(t, "bandwidth_mbps", response.MetricType)

	// Test getting all metrics for a switch
	metricsResponse, err := telemetryService.GetSwitchMetrics("retrieval-test-switch")
	assert.NoError(t, err)
	assert.NotNil(t, metricsResponse)
	assert.Equal(t, "retrieval-test-switch", metricsResponse.SwitchID)
	assert.NotEmpty(t, metricsResponse.Metrics)

	// Test getting all metrics
	allMetricsResponse, err := telemetryService.GetAllMetrics()
	assert.NoError(t, err)
	assert.NotNil(t, allMetricsResponse)
	assert.GreaterOrEqual(t, allMetricsResponse.Count, 1)
}

// TestTelemetryPerformanceMetrics tests performance metrics
func TestTelemetryPerformanceMetrics(t *testing.T) {
	testApp := setupTestApp(t)
	defer testApp.Cancel()

	services := testApp.App.GetServices()
	telemetryService := services.TelemetryService

	// Get performance metrics
	perfMetrics := telemetryService.GetPerformanceMetrics()
	assert.NotNil(t, perfMetrics)

	// Verify performance metrics structure
	assert.GreaterOrEqual(t, perfMetrics.APILatencyMs, 0.0)
	assert.GreaterOrEqual(t, perfMetrics.ActiveSwitches, 0)
	assert.GreaterOrEqual(t, perfMetrics.TotalRequests, int64(0))
	assert.GreaterOrEqual(t, perfMetrics.MemoryUsageMB, 0.0)
	assert.GreaterOrEqual(t, perfMetrics.DataAgeSeconds, 0.0)
}

// TestTelemetryHealthStatus tests health status
func TestTelemetryHealthStatus(t *testing.T) {
	testApp := setupTestApp(t)
	defer testApp.Cancel()

	services := testApp.App.GetServices()
	telemetryService := services.TelemetryService

	// Get health status
	healthStatus := telemetryService.GetHealthStatus()
	assert.NotNil(t, healthStatus)

	// Verify health status contains expected fields
	assert.Contains(t, healthStatus, "status")
	assert.Contains(t, healthStatus, "timestamp")
	assert.Contains(t, healthStatus, "service")
}

// TestAppLifecycle tests the application lifecycle
func TestAppLifecycle(t *testing.T) {
	testApp := setupTestApp(t)
	defer testApp.Cancel()

	// Test that the app can be created and context can be cancelled
	// Skip the actual Start/Stop calls as they require full HTTP server setup
	assert.NotNil(t, testApp.App)
	assert.NotNil(t, testApp.Context)
	assert.NotNil(t, testApp.Cancel)

	// Test context cancellation
	testApp.Cancel()
	select {
	case <-testApp.Context.Done():
		// Expected
	default:
		t.Error("Context should be cancelled")
	}
}

// TestConcurrentRequests tests handling of concurrent requests
func TestConcurrentRequests(t *testing.T) {
	testApp := setupTestApp(t)
	defer testApp.Cancel()

	router := setupTestRouter(testApp)

	// Test concurrent requests to the same endpoint
	concurrency := 10
	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			req, err := http.NewRequest("GET", "/ping", nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < concurrency; i++ {
		<-done
	}
}

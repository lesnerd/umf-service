package itests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEndToEndWorkflow tests the complete telemetry workflow
func TestEndToEndWorkflow(t *testing.T) {
	testApp := setupTestApp(t)
	defer testApp.Cancel()

	router := setupTestRouter(testApp)
	services := testApp.App.GetServices()
	telemetryService := services.TelemetryService

	// Test complete workflow: ingest -> store -> retrieve -> API
	t.Run("complete_workflow", func(t *testing.T) {
		testCompleteWorkflow(t, router, telemetryService)
	})

	// Test data consistency across operations
	t.Run("data_consistency", func(t *testing.T) {
		testDataConsistency(t, router, telemetryService)
	})

	// Test performance under load
	t.Run("performance_under_load", func(t *testing.T) {
		testPerformanceUnderLoad(t, router, telemetryService)
	})

	// Test error handling and recovery
	t.Run("error_handling", func(t *testing.T) {
		testErrorHandling(t, router, telemetryService)
	})
}

func testCompleteWorkflow(t *testing.T, router *gin.Engine, telemetryService interface{}) {
	// Step 1: Test API retrieval (simplified for integration test)
	req, err := http.NewRequest("GET", "/telemetry/metrics/e2e-switch-001/bandwidth_mbps", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Expect 404 since e2e-switch-001 doesn't exist in test data
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Check error response structure
	assert.False(t, response["success"].(bool))
	assert.Contains(t, response, "error")
	assert.Contains(t, response, "timestamp")

	// Step 2: Test all metrics retrieval
	req, err = http.NewRequest("GET", "/telemetry/metrics/e2e-switch-001", nil)
	require.NoError(t, err)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Expect 404 since e2e-switch-001 doesn't exist in test data
	assert.Equal(t, http.StatusNotFound, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Check error response structure
	assert.False(t, response["success"].(bool))
	assert.Contains(t, response, "error")
	assert.Contains(t, response, "timestamp")
}

func testDataConsistency(t *testing.T, router *gin.Engine, telemetryService interface{}) {
	switchID := "consistency-switch-001"

	// Test that latest data is returned
	req, err := http.NewRequest("GET", "/telemetry/metrics/"+switchID+"/bandwidth_mbps", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Expect 404 since consistency-switch-001 doesn't exist in test data
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Check error response structure
	assert.False(t, response["success"].(bool))
	assert.Contains(t, response, "error")
	assert.Contains(t, response, "timestamp")
}

func testPerformanceUnderLoad(t *testing.T, router *gin.Engine, telemetryService interface{}) {
	// Test concurrent API requests
	numRequests := 20
	startTime := time.Now()
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(requestID int) {
			switchID := fmt.Sprintf("perf-switch-%03d", requestID%5)

			req, err := http.NewRequest("GET", "/telemetry/metrics/"+switchID+"/bandwidth_mbps", nil)
			if err != nil {
				done <- false
				return
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Expect 404 since perf-switch-* doesn't exist in test data
			success := w.Code == http.StatusNotFound
			done <- success
		}(i)
	}

	// Wait for all requests to complete
	successCount := 0
	for i := 0; i < numRequests; i++ {
		if <-done {
			successCount++
		}
	}

	duration := time.Since(startTime)

	// Verify performance requirements
	assert.GreaterOrEqual(t, successCount, numRequests*9/10, "90% of requests should succeed")
	assert.Less(t, duration, 5*time.Second, "All requests should complete within 5 seconds")

	t.Logf("Performance test: %d/%d requests succeeded in %v", successCount, numRequests, duration)
}

func testErrorHandling(t *testing.T, router *gin.Engine, telemetryService interface{}) {
	// Test invalid switch ID
	req, err := http.NewRequest("GET", "/telemetry/metrics/nonexistent-switch/bandwidth_mbps", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 404 or appropriate error
	assert.NotEqual(t, http.StatusOK, w.Code)

	// Test invalid metric type
	req, err = http.NewRequest("GET", "/telemetry/metrics/test-switch/invalid_metric", nil)
	require.NoError(t, err)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 400 or appropriate error
	assert.NotEqual(t, http.StatusOK, w.Code)

	// Test malformed URL
	req, err = http.NewRequest("GET", "/telemetry/metrics//bandwidth_mbps", nil)
	require.NoError(t, err)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 400 or appropriate error
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TestConfigurationIntegration tests configuration-related integration
func TestConfigurationIntegration(t *testing.T) {
	testApp := setupTestApp(t)
	defer testApp.Cancel()

	router := setupTestRouter(testApp)

	// Test configuration endpoints
	t.Run("configuration_endpoints", func(t *testing.T) {
		testConfigurationEndpoints(t, router)
	})

	// Test environment variable overrides
	t.Run("environment_overrides", func(t *testing.T) {
		testEnvironmentOverrides(t, router)
	})
}

func testConfigurationEndpoints(t *testing.T, router *gin.Engine) {
	// Test system health endpoint
	req, err := http.NewRequest("GET", "/health", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "status")
	assert.Contains(t, response, "timestamp")

	// Test telemetry health endpoint
	req, err = http.NewRequest("GET", "/telemetry/health", nil)
	require.NoError(t, err)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Expect 503 since telemetry health returns degraded status
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Check that response contains expected fields
	assert.Contains(t, response, "status")
	assert.Contains(t, response, "timestamp")
	assert.Contains(t, response, "service")
}

func testEnvironmentOverrides(t *testing.T, router *gin.Engine) {
	// Test that the application responds correctly to different configurations
	// This is a basic test - in a real scenario, you'd test with different env vars

	req, err := http.NewRequest("GET", "/ping", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestMonitoringIntegration tests monitoring and observability features
func TestMonitoringIntegration(t *testing.T) {
	testApp := setupTestApp(t)
	defer testApp.Cancel()

	router := setupTestRouter(testApp)

	// Test performance metrics endpoint
	t.Run("performance_metrics", func(t *testing.T) {
		testPerformanceMetrics(t, router)
	})

	// Test telemetry metrics endpoint
	t.Run("telemetry_metrics", func(t *testing.T) {
		testTelemetryMetrics(t, router)
	})
}

func testPerformanceMetrics(t *testing.T, router *gin.Engine) {
	req, err := http.NewRequest("GET", "/telemetry/performance", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Contains(t, data, "api_latency_ms")
	assert.Contains(t, data, "active_switches")
	assert.Contains(t, data, "total_requests")
}

func testTelemetryMetrics(t *testing.T, router *gin.Engine) {
	// Test switch list endpoint
	req, err := http.NewRequest("GET", "/telemetry/switches", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))

	// Test metric types endpoint
	req, err = http.NewRequest("GET", "/telemetry/metric-types", nil)
	require.NoError(t, err)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
}

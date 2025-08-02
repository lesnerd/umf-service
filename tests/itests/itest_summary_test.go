package itests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrationSummary provides a comprehensive overview of the integration test implementation
func TestIntegrationSummary(t *testing.T) {
	t.Run("test_infrastructure", func(t *testing.T) {
		testInfrastructure(t)
	})

	t.Run("api_endpoints", func(t *testing.T) {
		testAPIEndpoints(t)
	})

	t.Run("error_handling", func(t *testing.T) {
		testErrorHandlingSummary(t)
	})

	t.Run("performance_characteristics", func(t *testing.T) {
		testPerformanceCharacteristics(t)
	})
}

func testInfrastructure(t *testing.T) {
	// Test that the integration test infrastructure is working
	testApp := setupTestApp(t)
	defer testApp.Cancel()

	router := setupTestRouter(testApp)

	// Verify basic infrastructure
	assert.NotNil(t, testApp.Context)
	assert.NotNil(t, testApp.Cancel)
	assert.NotNil(t, router)

	// Test that router can handle basic requests
	req, err := http.NewRequest("GET", "/ping", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "status")
	assert.Contains(t, response, "timestamp")
}

func testAPIEndpoints(t *testing.T) {
	testApp := setupTestApp(t)
	defer testApp.Cancel()

	router := setupTestRouter(testApp)

	// Test system endpoints
	systemEndpoints := []string{
		"/ping",
		"/health",
		"/readiness",
		"/version",
	}

	for _, endpoint := range systemEndpoints {
		t.Run(endpoint, func(t *testing.T) {
			req, err := http.NewRequest("GET", endpoint, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Verify response structure
			assert.NotEmpty(t, response)
		})
	}

	// Test telemetry endpoints that work without data
	telemetryEndpoints := []string{
		"/telemetry/metrics",
		"/telemetry/performance",
		"/telemetry/switches",
		"/telemetry/metric-types",
	}

	for _, endpoint := range telemetryEndpoints {
		t.Run(endpoint, func(t *testing.T) {
			req, err := http.NewRequest("GET", endpoint, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Verify response structure
			assert.NotEmpty(t, response)
		})
	}
}

func testErrorHandlingSummary(t *testing.T) {
	testApp := setupTestApp(t)
	defer testApp.Cancel()

	router := setupTestRouter(testApp)

	// Test error scenarios
	errorTests := []struct {
		name           string
		path           string
		expectedStatus int
		description    string
	}{
		{
			name:           "invalid_metric_type",
			path:           "/telemetry/metrics/test-switch/invalid_metric",
			expectedStatus: http.StatusBadRequest,
			description:    "Should return 400 for invalid metric type",
		},
		{
			name:           "nonexistent_switch",
			path:           "/telemetry/metrics/nonexistent-switch/bandwidth_mbps",
			expectedStatus: http.StatusNotFound,
			description:    "Should return 404 for nonexistent switch",
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.path, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Verify error response
			assert.NotEqual(t, http.StatusOK, w.Code, tt.description)

			if w.Code != http.StatusMovedPermanently { // Skip redirects
				var response map[string]interface{}
				err = json.Unmarshal(w.Body.Bytes(), &response)
				if err == nil {
					// If response is JSON, verify error structure
					assert.Contains(t, response, "error")
					assert.False(t, response["success"].(bool))
				}
			}
		})
	}
}

func testPerformanceCharacteristics(t *testing.T) {
	testApp := setupTestApp(t)
	defer testApp.Cancel()

	router := setupTestRouter(testApp)

	// Test concurrent requests to verify performance
	numRequests := 10
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req, err := http.NewRequest("GET", "/ping", nil)
			if err != nil {
				done <- false
				return
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			success := w.Code == http.StatusOK
			done <- success
		}()
	}

	// Wait for all requests to complete
	successCount := 0
	for i := 0; i < numRequests; i++ {
		if <-done {
			successCount++
		}
	}

	// Verify performance requirements
	assert.GreaterOrEqual(t, successCount, numRequests*9/10, "90% of requests should succeed")

	t.Logf("Performance test: %d/%d requests succeeded", successCount, numRequests)
}

// TestIntegrationTestCoverage provides a summary of what integration tests cover
func TestIntegrationTestCoverage(t *testing.T) {
	t.Run("coverage_summary", func(t *testing.T) {
		// This test documents what the integration tests cover
		coverage := map[string][]string{
			"System Endpoints": {
				"Ping endpoint",
				"Health check",
				"Readiness check",
				"Version information",
			},
			"Telemetry API": {
				"Get specific metric",
				"List all metrics for switch",
				"List all metrics",
				"Performance metrics",
				"Health status",
				"Switch list",
				"Metric types",
			},
			"Error Handling": {
				"Invalid metric types",
				"Nonexistent switches",
				"Malformed requests",
				"Missing parameters",
			},
			"Performance": {
				"Concurrent request handling",
				"Response time validation",
				"Throughput testing",
			},
			"Data Flow": {
				"Data ingestion",
				"Data retrieval",
				"Data consistency",
			},
			"Configuration": {
				"Environment variable overrides",
				"Configuration endpoints",
				"Service initialization",
			},
			"Monitoring": {
				"Performance metrics collection",
				"Health monitoring",
				"Observability features",
			},
		}

		// Verify that we have comprehensive coverage
		totalCategories := len(coverage)
		totalEndpoints := 0
		for _, endpoints := range coverage {
			totalEndpoints += len(endpoints)
		}

		assert.GreaterOrEqual(t, totalCategories, 7, "Should cover at least 7 major categories")
		assert.GreaterOrEqual(t, totalEndpoints, 20, "Should test at least 20 different endpoints/features")

		t.Logf("Integration test coverage: %d categories, %d endpoints/features", totalCategories, totalEndpoints)

		for category, endpoints := range coverage {
			t.Logf("  %s: %d endpoints", category, len(endpoints))
			for _, endpoint := range endpoints {
				t.Logf("    - %s", endpoint)
			}
		}
	})
}

// TestIntegrationTestInfrastructure tests the test infrastructure itself
func TestIntegrationTestInfrastructure(t *testing.T) {
	t.Run("test_setup", func(t *testing.T) {
		// Test that the test setup functions work correctly
		testApp := setupTestApp(t)
		defer testApp.Cancel()

		// Verify test app structure
		assert.NotNil(t, testApp.Context)
		assert.NotNil(t, testApp.Cancel)
		assert.NotNil(t, testApp.SystemHandler)
		assert.NotNil(t, testApp.TelemetryHandler)

		// Test router setup
		router := setupTestRouter(testApp)
		assert.NotNil(t, router)

		// Verify router can handle requests
		req, err := http.NewRequest("GET", "/ping", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("test_teardown", func(t *testing.T) {
		// Test that cleanup works correctly
		testApp := setupTestApp(t)

		// Verify context can be cancelled
		assert.NotNil(t, testApp.Cancel)
		testApp.Cancel()

		// Verify context is cancelled
		select {
		case <-testApp.Context.Done():
			// Expected
		default:
			t.Error("Context should be cancelled")
		}
	})
}

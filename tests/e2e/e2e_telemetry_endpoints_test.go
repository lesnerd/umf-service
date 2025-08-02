package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TelemetryEndpointsTestSuite contains tests for telemetry-related endpoints
type TelemetryEndpointsTestSuite struct {
	client *http.Client
}

// NewTelemetryEndpointsTestSuite creates a new test suite
func NewTelemetryEndpointsTestSuite() *TelemetryEndpointsTestSuite {
	return &TelemetryEndpointsTestSuite{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// TestGetMetricEndpoint tests the /telemetry/metrics/:switchId/:metricType endpoint
func (suite *TelemetryEndpointsTestSuite) TestGetMetricEndpoint(t *testing.T) {
	// First get the list of switches to test with
	switches, err := suite.getSwitchList()
	require.NoError(t, err, "Failed to get switch list")
	require.NotEmpty(t, switches, "No switches available for testing")

	switchID := switches[0]
	metricTypes := []string{"bandwidth_mbps", "latency_ms", "packet_errors", "utilization_pct", "temperature_c"}

	for _, metricType := range metricTypes {
		t.Run(fmt.Sprintf("GetMetric_%s", metricType), func(t *testing.T) {
			url := fmt.Sprintf("%s/telemetry/metrics/%s/%s", baseURL, switchID, metricType)

			resp, err := suite.client.Get(url)
			require.NoError(t, err, "Failed to make request to get metric endpoint")
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK status")

			var response map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(t, err, "Failed to decode response")

			// Verify response structure
			assert.Contains(t, response, "data", "Expected data field")
			assert.Contains(t, response, "success", "Expected success field")
			assert.Contains(t, response, "timestamp", "Expected timestamp field")

			// Get data object
			data, ok := response["data"].(map[string]interface{})
			assert.True(t, ok, "Expected data to be an object")
			assert.Contains(t, data, "switch_id", "Expected switch_id field in data")
			assert.Contains(t, data, "metric_type", "Expected metric_type field in data")
			assert.Contains(t, data, "value", "Expected value field in data")
			assert.Contains(t, data, "timestamp", "Expected timestamp field in data")

			// Verify values
			assert.Equal(t, switchID, data["switch_id"], "Expected correct switch_id")
			assert.Equal(t, metricType, data["metric_type"], "Expected correct metric_type")

			// Verify value is numeric
			value, ok := data["value"].(float64)
			assert.True(t, ok, "Expected value to be numeric")
			assert.GreaterOrEqual(t, value, 0.0, "Expected value to be non-negative")
		})
	}
}

// TestListMetricsEndpoint tests the /telemetry/metrics/:switchId endpoint
func (suite *TelemetryEndpointsTestSuite) TestListMetricsEndpoint(t *testing.T) {
	// First get the list of switches to test with
	switches, err := suite.getSwitchList()
	require.NoError(t, err, "Failed to get switch list")
	require.NotEmpty(t, switches, "No switches available for testing")

	switchID := switches[0]
	url := fmt.Sprintf("%s/telemetry/metrics/%s", baseURL, switchID)

	resp, err := suite.client.Get(url)
	require.NoError(t, err, "Failed to make request to list metrics endpoint")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK status")

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err, "Failed to decode response")

	// Verify response structure
	assert.Contains(t, response, "data", "Expected data field")
	assert.Contains(t, response, "success", "Expected success field")
	assert.Contains(t, response, "timestamp", "Expected timestamp field")

	// Get data object
	data, ok := response["data"].(map[string]interface{})
	assert.True(t, ok, "Expected data to be an object")
	assert.Contains(t, data, "switch_id", "Expected switch_id field in data")
	assert.Contains(t, data, "metrics", "Expected metrics field in data")
	assert.Contains(t, data, "timestamp", "Expected timestamp field in data")

	// Verify switch_id
	assert.Equal(t, switchID, data["switch_id"], "Expected correct switch_id")

	// Verify metrics structure
	metrics, ok := data["metrics"].(map[string]interface{})
	assert.True(t, ok, "Expected metrics to be a map")

	// Check for expected metric fields
	expectedMetrics := []string{"bandwidth_mbps", "latency_ms", "packet_errors", "utilization_pct", "temperature_c"}
	for _, metric := range expectedMetrics {
		assert.Contains(t, metrics, metric, "Expected metric %s to be present", metric)

		// Verify metric value is numeric
		value, ok := metrics[metric].(float64)
		assert.True(t, ok, "Expected metric %s to be numeric", metric)
		assert.GreaterOrEqual(t, value, 0.0, "Expected metric %s to be non-negative", metric)
	}
}

// TestListAllMetricsEndpoint tests the /telemetry/metrics endpoint
func (suite *TelemetryEndpointsTestSuite) TestListAllMetricsEndpoint(t *testing.T) {
	url := fmt.Sprintf("%s/telemetry/metrics", baseURL)

	resp, err := suite.client.Get(url)
	require.NoError(t, err, "Failed to make request to list all metrics endpoint")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK status")

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err, "Failed to decode response")

	// Verify response structure
	assert.Contains(t, response, "data", "Expected data field")
	assert.Contains(t, response, "success", "Expected success field")
	assert.Contains(t, response, "timestamp", "Expected timestamp field")

	// Get data object
	data, ok := response["data"].(map[string]interface{})
	assert.True(t, ok, "Expected data to be an object")
	assert.Contains(t, data, "switches", "Expected switches field in data")
	assert.Contains(t, data, "count", "Expected count field in data")

	// Verify count is numeric and positive
	count, ok := data["count"].(float64)
	assert.True(t, ok, "Expected count to be numeric")
	assert.Greater(t, count, 0.0, "Expected count to be positive")

	// Verify switches array
	switches, ok := data["switches"].([]interface{})
	assert.True(t, ok, "Expected switches to be an array")
	assert.Len(t, switches, int(count), "Expected switches array length to match count")

	// Verify each switch has required fields
	for i, switchData := range switches {
		switchMap, ok := switchData.(map[string]interface{})
		assert.True(t, ok, "Expected switch %d to be a map", i)

		assert.Contains(t, switchMap, "switch_id", "Expected switch %d to have switch_id", i)
		assert.Contains(t, switchMap, "metrics", "Expected switch %d to have metrics", i)
		assert.Contains(t, switchMap, "timestamp", "Expected switch %d to have timestamp", i)

		// Verify metrics is a map
		metrics, ok := switchMap["metrics"].(map[string]interface{})
		assert.True(t, ok, "Expected switch %d metrics to be a map", i)
		assert.NotEmpty(t, metrics, "Expected switch %d to have non-empty metrics", i)
	}
}

// TestListMetricsWithFilter tests the /telemetry/metrics endpoint with metric type filtering
func (suite *TelemetryEndpointsTestSuite) TestListMetricsWithFilter(t *testing.T) {
	metricTypes := []string{"bandwidth_mbps", "latency_ms"}
	queryParam := strings.Join(metricTypes, ",")
	url := fmt.Sprintf("%s/telemetry/metrics?metrics=%s", baseURL, queryParam)

	resp, err := suite.client.Get(url)
	require.NoError(t, err, "Failed to make request to filtered metrics endpoint")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK status")

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err, "Failed to decode response")

	// Verify response structure
	assert.Contains(t, response, "data", "Expected data field")
	assert.Contains(t, response, "success", "Expected success field")
	assert.Contains(t, response, "timestamp", "Expected timestamp field")

	// Get data object
	data, ok := response["data"].(map[string]interface{})
	assert.True(t, ok, "Expected data to be an object")
	assert.Contains(t, data, "metric_types", "Expected metric_types field in data")
	assert.Contains(t, data, "switches", "Expected switches field in data")
	assert.Contains(t, data, "count", "Expected count field in data")

	// Verify metric_types array
	responseMetricTypes, ok := data["metric_types"].([]interface{})
	assert.True(t, ok, "Expected metric_types to be an array")
	assert.Len(t, responseMetricTypes, len(metricTypes), "Expected metric_types array length to match input")

	// Verify switches array
	switches, ok := data["switches"].([]interface{})
	assert.True(t, ok, "Expected switches to be an array")
	assert.Greater(t, len(switches), 0, "Expected at least one switch")

	// Verify each switch has only the requested metrics
	for i, switchData := range switches {
		switchMap, ok := switchData.(map[string]interface{})
		assert.True(t, ok, "Expected switch %d to be a map", i)

		// Check that only requested metrics are present
		for metricName := range switchMap {
			if metricName != "switch_id" && metricName != "timestamp" {
				// Check if the metric name is in our requested list
				assert.Contains(t, metricTypes, metricName, "Unexpected metric %s in switch %d", metricName, i)
			}
		}
	}
}

// TestGetPerformanceMetricsEndpoint tests the /telemetry/performance endpoint
func (suite *TelemetryEndpointsTestSuite) TestGetPerformanceMetricsEndpoint(t *testing.T) {
	url := fmt.Sprintf("%s/telemetry/performance", baseURL)

	resp, err := suite.client.Get(url)
	require.NoError(t, err, "Failed to make request to performance metrics endpoint")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK status")

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err, "Failed to decode response")

	// Verify response contains performance-related fields
	// Note: Exact fields depend on implementation, but should contain some performance data
	assert.NotEmpty(t, response, "Expected non-empty performance metrics response")

	// Check for common performance metrics
	performanceFields := []string{"requests_per_second", "average_response_time", "error_rate", "uptime"}
	for _, field := range performanceFields {
		if _, exists := response[field]; exists {
			// If field exists, verify it's numeric
			value, ok := response[field].(float64)
			if ok {
				assert.GreaterOrEqual(t, value, 0.0, "Expected %s to be non-negative", field)
			}
		}
	}
}

// TestGetHealthStatusEndpoint tests the /telemetry/health endpoint
func (suite *TelemetryEndpointsTestSuite) TestGetHealthStatusEndpoint(t *testing.T) {
	url := fmt.Sprintf("%s/telemetry/health", baseURL)

	resp, err := suite.client.Get(url)
	require.NoError(t, err, "Failed to make request to telemetry health endpoint")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK status")

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err, "Failed to decode response")

	// Verify response structure
	assert.Contains(t, response, "status", "Expected status field")
	assert.Contains(t, response, "uptime", "Expected uptime field")
	assert.Contains(t, response, "response_time", "Expected response_time field")

	// Verify status
	assert.Equal(t, "healthy", response["status"], "Expected status to be 'healthy'")

	// Verify uptime is a string
	uptime, ok := response["uptime"].(string)
	assert.True(t, ok, "Expected uptime to be a string")
	assert.NotEmpty(t, uptime, "Expected non-empty uptime")
}

// TestGetSwitchListEndpoint tests the /telemetry/switches endpoint
func (suite *TelemetryEndpointsTestSuite) TestGetSwitchListEndpoint(t *testing.T) {
	url := fmt.Sprintf("%s/telemetry/switches", baseURL)

	resp, err := suite.client.Get(url)
	require.NoError(t, err, "Failed to make request to switch list endpoint")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK status")

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err, "Failed to decode response")

	// Verify response structure
	assert.Contains(t, response, "data", "Expected data field")
	assert.Contains(t, response, "success", "Expected success field")
	assert.Contains(t, response, "timestamp", "Expected timestamp field")

	// Get data object
	data, ok := response["data"].(map[string]interface{})
	assert.True(t, ok, "Expected data to be an object")
	assert.Contains(t, data, "switches", "Expected switches field in data")
	assert.Contains(t, data, "count", "Expected count field in data")

	// Verify count is numeric and positive
	count, ok := data["count"].(float64)
	assert.True(t, ok, "Expected count to be numeric")
	assert.Greater(t, count, 0.0, "Expected count to be positive")

	// Verify switches array
	switches, ok := data["switches"].([]interface{})
	assert.True(t, ok, "Expected switches to be an array")
	assert.Len(t, switches, int(count), "Expected switches array length to match count")

	// Verify each switch has the expected structure
	for i, switchObj := range switches {
		switchMap, ok := switchObj.(map[string]interface{})
		assert.True(t, ok, "Expected switch %d to be an object", i)
		assert.Contains(t, switchMap, "id", "Expected switch %d to have id field", i)
		assert.Contains(t, switchMap, "name", "Expected switch %d to have name field", i)
		assert.Contains(t, switchMap, "created", "Expected switch %d to have created field", i)
	}
}

// TestGetMetricTypesEndpoint tests the /telemetry/metric-types endpoint
func (suite *TelemetryEndpointsTestSuite) TestGetMetricTypesEndpoint(t *testing.T) {
	url := fmt.Sprintf("%s/telemetry/metric-types", baseURL)

	resp, err := suite.client.Get(url)
	require.NoError(t, err, "Failed to make request to metric types endpoint")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK status")

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err, "Failed to decode response")

	// Verify response structure
	assert.Contains(t, response, "data", "Expected data field")
	assert.Contains(t, response, "success", "Expected success field")
	assert.Contains(t, response, "timestamp", "Expected timestamp field")

	// Get data object
	data, ok := response["data"].(map[string]interface{})
	assert.True(t, ok, "Expected data to be an object")
	assert.Contains(t, data, "metric_types", "Expected metric_types field in data")
	assert.Contains(t, data, "count", "Expected count field in data")

	// Verify count is numeric and positive
	count, ok := data["count"].(float64)
	assert.True(t, ok, "Expected count to be numeric")
	assert.Greater(t, count, 0.0, "Expected count to be positive")

	// Verify metric_types array
	metricTypes, ok := data["metric_types"].([]interface{})
	assert.True(t, ok, "Expected metric_types to be an array")
	assert.Len(t, metricTypes, int(count), "Expected metric_types array length to match count")

	// Verify expected metric types are present
	expectedTypes := []string{"bandwidth_mbps", "latency_ms", "packet_errors", "utilization_pct", "temperature_c"}
	for _, expectedType := range expectedTypes {
		found := false
		for _, metricType := range metricTypes {
			if metricType.(string) == expectedType {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected metric type %s to be present", expectedType)
	}
}

// TestTelemetryEndpointsResponseTime tests that all telemetry endpoints respond within acceptable time
func (suite *TelemetryEndpointsTestSuite) TestTelemetryEndpointsResponseTime(t *testing.T) {
	endpoints := []string{
		"/telemetry/metrics",
		"/telemetry/performance",
		"/telemetry/health",
		"/telemetry/switches",
		"/telemetry/metric-types",
	}

	maxResponseTime := 5 * time.Second

	for _, endpoint := range endpoints {
		t.Run(fmt.Sprintf("ResponseTime_%s", endpoint), func(t *testing.T) {
			url := fmt.Sprintf("%s%s", baseURL, endpoint)

			start := time.Now()
			resp, err := suite.client.Get(url)
			duration := time.Since(start)

			require.NoError(t, err, "Failed to make request to %s", endpoint)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK status for %s", endpoint)
			assert.Less(t, duration, maxResponseTime, "Response time for %s exceeded %v", endpoint, maxResponseTime)

			t.Logf("Endpoint %s responded in %v", endpoint, duration)
		})
	}
}

// TestInvalidMetricType tests error handling for invalid metric types
func (suite *TelemetryEndpointsTestSuite) TestInvalidMetricType(t *testing.T) {
	switches, err := suite.getSwitchList()
	require.NoError(t, err, "Failed to get switch list")
	require.NotEmpty(t, switches, "No switches available for testing")

	switchID := switches[0]
	invalidMetricType := "invalid_metric"
	url := fmt.Sprintf("%s/telemetry/metrics/%s/%s", baseURL, switchID, invalidMetricType)

	resp, err := suite.client.Get(url)
	require.NoError(t, err, "Failed to make request to invalid metric endpoint")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Expected 400 Bad Request status")

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err, "Failed to decode error response")

	assert.Contains(t, response, "error", "Expected error field in response")
	assert.Contains(t, response["error"], "invalid metric type", "Expected error message about invalid metric type")
}

// TestInvalidSwitchId tests error handling for invalid switch IDs
func (suite *TelemetryEndpointsTestSuite) TestInvalidSwitchId(t *testing.T) {
	invalidSwitchID := "invalid_switch_12345"
	url := fmt.Sprintf("%s/telemetry/metrics/%s/bandwidth", baseURL, invalidSwitchID)

	resp, err := suite.client.Get(url)
	require.NoError(t, err, "Failed to make request to invalid switch endpoint")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Expected 400 Bad Request status")

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err, "Failed to decode error response")

	assert.Contains(t, response, "error", "Expected error field in response")
}

// getSwitchList is a helper method to get the list of available switches
func (suite *TelemetryEndpointsTestSuite) getSwitchList() ([]string, error) {
	url := fmt.Sprintf("%s/telemetry/switches", baseURL)

	resp, err := suite.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	data, ok := response["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("data field is not an object")
	}

	switchesInterface, ok := data["switches"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("switches field is not an array")
	}

	var switches []string
	for _, switchInterface := range switchesInterface {
		if switchMap, ok := switchInterface.(map[string]interface{}); ok {
			if switchID, ok := switchMap["id"].(string); ok {
				switches = append(switches, switchID)
			}
		}
	}

	return switches, nil
}

// RunTelemetryEndpointsTests runs all telemetry endpoint tests
func RunTelemetryEndpointsTests(t *testing.T) {
	suite := NewTelemetryEndpointsTestSuite()

	t.Run("GetMetric", suite.TestGetMetricEndpoint)
	t.Run("ListMetrics", suite.TestListMetricsEndpoint)
	t.Run("ListAllMetrics", suite.TestListAllMetricsEndpoint)
	t.Run("ListMetricsWithFilter", suite.TestListMetricsWithFilter)
	t.Run("GetPerformanceMetrics", suite.TestGetPerformanceMetricsEndpoint)
	t.Run("GetHealthStatus", suite.TestGetHealthStatusEndpoint)
	t.Run("GetSwitchList", suite.TestGetSwitchListEndpoint)
	t.Run("GetMetricTypes", suite.TestGetMetricTypesEndpoint)
	t.Run("ResponseTime", suite.TestTelemetryEndpointsResponseTime)
	t.Run("InvalidMetricType", suite.TestInvalidMetricType)
	t.Run("InvalidSwitchId", suite.TestInvalidSwitchId)
}

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	baseURL = "http://localhost:8080"
	timeout = 30 * time.Second
)

// SystemEndpointsTestSuite contains tests for system-related endpoints
type SystemEndpointsTestSuite struct {
	client *http.Client
}

// NewSystemEndpointsTestSuite creates a new test suite
func NewSystemEndpointsTestSuite() *SystemEndpointsTestSuite {
	return &SystemEndpointsTestSuite{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// TestPingEndpoint tests the /api/v1/system/ping endpoint
func (suite *SystemEndpointsTestSuite) TestPingEndpoint(t *testing.T) {
	url := fmt.Sprintf("%s/api/v1/system/ping", baseURL)

	resp, err := suite.client.Get(url)
	require.NoError(t, err, "Failed to make request to ping endpoint")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK status")

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err, "Failed to decode response")

	// Verify response structure
	assert.Equal(t, "ok", response["status"], "Expected status to be 'ok'")
	assert.Contains(t, response, "timestamp", "Expected timestamp field")
	assert.Contains(t, response, "service", "Expected service field")

	// Verify timestamp format
	timestamp, ok := response["timestamp"].(string)
	assert.True(t, ok, "Expected timestamp to be a string")
	_, err = time.Parse(time.RFC3339, timestamp)
	assert.NoError(t, err, "Expected timestamp to be in RFC3339 format")
}

// TestHealthEndpoint tests the /api/v1/system/health endpoint
func (suite *SystemEndpointsTestSuite) TestHealthEndpoint(t *testing.T) {
	url := fmt.Sprintf("%s/api/v1/system/health", baseURL)

	resp, err := suite.client.Get(url)
	require.NoError(t, err, "Failed to make request to health endpoint")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK status")

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err, "Failed to decode response")

	// Verify response structure
	assert.Equal(t, "healthy", response["status"], "Expected status to be 'healthy'")
	assert.Contains(t, response, "timestamp", "Expected timestamp field")
	assert.Contains(t, response, "service", "Expected service field")
	assert.Contains(t, response, "checks", "Expected checks field")

	// Verify checks structure
	checks, ok := response["checks"].(map[string]interface{})
	assert.True(t, ok, "Expected checks to be a map")
	assert.Equal(t, "ok", checks["database"], "Expected database check to be 'ok'")
}

// TestReadinessEndpoint tests the /api/v1/system/readiness endpoint
func (suite *SystemEndpointsTestSuite) TestReadinessEndpoint(t *testing.T) {
	url := fmt.Sprintf("%s/api/v1/system/readiness", baseURL)

	resp, err := suite.client.Get(url)
	require.NoError(t, err, "Failed to make request to readiness endpoint")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK status")

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err, "Failed to decode response")

	// Verify response structure
	assert.Equal(t, "ready", response["status"], "Expected status to be 'ready'")
	assert.Contains(t, response, "timestamp", "Expected timestamp field")
	assert.Contains(t, response, "service", "Expected service field")
}

// TestVersionEndpoint tests the /api/v1/system/version endpoint
func (suite *SystemEndpointsTestSuite) TestVersionEndpoint(t *testing.T) {
	url := fmt.Sprintf("%s/api/v1/system/version", baseURL)

	resp, err := suite.client.Get(url)
	require.NoError(t, err, "Failed to make request to version endpoint")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK status")

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err, "Failed to decode response")

	// Verify response structure
	assert.Contains(t, response, "service", "Expected service field")
	assert.Contains(t, response, "version", "Expected version field")
	assert.Contains(t, response, "build", "Expected build field")
	assert.Contains(t, response, "commit", "Expected commit field")

	// Verify service name
	assert.Equal(t, "Unified Fabric Manager Service", response["service"], "Expected correct service name")
}

// TestSystemEndpointsResponseTime tests that all system endpoints respond within acceptable time
func (suite *SystemEndpointsTestSuite) TestSystemEndpointsResponseTime(t *testing.T) {
	endpoints := []string{
		"/api/v1/system/ping",
		"/api/v1/system/health",
		"/api/v1/system/readiness",
		"/api/v1/system/version",
	}

	maxResponseTime := 2 * time.Second

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

// TestSystemEndpointsConcurrent tests concurrent access to system endpoints
func (suite *SystemEndpointsTestSuite) TestSystemEndpointsConcurrent(t *testing.T) {
	endpoints := []string{
		"/api/v1/system/ping",
		"/api/v1/system/health",
		"/api/v1/system/readiness",
		"/api/v1/system/version",
	}

	concurrency := 10
	results := make(chan error, len(endpoints)*concurrency)

	for _, endpoint := range endpoints {
		for i := 0; i < concurrency; i++ {
			go func(ep string) {
				url := fmt.Sprintf("%s%s", baseURL, ep)
				resp, err := suite.client.Get(url)
				if err != nil {
					results <- fmt.Errorf("failed to make request to %s: %v", ep, err)
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					results <- fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, ep)
					return
				}

				results <- nil
			}(endpoint)
		}
	}

	// Collect results
	for i := 0; i < len(endpoints)*concurrency; i++ {
		err := <-results
		assert.NoError(t, err, "Concurrent request failed")
	}
}

// RunSystemEndpointsTests runs all system endpoint tests
func RunSystemEndpointsTests(t *testing.T) {
	suite := NewSystemEndpointsTestSuite()

	t.Run("Ping", suite.TestPingEndpoint)
	t.Run("Health", suite.TestHealthEndpoint)
	t.Run("Readiness", suite.TestReadinessEndpoint)
	t.Run("Version", suite.TestVersionEndpoint)
	t.Run("ResponseTime", suite.TestSystemEndpointsResponseTime)
	t.Run("Concurrent", suite.TestSystemEndpointsConcurrent)
}

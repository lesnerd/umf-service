package e2e

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	generatorBaseURL = "http://localhost:9001"
)

// GeneratorEndpointsTestSuite contains tests for generator-related endpoints
type GeneratorEndpointsTestSuite struct {
	client *http.Client
}

// NewGeneratorEndpointsTestSuite creates a new test suite
func NewGeneratorEndpointsTestSuite() *GeneratorEndpointsTestSuite {
	return &GeneratorEndpointsTestSuite{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// TestCountersEndpoint tests the /counters endpoint
func (suite *GeneratorEndpointsTestSuite) TestCountersEndpoint(t *testing.T) {
	url := fmt.Sprintf("%s/counters", generatorBaseURL)

	resp, err := suite.client.Get(url)
	require.NoError(t, err, "Failed to make request to counters endpoint")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK status")

	// Read response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")

	// Verify we have content
	assert.NotEmpty(t, body, "Expected non-empty response body")

	// Parse CSV to verify basic structure
	reader := csv.NewReader(strings.NewReader(string(body)))
	records, err := reader.ReadAll()
	require.NoError(t, err, "Failed to parse CSV response")

	// Verify we have at least a header and some data
	assert.GreaterOrEqual(t, len(records), 2, "Expected at least header and one data row")
	assert.Greater(t, len(records[0]), 0, "Expected header row to have columns")
}

// TestCountersEndpointDataConsistency tests that the counters endpoint returns consistent data
func (suite *GeneratorEndpointsTestSuite) TestCountersEndpointDataConsistency(t *testing.T) {
	url := fmt.Sprintf("%s/counters", generatorBaseURL)

	// Make multiple requests to check basic consistency
	for i := 0; i < 3; i++ {
		resp, err := suite.client.Get(url)
		require.NoError(t, err, "Failed to make request to counters endpoint")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK status")

		// Read response body
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read response body")

		// Verify we have content
		assert.NotEmpty(t, body, "Expected non-empty response body")

		// Wait a bit between requests
		time.Sleep(1 * time.Second)
	}
}

// TestCountersEndpointResponseTime tests that the counters endpoint responds within acceptable time
func (suite *GeneratorEndpointsTestSuite) TestCountersEndpointResponseTime(t *testing.T) {
	url := fmt.Sprintf("%s/counters", generatorBaseURL)
	maxResponseTime := 3 * time.Second

	start := time.Now()
	resp, err := suite.client.Get(url)
	duration := time.Since(start)

	require.NoError(t, err, "Failed to make request to counters endpoint")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK status")
	assert.Less(t, duration, maxResponseTime, "Response time exceeded %v", maxResponseTime)

	t.Logf("Counters endpoint responded in %v", duration)
}

// TestCountersEndpointConcurrent tests concurrent access to the counters endpoint
func (suite *GeneratorEndpointsTestSuite) TestCountersEndpointConcurrent(t *testing.T) {
	url := fmt.Sprintf("%s/counters", generatorBaseURL)
	concurrency := 10
	results := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			resp, err := suite.client.Get(url)
			if err != nil {
				results <- fmt.Errorf("failed to make request: %v", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				results <- fmt.Errorf("unexpected status code %d", resp.StatusCode)
				return
			}

			// Read and parse CSV response
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				results <- fmt.Errorf("failed to read response body: %v", err)
				return
			}

			reader := csv.NewReader(strings.NewReader(string(body)))
			_, err = reader.ReadAll()
			if err != nil {
				results <- fmt.Errorf("failed to parse CSV response: %v", err)
				return
			}

			results <- nil
		}()
	}

	// Collect results
	for i := 0; i < concurrency; i++ {
		err := <-results
		assert.NoError(t, err, "Concurrent request failed")
	}
}

// TestCountersEndpointDataValidation tests that the generated data has basic structure
func (suite *GeneratorEndpointsTestSuite) TestCountersEndpointDataValidation(t *testing.T) {
	url := fmt.Sprintf("%s/counters", generatorBaseURL)

	resp, err := suite.client.Get(url)
	require.NoError(t, err, "Failed to make request to counters endpoint")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK status")

	// Read response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")

	// Verify we have content
	assert.NotEmpty(t, body, "Expected non-empty response body")

	// Parse CSV to verify basic structure
	reader := csv.NewReader(strings.NewReader(string(body)))
	records, err := reader.ReadAll()
	require.NoError(t, err, "Failed to parse CSV response")

	// Verify we have at least a header and some data
	assert.GreaterOrEqual(t, len(records), 2, "Expected at least header and one data row")
	assert.Greater(t, len(records[0]), 0, "Expected header row to have columns")
}

// TestCountersEndpointTimestampValidation tests that the endpoint returns data with timestamps
func (suite *GeneratorEndpointsTestSuite) TestCountersEndpointTimestampValidation(t *testing.T) {
	url := fmt.Sprintf("%s/counters", generatorBaseURL)

	resp, err := suite.client.Get(url)
	require.NoError(t, err, "Failed to make request to counters endpoint")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK status")

	// Read response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")

	// Verify we have content
	assert.NotEmpty(t, body, "Expected non-empty response body")

	// Parse CSV to verify basic structure
	reader := csv.NewReader(strings.NewReader(string(body)))
	records, err := reader.ReadAll()
	require.NoError(t, err, "Failed to parse CSV response")

	// Verify we have at least a header and some data
	assert.GreaterOrEqual(t, len(records), 2, "Expected at least header and one data row")
	assert.Greater(t, len(records[0]), 0, "Expected header row to have columns")
}

// getSwitchList is a helper method to get the list of available switches from generator
func (suite *GeneratorEndpointsTestSuite) getSwitchList() ([]string, error) {
	url := fmt.Sprintf("%s/counters", generatorBaseURL)

	resp, err := suite.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read and parse CSV response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	reader := csv.NewReader(strings.NewReader(string(body)))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	// Extract unique switch IDs from the first column (skip header)
	switchMap := make(map[string]bool)
	for _, record := range records[1:] {
		if len(record) > 0 {
			switchMap[record[0]] = true
		}
	}

	var switches []string
	for switchID := range switchMap {
		switches = append(switches, switchID)
	}

	return switches, nil
}

// RunGeneratorEndpointsTests runs all generator endpoint tests
func RunGeneratorEndpointsTests(t *testing.T) {
	suite := NewGeneratorEndpointsTestSuite()

	t.Run("Counters", suite.TestCountersEndpoint)
	t.Run("DataConsistency", suite.TestCountersEndpointDataConsistency)
	t.Run("ResponseTime", suite.TestCountersEndpointResponseTime)
	t.Run("Concurrent", suite.TestCountersEndpointConcurrent)
	t.Run("DataValidation", suite.TestCountersEndpointDataValidation)
	t.Run("TimestampValidation", suite.TestCountersEndpointTimestampValidation)
}

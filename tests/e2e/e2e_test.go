package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestE2EFullWorkflow tests the complete end-to-end workflow
func TestE2EFullWorkflow(t *testing.T) {
	// This test ensures all components work together
	t.Run("SystemEndpoints", RunSystemEndpointsTests)
	t.Run("TelemetryEndpoints", RunTelemetryEndpointsTests)
	t.Run("GeneratorEndpoints", RunGeneratorEndpointsTests)
}

// TestE2ESystemEndpoints runs only system endpoint tests
func TestE2ESystemEndpoints(t *testing.T) {
	RunSystemEndpointsTests(t)
}

// TestE2ETelemetryEndpoints runs only telemetry endpoint tests
func TestE2ETelemetryEndpoints(t *testing.T) {
	RunTelemetryEndpointsTests(t)
}

// TestE2EGeneratorEndpoints runs only generator endpoint tests
func TestE2EGeneratorEndpoints(t *testing.T) {
	RunGeneratorEndpointsTests(t)
}

// TestE2EPerformance tests performance characteristics of the system
func TestE2EPerformance(t *testing.T) {
	t.Run("SystemEndpointsPerformance", func(t *testing.T) {
		suite := NewSystemEndpointsTestSuite()
		suite.TestSystemEndpointsResponseTime(t)
		suite.TestSystemEndpointsConcurrent(t)
	})

	t.Run("TelemetryEndpointsPerformance", func(t *testing.T) {
		suite := NewTelemetryEndpointsTestSuite()
		suite.TestTelemetryEndpointsResponseTime(t)
	})

	t.Run("GeneratorEndpointsPerformance", func(t *testing.T) {
		suite := NewGeneratorEndpointsTestSuite()
		suite.TestCountersEndpointResponseTime(t)
		suite.TestCountersEndpointConcurrent(t)
	})
}

// TestE2EErrorHandling tests error handling across all endpoints
func TestE2EErrorHandling(t *testing.T) {
	t.Run("TelemetryErrorHandling", func(t *testing.T) {
		suite := NewTelemetryEndpointsTestSuite()
		suite.TestInvalidMetricType(t)
		suite.TestInvalidSwitchId(t)
	})
}

// TestE2EDataValidation tests data validation across all endpoints
func TestE2EDataValidation(t *testing.T) {
	t.Run("GeneratorDataValidation", func(t *testing.T) {
		suite := NewGeneratorEndpointsTestSuite()
		suite.TestCountersEndpointDataValidation(t)
		suite.TestCountersEndpointTimestampValidation(t)
		suite.TestCountersEndpointDataConsistency(t)
	})
}

// TestE2EIntegration tests integration between different components
func TestE2EIntegration(t *testing.T) {
	t.Run("GeneratorToTelemetryIntegration", func(t *testing.T) {
		// Test that data from generator is accessible through telemetry endpoints
		generatorSuite := NewGeneratorEndpointsTestSuite()
		telemetrySuite := NewTelemetryEndpointsTestSuite()

		// Get switches from generator
		generatorSwitches, err := generatorSuite.getSwitchList()
		assert.NoError(t, err, "Should be able to get switches from generator")
		assert.NotEmpty(t, generatorSwitches, "Should have switches from generator")

		// Get switches from telemetry
		telemetrySwitches, err := telemetrySuite.getSwitchList()
		assert.NoError(t, err, "Should be able to get switches from telemetry")
		assert.NotEmpty(t, telemetrySwitches, "Should have switches from telemetry")

		// Verify that both services have the same switches
		assert.ElementsMatch(t, generatorSwitches, telemetrySwitches,
			"Generator and telemetry should have the same switches")
	})
}

// TestE2EReliability tests system reliability under various conditions
func TestE2EReliability(t *testing.T) {
	t.Run("ConcurrentAccess", func(t *testing.T) {
		// Test concurrent access to all endpoints
		concurrency := 5
		results := make(chan error, concurrency*3) // 3 test suites

		// Run system endpoints concurrently
		for i := 0; i < concurrency; i++ {
			go func() {
				suite := NewSystemEndpointsTestSuite()
				suite.TestSystemEndpointsConcurrent(t)
				results <- nil
			}()
		}

		// Run telemetry endpoints concurrently
		for i := 0; i < concurrency; i++ {
			go func() {
				suite := NewTelemetryEndpointsTestSuite()
				suite.TestTelemetryEndpointsResponseTime(t)
				results <- nil
			}()
		}

		// Run generator endpoints concurrently
		for i := 0; i < concurrency; i++ {
			go func() {
				suite := NewGeneratorEndpointsTestSuite()
				suite.TestCountersEndpointConcurrent(t)
				results <- nil
			}()
		}

		// Collect results
		for i := 0; i < concurrency*3; i++ {
			err := <-results
			assert.NoError(t, err, "Concurrent test failed")
		}
	})

	t.Run("RepeatedAccess", func(t *testing.T) {
		// Test repeated access to endpoints
		iterations := 10

		for i := 0; i < iterations; i++ {
			t.Run("SystemEndpoints", func(t *testing.T) {
				suite := NewSystemEndpointsTestSuite()
				suite.TestPingEndpoint(t)
				suite.TestHealthEndpoint(t)
			})

			t.Run("TelemetryEndpoints", func(t *testing.T) {
				suite := NewTelemetryEndpointsTestSuite()
				suite.TestGetSwitchListEndpoint(t)
				suite.TestGetMetricTypesEndpoint(t)
			})

			t.Run("GeneratorEndpoints", func(t *testing.T) {
				suite := NewGeneratorEndpointsTestSuite()
				suite.TestCountersEndpoint(t)
			})

			// Small delay between iterations
			time.Sleep(100 * time.Millisecond)
		}
	})
}

// TestE2ESmoke runs a minimal set of tests to verify basic functionality
func TestE2ESmoke(t *testing.T) {
	t.Run("BasicConnectivity", func(t *testing.T) {
		// Test basic connectivity to all services
		suite := NewSystemEndpointsTestSuite()
		suite.TestPingEndpoint(t)
		suite.TestHealthEndpoint(t)
	})

	t.Run("BasicDataAccess", func(t *testing.T) {
		// Test basic data access
		telemetrySuite := NewTelemetryEndpointsTestSuite()
		telemetrySuite.TestGetSwitchListEndpoint(t)
		telemetrySuite.TestGetMetricTypesEndpoint(t)

		generatorSuite := NewGeneratorEndpointsTestSuite()
		generatorSuite.TestCountersEndpoint(t)
	})
}

// TestE2EStress runs stress tests on the system
func TestE2EStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress tests in short mode")
	}

	t.Run("HighConcurrency", func(t *testing.T) {
		concurrency := 50
		results := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			go func() {
				suite := NewSystemEndpointsTestSuite()
				suite.TestPingEndpoint(t)
				results <- nil
			}()
		}

		for i := 0; i < concurrency; i++ {
			err := <-results
			assert.NoError(t, err, "High concurrency test failed")
		}
	})

	t.Run("RapidRequests", func(t *testing.T) {
		requests := 100
		results := make(chan error, requests)

		for i := 0; i < requests; i++ {
			go func() {
				suite := NewTelemetryEndpointsTestSuite()
				suite.TestGetSwitchListEndpoint(t)
				results <- nil
			}()
		}

		for i := 0; i < requests; i++ {
			err := <-results
			assert.NoError(t, err, "Rapid requests test failed")
		}
	})
}

// TestE2EMonitoring tests monitoring and observability endpoints
func TestE2EMonitoring(t *testing.T) {
	t.Run("HealthChecks", func(t *testing.T) {
		suite := NewSystemEndpointsTestSuite()
		suite.TestHealthEndpoint(t)
		suite.TestReadinessEndpoint(t)
	})

	t.Run("PerformanceMetrics", func(t *testing.T) {
		suite := NewTelemetryEndpointsTestSuite()
		suite.TestGetPerformanceMetricsEndpoint(t)
	})

	t.Run("TelemetryHealth", func(t *testing.T) {
		suite := NewTelemetryEndpointsTestSuite()
		suite.TestGetHealthStatusEndpoint(t)
	})
}

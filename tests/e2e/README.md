# UFM Service E2E Tests

This directory contains end-to-end tests for the UFM Service that test the complete system in a real environment using Docker containers.

## Overview

The e2e test suite validates the complete UFM Service ecosystem including:
- **System Endpoints**: Health checks, ping, readiness, version
- **Telemetry Endpoints**: Metrics, performance, switches, health
- **Generator Endpoints**: Data generation and consistency
- **Integration**: Cross-service communication and data flow
- **Performance**: Response times and concurrent access
- **Reliability**: Error handling and stress testing

## Prerequisites

- Docker and Docker Compose installed
- Go 1.19+ installed
- Network access to pull Docker images

## Quick Start

### Run All E2E Tests
```bash
make e2e-test
```

This will:
1. Start the complete UFM Service environment using Docker Compose
2. Wait for all services to be healthy and responding
3. Run all e2e tests
4. Stop and clean up the environment
5. Report test results

### Manual Testing

If you want to run tests manually:

1. **Start the environment**:
   ```bash
   ./scripts/e2e/start_e2e.sh
   ```

2. **Run specific test suites**:
   ```bash
   # Run all e2e tests
   go test -v ./tests/e2e/...
   
   # Run only system endpoint tests
   go test -v -run TestE2ESystemEndpoints ./tests/e2e/...
   
   # Run only telemetry endpoint tests
   go test -v -run TestE2ETelemetryEndpoints ./tests/e2e/...
   
   # Run only generator endpoint tests
   go test -v -run TestE2EGeneratorEndpoints ./tests/e2e/...
   
   # Run smoke tests only
   go test -v -run TestE2ESmoke ./tests/e2e/...
   
   # Run performance tests
   go test -v -run TestE2EPerformance ./tests/e2e/...
   
   # Run stress tests (skipped in short mode)
   go test -v -run TestE2EStress ./tests/e2e/...
   ```

3. **Stop the environment**:
   ```bash
   ./scripts/e2e/stop_e2e.sh
   ```

## Test Categories

### System Endpoints (`e2e_system_endpoints_test.go`)
Tests the basic system functionality:
- `GET /api/v1/system/ping` - Basic connectivity
- `GET /api/v1/system/health` - Health status
- `GET /api/v1/system/readiness` - Readiness status
- `GET /api/v1/system/version` - Version information

### Telemetry Endpoints (`e2e_telemetry_endpoints_test.go`)
Tests the telemetry data access:
- `GET /telemetry/metrics/:switchId/:metricType` - Get specific metric
- `GET /telemetry/metrics/:switchId` - Get all metrics for a switch
- `GET /telemetry/metrics` - Get all metrics (with optional filtering)
- `GET /telemetry/performance` - Performance metrics
- `GET /telemetry/health` - Telemetry health status
- `GET /telemetry/switches` - List of available switches
- `GET /telemetry/metric-types` - Available metric types

### Generator Endpoints (`e2e_generator_endpoints_test.go`)
Tests the data generator service:
- `GET /counters` - Generated telemetry data
- Data consistency and validation
- Response time and concurrent access

### Integration Tests (`e2e_test.go`)
Tests that verify the complete system works together:
- Cross-service data consistency
- End-to-end data flow
- System reliability under load

## Test Types

### Smoke Tests
Quick tests to verify basic functionality:
```bash
go test -v -run TestE2ESmoke ./tests/e2e/...
```

### Performance Tests
Tests response times and concurrent access:
```bash
go test -v -run TestE2EPerformance ./tests/e2e/...
```

### Stress Tests
High-load tests (skipped in short mode):
```bash
go test -v -run TestE2EStress ./tests/e2e/...
```

### Error Handling Tests
Tests error conditions and edge cases:
```bash
go test -v -run TestE2EErrorHandling ./tests/e2e/...
```

## Configuration

### Environment Variables
The e2e tests use the following configuration:
- `baseURL`: `http://localhost:8080` (main API)
- `generatorBaseURL`: `http://localhost:9001` (generator API)
- `timeout`: `30s` (HTTP client timeout)

### Docker Services
The e2e environment includes:
- **postgres**: PostgreSQL database
- **ufm-generator**: Telemetry data generator
- **ufm-server**: Main UFM service
- **jaeger**: Distributed tracing
- **prometheus**: Metrics collection

## Troubleshooting

### Services Not Starting
If services fail to start:
1. Check Docker is running
2. Check ports 8080, 9001, 5432 are available
3. Check Docker Compose logs:
   ```bash
   docker-compose logs
   ```

### Tests Failing
If tests are failing:
1. Check service health:
   ```bash
   curl http://localhost:8080/api/v1/system/health
   curl http://localhost:9001/counters
   ```
2. Check service logs:
   ```bash
   docker-compose logs ufm-server
   docker-compose logs ufm-generator
   ```

### Cleanup
To completely clean up:
```bash
./scripts/e2e/stop_e2e.sh --clean --prune
```

## Adding New Tests

### Adding New Endpoint Tests
1. Create a new test file or add to existing test suite
2. Follow the existing pattern using test suites
3. Add the test to the appropriate runner function
4. Update this README if needed

### Test Structure
```go
func (suite *TestSuite) TestNewEndpoint(t *testing.T) {
    url := fmt.Sprintf("%s/new/endpoint", baseURL)
    
    resp, err := suite.client.Get(url)
    require.NoError(t, err)
    defer resp.Body.Close()
    
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    
    var response map[string]interface{}
    err = json.NewDecoder(resp.Body).Decode(&response)
    require.NoError(t, err)
    
    // Add assertions for response structure and content
}
```

## Continuous Integration

The e2e tests are designed to run in CI environments:
- Use `make e2e-test` for automated testing
- Tests include proper cleanup and error handling
- Exit codes are properly propagated
- Logs are collected for debugging

## Performance Benchmarks

The e2e tests include performance benchmarks:
- Response time thresholds (2s for system, 5s for telemetry)
- Concurrent access testing (10-50 concurrent requests)
- Stress testing (100+ rapid requests)

These help ensure the system performs well under various load conditions. 
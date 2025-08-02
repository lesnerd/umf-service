# UFM Telemetry System

A high-performance network telemetry aggregation system inspired by NVIDIA's UFM, designed to collect, store, and serve fabric switch metrics in real-time.

## Architecture Overview

The UFM Telemetry System consists of:

1. **Data Generator Server** (Port 9001): Simulates telemetry data and serves CSV endpoints
2. **Metrics API Server** (Port 8080): Serves telemetry data via REST endpoints  
3. **Hybrid Storage**: Combines in-memory cache with PostgreSQL persistence
4. **Real-time Simulation**: Generates realistic switch metrics with variations and spikes

## System Components

### Core Components

- **In-Memory Cache**: Sub-millisecond API responses with thread-safe operations
- **PostgreSQL Repository**: Persistent storage with bulk insert optimization
- **Hybrid Store**: Write-through cache with background database sync
- **Data Simulator**: Realistic telemetry generation with configurable parameters
- **Performance Monitoring**: Built-in observability and metrics collection

### Telemetry Metrics

Each switch provides the following metrics:
- **Bandwidth**: Network throughput in Mbps
- **Latency**: Network latency in milliseconds  
- **Packet Errors**: Error count over time
- **Utilization**: Port utilization percentage
- **Temperature**: Switch temperature in Celsius

## Quick Start

### Prerequisites

1. **Go 1.23+**: `brew install go` or download from golang.org
2. **PostgreSQL**: For persistent storage
3. **Docker**: For easy PostgreSQL setup

### Setup

1. **Start PostgreSQL** (using Docker):
```bash
docker run -d --name postgres-ufm \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=umf_db \
  -p 5432:5432 postgres:13
```

2. **Initialize Database**:
```bash
make init-telemetry-db
```

3. **Create Configuration**:
```bash
make create-config
```

### Running the System

#### Option 1: Full Demo (Recommended)
```bash
make demo
```
This starts both servers and provides endpoint URLs.

#### Option 2: Manual Startup
```bash
# Terminal 1: Start data generator
make run-generator

# Terminal 2: Start main server  
make run-with-telemetry
```

#### Option 3: Individual Commands
```bash
# Generator server only
go run ./cmd/generator

# Main server only  
go run ./cmd/server
```

## API Endpoints

### Main Telemetry API (Port 8080)

#### Core Endpoints (as per requirements)

**GetMetric**: Fetch specific metric for a switch
```bash
GET /telemetry/metrics/{switchId}/{metricType}

# Examples:
curl http://localhost:8080/telemetry/metrics/switch-001/bandwidth_mbps
curl http://localhost:8080/telemetry/metrics/switch-002/latency_ms
```

**ListMetrics**: Fetch metrics for switch(es)
```bash
# All metrics for specific switch
GET /telemetry/metrics/{switchId}
curl http://localhost:8080/telemetry/metrics/switch-001

# All metrics for all switches  
GET /telemetry/metrics
curl http://localhost:8080/telemetry/metrics
```

#### Additional Endpoints

**Performance Metrics**:
```bash
GET /telemetry/performance
curl http://localhost:8080/telemetry/performance
```

**Health Status**:
```bash
GET /telemetry/health  
curl http://localhost:8080/telemetry/health
```

**Switch List**:
```bash
GET /telemetry/switches
curl http://localhost:8080/telemetry/switches
```

### Generator Server (Port 9001)

**CSV Data Export** (as per requirements):
```bash
GET /counters
curl http://localhost:9001/counters
```

**Status Information**:
```bash
GET /status
curl http://localhost:9001/status

GET /health
curl http://localhost:9001/health
```

## Configuration

Configuration is managed through `system.yaml`:

```yaml
telemetry:
  enabled: true
  generator:
    port: "9001"
    switch_count: 10
    update_interval: "10s"
    enable_spikes: true
    enable_errors: true
    variation_pct: 0.2
  storage:
    cache_ttl: "5m"
    batch_size: 100
    flush_interval: "30s"
    max_retries: 3
  simulator:
    switch_count: 10
    update_interval: "10s"
    enable_spikes: true
    enable_errors: true
```

### Environment Variables

Override configuration with environment variables:
- `TELEMETRY_ENABLED=true`
- `TELEMETRY_GENERATOR_PORT=9001`
- `DATABASE_URL=postgres://user:pass@host:port/db`

## Performance Characteristics

### Expected Performance

- **API Response Time**: < 1ms for GetMetric calls
- **Concurrent Requests**: 1000+ RPS without degradation
- **Memory Usage**: ~100KB per switch with full metrics
- **Update Frequency**: 10-second intervals for 100+ switches
- **Cache Hit Rate**: 95%+ for recent data queries

### Performance Monitoring

The system includes comprehensive performance monitoring:

```bash
# Get performance metrics
curl http://localhost:8080/telemetry/performance

# Response includes:
{
  "success": true,
  "data": {
    "api_latency_ms": 0.8,
    "active_switches": 10,
    "total_requests": 1547,
    "memory_usage_mb": 2.1,
    "data_age_seconds": 3.2,
    "last_update": "2024-01-01T10:00:00Z"
  }
}
```

## Data Flow

1. **Simulation**: Background workers generate realistic telemetry data every 10 seconds
2. **Ingestion**: Data updates in-memory cache immediately (< 1ms)
3. **Persistence**: Background process flushes to PostgreSQL every 30 seconds
4. **API Queries**: Served from in-memory cache for sub-millisecond responses
5. **CSV Export**: Generator serves current snapshot in CSV format

## Testing

### Automated Testing
```bash
# Run unit tests
make test

# Test API endpoints
make test-telemetry-endpoints

# Test with coverage
make test-coverage
```

### Manual Testing

1. **Start the system**:
```bash
make demo
```

2. **Test core endpoints**:
```bash
# Get all metrics
curl http://localhost:8080/telemetry/metrics | jq

# Get specific switch
curl http://localhost:8080/telemetry/metrics/switch-001 | jq

# Get specific metric
curl http://localhost:8080/telemetry/metrics/switch-001/bandwidth_mbps | jq

# Get CSV data
curl http://localhost:9001/counters
```

3. **Monitor performance**:
```bash
curl http://localhost:8080/telemetry/performance | jq
```

## Development

### Project Structure

```
internal/telemetry/
├── models/          # Data structures and types
├── storage/         # Cache and database layers
├── simulator/       # Data generation logic
├── interfaces.go    # Service interfaces
└── service.go       # Business logic

cmd/
├── server/          # Main API server (port 8080)
└── generator/       # Data generator server (port 9001)
```

### Adding New Metrics

1. **Update models** in `internal/telemetry/models/models.go`
2. **Add to TelemetryData struct** with appropriate JSON/DB tags
3. **Update CSV serialization** methods
4. **Add validation** in service layer
5. **Update simulator** to generate realistic data

### Extending the API

1. **Add handler methods** in `internal/http/handler/http_handler_telemetry.go`
2. **Register routes** in `internal/http/router.go`
3. **Implement business logic** in `internal/telemetry/service.go`
4. **Add tests** for new functionality

## Troubleshooting

### Common Issues

**Database Connection Errors**:
```bash
# Check PostgreSQL is running
docker ps | grep postgres

# Test connection
psql -h localhost -p 5432 -U postgres -d umf_db -c "SELECT 1;"
```

**No Telemetry Data**:
```bash
# Check if telemetry is enabled
curl http://localhost:8080/telemetry/health

# Verify generator is running
curl http://localhost:9001/health

# Check logs for errors
tail -f ~/.ufm/logs/service.log
```

**Performance Issues**:
```bash
# Monitor performance metrics
curl http://localhost:8080/telemetry/performance

# Check cache statistics  
curl http://localhost:8080/telemetry/health | jq '.performance'
```

### Debugging

Enable debug logging:
```bash
LOG_LEVEL=debug go run ./cmd/server
```

View detailed logs:
```bash
tail -f ~/.ufm/logs/service.log | grep telemetry
```

## Production Deployment

### Docker Deployment

```dockerfile
# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o ufm-server ./cmd/server
RUN go build -o ufm-generator ./cmd/generator

# Runtime stage  
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /app/ufm-server .
COPY --from=builder /app/ufm-generator .
COPY system.yaml .

# Expose ports
EXPOSE 8080 9001

# Run servers
CMD ["./ufm-server"]
```

### Scaling Considerations

- **Horizontal Scaling**: Multiple API servers behind load balancer
- **Database Scaling**: Read replicas for query distribution
- **Cache Scaling**: Redis cluster for distributed caching
- **Data Partitioning**: Partition by switch ID or time ranges

## Security

- **Input Validation**: All API inputs validated and sanitized
- **Rate Limiting**: Built-in rate limiting for API endpoints
- **Authentication**: Integrate with existing auth systems
- **Database Security**: Parameterized queries prevent SQL injection
- **TLS**: Enable HTTPS in production environments

## License

This UFM Telemetry System is part of the UFM service and follows the same licensing terms.
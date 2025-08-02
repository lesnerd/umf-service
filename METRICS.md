# UMF Service Prometheus Metrics

This document describes the comprehensive Prometheus metrics implementation for the UMF service.

## Overview

The UMF service now includes extensive Prometheus metrics for monitoring and observability. Metrics are collected from both the main UMF server and the data generator service.

## Metrics Endpoints

### UMF Server
- **Endpoint**: `/api/v1/system/metrics`
- **Port**: 8080
- **Service**: Main UMF telemetry service

## Prometheus Configuration

The Prometheus configuration (`prometheus.yml`) is set up to scrape the UMF server:

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'ufm-server'
    static_configs:
      - targets: ['ufm-server:8080']
    metrics_path: /api/v1/system/metrics
    scrape_interval: 15s
```

## Available Metrics

### HTTP Metrics

#### Request Counters
- `http_requests_total` - Total number of HTTP requests
  - Labels: `method`, `endpoint`, `status_code`

#### Request Duration
- `http_request_duration_seconds` - HTTP request duration in seconds
  - Labels: `method`, `endpoint`

#### Request In Flight
- `http_requests_in_flight` - Current number of HTTP requests being processed

### Telemetry Service Metrics

#### Ingestion Metrics
- `telemetry_ingest_total` - Total number of telemetry data points ingested
  - Labels: `switch_id`, `metric_type`, `status`

#### Ingestion Duration
- `telemetry_ingest_duration_seconds` - Telemetry ingestion duration in seconds
  - Labels: `switch_id`, `metric_type`

#### Query Metrics
- `telemetry_query_total` - Total number of telemetry queries
  - Labels: `switch_id`, `metric_type`, `status`

#### Query Duration
- `telemetry_query_duration_seconds` - Telemetry query duration in seconds
  - Labels: `switch_id`, `metric_type`

### Database Metrics

#### Operation Counters
- `database_operations_total` - Total number of database operations
  - Labels: `operation`, `table`, `status`

#### Operation Duration
- `database_operation_duration_seconds` - Database operation duration in seconds
  - Labels: `operation`, `table`

#### Connection Metrics
- `database_connections_active` - Number of active database connections

### Queue Metrics

#### Queue State
- `queue_depth` - Current number of items in the processing queue
- `queue_worker_utilization_percent` - Queue worker utilization percentage

#### Queue Operations
- `queue_operations_total` - Total number of queue operations
  - Labels: `operation`, `status`

### System Metrics

#### Runtime Metrics
- `system_uptime_seconds` - System uptime in seconds
- `system_memory_usage_bytes` - Current memory usage in bytes
- `system_goroutines` - Current number of goroutines



### Cache Metrics

#### Cache Performance
- `cache_hits_total` - Total number of cache hits
- `cache_misses_total` - Total number of cache misses
- `cache_size` - Current number of items in cache

### Business Metrics

#### Switch Metrics
- `active_switches` - Number of active switches
- `metrics_per_switch` - Number of metrics per switch
  - Labels: `switch_id`

### Error Metrics

#### Error Tracking
- `errors_total` - Total number of errors
  - Labels: `component`, `error_type`

## Metric Types

### Counters
- `http_requests_total`
- `telemetry_ingest_total`
- `telemetry_query_total`
- `database_operations_total`
- `queue_operations_total`
- `cache_hits_total`
- `cache_misses_total`
- `errors_total`

### Gauges
- `http_requests_in_flight`
- `database_connections_active`
- `queue_depth`
- `queue_worker_utilization_percent`
- `system_uptime_seconds`
- `system_memory_usage_bytes`
- `system_goroutines`
- `cache_size`
- `active_switches`
- `metrics_per_switch`

### Histograms
- `http_request_duration_seconds`
- `telemetry_ingest_duration_seconds`
- `telemetry_query_duration_seconds`
- `database_operation_duration_seconds`

## Implementation Details

### Metrics Collection

1. **HTTP Middleware**: Automatically collects request metrics for all HTTP endpoints
2. **Service Layer**: Telemetry service methods record ingestion and query metrics
3. **Database Layer**: Repository operations are wrapped with metrics collection
4. **Cache Layer**: Cache hits/misses and size are tracked
5. **Queue Layer**: Queue depth and worker utilization are monitored
6. **System Layer**: Runtime metrics are collected periodically

### Metrics Updates

- **HTTP Metrics**: Updated on every request via middleware
- **System Metrics**: Updated every 30 seconds via background goroutine
- **Queue Metrics**: Updated every 10 seconds via background goroutine
- **Cache Metrics**: Updated on cache operations
- **Database Metrics**: Updated on database operations

## Monitoring and Alerting

### Key Metrics to Monitor

1. **Response Time**: `http_request_duration_seconds`
2. **Error Rate**: `errors_total`
3. **System Health**: `system_memory_usage_bytes`, `system_goroutines`
4. **Database Performance**: `database_operation_duration_seconds`
5. **Cache Performance**: `cache_hits_total` vs `cache_misses_total`
6. **Queue Health**: `queue_depth`, `queue_worker_utilization_percent`

### Example Queries

```promql
# Request rate
rate(http_requests_total[5m])

# Error rate
rate(errors_total[5m])

# 95th percentile response time
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# Cache hit ratio
rate(cache_hits_total[5m]) / (rate(cache_hits_total[5m]) + rate(cache_misses_total[5m]))

# Database operation rate
rate(database_operations_total[5m])

# Queue depth
queue_depth

# System memory usage
system_memory_usage_bytes
```

## Deployment

The metrics are automatically available when running the services with Docker Compose:

```bash
docker-compose up -d
```

Prometheus will be available at `http://localhost:9090` and will automatically start collecting metrics from the UMF server.

## Customization

### Adding New Metrics

1. Define the metric in `internal/monitoring/metrics/metrics.go`
2. Add metric collection in the appropriate service layer
3. Update this documentation

### Modifying Scrape Intervals

Update the `scrape_interval` in `prometheus.yml` to change how frequently Prometheus collects metrics.

### Adding New Labels

When adding new labels to existing metrics, ensure backward compatibility by providing default values for existing time series. 
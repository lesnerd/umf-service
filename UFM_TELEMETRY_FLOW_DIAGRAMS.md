# UFM Telemetry Service Flow Diagrams

## Overview

This document provides detailed flow diagrams for the UFM (Unified Fabric Manager) telemetry microservice, showing both data ingestion and REST API request flows.

---

## Diagram 1: Data Ingestion Flow (Generator → Storage)

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                            UFM TELEMETRY DATA INGESTION FLOW                        │
└─────────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐     ┌──────────────────┐
│   Generator     │     │ Generator Client │     │ Telemetry       │     │ Hybrid Storage   │
│   (Port 9001)   │     │ (Polling)        │     │ Service         │     │ (Cache + DB)     │
└─────────────────┘     └──────────────────┘     └─────────────────┘     └──────────────────┘

Step 1: Data Generation (Every 10 seconds)
┌─────────────────┐
│ DataGenerator   │
│ - Generate CSV  │ ──────────────┐
│ - Switch data   │               │
│ - Timestamps    │               │
│ - Gen ID: 1001  │               │
└─────────────────┘               │
                                  │
                ┌─────────────────▼──────────────────┐
                │ Pre-generated CSV Buffer:          │
                │ switch-001,2024-08-02T10:00:00Z,   │
                │ 450.5,2.1,0,75.2,42.3\n           │
                │ switch-002,2024-08-02T10:00:01Z,   │
                │ 380.2,1.8,1,68.9,41.1\n           │
                │ ... (1-100 messages per switch)    │
                │ X-Generation-ID: 1001              │
                └────────────────────────────────────┘

Step 2: Polling Request (Every second)
                ┌──────────────────┐
                │ HTTP GET Request │
                │ /counters        │ ─────────────────┐
                │ Timeout: 5s      │                  │
                └──────────────────┘                  │
                                                      │
┌─────────────────┐                                   │
│ Generator       │ ◄─────────────────────────────────┘
│ handleCounters()│
│ - Check ready   │ ──────────────┐
│ - Return CSV    │               │
│ - Add headers   │               │
└─────────────────┘               │
                                  │
        ┌─────────────────────────▼─────────────────────────┐
        │ HTTP Response:                                    │
        │ Status: 200 OK                                    │
        │ Content-Type: text/plain                          │
        │ X-Generation-ID: 1001                             │
        │ X-Switch-Count: 10                                │
        │ Body: CSV data (500-5000 records)                 │
        └───────────────────────────────────────────────────┘

Step 3: Deduplication Check
                ┌──────────────────┐
                │ Generator Client │
                │ Check Generation │ ──────────────┐
                │ ID: 1001        │               │
                └──────────────────┘               │
                                                   │
        ┌───────────────────────────────────────────▼──────────────────────────────────────┐
        │ Deduplication Logic:                                                              │
        │ 1. Compare X-Generation-ID with lastGenerationID (1000)                          │
        │ 2. If new (1001 > 1000): PROCESS                                                 │
        │ 3. If same (1001 == 1001): SKIP (duplicate)                                      │
        │ 4. Fallback: Compare timestamps if no generation ID                              │
        │ 5. Update lastGenerationID = 1001                                                │
        └───────────────────────────────────────────────────────────────────────────────────┘

Step 4: CSV Parsing & Validation
                ┌──────────────────┐
                │ Parse CSV Data   │ ──────────────┐
                │ Split lines      │               │
                │ Parse fields     │               │
                └──────────────────┘               │
                                                   │
        ┌───────────────────────────────────────────▼──────────────────────────────────────┐
        │ Per Record Validation:                                                            │
        │ - Switch ID not empty                                                             │
        │ - Timestamp valid RFC3339                                                         │
        │ - Bandwidth ≥ 0 Mbps                                                              │
        │ - Latency ≥ 0 ms                                                                  │
        │ - Packet errors ≥ 0                                                               │
        │ - Utilization 0-100%                                                              │
        │ - Temperature valid range                                                         │
        │ Invalid records: LOGGED & SKIPPED                                                 │
        └───────────────────────────────────────────────────────────────────────────────────┘

Step 5: Service Layer Processing
                ┌─────────────────┐
                │ IngestBatch()   │ ──────────────────────────────┐
                │ - Batch process │                               │
                │ - Performance   │                               │
                │ - Error handle  │                               │
                └─────────────────┘                               │
                                                                  │
        ┌─────────────────────────────────────────────────────────▼────────────────────────┐
        │ Batch Processing:                                                                 │
        │ 1. Group by switch ID for efficiency                                             │
        │ 2. Validate each TelemetryData record                                            │
        │ 3. Track metrics: processed_count, error_count, duration                         │
        │ 4. Log batch statistics                                                          │
        │ 5. Pass valid records to storage layer                                           │
        └──────────────────────────────────────────────────────────────────────────────────┘

Step 6: Hybrid Storage - Cache Layer (Immediate)
                ┌──────────────────┐
                │ In-Memory Cache  │ ──────────────┐
                │ (Thread-safe)    │               │
                └──────────────────┘               │
                                                   │
        ┌───────────────────────────────────────────▼──────────────────────────────────────┐
        │ Cache Operations (RWMutex protected):                                            │
        │ 1. Store latest metrics per switch ID                                            │
        │ 2. Update timestamp for each metric type                                         │
        │ 3. Increment request counters                                                    │
        │ 4. Calculate memory usage                                                        │
        │ 5. Response time: < 1ms                                                          │
        │ 6. Data structure: map[switchID]map[metricType]LatestValue                       │
        └───────────────────────────────────────────────────────────────────────────────────┘

Step 7: Background Database Queue (Asynchronous)
                ┌──────────────────┐
                │ Database Queue   │ ──────────────┐
                │ (Channel-based)  │               │
                └──────────────────┘               │
                                                   │
        ┌───────────────────────────────────────────▼──────────────────────────────────────┐
        │ Queue Processing:                                                                 │
        │ 1. Non-blocking queue writes (buffered channel)                                  │
        │ 2. Background flush worker processes queue                                       │
        │ 3. Batch size: 100 records per flush                                             │
        │ 4. Flush interval: 30 seconds                                                    │
        │ 5. Retry logic: exponential backoff on DB errors                                 │
        │ 6. Metrics: queue_depth, flush_duration, error_count                             │
        └───────────────────────────────────────────────────────────────────────────────────┘

Step 8: PostgreSQL Persistence
                ┌──────────────────┐
                │ PostgreSQL       │ ──────────────┐
                │ Repository       │               │
                └──────────────────┘               │
                                                   │
        ┌───────────────────────────────────────────▼──────────────────────────────────────┐
        │ Database Operations:                                                              │
        │ 1. Use PostgreSQL COPY command for bulk inserts                                  │
        │ 2. Connection pooling for performance                                            │
        │ 3. Batch processing: 100+ records per transaction                                │
        │ 4. Switch registration: upsert switch metadata                                   │
        │ 5. Metric storage: timestamped telemetry data                                    │
        │ 6. Cleanup worker: remove old metrics (configurable retention)                   │
        └───────────────────────────────────────────────────────────────────────────────────┘

Step 9: Background Workers
        ┌─────────────────────────────────────────────────────────────────────────────────┐
        │ Background Worker Pool:                                                         │
        │                                                                                 │
        │ ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐                 │
        │ │ Flush Worker    │  │ Cleanup Worker  │  │ Metrics Worker  │                 │
        │ │ - Process queue │  │ - Remove stale  │  │ - Report stats  │                 │
        │ │ - Batch to DB   │  │ - Free memory   │  │ - Health check  │                 │
        │ │ - Retry failed  │  │ - Every 5 min   │  │ - Every 1 min   │                 │
        │ └─────────────────┘  └─────────────────┘  └─────────────────┘                 │
        └─────────────────────────────────────────────────────────────────────────────────┘

Performance Characteristics:
┌─────────────────────────────────────────────────────────────────────────────────────┐
│ • Generator: 500-5000 records/10s (pre-generated, instant response)                │
│ • Client: 10s polling interval, <1s processing time                                │
│ • Cache: <1ms read/write, thread-safe operations                                   │
│ • Database: Bulk inserts, 100+ records/transaction, async processing              │
│ • Deduplication: Generation ID + timestamp based, 99.9% accuracy                  │
│ • Error resilience: Continues on temporary failures, comprehensive logging        │
└─────────────────────────────────────────────────────────────────────────────────────┘
```

---

## Diagram 2: REST API Flow (HTTP → Response)

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                            UFM TELEMETRY REST API FLOW                             │
└─────────────────────────────────────────────────────────────────────────────────────┘

HTTP Client ──────────────────────► HTTP Server (Port 8080) ──────────► Response

API Endpoints:
• GET /telemetry/metrics/:switchId/:metricType  (GetMetric)
• GET /telemetry/metrics/:switchId              (ListMetrics for switch)
• GET /telemetry/metrics                        (ListMetrics for all)
• GET /telemetry/metrics?metrics=bandwidth_mbps,latency_ms

┌─────────────────────────────────────────────────────────────────────────────────────┐
│                                  REQUEST FLOW                                       │
└─────────────────────────────────────────────────────────────────────────────────────┘

Step 1: HTTP Request Processing
┌─────────────────┐
│ Client Request  │
│ GET /telemetry/ │ ─────────────────┐
│ metrics/switch- │                  │
│ 001/bandwidth_  │                  │
│ mbps           │                  │
└─────────────────┘                  │
                                     │
        ┌────────────────────────────▼─────────────────────────────┐
        │ Gin HTTP Router:                                         │
        │ 1. Route matching: /telemetry/metrics/:switchId/:metric  │
        │ 2. Extract path parameters                               │
        │ 3. Parse query parameters                                │
        │ 4. Apply middleware chain                                │
        │ 5. Forward to handler                                    │
        └──────────────────────────────────────────────────────────┘

Step 2: Middleware Chain Processing
        ┌──────────────────────────────────────────────────────────┐
        │ Middleware Stack:                                        │
        │                                                          │
        │ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐         │
        │ │ CORS        │ │ Logging     │ │ Recovery    │         │
        │ │ Headers     │ │ Request ID  │ │ Panic       │         │
        │ │ Allow: *    │ │ Tracing     │ │ Handling    │         │
        │ └─────────────┘ └─────────────┘ └─────────────┘         │
        │                                                          │
        │ ┌─────────────┐ ┌─────────────┐                         │
        │ │ Telemetry   │ │ Auth        │                         │
        │ │ Tracing     │ │ (Optional)  │                         │
        │ │ Metrics     │ │ JWT/API Key │                         │
        │ └─────────────┘ └─────────────┘                         │
        └──────────────────────────────────────────────────────────┘

Step 3: Handler Routing & Validation
                ┌──────────────────┐
                │ Telemetry        │
                │ Handler          │ ─────────────────┐
                │ GetMetric()      │                  │
                │ ListMetrics()    │                  │
                └──────────────────┘                  │
                                                      │
        ┌─────────────────────────────────────────────▼────────────────────────────────┐
        │ Request Validation:                                                           │
        │                                                                               │
        │ GetMetric Request:                     ListMetrics Request:                   │
        │ 1. Validate switchId format           1. Optional switchId validation        │
        │ 2. Validate metricType enum           2. Parse metrics query parameter       │
        │ 3. Check parameter presence           3. Validate metric type filters        │
        │ 4. Set response headers               4. Set pagination headers              │
        │                                                                               │
        │ Error Cases:                                                                  │
        │ • 400 Bad Request: Invalid parameters                                         │
        │ • 404 Not Found: Switch/metric not found                                      │
        │ • 500 Internal Error: Service failures                                        │
        └───────────────────────────────────────────────────────────────────────────────┘

Step 4: Service Layer Delegation
                ┌─────────────────┐
                │ TelemetryService│ ─────────────────┐
                │ Interface       │                  │
                │ GetMetric()     │                  │
                │ GetAllMetrics() │                  │
                └─────────────────┘                  │
                                                     │
        ┌────────────────────────────────────────────▼───────────────────────────────────┐
        │ Service Operations:                                                            │
        │                                                                                │
        │ GetMetric(switchId, metricType):           GetAllMetrics(filters):             │
        │ 1. Input validation                        1. Parse metric filters            │
        │ 2. Call storage layer                     2. Call storage for all switches   │
        │ 3. Handle not found cases                 3. Apply filters if specified       │
        │ 4. Return formatted data                  4. Aggregate results                │
        │                                                                                │
        │ Performance Tracking:                                                          │
        │ • Request counters                                                             │
        │ • Response time metrics                                                        │
        │ • Error rate tracking                                                          │
        │ • Cache hit/miss ratios                                                        │
        └────────────────────────────────────────────────────────────────────────────────┘

Step 5: Storage Layer - Cache First Strategy
                ┌──────────────────┐
                │ Hybrid Storage   │ ─────────────────┐
                │ Cache First      │                  │
                └──────────────────┘                  │
                                                      │
        ┌─────────────────────────────────────────────▼────────────────────────────────┐
        │ Cache Lookup (Primary):                                                      │
        │                                                                              │
        │ ┌─────────────────┐    Hit     ┌─────────────────┐                          │
        │ │ In-Memory Cache │ ────────►  │ Return Data     │                          │
        │ │ RWMutex Lock    │            │ < 1ms Response  │                          │
        │ │ Latest Metrics  │            │ Update Counters │                          │
        │ └─────────────────┘            └─────────────────┘                          │
        │                                                                              │
        │ │                                                                            │
        │ │ Miss                                                                       │
        │ ▼                                                                            │
        │                                                                              │
        │ ┌─────────────────┐            ┌─────────────────┐                          │
        │ │ Database Query  │ ────────►  │ Cache Update    │                          │
        │ │ PostgreSQL      │            │ Store Result    │                          │
        │ │ Historical Data │            │ Return Data     │                          │
        │ └─────────────────┘            └─────────────────┘                          │
        └──────────────────────────────────────────────────────────────────────────────┘

Step 6: Database Fallback (Cache Miss)
                ┌──────────────────┐
                │ PostgreSQL       │ ─────────────────┐
                │ Repository       │                  │
                └──────────────────┘                  │
                                                      │
        ┌─────────────────────────────────────────────▼────────────────────────────────┐
        │ Database Query Operations:                                                    │
        │                                                                               │
        │ Single Metric Query:                   Multi-Metric Query:                   │
        │ SELECT bandwidth_mbps                  SELECT switch_id, metric_type,        │
        │ FROM telemetry_metrics                       metric_value, timestamp         │
        │ WHERE switch_id = 'switch-001'         FROM telemetry_metrics                │
        │   AND metric_type = 'bandwidth_mbps'   WHERE timestamp > NOW() - INTERVAL    │
        │   AND timestamp > NOW() - '1 hour'           '1 hour'                        │
        │ ORDER BY timestamp DESC                ORDER BY switch_id, timestamp DESC    │
        │ LIMIT 1;                               LIMIT 1000;                           │
        │                                                                               │
        │ Connection Management:                                                        │
        │ • Connection pooling (max 20 connections)                                    │
        │ • Query timeout: 10 seconds                                                  │
        │ • Retry logic: 3 attempts with backoff                                       │
        │ • Performance indexing on (switch_id, metric_type, timestamp)                │
        └───────────────────────────────────────────────────────────────────────────────┘

Step 7: Data Transformation & Response Formation
        ┌─────────────────────────────────────────────────────────────────────────────┐
        │ Response Data Processing:                                                   │
        │                                                                             │
        │ GetMetric Response:                    ListMetrics Response:                │
        │ {                                      {                                    │
        │   "switch_id": "switch-001",             "switches": [                       │
        │   "metric_type": "bandwidth_mbps",         {                                 │
        │   "value": 450.5,                           "switch_id": "switch-001",      │
        │   "timestamp": "2024-08-02T10:05:30Z",      "metrics": {                    │
        │   "unit": "Mbps"                              "bandwidth_mbps": 450.5,      │
        │ }                                             "latency_ms": 2.1,           │
        │                                               "packet_errors": 0,           │
        │ Error Response:                               "utilization_pct": 75.2,      │
        │ {                                             "temperature_c": 42.3         │
        │   "error": "metric not found",              },                               │
        │   "code": "METRIC_NOT_FOUND",               "timestamp": "2024-08-02T10:05:30Z" │
        │   "details": "switch-001/invalid_metric"    }                               │
        │ }                                         ],                                 │
        │                                          "total_switches": 10,              │
        │                                          "filtered": false                   │
        │                                        }                                     │
        └─────────────────────────────────────────────────────────────────────────────┘

Step 8: HTTP Response Headers & Performance Tracking
        ┌─────────────────────────────────────────────────────────────────────────────┐
        │ Response Headers:                                                           │
        │                                                                             │
        │ Standard Headers:                      Performance Headers:                 │
        │ Content-Type: application/json         X-Response-Time: 2.3ms               │
        │ Content-Length: 256                    X-Cache-Hit: true                    │
        │ Date: Thu, 02 Aug 2024 10:05:30 GMT   X-Switch-Count: 10                   │
        │                                        X-Metric-Count: 5                    │
        │ CORS Headers:                          X-Source: cache                      │
        │ Access-Control-Allow-Origin: *         X-Request-ID: req-123456789          │
        │ Access-Control-Allow-Methods: GET      X-Service-Version: 1.0.0             │
        │ Access-Control-Allow-Headers: *                                             │
        │                                                                             │
        │ Status Codes:                                                               │
        │ 200 OK: Successful data retrieval                                          │
        │ 400 Bad Request: Invalid parameters                                         │
        │ 404 Not Found: Switch/metric not found                                      │
        │ 500 Internal Server Error: Service failure                                 │
        │ 503 Service Unavailable: Cache/DB unavailable                              │
        └─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────────┐
│                              PERFORMANCE SUMMARY                                    │
└─────────────────────────────────────────────────────────────────────────────────────┘

Cache Hit Scenario:                          Cache Miss Scenario:
┌─────────────────────────────────────────┐  ┌─────────────────────────────────────────┐
│ • Total Response Time: < 5ms            │  │ • Total Response Time: 50-200ms         │
│ • Cache Lookup: < 1ms                   │  │ • Cache Lookup: < 1ms (miss)            │
│ • JSON Serialization: 1-2ms             │  │ • Database Query: 20-100ms              │
│ • HTTP Response: 1-2ms                  │  │ • Cache Update: 1-2ms                   │
│ • Throughput: 10,000+ req/sec           │  │ • JSON Serialization: 1-2ms             │
│                                         │  │ • HTTP Response: 1-2ms                  │
│ • Memory Usage: ~50MB for 10 switches   │  │ • Throughput: 100-500 req/sec           │
│ • CPU Usage: <1% per request            │  │                                         │
│                                         │  │ • Database Connection: Pooled           │
│                                         │  │ • Query Optimization: Indexed           │
└─────────────────────────────────────────┘  └─────────────────────────────────────────┘

Error Handling Flow:
┌─────────────────────────────────────────────────────────────────────────────────────┐
│ Request Error → Handler Catch → Service Error → Storage Error → Response             │
│                                                                                     │
│ • Validation errors: 400 Bad Request with detailed message                         │
│ • Not found errors: 404 Not Found with resource details                           │
│ • Service errors: 500 Internal Server Error with correlation ID                   │
│ • Timeout errors: 503 Service Unavailable with retry headers                      │
│ • All errors logged with request context and stack trace                          │
└─────────────────────────────────────────────────────────────────────────────────────┘
```

---

## Key Flow Characteristics

### Data Ingestion Flow:
- **High Performance**: 500-5000 records every 10 seconds
- **Reliability**: Deduplication, validation, and error recovery
- **Scalability**: Async processing, background workers, connection pooling
- **Monitoring**: Comprehensive metrics and health checks

### REST API Flow:
- **Low Latency**: <5ms for cached data, <200ms for database queries
- **High Throughput**: 10,000+ requests/second for cached data
- **Fault Tolerance**: Graceful degradation and error handling
- **Observability**: Request tracing, performance headers, and metrics

### Storage Strategy:
- **Cache-First**: Immediate response from in-memory cache
- **Async Persistence**: Non-blocking database writes
- **Data Consistency**: Background synchronization between cache and database
- **Performance Optimization**: Bulk database operations and connection pooling

This architecture provides a robust, scalable telemetry system capable of handling high-frequency data ingestion while serving low-latency API requests with comprehensive monitoring and error handling.
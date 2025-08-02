# UFM Service

A production-ready UFM (Unified Fabric Manager) service. This service provides a solid foundation for building scalable, maintainable microservices with comprehensive observability and security features.

## Architecture

This template follows a clean, layered architecture with dependency injection:

- **cmd/server/** - Application entry point
- **internal/app/** - Application bootstrap and lifecycle management
- **internal/config/** - Configuration management with environment variables
- **internal/service/** - Business logic and service layer
- **internal/http/** - HTTP transport layer (handlers, middleware, routing)
- **internal/log/** - Structured logging infrastructure
- **internal/monitoring/** - Metrics and distributed tracing
- **internal/utils/** - Shared utilities
- **pkg/** - Public packages (if any)

## Features

### Core Features
- **Clean Architecture**: Layered architecture with dependency injection
- **HTTP Server**: Gin-based web server with middleware support
- **Configuration Management**: Environment-driven configuration with validation
- **Graceful Shutdown**: Proper resource cleanup and shutdown handling
- **Health Checks**: Standard health, readiness, and ping endpoints

### Observability
- **Structured Logging**: JSON-formatted logs with request correlation
- **Distributed Tracing**: OpenTracing integration with Jaeger
- **Metrics**: Prometheus-compatible metrics collection
- **Request Logging**: Detailed HTTP request/response logging

### Security & Operations
- **JWT Authentication**: Token-based authentication (configurable)
- **Multi-tenant Support**: Tenant isolation patterns (optional)
- **Panic Recovery**: Graceful error handling and recovery
- **Docker Support**: Multi-stage Dockerfile with health checks

## Quick Start

### Prerequisites
- Go 1.23 or later
- Docker (for supporting services)
- Make

### Setup

1. **Clone the template**:
   ```bash
   git clone <ufm-repo> ufm
   cd ufm
   ```

2. **Install dependencies**:
   ```bash
   make prereq
   go mod tidy
   ```

3. **Configure environment**:
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

4. **Start supporting services** (optional):
   ```bash
   docker-compose up -d
   ```

5. **Run the service**:
   ```bash
   make run
   ```

The service will be available at `http://localhost:8080`

## Development

### Available Commands

```bash
# Development
make run                    # Run the service
make build                  # Build for current platform
make build-all             # Cross-platform builds

# Testing
make test                  # Run tests
make test-coverage         # Run tests with coverage

# Code Quality
make lint                  # Run linter
make format               # Format code
make generate             # Generate mocks and swagger docs

# Docker
docker-compose up -d      # Start supporting services
docker build -t ufm-service .  # Build Docker image

# Cleanup
make clean                # Clean build artifacts
```

### Project Structure

```
ufm/
├── cmd/
│   └── server/           # Main application entry point
├── internal/
│   ├── app/              # Application bootstrap
│   ├── config/           # Configuration management
│   ├── service/          # Business logic
│   ├── http/             # HTTP transport
│   │   ├── handler/      # HTTP handlers
│   │   └── middleware/   # HTTP middleware
│   ├── log/              # Logging infrastructure
│   ├── monitoring/       # Metrics and tracing
│   └── utils/            # Shared utilities
├── pkg/                  # Public packages
├── scripts/              # Build and deployment scripts
├── tests/                # Integration tests
├── Makefile
├── docker-compose.yml    # Supporting services
└── Dockerfile           # Service container
```

## Configuration

The service is configured through environment variables. See `.env.example` for all available options.

### Key Configuration Areas

- **Server**: Port, timeouts, host binding
- **Database**: Connection strings, pool settings
- **Logging**: Level, format, output
- **Tracing**: Jaeger endpoint, service name
- **Metrics**: Prometheus endpoint, collection interval
- **Authentication**: JWT settings, token expiration

## API Endpoints

### System Endpoints
- `GET /ping` - Simple ping endpoint
- `GET /health` - Health check endpoint
- `GET /readiness` - Readiness check endpoint
- `GET /version` - Version information

### Metrics & Monitoring
- `GET /metrics` - Prometheus metrics endpoint
- `GET /debug/pprof/` - Go pprof debugging endpoints (if enabled)

## Deployment

### Docker Deployment

1. **Build the image**:
   ```bash
   docker build -t ufm-service .
   ```

2. **Run the container**:
   ```bash
   docker run -p 8080:8080 --env-file .env ufm-service
   ```

### Docker Compose

Use the provided `docker-compose.yml` for local development with supporting services:

```bash
docker-compose up -d  # Start all services
docker-compose logs -f ufm-service  # View logs
```

## Extending the Template

### Adding New Endpoints

1. Create a new handler in `internal/http/handler/`
2. Register routes in `internal/http/router.go`
3. Add business logic in `internal/service/`

### Adding New Services

1. Define interface in `internal/service/`
2. Implement service with constructor
3. Register in `internal/app/app.go`
4. Inject dependencies as needed

### Adding Middleware

1. Create middleware in `internal/http/middleware/`
2. Register in `internal/http/server.go`
3. Configure order in middleware chain

## Testing

### Unit Tests
```bash
make test                  # Run all tests
go test ./internal/...     # Run specific package tests
```

### Integration Tests
```bash
go test ./tests/...        # Run integration tests
```

### Test Coverage
```bash
make test-coverage         # Generate coverage report
open coverage.html         # View coverage in browser
```

## Monitoring

### Logs
- JSON-structured logs with correlation IDs
- Configurable log levels (debug, info, warn, error)
- Request/response logging with timing

### Metrics
- Prometheus-compatible metrics on `/metrics`
- HTTP request metrics (duration, status codes)
- Custom business metrics support

### Tracing
- OpenTracing integration with Jaeger
- Distributed trace propagation
- Request span creation and tagging

## License

This template is provided as-is for educational and development purposes. Modify as needed for your specific use case.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

## Support

For questions or issues with this template, please check the documentation or create an issue in the repository.

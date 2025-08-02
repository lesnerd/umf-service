SHELL := /bin/bash

.DEFAULT_GOAL = build

GOCMD = go
GOOS = $(shell go env GOOS)
GOARCH = $(shell go env GOARCH)

# ----------------------------------------------------------------------------------------------------------------------

export PROJECT_DIR = $(CURDIR)

DOC_DIR = ${PROJECT_DIR}/doc
BIN_DIR = ${PROJECT_DIR}/bin

MODULE_NAME = server
BINARY_NAME = ufm-$(MODULE_NAME)

# ----------------------------------------------------------------------------------------------------------------------

APP_INFO_VERSION ?= undefined
APP_INFO_REVISION ?= undefined
APP_INFO_BUILD_TIME ?= $(shell date -u "+%Y-%m-%dT%H:%M:%SZ")

LINKERFLAGS = -s -w \
  -X 'ufm/internal/utils.version=$(APP_INFO_VERSION)' \
  -X 'ufm/internal/utils.revision=$(APP_INFO_REVISION)' \
  -X 'ufm/internal/utils.buildTime=$(APP_INFO_BUILD_TIME)'

# ======================================================================================================================

prereq:
	$(GOCMD) install google.golang.org/protobuf/cmd/protoc-gen-go@v1.33
	$(GOCMD) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3
	$(GOCMD) install go.uber.org/mock/mockgen@v0.4.0
	$(GOCMD) install github.com/jstemmer/go-junit-report@v1.0.0
	$(GOCMD) install gotest.tools/gotestsum@v1.12.0
	$(GOCMD) install github.com/swaggo/swag/cmd/swag@v1.16.4

# ----------------------------------------------------------------------------------------------------------------------

clean-bin:
	@rm -rf bin/${BINARY_NAME}-linux-amd64
	@rm -rf bin/${BINARY_NAME}-linux-arm64
	@rm -rf bin/${BINARY_NAME}-darwin-${GOARCH}
	@rm -rf bin/${BINARY_NAME}-windows-amd64.exe

clean-mock:
	@find . -name "*_mock.go" -delete

clean-swagger:
	@rm -f ${PROJECT_DIR}/doc/swagger.json
	@rm -f ${PROJECT_DIR}/doc/swagger.yaml
	@rm -f ${PROJECT_DIR}/doc/swagger.html

clean: clean-bin clean-mock

# ----------------------------------------------------------------------------------------------------------------------

generate: clean-mock
	$(GOCMD) generate ./...

generate-swagger: clean-swagger
	swag init -g ./cmd/server/main.go -o ${DOC_DIR}/

# ----------------------------------------------------------------------------------------------------------------------

build: clean generate
	$(GOCMD) build -trimpath -o bin/${BINARY_NAME}-${GOOS}-${GOARCH} ./cmd/server

build-linux-amd64:
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 $(GOCMD) build -ldflags="${LINKERFLAGS}" -trimpath -o ${BIN_DIR}/${BINARY_NAME}-linux-amd64 ./cmd/server

build-linux-arm64:
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 $(GOCMD) build -ldflags="${LINKERFLAGS}" -trimpath -o ${BIN_DIR}/${BINARY_NAME}-linux-arm64 ./cmd/server

build-darwin:
	GOOS=darwin GOARCH=${GOARCH} $(GOCMD) build -ldflags="${LINKERFLAGS}" -trimpath -o ${BIN_DIR}/${BINARY_NAME}-darwin-${GOARCH} ./cmd/server

build-windows:
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 $(GOCMD) build -ldflags="${LINKERFLAGS}" -trimpath -o ${BIN_DIR}/${BINARY_NAME}-windows-amd64.exe ./cmd/server

build-all: generate build-linux-amd64 build-linux-arm64 build-darwin build-windows

# ----------------------------------------------------------------------------------------------------------------------

utest: generate
	gotestsum  --format testname --junitfile-hide-empty-pkg=true --junitfile=target/reports/unittest.xml --packages="$(shell go list ./internal/... | grep -v /mocks)" -- -race -v  -count=1 -coverprofile=target/reports/ucoverage.out

itest: generate
	gotestsum --format testname --junitfile=target/reports/itestresults.xml --packages="$(shell go list ./tests/itests/...)" -- -tags=itest -p 1 -count=1 -coverprofile=target/reports/icoverage.out -coverpkg=github.com/ufm/...

e2e-test:
	cd ${PROJECT_DIR}
	${PROJECT_DIR}/scripts/e2e/start_e2e.sh
	$(GOCMD) test -v -count=1 ./tests/e2e/...; exit_status=$$?
	${PROJECT_DIR}/scripts/e2e/stop_e2e.sh
	exit $$exit_status

test: utest

test-all: utest itest e2e-test

# ----------------------------------------------------------------------------------------------------------------------

run:
	$(GOCMD) run ./cmd/server

tidy:
	$(GOCMD) mod tidy

lint:
	golangci-lint run

format:
	$(GOCMD) fmt ./...

create-config:
	@echo "Creating configuration from system.yaml..."
	@mkdir -p ~/.ufm/config
	@cp system.yaml ~/.ufm/config/system.yaml
	@echo "Configuration created at ~/.ufm/config/system.yaml"

# Telemetry-specific targets
init-telemetry-db:
	@echo "Initializing telemetry database..."
	@./scripts/init-telemetry-db.sh

run-generator:
	@echo "Starting telemetry data generator server on port 9001..."
	$(GOCMD) run ./cmd/generator

run-with-telemetry: create-config
	@echo "Starting UFM server with telemetry enabled..."
	TELEMETRY_ENABLED=true $(GOCMD) run ./cmd/server

run-with-queue: create-config
	@echo "Starting UFM server with telemetry and request queuing enabled..."
	@echo "WARNING: Queuing is for extreme high loads (10k+ RPS). Use run-with-telemetry for normal loads."
	TELEMETRY_ENABLED=true TELEMETRY_QUEUE_ENABLED=true $(GOCMD) run ./cmd/server

test-telemetry-endpoints:
	@echo "Testing telemetry API endpoints..."
	@echo "GET /telemetry/metrics"
	@curl -s http://localhost:8080/telemetry/metrics | jq . || echo "Server not running or no data"
	@echo "\nGET /telemetry/health"
	@curl -s http://localhost:8080/telemetry/health | jq . || echo "Server not running"
	@echo "\nGET /counters (generator)"
	@curl -s http://localhost:9001/counters || echo "Generator not running"

demo:
	@echo "🚀 Starting UFM Telemetry Demo with Docker..."
	@echo "Building and starting all services..."
	@docker-compose up --build -d
	@echo ""
	@echo "Waiting for services to be ready..."
	@sleep 10
	@echo ""
	@echo "✅ UFM Telemetry System Running!"
	@echo "   - Main API: http://localhost:8080/telemetry/metrics"
	@echo "   - Generator CSV: http://localhost:9001/counters"
	@echo "   - Performance: http://localhost:8080/telemetry/performance"
	@echo "   - Health: http://localhost:8080/telemetry/health"
	@echo "   - System Health: http://localhost:8080/api/v1/system/health"
	@echo ""
	@echo "📊 Check service status:"
	@echo "   docker-compose ps"
	@echo ""
	@echo "🔍 View logs:"
	@echo "   docker-compose logs -f [service-name]"
	@echo ""
	@echo "🛑 Stop demo:"
	@echo "   make stop-demo"

debug-env:
	@echo "🚀 Starting UFM Telemetry Demo with Docker for Debugging..."
	@echo "Building and starting all services..."
	@docker-compose -f docker-compose-for-debug.yml up --build -d
	@echo ""
	@echo "Waiting for services to be ready..."
	@sleep 10
	@echo ""
	@echo "✅ UFM Telemetry System Running!"
	@echo "   - Main API: http://localhost:8080/telemetry/metrics"
	@echo "   - Generator CSV: http://localhost:9001/counters"
	@echo "   - Performance: http://localhost:8080/telemetry/performance"
	@echo "   - Health: http://localhost:8080/telemetry/health"
	@echo "   - System Health: http://localhost:8080/api/v1/system/health"
	@echo ""
	@echo "📊 Check service status:"
	@echo "   docker-compose -f docker-compose-for-debug.yml ps"
	@echo ""
	@echo "🔍 View logs:"
	@echo "   docker-compose -f docker-compose-for-debug.yml logs -f [service-name]"
	@echo ""
	@echo "🛑 Stop demo:"
	@echo "   make stop-demo"

stop-demo:
	@echo "🛑 Stopping UFM Telemetry Demo..."
	@docker-compose down --remove-orphans
	@echo "✅ Demo stopped - all containers and networks removed"

# Docker development targets
docker-build:
	@echo "🐳 Building Docker images..."
	@docker-compose build

docker-logs:
	@docker-compose logs -f

docker-status:
	@echo "📊 UFM Service Status:"
	@docker-compose ps

docker-clean:
	@echo "🧹 Cleaning up Docker resources..."
	@docker-compose down --remove-orphans --volumes
	@docker system prune -f
	@echo "✅ Cleanup completed"

# ----------------------------------------------------------------------------------------------------------------------

.PHONY: prereq clean-bin clean-mock clean-swagger clean \
        generate generate-swagger \
        build build-linux-amd64 build-linux-arm64 build-darwin build-windows build-all \
        test test-coverage run tidy lint format create-config \
        init-telemetry-db run-generator run-with-telemetry run-with-queue \
        test-telemetry-endpoints demo stop-demo \
        docker-build docker-logs docker-status docker-clean \
        e2e-test test-all
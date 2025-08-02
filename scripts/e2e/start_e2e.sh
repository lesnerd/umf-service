#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
COMPOSE_FILE="$PROJECT_DIR/docker-compose.yml"

echo -e "${BLUE}🚀 Starting UFM E2E Test Environment${NC}"
echo -e "${BLUE}===============================================${NC}"

# Check if docker-compose.yml exists
if [ ! -f "$COMPOSE_FILE" ]; then
    echo -e "${RED}❌ Error: docker-compose.yml not found at $COMPOSE_FILE${NC}"
    exit 1
fi

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Error: Docker is not running${NC}"
    exit 1
fi

# Stop any existing containers
echo -e "${YELLOW}🛑 Stopping any existing containers...${NC}"
cd "$PROJECT_DIR"
docker-compose down --remove-orphans 2>/dev/null || true

# Build and start services
echo -e "${YELLOW}🔨 Building and starting services...${NC}"
docker-compose up --build -d

# Wait for services to be healthy
echo -e "${YELLOW}⏳ Waiting for services to be ready...${NC}"

# Configuration
TIMEOUT_MINUTES=5

# Function to collect logs for debugging
collect_logs() {
    echo -e "${YELLOW}📋 Collecting service logs for debugging...${NC}"
    echo -e "${BLUE}=== Postgres Logs ===${NC}"
    docker-compose logs postgres
    echo -e "${BLUE}=== UFM Generator Logs ===${NC}"
    docker-compose logs ufm-generator
    echo -e "${BLUE}=== UFM Server Logs ===${NC}"
    docker-compose logs ufm-server
}

# Function to check service health by testing endpoints
check_service_health() {
    local url="$1"
    local description="$2"
    local max_retries=$((TIMEOUT_MINUTES * 60 / 10))  # Check every 10 seconds
    local retry=0

    echo -e "${YELLOW}🔍 Checking $description at $url${NC}"
    
    while [[ $retry -lt $max_retries ]]; do
        status_code=$(curl -s -o /dev/null -w "%{http_code}" "$url")
        echo -e "${BLUE}Service $description responded with $status_code${NC}"
        
        if [[ $status_code -eq 200 ]]; then
            echo -e "${GREEN}✅ $description is healthy.${NC}"
            return 0
        fi
        
        echo -e "${YELLOW}⏳ Waiting for $description to be ready... (attempt $((retry + 1))/$max_retries)${NC}"
        sleep 10
        ((retry++))
    done
    
    echo -e "${RED}❌ $description failed to respond with 200 after $TIMEOUT_MINUTES minutes. Exiting...${NC}"
    collect_logs
    exit 1
}

# Wait for services to be healthy
echo -e "${YELLOW}⏳ Waiting for services to be ready...${NC}"

# Give services a moment to fully start up
sleep 10

# Check essential service health by testing their endpoints
check_service_health "http://localhost:8080/api/v1/system/ping" "Main API Ping"
check_service_health "http://localhost:9001/counters" "Generator API"
check_service_health "http://localhost:8080/telemetry/metrics" "Telemetry Metrics"

echo -e "${GREEN}🎉 UFM E2E Test Environment is ready!${NC}"
echo -e "${BLUE}📊 Service URLs:${NC}"
echo -e "   - Main API: http://localhost:8080"
echo -e "   - Generator: http://localhost:9001"
echo -e "   - Jaeger UI: http://localhost:16686"
echo -e "   - Prometheus: http://localhost:9090"
echo -e ""
echo -e "${BLUE}🧪 Ready to run e2e tests!${NC}" 
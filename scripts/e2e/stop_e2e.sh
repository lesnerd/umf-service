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

echo -e "${BLUE}🛑 Stopping UFM E2E Test Environment${NC}"
echo -e "${BLUE}================================================${NC}"

# Change to project directory
cd "$PROJECT_DIR"

# Stop all containers
echo -e "${YELLOW}🛑 Stopping all containers...${NC}"
docker-compose down --remove-orphans

# Remove volumes if requested
if [ "$1" = "--clean" ] || [ "$1" = "-c" ]; then
    echo -e "${YELLOW}🧹 Removing volumes...${NC}"
    docker-compose down --volumes --remove-orphans
fi

# Clean up any dangling images (optional)
if [ "$1" = "--prune" ] || [ "$1" = "-p" ]; then
    echo -e "${YELLOW}🧹 Pruning unused Docker resources...${NC}"
    docker system prune -f
fi

# Show final status
echo -e "${YELLOW}📊 Checking final status...${NC}"
if docker-compose ps | grep -q "Up"; then
    echo -e "${RED}⚠️  Some containers are still running:${NC}"
    docker-compose ps
else
    echo -e "${GREEN}✅ All containers stopped successfully${NC}"
fi

echo -e "${GREEN}🎉 UFM E2E Test Environment stopped!${NC}"
echo -e "${BLUE}💡 Usage:${NC}"
echo -e "   $0              # Stop containers"
echo -e "   $0 --clean      # Stop containers and remove volumes"
echo -e "   $0 --prune      # Stop containers and prune unused resources" 
#!/bin/bash

# Initialize telemetry database script
# This script sets up the PostgreSQL database for telemetry data

set -e

# Configuration - Using PostgreSQL default admin credentials
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_NAME=${DB_NAME:-umf_db}
DB_USER=${DB_USER:-postgres}
DB_PASSWORD=${DB_PASSWORD:-password}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Initializing UFM Telemetry Database...${NC}"

# Check if PostgreSQL is running
echo -e "${YELLOW}Checking PostgreSQL connection...${NC}"
if ! PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c '\q' 2>/dev/null; then
    echo -e "${RED}Error: Cannot connect to PostgreSQL at $DB_HOST:$DB_PORT${NC}"
    echo -e "${YELLOW}Make sure PostgreSQL is running and the connection parameters are correct.${NC}"
    echo -e "${YELLOW}You can start PostgreSQL with: docker-compose up -d postgres${NC}"
    exit 1
fi

echo -e "${GREEN}✓ PostgreSQL connection successful${NC}"

# Create database if it doesn't exist
echo -e "${YELLOW}Creating database '$DB_NAME' if it doesn't exist...${NC}"
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c "CREATE DATABASE $DB_NAME;" 2>/dev/null || true
echo -e "${GREEN}✓ Database '$DB_NAME' ready${NC}"

# Run migrations
echo -e "${YELLOW}Running telemetry database migrations...${NC}"
MIGRATION_DIR="$(dirname "$0")/migrations"

# Run first migration (create tables)
MIGRATION_FILE_1="$MIGRATION_DIR/001_create_telemetry_tables.sql"
if [ ! -f "$MIGRATION_FILE_1" ]; then
    echo -e "${RED}Error: Migration file not found: $MIGRATION_FILE_1${NC}"
    exit 1
fi

PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f "$MIGRATION_FILE_1"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Telemetry tables migration completed successfully${NC}"
else
    echo -e "${RED}✗ Tables migration failed${NC}"
    exit 1
fi

# Run second migration (create user and permissions)
MIGRATION_FILE_2="$MIGRATION_DIR/002_create_umf_user.sql"
if [ ! -f "$MIGRATION_FILE_2" ]; then
    echo -e "${RED}Error: Migration file not found: $MIGRATION_FILE_2${NC}"
    exit 1
fi

PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f "$MIGRATION_FILE_2"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ User creation and permissions migration completed successfully${NC}"
else
    echo -e "${RED}✗ User migration failed${NC}"
    exit 1
fi

# Verify the setup
echo -e "${YELLOW}Verifying database setup...${NC}"
TABLE_COUNT=$(PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('switches', 'telemetry_metrics');" | tr -d ' ')

echo -e "${GREEN}✓ Database verification complete${NC}"
echo -e "${GREEN}✓ Tables created: $TABLE_COUNT${NC}"

echo -e "${GREEN}"
echo "=========================================="
echo "UFM Telemetry Database Ready!"
echo "=========================================="
echo -e "${NC}"
echo "Database: $DB_NAME"
echo "Host: $DB_HOST:$DB_PORT"
echo "Admin User: $DB_USER"
echo "App User: umf_user"
echo "Tables: $TABLE_COUNT"
echo ""
echo "You can now start the UFM service with:"
echo "  make run"
echo ""
echo "Or start the generator server with:"
echo "  go run ./cmd/generator"
echo -e "${NC}"
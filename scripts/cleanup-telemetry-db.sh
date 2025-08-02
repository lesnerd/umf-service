#!/bin/bash

# Cleanup telemetry database script
# This script removes the UFM telemetry database and user

set -e

# Configuration - Using PostgreSQL default admin credentials
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_NAME=${DB_NAME:-umf_db}
DB_USER=${DB_USER:-umf_user}
DB_PASSWORD=${DB_PASSWORD:-password}
ADMIN_USER=${ADMIN_USER:-postgres}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}UFM Telemetry Database Cleanup${NC}"
echo "This will permanently delete the database and user."
echo ""

# Confirmation prompt
read -p "Are you sure you want to delete the database '$DB_NAME' and user '$DB_USER'? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}Cleanup cancelled.${NC}"
    exit 0
fi

echo -e "${YELLOW}Starting database cleanup...${NC}"

# Check if PostgreSQL is running
echo -e "${YELLOW}Checking PostgreSQL connection...${NC}"
if ! PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $ADMIN_USER -d postgres -c '\q' 2>/dev/null; then
    echo -e "${RED}Error: Cannot connect to PostgreSQL at $DB_HOST:$DB_PORT${NC}"
    echo -e "${YELLOW}Make sure PostgreSQL is running and the connection parameters are correct.${NC}"
    exit 1
fi

echo -e "${GREEN}✓ PostgreSQL connection successful${NC}"

# Terminate active connections to the database
echo -e "${YELLOW}Terminating active connections to database '$DB_NAME'...${NC}"
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $ADMIN_USER -d postgres -c "
SELECT pg_terminate_backend(pid) 
FROM pg_stat_activity 
WHERE datname = '$DB_NAME' AND pid <> pg_backend_pid();" 2>/dev/null || true

# Drop the database
echo -e "${YELLOW}Dropping database '$DB_NAME'...${NC}"
if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $ADMIN_USER -d postgres -c "DROP DATABASE IF EXISTS $DB_NAME;" 2>/dev/null; then
    echo -e "${GREEN}✓ Database '$DB_NAME' dropped successfully${NC}"
else
    echo -e "${RED}✗ Failed to drop database '$DB_NAME'${NC}"
    exit 1
fi

# Drop the user
echo -e "${YELLOW}Dropping user '$DB_USER'...${NC}"
if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $ADMIN_USER -d postgres -c "DROP USER IF EXISTS $DB_USER;" 2>/dev/null; then
    echo -e "${GREEN}✓ User '$DB_USER' dropped successfully${NC}"
else
    echo -e "${RED}✗ Failed to drop user '$DB_USER'${NC}"
    exit 1
fi

echo -e "${GREEN}"
echo "=========================================="
echo "UFM Telemetry Database Cleanup Complete!"
echo "=========================================="
echo -e "${NC}"
echo "Database: $DB_NAME - DELETED"
echo "User: $DB_USER - DELETED"
echo ""
echo "To recreate the database, run:"
echo "  ./scripts/init-telemetry-db.sh"
echo -e "${NC}" 
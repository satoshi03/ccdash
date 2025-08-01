#!/bin/bash

# Test migration system with separate port and database

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
TEST_DB_PATH="/tmp/ccdash-test-migration.db"
TEST_PORT=7070
TEST_LOG_PATH="/tmp/ccdash-test"

echo -e "${YELLOW}Testing Migration System${NC}"
echo "================================"
echo "Database: $TEST_DB_PATH"
echo "Port: $TEST_PORT"
echo ""

# Clean up previous test database
if [ -f "$TEST_DB_PATH" ]; then
    echo -e "${YELLOW}Removing existing test database...${NC}"
    rm -f "$TEST_DB_PATH"*
fi

# Create test log directory
mkdir -p "$TEST_LOG_PATH"

# Build the migration tool
echo -e "${YELLOW}Building migration tool...${NC}"
cd backend
go build -o ../bin/migrate cmd/migrate/main.go
if [ $? -ne 0 ]; then
    echo -e "${RED}Failed to build migration tool${NC}"
    exit 1
fi
cd ..

# Test 1: Check migration status on empty database
echo -e "\n${YELLOW}Test 1: Check migration status (empty database)${NC}"
./bin/migrate -db "$TEST_DB_PATH" -cmd status

# Test 2: Run migrations
echo -e "\n${YELLOW}Test 2: Run migrations${NC}"
./bin/migrate -db "$TEST_DB_PATH" -cmd up

# Test 3: Check migration status after running migrations
echo -e "\n${YELLOW}Test 3: Check migration status (after migrations)${NC}"
./bin/migrate -db "$TEST_DB_PATH" -cmd status

# Test 4: Run migrations again (should do nothing)
echo -e "\n${YELLOW}Test 4: Run migrations again (idempotency test)${NC}"
./bin/migrate -db "$TEST_DB_PATH" -cmd up

# Test 5: Build and start server with test configuration
echo -e "\n${YELLOW}Test 5: Starting server on port $TEST_PORT...${NC}"
cd backend
go build -o ../bin/server cmd/server/main.go
if [ $? -ne 0 ]; then
    echo -e "${RED}Failed to build server${NC}"
    exit 1
fi
cd ..

# Start server in background with environment variables
CCDASH_DB_PATH="$TEST_DB_PATH" PORT=$TEST_PORT ./bin/server &
SERVER_PID=$!

# Wait for server to start
sleep 3

# Test server health
echo -e "\n${YELLOW}Test 6: Check server health${NC}"
curl -s http://localhost:$TEST_PORT/api/v1/health | jq .

# Test API endpoints
echo -e "\n${YELLOW}Test 7: Test API endpoints${NC}"
echo -e "Testing /api/token-usage..."
curl -s http://localhost:$TEST_PORT/api/token-usage | jq .

echo -e "\nTesting /api/claude/sessions/recent..."
curl -s http://localhost:$TEST_PORT/api/claude/sessions/recent | jq .

# Stop server
echo -e "\n${YELLOW}Stopping server...${NC}"
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null

# Clean up
echo -e "\n${YELLOW}Cleaning up...${NC}"
rm -f bin/migrate bin/server

echo -e "\n${GREEN}Migration system test completed!${NC}"
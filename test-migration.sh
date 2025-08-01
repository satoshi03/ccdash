#!/bin/bash
set -e

echo "=== Testing Database Migration System Phase 1 ==="

# Clean up any previous test database
rm -rf ~/.ccdash-test
mkdir -p ~/.ccdash-test

# Build the migration CLI tool
echo "Building migration CLI..."
cd backend && go build -o ../bin/migrate ./cmd/migrate
cd ..

# Test migration status on empty database
echo -e "\n1. Testing migration status on empty database..."
CCDASH_DB_PATH=~/.ccdash-test/migration-test.db ./bin/migrate status

# Run migrations
echo -e "\n2. Running migrations..."
CCDASH_DB_PATH=~/.ccdash-test/migration-test.db ./bin/migrate up

# Check migration status after running
echo -e "\n3. Checking migration status after running..."
CCDASH_DB_PATH=~/.ccdash-test/migration-test.db ./bin/migrate status

# Test with different database path
echo -e "\n4. Testing with custom database path..."
./bin/migrate -db ~/.ccdash-test/alternate-test.db status
./bin/migrate -db ~/.ccdash-test/alternate-test.db up
./bin/migrate -db ~/.ccdash-test/alternate-test.db status

# Build and test the server with migrations
echo -e "\n5. Building and testing server with migration database..."
cd backend && go build -o ../bin/server-test ./cmd/server
cd ..

# Start server with test config in background
echo "Starting server on port 7070..."
CCDASH_DB_PATH=~/.ccdash-test/migration-test.db PORT=7070 ./bin/server-test > test-migration-server.log 2>&1 &
SERVER_PID=$!

# Wait for server to start
sleep 2

# Test API endpoints
echo -e "\n6. Testing API endpoints..."
echo "Health check:"
curl -s http://localhost:7070/api/v1/health | jq .

echo -e "\nToken usage:"
curl -s http://localhost:7070/api/token-usage | jq .

# Kill the server
kill $SERVER_PID

echo -e "\n=== Migration System Test Complete ==="
echo "Check test-migration-server.log for server output"
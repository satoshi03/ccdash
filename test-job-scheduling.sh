#!/bin/bash

# Test job scheduling API

API_URL="http://localhost:7070/api"
PROJECT_ID="3aeb3c6b-bc5a-49a1-a6a1-6c506263fdbf"  # claudeee project

echo "=== Testing Job Scheduling API ==="
echo ""

# Test 1: Create immediate job
echo "1. Creating immediate job..."
curl -X POST $API_URL/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "'$PROJECT_ID'",
    "command": "echo Immediate job test",
    "yolo_mode": false,
    "schedule_type": "immediate"
  }' | jq .

echo ""

# Test 2: Create delayed job (3 hours)
echo "2. Creating delayed job (3 hours)..."
curl -X POST $API_URL/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "'$PROJECT_ID'",
    "command": "echo Delayed job test",
    "yolo_mode": false,
    "schedule_type": "delayed",
    "schedule_params": {
      "delay_hours": 3
    }
  }' | jq .

echo ""

# Test 3: Create scheduled job (24 hours from now)
echo "3. Creating scheduled job (24 hours from now)..."
SCHEDULED_TIME=$(date -u -v+24H '+%Y-%m-%dT%H:%M:%SZ')
curl -X POST $API_URL/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "'$PROJECT_ID'",
    "command": "echo Scheduled job test",
    "yolo_mode": true,
    "schedule_type": "scheduled",
    "schedule_params": {
      "scheduled_time": "'$SCHEDULED_TIME'"
    }
  }' | jq .

echo ""

# Test 4: Test validation - delayed job without params
echo "4. Testing validation - delayed job without params (should fail)..."
curl -X POST $API_URL/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "'$PROJECT_ID'",
    "command": "echo Invalid job test",
    "yolo_mode": false,
    "schedule_type": "delayed"
  }' | jq .

echo ""

# Test 5: List all jobs
echo "5. Listing all jobs..."
curl -X GET "$API_URL/jobs?limit=10" | jq '.jobs[] | {id: .id, command: .command, schedule_type: .schedule_type, scheduled_at: .scheduled_at, schedule_params: .schedule_params}'

echo ""
echo "=== Test completed ==="
#!/bin/bash
# Run CCDash backend with safety checks disabled
# Usage: ./scripts/run-no-safety.sh

echo "🚀 Starting CCDash backend with safety checks DISABLED..."
echo "⚠️  WARNING: All commands will be executed without safety validation!"
echo ""

# Export the disable flag
export CCDASH_DISABLE_SAFETY_CHECK=true

# Run the server
cd "$(dirname "$0")/.." && go run cmd/server/main.go
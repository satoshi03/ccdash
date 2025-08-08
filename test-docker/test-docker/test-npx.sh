#\!/bin/bash
echo "=== Testing npx ccdash@latest on Linux ==="
echo "Node version: $(node --version)"
echo "NPM version: $(npm --version)"
echo ""

echo "=== Installing ccdash@latest globally ==="
npm install -g ccdash@latest
echo ""

echo "=== Checking installed binary location ==="
INSTALLED_BIN=$(which ccdash)
if [ -n "$INSTALLED_BIN" ]; then
  echo "ccdash installed at: $INSTALLED_BIN"
  
  # Find actual binary location
  ACTUAL_BIN=$(readlink -f "$INSTALLED_BIN")
  echo "Actual location: $ACTUAL_BIN"
  
  # Find binaries in the package
  BIN_DIR=$(dirname "$ACTUAL_BIN")
  echo ""
  echo "=== Binaries in package ==="
  ls -la "$BIN_DIR"/ccdash-server* 2>/dev/null || echo "No ccdash-server binaries found"
  
  # Check Linux binary specifically
  LINUX_BIN="$BIN_DIR/ccdash-server-linux-amd64"
  if [ -f "$LINUX_BIN" ]; then
    echo ""
    echo "=== Linux binary analysis ==="
    echo "Path: $LINUX_BIN"
    echo "File type:"
    file "$LINUX_BIN"
    echo ""
    echo "First 32 bytes (hex):"
    hexdump -C "$LINUX_BIN" | head -n 2
    echo ""
    echo "Permissions:"
    ls -la "$LINUX_BIN"
  fi
fi

echo ""
echo "=== Testing help command ==="
ccdash help || echo "Help command failed"

echo ""
echo "=== Testing backend server start (10 second timeout) ==="
timeout 10 ccdash start --no-open 2>&1 || echo "Server start test completed (timeout expected)"

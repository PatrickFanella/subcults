#!/bin/bash
# Manual verification script for profiling endpoints

set -e

echo "=== Profiling Manual Verification ==="
echo ""

# Check if server is running
if ! curl -s http://localhost:8080/health/live > /dev/null 2>&1; then
    echo "ERROR: API server is not running at localhost:8080"
    echo "Start the server with: PROFILING_ENABLED=true SUBCULT_ENV=development ./bin/api"
    exit 1
fi

echo "✓ API server is running"
echo ""

# Check profiling status
echo "1. Checking profiling status..."
STATUS=$(curl -s http://localhost:8080/debug/profiling/status)
echo "$STATUS" | python3 -m json.tool 2>/dev/null || echo "$STATUS"
echo ""

# Extract profiling_enabled from JSON
ENABLED=$(echo "$STATUS" | grep -o '"profiling_enabled": [^,}]*' | cut -d' ' -f2)

if [ "$ENABLED" != "true" ]; then
    echo "WARNING: Profiling is disabled. Enable it with: PROFILING_ENABLED=true"
    echo "Remaining tests will fail."
    exit 0
fi

echo "✓ Profiling is enabled"
echo ""

# Test profiling index
echo "2. Testing /debug/pprof/ (index page)..."
if curl -s http://localhost:8080/debug/pprof/ | grep -q "Profile"; then
    echo "✓ Index page accessible"
else
    echo "✗ Index page not accessible"
    exit 1
fi
echo ""

# Test goroutine profile
echo "3. Testing /debug/pprof/goroutine (goroutine profile)..."
if curl -s http://localhost:8080/debug/pprof/goroutine > /tmp/goroutine.prof; then
    GOROUTINES=$(head -1 /tmp/goroutine.prof | cut -d' ' -f1)
    echo "✓ Goroutine profile collected ($GOROUTINES goroutines)"
else
    echo "✗ Goroutine profile failed"
    exit 1
fi
echo ""

# Test heap profile
echo "4. Testing /debug/pprof/heap (heap profile)..."
if curl -s http://localhost:8080/debug/pprof/heap > /tmp/heap.prof 2>&1; then
    SIZE=$(stat -f%z /tmp/heap.prof 2>/dev/null || stat -c%s /tmp/heap.prof 2>/dev/null)
    echo "✓ Heap profile collected (${SIZE} bytes)"
else
    echo "✗ Heap profile failed"
    exit 1
fi
echo ""

# Test CPU profile (short duration)
echo "5. Testing /debug/pprof/profile?seconds=2 (CPU profile)..."
echo "   (This will take 2 seconds...)"
if curl -s 'http://localhost:8080/debug/pprof/profile?seconds=2' > /tmp/cpu.prof 2>&1; then
    SIZE=$(stat -f%z /tmp/cpu.prof 2>/dev/null || stat -c%s /tmp/cpu.prof 2>/dev/null)
    echo "✓ CPU profile collected (${SIZE} bytes)"
else
    echo "✗ CPU profile failed"
    exit 1
fi
echo ""

# Test with go tool pprof
echo "6. Testing with go tool pprof..."
if command -v go &> /dev/null; then
    echo "   Analyzing goroutine profile..."
    if go tool pprof -text /tmp/goroutine.prof | head -10 > /dev/null 2>&1; then
        echo "✓ go tool pprof can analyze profiles"
    else
        echo "✗ go tool pprof failed"
        exit 1
    fi
else
    echo "⊘ go tool not available (skipped)"
fi
echo ""

# Summary
echo "=== Summary ==="
echo "✓ All profiling endpoints are working correctly"
echo ""
echo "Profile files saved:"
echo "  - /tmp/cpu.prof (CPU profile)"
echo "  - /tmp/heap.prof (heap profile)"
echo "  - /tmp/goroutine.prof (goroutine profile)"
echo ""
echo "Next steps:"
echo "  1. Analyze with go tool pprof:"
echo "     go tool pprof -http=:8081 /tmp/cpu.prof"
echo "  2. Generate flame graph:"
echo "     go tool pprof -http=:8081 http://localhost:8080/debug/pprof/profile?seconds=30"
echo "  3. View interactive profile:"
echo "     go tool pprof http://localhost:8080/debug/pprof/heap"
echo ""

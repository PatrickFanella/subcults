#!/bin/bash
# Manual CORS verification script
# This script tests CORS headers with curl

set -e

API_URL="http://localhost:8080"
ALLOWED_ORIGIN="http://localhost:3000"
FORBIDDEN_ORIGIN="http://malicious.com"

echo "==================================================================="
echo "CORS Manual Verification Script"
echo "==================================================================="
echo ""
echo "Prerequisites:"
echo "1. API server must be running on $API_URL"
echo "2. CORS must be enabled with CORS_ALLOWED_ORIGINS=$ALLOWED_ORIGIN"
echo ""
echo "==================================================================="
echo ""

# Test 1: Preflight request from allowed origin
echo "Test 1: Preflight OPTIONS request from allowed origin"
echo "Expected: 204 No Content with CORS headers"
echo "-------------------------------------------------------------------"
curl -i -X OPTIONS "$API_URL/health/live" \
  -H "Origin: $ALLOWED_ORIGIN" \
  -H "Access-Control-Request-Method: GET" \
  -H "Access-Control-Request-Headers: Content-Type" \
  2>/dev/null | grep -E "(HTTP/|Access-Control-)"
echo ""

# Test 2: Actual GET request from allowed origin
echo "Test 2: GET request from allowed origin"
echo "Expected: 200 OK with Access-Control-Allow-Origin header"
echo "-------------------------------------------------------------------"
curl -i -X GET "$API_URL/health/live" \
  -H "Origin: $ALLOWED_ORIGIN" \
  2>/dev/null | grep -E "(HTTP/|Access-Control-)"
echo ""

# Test 3: Request from unauthorized origin
echo "Test 3: GET request from unauthorized origin"
echo "Expected: 403 Forbidden"
echo "-------------------------------------------------------------------"
curl -i -X GET "$API_URL/health/live" \
  -H "Origin: $FORBIDDEN_ORIGIN" \
  2>/dev/null | grep -E "(HTTP/|Access-Control-)" || echo "No CORS headers (expected)"
echo ""

# Test 4: Same-origin request (no Origin header)
echo "Test 4: Same-origin request (no Origin header)"
echo "Expected: 200 OK, no CORS headers needed"
echo "-------------------------------------------------------------------"
curl -i -X GET "$API_URL/health/live" \
  2>/dev/null | grep -E "HTTP/"
echo ""

echo "==================================================================="
echo "CORS verification complete!"
echo "==================================================================="

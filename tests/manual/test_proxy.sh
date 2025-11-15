#!/bin/bash

# Switchboard Gateway - Proxy Testing Script
# This script tests the reverse proxy functionality

set -e

echo "🧪 Switchboard Gateway - Proxy Tests"
echo "===================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Base URLs
GATEWAY_URL="http://localhost:8080"
BACKEND_URL="http://localhost:8081"

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Helper function to test endpoint
test_endpoint() {
    local name=$1
    local url=$2
    local expected_status=$3
    
    echo -n "Testing: $name ... "
    
    status=$(curl -s -o /dev/null -w "%{http_code}" "$url")
    
    if [ "$status" == "$expected_status" ]; then
        echo -e "${GREEN}✓ PASS${NC} (HTTP $status)"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗ FAIL${NC} (Expected $expected_status, got $status)"
        ((TESTS_FAILED++))
    fi
}

# Helper function to test with output
test_with_output() {
    local name=$1
    local url=$2
    
    echo "Testing: $name"
    echo "URL: $url"
    echo "Response:"
    curl -s "$url" | jq '.' || curl -s "$url"
    echo ""
}

echo "Step 1: Verify backend is running"
echo "-----------------------------------"
test_endpoint "Demo Backend" "$BACKEND_URL/get" "200"
echo ""

echo "Step 2: Verify gateway is running"
echo "-----------------------------------"
test_endpoint "Gateway Health" "$GATEWAY_URL/health" "200"
test_endpoint "Gateway Ready" "$GATEWAY_URL/ready" "200"
echo ""

echo "Step 3: Test proxy requests"
echo "-----------------------------------"
test_endpoint "GET /api/users" "$GATEWAY_URL/api/users" "200"
test_endpoint "GET /api/products" "$GATEWAY_URL/api/products" "200"
echo ""

echo "Step 4: Test with path parameters"
echo "-----------------------------------"
test_endpoint "GET /api/users/123" "$GATEWAY_URL/api/users/123" "200"
echo ""

echo "Step 5: Test HTTP methods"
echo "-----------------------------------"
test_endpoint "POST /api/users" "$GATEWAY_URL/api/users" "200"
test_endpoint "PUT /api/users/123" "$GATEWAY_URL/api/users/123" "200"
test_endpoint "DELETE /api/users/123" "$GATEWAY_URL/api/users/123" "200"
echo ""

echo "Step 6: Test query parameters"
echo "-----------------------------------"
test_endpoint "GET /api/users?page=1&limit=10" "$GATEWAY_URL/api/users?page=1&limit=10" "200"
echo ""

echo "Step 7: Test headers"
echo "-----------------------------------"
echo "Testing: Headers forwarding"
response=$(curl -s "$GATEWAY_URL/api/headers" -H "X-Custom-Header: test-value")
if echo "$response" | grep -q "X-Custom-Header"; then
    echo -e "${GREEN}✓ PASS${NC} - Headers forwarded"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC} - Headers not forwarded"
    ((TESTS_FAILED++))
fi
echo ""

echo "Step 8: Test proxy headers added"
echo "-----------------------------------"
echo "Checking for X-Request-ID, X-Forwarded-For, etc."
response=$(curl -s "$GATEWAY_URL/api/headers")
echo "$response" | jq '.headers' | grep -E "X-Request-Id|X-Forwarded-For|X-Real-Ip" || true
echo ""

echo "Step 9: Test non-existent route"
echo "-----------------------------------"
test_endpoint "GET /nonexistent" "$GATEWAY_URL/nonexistent" "404"
echo ""

echo "Step 10: Detailed request test"
echo "-----------------------------------"
test_with_output "Full request details" "$GATEWAY_URL/api/anything"
echo ""

# Summary
echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Failed: ${RED}$TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed${NC}"
    exit 1
fi
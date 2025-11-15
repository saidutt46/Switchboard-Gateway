#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Base URLs
ADMIN_API="http://localhost:8000"
GATEWAY="http://localhost:8080"

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

echo "=========================================="
echo "Admin API & Hot Reload Test Suite"
echo "=========================================="
echo ""

# Helper function to run test
run_test() {
    local test_name=$1
    local command=$2
    local expected=$3
    
    echo -n "Testing: $test_name... "
    
    result=$(eval $command 2>&1)
    status=$?
    
    if [ $status -eq 0 ] && [[ $result == *"$expected"* ]]; then
        echo -e "${GREEN}✓ PASSED${NC}"
        ((TESTS_PASSED++))
        return 0
    else
        echo -e "${RED}✗ FAILED${NC}"
        echo "  Expected: $expected"
        echo "  Got: $result"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Test 1: Health Check
echo "=== Health Checks ==="
run_test "Admin API health" \
    "curl -s $ADMIN_API/health | jq -r '.status'" \
    "healthy"

run_test "Gateway health" \
    "curl -s $GATEWAY/health | jq -r '.status'" \
    "healthy"

echo ""

# Test 2: Services CRUD
echo "=== Services CRUD ==="

# Create service
echo -n "Creating service... "
SERVICE_RESPONSE=$(curl -s -X POST $ADMIN_API/services \
    -H "Content-Type: application/json" \
    -d '{
        "name": "test-service",
        "protocol": "http",
        "host": "test.example.com",
        "port": 8080
    }')

SERVICE_ID=$(echo $SERVICE_RESPONSE | jq -r '.id')

if [ "$SERVICE_ID" != "null" ] && [ -n "$SERVICE_ID" ]; then
    echo -e "${GREEN}✓ Created (ID: $SERVICE_ID)${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAILED${NC}"
    ((TESTS_FAILED++))
fi

# List services
run_test "List services" \
    "curl -s $ADMIN_API/services | jq 'length'" \
    "1"

# Get service
run_test "Get service by ID" \
    "curl -s $ADMIN_API/services/$SERVICE_ID | jq -r '.name'" \
    "test-service"

# Update service
run_test "Update service" \
    "curl -s -X PUT $ADMIN_API/services/$SERVICE_ID -H 'Content-Type: application/json' -d '{\"port\": 9090}' | jq -r '.port'" \
    "9090"

echo ""

# Test 3: Routes CRUD
echo "=== Routes CRUD ==="

# Create route
echo -n "Creating route... "
ROUTE_RESPONSE=$(curl -s -X POST $ADMIN_API/routes \
    -H "Content-Type: application/json" \
    -d "{
        \"service_id\": \"$SERVICE_ID\",
        \"name\": \"test-route\",
        \"paths\": [\"/api/test\"],
        \"methods\": [\"GET\", \"POST\"]
    }")

ROUTE_ID=$(echo $ROUTE_RESPONSE | jq -r '.id')

if [ "$ROUTE_ID" != "null" ] && [ -n "$ROUTE_ID" ]; then
    echo -e "${GREEN}✓ Created (ID: $ROUTE_ID)${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAILED${NC}"
    ((TESTS_FAILED++))
fi

# List routes
run_test "List routes" \
    "curl -s $ADMIN_API/routes | jq 'length'" \
    "1"

echo ""

# Test 4: Hot Reload
echo "=== Hot Reload Test ==="

echo -n "Waiting for gateway to reload... "
sleep 2
echo "done"

# Test if route is accessible on gateway
echo -n "Testing route on gateway... "
GATEWAY_RESPONSE=$(curl -s $GATEWAY/api/test)

if [[ $GATEWAY_RESPONSE == *"matched"* ]]; then
    echo -e "${GREEN}✓ Hot reload working!${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ Route not found on gateway${NC}"
    ((TESTS_FAILED++))
fi

# Update route
echo -n "Updating route paths... "
curl -s -X PUT $ADMIN_API/routes/$ROUTE_ID \
    -H "Content-Type: application/json" \
    -d '{"paths": ["/api/test", "/api/test-v2"]}' > /dev/null

echo "done"
sleep 2

# Test new path
echo -n "Testing updated route... "
GATEWAY_RESPONSE=$(curl -s $GATEWAY/api/test-v2)

if [[ $GATEWAY_RESPONSE == *"matched"* ]]; then
    echo -e "${GREEN}✓ Updated route working!${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ Updated route not working${NC}"
    ((TESTS_FAILED++))
fi

echo ""

# Test 5: Consumers & API Keys
echo "=== Consumers & API Keys ==="

# Create consumer
echo -n "Creating consumer... "
CONSUMER_RESPONSE=$(curl -s -X POST $ADMIN_API/consumers \
    -H "Content-Type: application/json" \
    -d '{
        "username": "test-app",
        "email": "test@example.com"
    }')

CONSUMER_ID=$(echo $CONSUMER_RESPONSE | jq -r '.id')

if [ "$CONSUMER_ID" != "null" ] && [ -n "$CONSUMER_ID" ]; then
    echo -e "${GREEN}✓ Created (ID: $CONSUMER_ID)${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAILED${NC}"
    ((TESTS_FAILED++))
fi

# Generate API key
echo -n "Generating API key... "
KEY_RESPONSE=$(curl -s -X POST "$ADMIN_API/consumers/$CONSUMER_ID/keys?name=Test%20Key")
API_KEY=$(echo $KEY_RESPONSE | jq -r '.key')

if [ "$API_KEY" != "null" ] && [[ $API_KEY == gw_* ]]; then
    echo -e "${GREEN}✓ Generated (Key: ${API_KEY:0:20}...)${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAILED${NC}"
    ((TESTS_FAILED++))
fi

# List API keys
run_test "List API keys" \
    "curl -s $ADMIN_API/consumers/$CONSUMER_ID/keys | jq 'length'" \
    "1"

echo ""

# Test 6: Plugins
echo "=== Plugins ==="

# List available plugins
run_test "List available plugins" \
    "curl -s $ADMIN_API/plugins/available | jq '.authentication | length'" \
    "3"

# Create global plugin
echo -n "Creating global plugin... "
PLUGIN_RESPONSE=$(curl -s -X POST $ADMIN_API/plugins \
    -H "Content-Type: application/json" \
    -d '{
        "name": "rate-limit",
        "scope": "global",
        "priority": 20,
        "config": {
            "limit": 1000,
            "window": "1m"
        }
    }')

PLUGIN_ID=$(echo $PLUGIN_RESPONSE | jq -r '.id')

if [ "$PLUGIN_ID" != "null" ] && [ -n "$PLUGIN_ID" ]; then
    echo -e "${GREEN}✓ Created (ID: $PLUGIN_ID)${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAILED${NC}"
    ((TESTS_FAILED++))
fi

# Create route-specific plugin
echo -n "Creating route plugin... "
ROUTE_PLUGIN_RESPONSE=$(curl -s -X POST $ADMIN_API/plugins \
    -H "Content-Type: application/json" \
    -d "{
        \"name\": \"cache\",
        \"scope\": \"route\",
        \"route_id\": \"$ROUTE_ID\",
        \"priority\": 30,
        \"config\": {
            \"ttl\": 300
        }
    }")

ROUTE_PLUGIN_ID=$(echo $ROUTE_PLUGIN_RESPONSE | jq -r '.id')

if [ "$ROUTE_PLUGIN_ID" != "null" ] && [ -n "$ROUTE_PLUGIN_ID" ]; then
    echo -e "${GREEN}✓ Created (ID: $ROUTE_PLUGIN_ID)${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAILED${NC}"
    ((TESTS_FAILED++))
fi

echo ""

# Test 7: Cleanup
echo "=== Cleanup ==="

echo -n "Deleting plugin... "
curl -s -X DELETE $ADMIN_API/plugins/$PLUGIN_ID > /dev/null
echo -e "${GREEN}✓${NC}"

echo -n "Deleting route plugin... "
curl -s -X DELETE $ADMIN_API/plugins/$ROUTE_PLUGIN_ID > /dev/null
echo -e "${GREEN}✓${NC}"

echo -n "Deleting route... "
curl -s -X DELETE $ADMIN_API/routes/$ROUTE_ID > /dev/null
sleep 2
echo -e "${GREEN}✓${NC}"

echo -n "Deleting consumer... "
curl -s -X DELETE $ADMIN_API/consumers/$CONSUMER_ID > /dev/null
echo -e "${GREEN}✓${NC}"

echo -n "Deleting service... "
curl -s -X DELETE $ADMIN_API/services/$SERVICE_ID > /dev/null
echo -e "${GREEN}✓${NC}"

echo ""

# Summary
echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}✗ Some tests failed${NC}"
    exit 1
fi
#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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
        echo -e "${YELLOW}  Expected: $expected${NC}"
        echo -e "${YELLOW}  Got: $result${NC}"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Cleanup function
cleanup_all() {
    echo -e "${BLUE}=== Pre-Test Cleanup ===${NC}"
    
    # Delete all test consumers
    echo -n "Cleaning up test consumers... "
    CONSUMERS=$(curl -s $ADMIN_API/consumers | jq -r '.[].id')
    for id in $CONSUMERS; do
        curl -s -X DELETE $ADMIN_API/consumers/$id > /dev/null 2>&1
    done
    echo -e "${GREEN}✓${NC}"
    
    # Delete all test plugins
    echo -n "Cleaning up test plugins... "
    PLUGINS=$(curl -s $ADMIN_API/plugins | jq -r '.[].id')
    for id in $PLUGINS; do
        curl -s -X DELETE $ADMIN_API/plugins/$id > /dev/null 2>&1
    done
    echo -e "${GREEN}✓${NC}"
    
    # Delete all test routes
    echo -n "Cleaning up test routes... "
    ROUTES=$(curl -s $ADMIN_API/routes | jq -r '.[].id')
    for id in $ROUTES; do
        curl -s -X DELETE $ADMIN_API/routes/$id > /dev/null 2>&1
    done
    echo -e "${GREEN}✓${NC}"
    
    # Delete all test services
    echo -n "Cleaning up test services... "
    SERVICES=$(curl -s $ADMIN_API/services | jq -r '.[].id')
    for id in $SERVICES; do
        curl -s -X DELETE $ADMIN_API/services/$id > /dev/null 2>&1
    done
    echo -e "${GREEN}✓${NC}"
    
    echo ""
    sleep 1
}

# Run cleanup before tests
cleanup_all

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

# Verify empty state
INITIAL_SERVICES=$(curl -s $ADMIN_API/services | jq 'length')
echo "Initial services count: $INITIAL_SERVICES"

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
    echo -e "${YELLOW}Response: $SERVICE_RESPONSE${NC}"
    ((TESTS_FAILED++))
fi

# List services
CURRENT_SERVICES=$(curl -s $ADMIN_API/services | jq 'length')
EXPECTED_SERVICES=$((INITIAL_SERVICES + 1))
run_test "List services (count should increase)" \
    "echo $CURRENT_SERVICES" \
    "$EXPECTED_SERVICES"

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

# Verify empty state
INITIAL_ROUTES=$(curl -s $ADMIN_API/routes | jq 'length')
echo "Initial routes count: $INITIAL_ROUTES"

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
    echo -e "${YELLOW}Response: $ROUTE_RESPONSE${NC}"
    ((TESTS_FAILED++))
fi

# List routes
CURRENT_ROUTES=$(curl -s $ADMIN_API/routes | jq 'length')
EXPECTED_ROUTES=$((INITIAL_ROUTES + 1))
run_test "List routes (count should increase)" \
    "echo $CURRENT_ROUTES" \
    "$EXPECTED_ROUTES"

echo ""

# Test 4: Hot Reload
echo "=== Hot Reload Test ==="

echo -n "Waiting for gateway to reload... "
sleep 2
echo "done"

# Test if route is accessible on gateway
echo -n "Testing route on gateway... "
GATEWAY_RESPONSE=$(curl -s $GATEWAY/api/test)

if [[ $GATEWAY_RESPONSE == *"matched"* ]] || [[ $GATEWAY_RESPONSE == *"test-route"* ]]; then
    echo -e "${GREEN}✓ Hot reload working!${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ Route not found on gateway${NC}"
    echo -e "${YELLOW}Response: $GATEWAY_RESPONSE${NC}"
    ((TESTS_FAILED++))
fi

# Update route
echo -n "Updating route paths... "
UPDATE_RESPONSE=$(curl -s -X PUT $ADMIN_API/routes/$ROUTE_ID \
    -H "Content-Type: application/json" \
    -d '{"paths": ["/api/test", "/api/test-v2"]}')

echo "done"
sleep 2

# Test new path
echo -n "Testing updated route... "
GATEWAY_RESPONSE=$(curl -s $GATEWAY/api/test-v2)

if [[ $GATEWAY_RESPONSE == *"matched"* ]] || [[ $GATEWAY_RESPONSE == *"test-route"* ]]; then
    echo -e "${GREEN}✓ Updated route working!${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ Updated route not working${NC}"
    echo -e "${YELLOW}Response: $GATEWAY_RESPONSE${NC}"
    ((TESTS_FAILED++))
fi

echo ""

# Test 5: Consumers & API Keys
echo "=== Consumers & API Keys ==="

# Verify empty state
INITIAL_CONSUMERS=$(curl -s $ADMIN_API/consumers | jq 'length')
echo "Initial consumers count: $INITIAL_CONSUMERS"

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
    echo -e "${YELLOW}Response: $CONSUMER_RESPONSE${NC}"
    ((TESTS_FAILED++))
    
    # Try to find existing consumer
    EXISTING_CONSUMER=$(curl -s $ADMIN_API/consumers | jq -r '.[] | select(.username=="test-app") | .id')
    if [ -n "$EXISTING_CONSUMER" ] && [ "$EXISTING_CONSUMER" != "null" ]; then
        echo -e "${YELLOW}  Using existing consumer: $EXISTING_CONSUMER${NC}"
        CONSUMER_ID=$EXISTING_CONSUMER
    fi
fi

# Only proceed with API key tests if we have a consumer ID
if [ -n "$CONSUMER_ID" ] && [ "$CONSUMER_ID" != "null" ]; then
    # Generate API key
    echo -n "Generating API key... "
    KEY_RESPONSE=$(curl -s -X POST "$ADMIN_API/consumers/$CONSUMER_ID/keys?name=Test%20Key")
    API_KEY=$(echo $KEY_RESPONSE | jq -r '.key')

    if [ "$API_KEY" != "null" ] && [[ $API_KEY == gw_* ]]; then
        echo -e "${GREEN}✓ Generated (Key: ${API_KEY:0:20}...)${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗ FAILED${NC}"
        echo -e "${YELLOW}Response: $KEY_RESPONSE${NC}"
        ((TESTS_FAILED++))
    fi

    # List API keys
    KEYS_COUNT=$(curl -s $ADMIN_API/consumers/$CONSUMER_ID/keys | jq 'length')
    if [ "$KEYS_COUNT" -ge 1 ]; then
        echo -e "Testing: List API keys... ${GREEN}✓ PASSED (Count: $KEYS_COUNT)${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "Testing: List API keys... ${RED}✗ FAILED${NC}"
        ((TESTS_FAILED++))
    fi
else
    echo -e "${RED}Skipping API key tests (no valid consumer)${NC}"
    ((TESTS_FAILED+=2))
fi

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
    echo -e "${YELLOW}Response: $PLUGIN_RESPONSE${NC}"
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
    echo -e "${YELLOW}Response: $ROUTE_PLUGIN_RESPONSE${NC}"
    ((TESTS_FAILED++))
fi

echo ""

# Test 7: Cleanup
echo "=== Cleanup ==="

echo -n "Deleting plugins... "
curl -s -X DELETE $ADMIN_API/plugins/$PLUGIN_ID > /dev/null 2>&1
curl -s -X DELETE $ADMIN_API/plugins/$ROUTE_PLUGIN_ID > /dev/null 2>&1
echo -e "${GREEN}✓${NC}"

echo -n "Deleting route... "
curl -s -X DELETE $ADMIN_API/routes/$ROUTE_ID > /dev/null 2>&1
sleep 2
echo -e "${GREEN}✓${NC}"

echo -n "Deleting consumer... "
if [ -n "$CONSUMER_ID" ] && [ "$CONSUMER_ID" != "null" ]; then
    curl -s -X DELETE $ADMIN_API/consumers/$CONSUMER_ID > /dev/null 2>&1
fi
echo -e "${GREEN}✓${NC}"

echo -n "Deleting service... "
curl -s -X DELETE $ADMIN_API/services/$SERVICE_ID > /dev/null 2>&1
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
    echo -e "${YELLOW}⚠ Some tests failed (may be due to leftover data)${NC}"
    echo -e "${BLUE}💡 Tip: Script now cleans up before running${NC}"
    exit 1
fi
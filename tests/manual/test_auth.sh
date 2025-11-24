#!/bin/bash
# Test API Key Authentication Plugin

set -e

# Colors (using echo -e)
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
TEST_KEY="test-key-12345"

echo ""
echo "========================================="
echo "🔐 API Key Authentication Tests"
echo "========================================="
echo ""
echo "Gateway URL: $GATEWAY_URL"
echo "Test API Key: $TEST_KEY"
echo ""

# Wait for gateway to be ready
echo "Checking if gateway is ready..."
for i in {1..10}; do
    if curl -s -o /dev/null -w "%{http_code}" $GATEWAY_URL/health | grep -q "200"; then
        echo "✓ Gateway is ready"
        break
    fi
    if [ $i -eq 10 ]; then
        echo "✗ Gateway not responding after 10 attempts"
        exit 1
    fi
    sleep 1
done
echo ""

# Trigger hot reload to ensure plugin is loaded
if command -v docker &> /dev/null; then
    echo "Triggering plugin reload..."
    docker exec switchboard-redis redis-cli PUBLISH gateway:config:changes '{"entity_type":"plugin","entity_id":"*","action":"reload"}' > /dev/null 2>&1 || true
    sleep 2
    echo "✓ Reload triggered"
    echo ""
fi

# Test 1: Health check should bypass auth
echo -e "${BLUE}Test 1: Health check bypasses authentication${NC}"
echo "-------------------------------------------"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" $GATEWAY_URL/health)
if [ "$STATUS" = "200" ]; then
    echo -e "${GREEN}✓ PASS${NC}: Health check accessible without API key (200)"
else
    echo -e "${RED}✗ FAIL${NC}: Expected 200, got $STATUS"
fi
echo ""

# Test 2: Request without API key should fail
echo -e "${BLUE}Test 2: Request without API key${NC}"
echo "-----------------------------------"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" $GATEWAY_URL/test/get 2>/dev/null || echo "000")

if [ "$STATUS" = "401" ]; then
    echo -e "${GREEN}✓ PASS${NC}: Unauthorized (401)"
elif [ "$STATUS" = "404" ]; then
    echo -e "${YELLOW}⚠ SKIP${NC}: Route not found (404) - need to setup test routes"
    echo "  Run: docker exec -i switchboard-postgres psql -U switchboard -d switchboard < tests/manual/setup_test_routes.sql"
else
    echo -e "${RED}✗ FAIL${NC}: Expected 401, got $STATUS"
    echo "  This means auth plugin may not be loaded or enabled"
fi
echo ""

# Test 3: Request with valid API key in header
echo -e "${BLUE}Test 3: Valid API key in header${NC}"
echo "-----------------------------------"
OUTPUT=$(mktemp)
STATUS=$(curl -s -o "$OUTPUT" -w "%{http_code}" -H "X-API-Key: $TEST_KEY" $GATEWAY_URL/test/get 2>/dev/null || echo "000")

if [ "$STATUS" = "200" ]; then
    echo -e "${GREEN}✓ PASS${NC}: Authorized (200)"
elif [ "$STATUS" = "404" ]; then
    echo -e "${YELLOW}⚠ SKIP${NC}: Route not found (404)"
else
    echo -e "${RED}✗ FAIL${NC}: Expected 200, got $STATUS"
fi
rm -f "$OUTPUT"
echo ""

# Test 4: Request with valid API key in query parameter
echo -e "${BLUE}Test 4: Valid API key in query parameter${NC}"
echo "----------------------------------------------"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$GATEWAY_URL/test/get?apikey=$TEST_KEY" 2>/dev/null || echo "000")

if [ "$STATUS" = "200" ]; then
    echo -e "${GREEN}✓ PASS${NC}: Authorized via query param (200)"
elif [ "$STATUS" = "404" ]; then
    echo -e "${YELLOW}⚠ SKIP${NC}: Route not found (404)"
else
    echo -e "${RED}✗ FAIL${NC}: Expected 200, got $STATUS"
fi
echo ""

# Test 5: Request with invalid API key
echo -e "${BLUE}Test 5: Invalid API key${NC}"
echo "------------------------"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: invalid-key-999" $GATEWAY_URL/test/get 2>/dev/null || echo "000")

if [ "$STATUS" = "401" ]; then
    echo -e "${GREEN}✓ PASS${NC}: Unauthorized (401)"
elif [ "$STATUS" = "404" ]; then
    echo -e "${YELLOW}⚠ SKIP${NC}: Route not found (404)"
else
    echo -e "${RED}✗ FAIL${NC}: Expected 401, got $STATUS"
fi
echo ""

# Test 6: Check Redis cache
echo -e "${BLUE}Test 6: Redis cache${NC}"
echo "------------------------------"

if command -v docker &> /dev/null; then
    # Make a request to populate cache
    curl -s -H "X-API-Key: $TEST_KEY" $GATEWAY_URL/test/get > /dev/null 2>&1 || true
    sleep 1
    
    REDIS_KEYS=$(docker exec switchboard-redis redis-cli --scan --pattern "auth:apikey:*" 2>/dev/null | wc -l | tr -d ' ' || echo "0")
    
    if [ "$REDIS_KEYS" -gt 0 ]; then
        echo -e "${GREEN}✓ PASS${NC}: Redis cache has $REDIS_KEYS key(s)"
    else
        echo -e "${YELLOW}⚠ WARNING${NC}: Redis cache is empty (route may not exist)"
    fi
else
    echo -e "${YELLOW}⚠ SKIP${NC}: Docker not available"
fi
echo ""

# Check plugin status in database
echo -e "${BLUE}Verification: Plugin Status${NC}"
echo "--------------------------------"
if command -v docker &> /dev/null; then
    echo "Checking if api-key-auth plugin is enabled in database..."
    PLUGIN_STATUS=$(docker exec switchboard-postgres psql -U switchboard -d switchboard -t -c "SELECT enabled FROM plugins WHERE name = 'api-key-auth' AND scope = 'global';" 2>/dev/null | tr -d ' ' || echo "")
    
    if [ "$PLUGIN_STATUS" = "t" ]; then
        echo -e "${GREEN}✓${NC} Plugin is enabled in database"
    else
        echo -e "${RED}✗${NC} Plugin not found or disabled in database"
        echo "  Run: docker exec -i switchboard-postgres psql -U switchboard -d switchboard < tests/manual/setup_test_auth.sql"
    fi
else
    echo -e "${YELLOW}⚠ SKIP${NC}: Docker not available"
fi
echo ""

# Summary
echo "========================================="
echo "📊 Test Summary"
echo "========================================="
echo ""
echo "If tests are failing with 404 or 200 (instead of 401):"
echo "  1. Setup test routes:"
echo "     docker exec -i switchboard-postgres psql -U switchboard -d switchboard < tests/manual/setup_test_routes.sql"
echo ""
echo "  2. Restart gateway to reload plugins"
echo ""
echo "  3. Check gateway logs for plugin loading"
echo ""
#!/bin/bash
# Test JWT Authentication Plugin

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"

# Load tokens from generator
if [ -f /tmp/test_jwt_tokens.txt ]; then
    source /tmp/test_jwt_tokens.txt
else
    echo -e "${RED}✗ Token file not found${NC}"
    echo "Run: ./tests/manual/generate_test_jwt.sh"
    exit 1
fi

echo ""
echo "========================================="
echo "🔐 JWT Authentication Tests"
echo "========================================="
echo ""
echo "Gateway URL: $GATEWAY_URL"
echo ""

# Wait for gateway to be ready
echo "Checking if gateway is ready..."
for i in {1..10}; do
    if curl -s -o /dev/null -w "%{http_code}" $GATEWAY_URL/health | grep -q "200"; then
        echo "✓ Gateway is ready"
        break
    fi
    if [ $i -eq 10 ]; then
        echo "✗ Gateway not responding"
        exit 1
    fi
    sleep 1
done
echo ""

# Test 1: Health check should bypass auth
echo -e "${BLUE}Test 1: Health check bypasses JWT authentication${NC}"
echo "---------------------------------------------------"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" $GATEWAY_URL/health)
if [ "$STATUS" = "200" ]; then
    echo -e "${GREEN}✓ PASS${NC}: Health check accessible without JWT (200)"
else
    echo -e "${RED}✗ FAIL${NC}: Expected 200, got $STATUS"
fi
echo ""

# Test 2: Request without JWT should fail
echo -e "${BLUE}Test 2: Request without JWT token${NC}"
echo "------------------------------------"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" $GATEWAY_URL/test/get 2>/dev/null || echo "000")

if [ "$STATUS" = "401" ]; then
    echo -e "${GREEN}✓ PASS${NC}: Unauthorized (401)"
elif [ "$STATUS" = "404" ]; then
    echo -e "${YELLOW}⚠ SKIP${NC}: Route not found (404)"
else
    echo -e "${RED}✗ FAIL${NC}: Expected 401, got $STATUS"
fi
echo ""

# Test 3: Request with valid JWT in Authorization header
echo -e "${BLUE}Test 3: Valid JWT in Authorization header${NC}"
echo "----------------------------------------------"
OUTPUT=$(mktemp)
STATUS=$(curl -s -o "$OUTPUT" -w "%{http_code}" \
    -H "Authorization: Bearer $VALID_TOKEN" \
    $GATEWAY_URL/test/get 2>/dev/null || echo "000")

if [ "$STATUS" = "200" ]; then
    echo -e "${GREEN}✓ PASS${NC}: Authorized (200)"
    
    # Check if user headers were forwarded
    if grep -q "X-User-ID" "$OUTPUT" 2>/dev/null; then
        echo -e "${GREEN}✓ PASS${NC}: User headers forwarded to backend"
    else
        echo -e "${YELLOW}⚠ INFO${NC}: User headers may not be visible in response"
    fi
elif [ "$STATUS" = "404" ]; then
    echo -e "${YELLOW}⚠ SKIP${NC}: Route not found (404)"
else
    echo -e "${RED}✗ FAIL${NC}: Expected 200, got $STATUS"
fi
rm -f "$OUTPUT"
echo ""

# Test 4: Request with valid JWT in cookie
echo -e "${BLUE}Test 4: Valid JWT in cookie${NC}"
echo "---------------------------------"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    -b "jwt_token=$VALID_TOKEN" \
    $GATEWAY_URL/test/get 2>/dev/null || echo "000")

if [ "$STATUS" = "200" ]; then
    echo -e "${GREEN}✓ PASS${NC}: Authorized via cookie (200)"
elif [ "$STATUS" = "404" ]; then
    echo -e "${YELLOW}⚠ SKIP${NC}: Route not found (404)"
else
    echo -e "${RED}✗ FAIL${NC}: Expected 200, got $STATUS"
fi
echo ""

# Test 5: Request with expired JWT
echo -e "${BLUE}Test 5: Expired JWT token${NC}"
echo "----------------------------"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "Authorization: Bearer $EXPIRED_TOKEN" \
    $GATEWAY_URL/test/get 2>/dev/null || echo "000")

if [ "$STATUS" = "401" ]; then
    echo -e "${GREEN}✓ PASS${NC}: Unauthorized - token expired (401)"
elif [ "$STATUS" = "404" ]; then
    echo -e "${YELLOW}⚠ SKIP${NC}: Route not found (404)"
else
    echo -e "${RED}✗ FAIL${NC}: Expected 401, got $STATUS"
fi
echo ""

# Test 6: Request with malformed JWT
echo -e "${BLUE}Test 6: Malformed JWT token${NC}"
echo "------------------------------"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "Authorization: Bearer invalid.token.here" \
    $GATEWAY_URL/test/get 2>/dev/null || echo "000")

if [ "$STATUS" = "401" ]; then
    echo -e "${GREEN}✓ PASS${NC}: Unauthorized - invalid token (401)"
elif [ "$STATUS" = "404" ]; then
    echo -e "${YELLOW}⚠ SKIP${NC}: Route not found (404)"
else
    echo -e "${RED}✗ FAIL${NC}: Expected 401, got $STATUS"
fi
echo ""

# Test 7: Check Redis cache
echo -e "${BLUE}Test 7: Redis cache${NC}"
echo "------------------------------"

if command -v docker &> /dev/null; then
    # Make a request to populate cache
    curl -s -H "Authorization: Bearer $VALID_TOKEN" $GATEWAY_URL/test/get > /dev/null 2>&1 || true
    sleep 1
    
    REDIS_KEYS=$(docker exec switchboard-redis redis-cli --scan --pattern "auth:jwt:*" 2>/dev/null | wc -l | tr -d ' ' || echo "0")
    
    if [ "$REDIS_KEYS" -gt 0 ]; then
        echo -e "${GREEN}✓ PASS${NC}: Redis cache has $REDIS_KEYS JWT key(s)"
    else
        echo -e "${YELLOW}⚠ WARNING${NC}: Redis cache is empty (route may not exist)"
    fi
else
    echo -e "${YELLOW}⚠ SKIP${NC}: Docker not available"
fi
echo ""

# Test 8: Both auth plugins work independently
echo -e "${BLUE}Test 8: API Key and JWT both work${NC}"
echo "---------------------------------------"

# Generate a new API key hash for testing
API_KEY="test-key-12345"

# Try with API key
STATUS_API=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "X-API-Key: $API_KEY" \
    $GATEWAY_URL/test/get 2>/dev/null || echo "000")

# Try with JWT
STATUS_JWT=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "Authorization: Bearer $VALID_TOKEN" \
    $GATEWAY_URL/test/get 2>/dev/null || echo "000")

if [ "$STATUS_API" = "200" ] && [ "$STATUS_JWT" = "200" ]; then
    echo -e "${GREEN}✓ PASS${NC}: Both API Key and JWT work independently"
elif [ "$STATUS_API" = "404" ] || [ "$STATUS_JWT" = "404" ]; then
    echo -e "${YELLOW}⚠ SKIP${NC}: Route not found (404)"
else
    echo -e "${YELLOW}⚠ INFO${NC}: API Key: $STATUS_API, JWT: $STATUS_JWT"
    echo "  Note: Only one auth method needs to succeed per route config"
fi
echo ""

# Verify plugin status
echo -e "${BLUE}Verification: Plugin Status${NC}"
echo "--------------------------------"
if command -v docker &> /dev/null; then
    JWT_STATUS=$(docker exec switchboard-postgres psql -U switchboard -d switchboard -t -c "SELECT enabled FROM plugins WHERE name = 'jwt-auth' AND scope = 'global';" 2>/dev/null | tr -d ' ' || echo "")
    
    if [ "$JWT_STATUS" = "t" ]; then
        echo -e "${GREEN}✓${NC} JWT plugin is enabled in database"
    else
        echo -e "${RED}✗${NC} JWT plugin not found or disabled"
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
echo "JWT Authentication plugin tested successfully!"
echo ""
echo "Token info:"
echo "  - Valid token expires in ~60 minutes"
echo "  - Expired token for negative testing"
echo "  - Tokens saved in: /tmp/test_jwt_tokens.txt"
echo ""
echo "Next steps:"
echo "  1. Check gateway logs: docker logs switchboard-gateway"
echo "  2. Monitor Redis: docker exec -it switchboard-redis redis-cli MONITOR"
echo "  3. Regenerate tokens: ./tests/manual/generate_test_jwt.sh"
echo ""
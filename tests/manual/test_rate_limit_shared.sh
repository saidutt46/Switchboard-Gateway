#!/bin/bash
# Test Rate Limiting with Shared Identifier
#
# This test verifies that the "shared" identifier causes all users
# to share a single rate limit bucket (total limit).

set -e

ADMIN_API=${ADMIN_API:-http://localhost:8000}
GATEWAY=${GATEWAY:-http://localhost:8080}

echo "🧪 Rate Limiting: Shared Identifier Test"
echo "========================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Step 1: Clean up any existing rate limit plugins
echo "Step 1: Cleaning up existing rate limit plugins..."
PLUGIN_IDS=$(curl -s "$ADMIN_API/plugins" | jq -r '.[] | select(.name == "rate-limit") | .id')
for id in $PLUGIN_IDS; do
  curl -s -X DELETE "$ADMIN_API/plugins/$id" > /dev/null
  echo "  Deleted plugin: $id"
done
sleep 1

# Step 2: Clear Redis
echo ""
echo "Step 2: Clearing Redis rate limit keys..."
docker exec switchboard-redis redis-cli --scan --pattern "rate_limit:*" | \
  xargs -r docker exec -i switchboard-redis redis-cli DEL > /dev/null 2>&1 || true
echo "  ✓ Redis cleared"
sleep 1

# Step 3: Create shared rate limit (10 req/min TOTAL)
echo ""
echo "Step 3: Creating shared rate limit (10 req/min TOTAL)..."
PLUGIN_RESPONSE=$(curl -s -X POST "$ADMIN_API/plugins" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "rate-limit",
    "scope": "global",
    "config": {
      "algorithm": "token-bucket",
      "limit": 10,
      "window": "1m",
      "identifier": "shared",
      "redis_url": "redis://localhost:6379/0",
      "headers": true,
      "response_code": 429
    },
    "enabled": true,
    "priority": 10
  }')

PLUGIN_ID=$(echo "$PLUGIN_RESPONSE" | jq -r '.id')
echo "  ✓ Plugin created: $PLUGIN_ID"
echo "  Config: 10 requests/minute TOTAL (all users share)"
sleep 2

# Step 4: Test with requests from "different IPs" (simulated)
echo ""
echo "Step 4: Testing shared bucket behavior..."
echo "  Simulating 3 different users (X-Forwarded-For headers)"
echo ""

# Simulate 3 different users
USER_1_IP="1.2.3.4"
USER_2_IP="5.6.7.8"
USER_3_IP="9.10.11.12"

# Each user tries 5 requests (15 total)
# With shared bucket (10 req/min), we expect:
# - Requests 1-10: Success (200)
# - Requests 11-15: Denied (429)

ALL_RESULTS=()
ALLOWED_COUNT=0
DENIED_COUNT=0

echo "User 1 (IP $USER_1_IP) - 5 requests:"
for i in {1..5}; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "X-Forwarded-For: $USER_1_IP" \
    "$GATEWAY/test/get")
  
  if [ "$STATUS" = "200" ]; then
    echo -e "  Request $i: ${GREEN}$STATUS${NC}"
    ALLOWED_COUNT=$((ALLOWED_COUNT + 1))
  else
    echo -e "  Request $i: ${RED}$STATUS${NC}"
    DENIED_COUNT=$((DENIED_COUNT + 1))
  fi
  ALL_RESULTS+=("$STATUS")
done

echo ""
echo "User 2 (IP $USER_2_IP) - 5 requests:"
for i in {1..5}; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "X-Forwarded-For: $USER_2_IP" \
    "$GATEWAY/test/get")
  
  if [ "$STATUS" = "200" ]; then
    echo -e "  Request $i: ${GREEN}$STATUS${NC}"
    ALLOWED_COUNT=$((ALLOWED_COUNT + 1))
  else
    echo -e "  Request $i: ${RED}$STATUS${NC}"
    DENIED_COUNT=$((DENIED_COUNT + 1))
  fi
  ALL_RESULTS+=("$STATUS")
done

echo ""
echo "User 3 (IP $USER_3_IP) - 5 requests:"
for i in {1..5}; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "X-Forwarded-For: $USER_3_IP" \
    "$GATEWAY/test/get")
  
  if [ "$STATUS" = "200" ]; then
    echo -e "  Request $i: ${GREEN}$STATUS${NC}"
    ALLOWED_COUNT=$((ALLOWED_COUNT + 1))
  else
    echo -e "  Request $i: ${RED}$STATUS${NC}"
    DENIED_COUNT=$((DENIED_COUNT + 1))
  fi
  ALL_RESULTS+=("$STATUS")
done

# Step 5: Verify results
echo ""
echo "Step 5: Verifying shared bucket behavior..."
echo "=========================================="
echo ""
echo "Total requests: 15 (3 users × 5 requests)"
echo "Rate limit: 10 requests/minute TOTAL (shared)"
echo ""
echo "Results:"
echo "  ✓ Allowed (200): $ALLOWED_COUNT"
echo "  ✗ Denied (429): $DENIED_COUNT"
echo ""

# Expected: 10-11 allowed (Token Bucket), 4-5 denied
EXPECTED_MIN_ALLOWED=10
EXPECTED_MAX_ALLOWED=11
EXPECTED_MIN_DENIED=4
EXPECTED_MAX_DENIED=5

SUCCESS=true

if [ "$ALLOWED_COUNT" -ge "$EXPECTED_MIN_ALLOWED" ] && [ "$ALLOWED_COUNT" -le "$EXPECTED_MAX_ALLOWED" ]; then
  echo -e "${GREEN}✓ PASS${NC}: Allowed count correct ($ALLOWED_COUNT, expected $EXPECTED_MIN_ALLOWED-$EXPECTED_MAX_ALLOWED)"
else
  echo -e "${RED}✗ FAIL${NC}: Allowed count incorrect ($ALLOWED_COUNT, expected $EXPECTED_MIN_ALLOWED-$EXPECTED_MAX_ALLOWED)"
  SUCCESS=false
fi

if [ "$DENIED_COUNT" -ge "$EXPECTED_MIN_DENIED" ] && [ "$DENIED_COUNT" -le "$EXPECTED_MAX_DENIED" ]; then
  echo -e "${GREEN}✓ PASS${NC}: Denied count correct ($DENIED_COUNT, expected $EXPECTED_MIN_DENIED-$EXPECTED_MAX_DENIED)"
else
  echo -e "${RED}✗ FAIL${NC}: Denied count incorrect ($DENIED_COUNT, expected $EXPECTED_MIN_DENIED-$EXPECTED_MAX_DENIED)"
  SUCCESS=false
fi

# Step 6: Verify Redis key
echo ""
echo "Step 6: Verifying shared Redis key..."
REDIS_KEYS=$(docker exec switchboard-redis redis-cli --scan --pattern "rate_limit:*")
KEY_COUNT=$(echo "$REDIS_KEYS" | wc -l)

echo "  Redis keys found: $KEY_COUNT"
echo "  Keys:"
echo "$REDIS_KEYS" | sed 's/^/    /'

if [ "$KEY_COUNT" -eq 1 ]; then
  echo -e "${GREEN}✓ PASS${NC}: Only 1 Redis key (all users shared)"
  
  # Check if key contains "shared"
  if echo "$REDIS_KEYS" | grep -q "shared"; then
    echo -e "${GREEN}✓ PASS${NC}: Key contains 'shared' identifier"
  else
    echo -e "${RED}✗ FAIL${NC}: Key does not contain 'shared' identifier"
    SUCCESS=false
  fi
else
  echo -e "${RED}✗ FAIL${NC}: Expected 1 Redis key, found $KEY_COUNT (users not sharing bucket!)"
  SUCCESS=false
fi

# Step 7: Compare with per-IP behavior
echo ""
echo "Step 7: Testing per-IP for comparison..."
echo "=========================================="

# Update plugin to per-IP
curl -s -X PUT "$ADMIN_API/plugins/$PLUGIN_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "rate-limit",
    "scope": "global",
    "config": {
      "algorithm": "token-bucket",
      "limit": 10,
      "window": "1m",
      "identifier": "ip",
      "redis_url": "redis://localhost:6379/0",
      "headers": true,
      "response_code": 429
    },
    "enabled": true,
    "priority": 10
  }' > /dev/null

echo "  Updated to per-IP identifier"
sleep 2

# Clear Redis
docker exec switchboard-redis redis-cli --scan --pattern "rate_limit:*" | \
  xargs -r docker exec -i switchboard-redis redis-cli DEL > /dev/null 2>&1 || true

# Test with same 3 IPs
PER_IP_ALLOWED=0
for ip in "$USER_1_IP" "$USER_2_IP" "$USER_3_IP"; do
  for i in {1..5}; do
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
      -H "X-Forwarded-For: $ip" \
      "$GATEWAY/test/get")
    
    if [ "$STATUS" = "200" ]; then
      PER_IP_ALLOWED=$((PER_IP_ALLOWED + 1))
    fi
  done
done

echo ""
echo "Per-IP Results:"
echo "  ✓ Allowed: $PER_IP_ALLOWED / 15"
echo ""

if [ "$PER_IP_ALLOWED" -gt "$ALLOWED_COUNT" ]; then
  echo -e "${GREEN}✓ PASS${NC}: Per-IP allows more requests than shared ($PER_IP_ALLOWED vs $ALLOWED_COUNT)"
  echo "  This proves shared bucket is working correctly!"
else
  echo -e "${YELLOW}⚠ WARNING${NC}: Per-IP didn't allow more requests"
fi

# Check Redis keys for per-IP
REDIS_KEYS_PER_IP=$(docker exec switchboard-redis redis-cli --scan --pattern "rate_limit:*")
KEY_COUNT_PER_IP=$(echo "$REDIS_KEYS_PER_IP" | wc -l)

echo ""
echo "  Redis keys (per-IP): $KEY_COUNT_PER_IP"
if [ "$KEY_COUNT_PER_IP" -eq 3 ]; then
  echo -e "${GREEN}✓ PASS${NC}: 3 Redis keys (one per IP)"
else
  echo -e "${YELLOW}⚠ WARNING${NC}: Expected 3 keys, found $KEY_COUNT_PER_IP"
fi

# Cleanup
echo ""
echo "Step 8: Cleanup..."
curl -s -X DELETE "$ADMIN_API/plugins/$PLUGIN_ID" > /dev/null
docker exec switchboard-redis redis-cli --scan --pattern "rate_limit:*" | \
  xargs -r docker exec -i switchboard-redis redis-cli DEL > /dev/null 2>&1 || true
echo "  ✓ Cleanup complete"

# Final result
echo ""
echo "=========================================="
if [ "$SUCCESS" = true ]; then
  echo -e "${GREEN}✅ ALL TESTS PASSED${NC}"
  echo ""
  echo "Shared identifier is working correctly!"
  echo "  - All users share one bucket"
  echo "  - Total limit enforced (10 req/min)"
  echo "  - Only 1 Redis key created"
  exit 0
else
  echo -e "${RED}❌ SOME TESTS FAILED${NC}"
  echo ""
  echo "Please review the failures above."
  exit 1
fi
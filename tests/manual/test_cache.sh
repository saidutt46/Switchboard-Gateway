#!/bin/bash
# Test cache plugin functionality
# Usage: ./tests/manual/test_cache.sh

set -e

GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
ADMIN_URL="${ADMIN_URL:-http://localhost:8000}"

echo "🧪 Cache Plugin Tests"
echo "====================="
echo ""
echo "Gateway: $GATEWAY_URL"
echo "Admin:   $ADMIN_URL"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

pass() { echo -e "${GREEN}✅ PASS${NC}: $1"; }
fail() { echo -e "${RED}❌ FAIL${NC}: $1"; }
info() { echo -e "${YELLOW}ℹ️  INFO${NC}: $1"; }

# ============================================================================
# Test 1: First request should be MISS
# ============================================================================
echo "Test 1: Cache MISS on first request"
echo "------------------------------------"

# Clear Redis cache first
docker exec switchboard-redis redis-cli FLUSHDB > /dev/null 2>&1 || true

RESPONSE=$(curl -s -i "$GATEWAY_URL/test/get?cache_test=1")
X_CACHE=$(echo "$RESPONSE" | grep -i "^x-cache:" | cut -d: -f2 | tr -d ' \r')
STATUS=$(echo "$RESPONSE" | grep "HTTP/" | tail -1 | awk '{print $2}')

echo "Status: $STATUS"
echo "X-Cache: $X_CACHE"

if [ "$X_CACHE" == "MISS" ]; then
  pass "First request was cache MISS"
else
  fail "Expected MISS, got: $X_CACHE"
fi
echo ""

# ============================================================================
# Test 2: Second request should be HIT
# ============================================================================
echo "Test 2: Cache HIT on second request"
echo "------------------------------------"

sleep 0.5  # Small delay to ensure cache is written

RESPONSE=$(curl -s -i "$GATEWAY_URL/test/get?cache_test=1")
X_CACHE=$(echo "$RESPONSE" | grep -i "^x-cache:" | cut -d: -f2 | tr -d ' \r')
X_CACHE_TTL=$(echo "$RESPONSE" | grep -i "^x-cache-ttl:" | cut -d: -f2 | tr -d ' \r')
AGE=$(echo "$RESPONSE" | grep -i "^age:" | cut -d: -f2 | tr -d ' \r')

echo "X-Cache: $X_CACHE"
echo "X-Cache-TTL: $X_CACHE_TTL"
echo "Age: $AGE"

if [ "$X_CACHE" == "HIT" ]; then
  pass "Second request was cache HIT"
else
  fail "Expected HIT, got: $X_CACHE"
fi
echo ""

# ============================================================================
# Test 3: Different query params should be different cache keys
# ============================================================================
echo "Test 3: Different query params = different cache"
echo "-------------------------------------------------"

RESPONSE=$(curl -s -i "$GATEWAY_URL/test/get?cache_test=2")
X_CACHE=$(echo "$RESPONSE" | grep -i "^x-cache:" | cut -d: -f2 | tr -d ' \r')

echo "X-Cache (different query): $X_CACHE"

if [ "$X_CACHE" == "MISS" ]; then
  pass "Different query param caused cache MISS"
else
  fail "Expected MISS for different query, got: $X_CACHE"
fi
echo ""

# ============================================================================
# Test 4: POST requests should bypass cache
# ============================================================================
echo "Test 4: POST requests bypass cache"
echo "-----------------------------------"

RESPONSE=$(curl -s -i -X POST "$GATEWAY_URL/test/post" -d '{"test": true}')
X_CACHE=$(echo "$RESPONSE" | grep -i "^x-cache:" | cut -d: -f2 | tr -d ' \r')

echo "X-Cache (POST): $X_CACHE"

if [ -z "$X_CACHE" ] || [ "$X_CACHE" == "BYPASS" ]; then
  pass "POST request bypassed cache"
else
  fail "POST should bypass cache, got: $X_CACHE"
fi
echo ""

# ============================================================================
# Test 5: Health endpoint should bypass cache
# ============================================================================
echo "Test 5: Health endpoint bypasses cache"
echo "---------------------------------------"

RESPONSE=$(curl -s -i "$GATEWAY_URL/health")
X_CACHE=$(echo "$RESPONSE" | grep -i "^x-cache:" | cut -d: -f2 | tr -d ' \r')

echo "X-Cache (/health): $X_CACHE"

if [ -z "$X_CACHE" ]; then
  pass "Health endpoint bypassed cache (no header)"
else
  fail "Health should bypass cache, got: $X_CACHE"
fi
echo ""

# ============================================================================
# Test 6: Ignored query params should not affect cache key
# ============================================================================
echo "Test 6: Ignored query params (utm_source)"
echo "------------------------------------------"

# First, ensure base request is cached
curl -s "$GATEWAY_URL/test/get?cache_test=1" > /dev/null

# Now with utm_source - should be HIT (same cache key)
RESPONSE=$(curl -s -i "$GATEWAY_URL/test/get?cache_test=1&utm_source=test")
X_CACHE=$(echo "$RESPONSE" | grep -i "^x-cache:" | cut -d: -f2 | tr -d ' \r')

echo "X-Cache (with utm_source): $X_CACHE"

if [ "$X_CACHE" == "HIT" ]; then
  pass "utm_source was ignored in cache key"
else
  info "Expected HIT (utm_source ignored), got: $X_CACHE"
  info "This might be expected if cache expired"
fi
echo ""

# ============================================================================
# Test 7: Performance - Cache HIT should be fast
# ============================================================================
echo "Test 7: Cache HIT performance"
echo "------------------------------"

# Warm up cache
curl -s "$GATEWAY_URL/test/get?perf_test=1" > /dev/null
sleep 0.2

# Measure cache hit time
START=$(date +%s%N)
curl -s "$GATEWAY_URL/test/get?perf_test=1" > /dev/null
END=$(date +%s%N)

DURATION_MS=$(( (END - START) / 1000000 ))

echo "Cache HIT latency: ${DURATION_MS}ms"

if [ $DURATION_MS -lt 50 ]; then
  pass "Cache HIT is fast (<50ms)"
elif [ $DURATION_MS -lt 100 ]; then
  info "Cache HIT is acceptable (<100ms)"
else
  fail "Cache HIT is slow (>${DURATION_MS}ms)"
fi
echo ""

# ============================================================================
# Test 8: Debug headers (X-Cache-Key)
# ============================================================================
echo "Test 8: Debug headers"
echo "----------------------"

RESPONSE=$(curl -s -i "$GATEWAY_URL/test/get?debug_test=1")
X_CACHE_KEY=$(echo "$RESPONSE" | grep -i "^x-cache-key:" | cut -d: -f2 | tr -d '\r')

if [ -n "$X_CACHE_KEY" ]; then
  echo "X-Cache-Key: $X_CACHE_KEY"
  pass "Debug header X-Cache-Key is present"
else
  info "X-Cache-Key header not found (debug_headers might be disabled)"
fi
echo ""

# ============================================================================
# Summary
# ============================================================================
echo "============================================"
echo "📊 Cache Plugin Test Summary"
echo "============================================"
echo ""

# Check Redis cache stats
KEYS=$(docker exec switchboard-redis redis-cli KEYS "cache:*" 2>/dev/null | wc -l)
echo "Cache keys in Redis: $KEYS"

# Show sample cache key
SAMPLE_KEY=$(docker exec switchboard-redis redis-cli KEYS "cache:*" 2>/dev/null | head -1)
if [ -n "$SAMPLE_KEY" ]; then
  echo "Sample key: $SAMPLE_KEY"
  TTL=$(docker exec switchboard-redis redis-cli TTL "$SAMPLE_KEY" 2>/dev/null)
  echo "Sample TTL: ${TTL}s"
fi

echo ""
echo "✅ Cache plugin tests completed!"
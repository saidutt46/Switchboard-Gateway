#!/bin/bash

echo "🧪 Rate Limiting Tests (Corrected)"
echo "=================================="
echo ""

# Reset rate limit before testing
echo "Resetting rate limit..."
docker exec switchboard-redis redis-cli DEL "rate_limit:token-bucket:ip:::1"
sleep 1

# Test 1: Rapid fire (no delays)
echo "Test 1: Rapid fire (should allow exactly 10)"
echo "--------------------------------------------"
RESULTS=""
for i in {1..15}; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/test/get)
  RESULTS="$RESULTS$STATUS "
  echo -n "$STATUS "
done
echo ""
echo ""

# Count 200s and 429s
COUNT_200=$(echo "$RESULTS" | grep -o "200" | wc -l)
COUNT_429=$(echo "$RESULTS" | grep -o "429" | wc -l)
echo "✅ Allowed: $COUNT_200 (expected: 10)"
echo "❌ Denied: $COUNT_429 (expected: 5)"
echo ""

# Wait for reset
echo "Waiting 60 seconds for rate limit reset..."
sleep 60

# Test 2: With delays (should see gradual refill)
echo "Test 2: With 0.5s delays (should see refill)"
echo "---------------------------------------------"
for i in {1..15}; do
  # Single request, capture both status and headers
  RESPONSE=$(curl -s -i http://localhost:8080/test/get)
  STATUS=$(echo "$RESPONSE" | grep "HTTP/" | tail -1 | awk '{print $2}')
  REMAINING=$(echo "$RESPONSE" | grep -i "^x-ratelimit-remaining:" | cut -d: -f2 | tr -d ' \r')
  
  echo "Request $i: HTTP $STATUS | Remaining: $REMAINING tokens"
  sleep 0.5
done
echo ""
echo "✅ Tests complete!"
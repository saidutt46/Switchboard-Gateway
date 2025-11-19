#!/bin/bash
# Switch Rate Limit Algorithm

ALGORITHM=$1
ADMIN_API=${ADMIN_API:-http://localhost:8000}

if [ -z "$ALGORITHM" ]; then
  echo "Usage: ./switch_algorithm.sh [token-bucket|sliding-window]"
  exit 1
fi

if [ "$ALGORITHM" != "token-bucket" ] && [ "$ALGORITHM" != "sliding-window" ]; then
  echo "Error: Algorithm must be 'token-bucket' or 'sliding-window'"
  exit 1
fi

echo "🔄 Switching to $ALGORITHM algorithm..."

# Get plugin ID
PLUGIN_ID=$(curl -s "$ADMIN_API/plugins" | jq -r '.[] | select(.name == "rate-limit") | .id')

if [ -z "$PLUGIN_ID" ]; then
  echo "❌ Error: Rate limit plugin not found"
  exit 1
fi

# Update plugin
curl -X PUT "$ADMIN_API/plugins/$PLUGIN_ID" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"rate-limit\",
    \"scope\": \"global\",
    \"config\": {
      \"algorithm\": \"$ALGORITHM\",
      \"limit\": 10,
      \"window\": \"1m\",
      \"identifier\": \"auto\",
      \"redis_url\": \"redis://localhost:6379/0\",
      \"key_prefix\": \"rate_limit:\",
      \"headers\": true,
      \"response_code\": 429,
      \"response_message\": \"Too many requests - please try again later\"
    },
    \"enabled\": true,
    \"priority\": 10
  }" | jq

echo ""
echo "✅ Algorithm switched to: $ALGORITHM"
echo "⏳ Waiting 2 seconds for hot reload..."
sleep 2

# Clear Redis keys
echo "🧹 Clearing rate limit keys from Redis..."
docker exec switchboard-redis redis-cli --scan --pattern "rate_limit:*" | xargs -r docker exec -i switchboard-redis redis-cli DEL

echo "✅ Ready to test!"
#!/bin/bash
# Rate limit test — sends 12 requests, expects 429 after the limit is hit

API_KEY="${1:-gw_prod_5gJ7IMOYyYSJxA9A2gE5O5OPWa2hkK8Yo4UHKTcN_s4}"

echo "Testing rate limit with key: ${API_KEY:0:20}..."
echo "---"

for i in $(seq 1 12); do
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: $API_KEY" http://localhost:8080/get)
    if [ "$STATUS" = "200" ]; then
        echo "Request $i: $STATUS OK"
    elif [ "$STATUS" = "429" ]; then
        echo "Request $i: $STATUS RATE LIMITED"
    else
        echo "Request $i: $STATUS"
    fi
done

echo "---"
echo "Done. Requests after the limit should show 429."

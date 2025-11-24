#!/bin/bash
# Generate test JWT tokens for testing

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

SECRET_KEY="my-test-secret-key-12345"

echo ""
echo "========================================="
echo "🔑 JWT Token Generator"
echo "========================================="
echo ""

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo -e "${YELLOW}⚠ WARNING: jq not installed${NC}"
    echo "Install with: brew install jq (macOS) or apt-get install jq (Linux)"
    echo ""
fi

# Function to generate JWT token
generate_jwt() {
    local user_id=$1
    local username=$2
    local email=$3
    local exp_minutes=$4
    
    # Calculate timestamps
    local now=$(date +%s)
    local exp=$((now + exp_minutes * 60))
    
    # Create header
    local header='{"alg":"HS256","typ":"JWT"}'
    local header_b64=$(echo -n "$header" | openssl base64 -e -A | tr '+/' '-_' | tr -d '=')
    
    # Create payload
    local payload=$(jq -n \
        --arg user_id "$user_id" \
        --arg username "$username" \
        --arg email "$email" \
        --arg iss "test-gateway" \
        --arg aud "test-api" \
        --argjson iat $now \
        --argjson exp $exp \
        '{user_id: $user_id, username: $username, email: $email, iss: $iss, aud: $aud, iat: $iat, exp: $exp}')
    
    local payload_b64=$(echo -n "$payload" | openssl base64 -e -A | tr '+/' '-_' | tr -d '=')
    
    # Create signature
    local signature=$(echo -n "${header_b64}.${payload_b64}" | openssl dgst -sha256 -hmac "$SECRET_KEY" -binary | openssl base64 -e -A | tr '+/' '-_' | tr -d '=')
    
    # Combine to create JWT
    echo "${header_b64}.${payload_b64}.${signature}"
}

# Generate valid token (expires in 60 minutes)
echo -e "${BLUE}Generating VALID token (expires in 60 minutes)...${NC}"
VALID_TOKEN=$(generate_jwt "user-123" "testuser" "test@example.com" 60)
echo ""
echo -e "${GREEN}Valid Token:${NC}"
echo "$VALID_TOKEN"
echo ""

# Generate expired token (expired 5 minutes ago)
echo -e "${BLUE}Generating EXPIRED token (for testing)...${NC}"
EXPIRED_TOKEN=$(generate_jwt "user-456" "expireduser" "expired@example.com" -5)
echo ""
echo -e "${YELLOW}Expired Token:${NC}"
echo "$EXPIRED_TOKEN"
echo ""

# Save tokens to file for easy access
cat > /tmp/test_jwt_tokens.txt << EOF
# Test JWT Tokens
# Generated: $(date)
# Secret Key: $SECRET_KEY

# Valid Token (expires in 60 minutes)
VALID_TOKEN="$VALID_TOKEN"

# Expired Token (for negative testing)
EXPIRED_TOKEN="$EXPIRED_TOKEN"

# Usage Examples:
# curl -H "Authorization: Bearer \$VALID_TOKEN" http://localhost:8080/test/get
# curl -b "jwt_token=\$VALID_TOKEN" http://localhost:8080/test/get
EOF

echo "========================================="
echo "✅ Tokens saved to: /tmp/test_jwt_tokens.txt"
echo ""
echo -e "${GREEN}Quick Test:${NC}"
echo "export VALID_TOKEN=\"$VALID_TOKEN\""
echo "curl -H \"Authorization: Bearer \$VALID_TOKEN\" http://localhost:8080/test/get"
echo ""
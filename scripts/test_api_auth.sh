#!/bin/bash

# Test API Authentication

API_BASE="http://localhost:8080"
EMAIL="koopa@assistant.local"
PASSWORD="KoopaAssistant2024!"

echo "🔐 Testing API Authentication..."
echo "================================"

# Test 1: Login
echo -n "1. Testing login endpoint... "
LOGIN_RESPONSE=$(curl -s -X POST $API_BASE/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")

ACCESS_TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.accessToken // empty')

if [ -n "$ACCESS_TOKEN" ]; then
  echo "✅ Success!"
  echo "   Access Token: ${ACCESS_TOKEN:0:20}..."
else
  echo "❌ Failed!"
  echo "   Response: $LOGIN_RESPONSE"
  exit 1
fi

# Test 2: Access protected endpoint without token
echo -n "2. Testing protected endpoint without token... "
UNAUTH_RESPONSE=$(curl -s -w "\n%{http_code}" $API_BASE/api/v1/conversations)
HTTP_CODE=$(echo "$UNAUTH_RESPONSE" | tail -n 1)

if [ "$HTTP_CODE" = "401" ]; then
  echo "✅ Correctly rejected (401)"
else
  echo "❌ Expected 401, got $HTTP_CODE"
fi

# Test 3: Access protected endpoint with token
echo -n "3. Testing protected endpoint with token... "
AUTH_RESPONSE=$(curl -s -w "\n%{http_code}" $API_BASE/api/v1/conversations \
  -H "Authorization: Bearer $ACCESS_TOKEN")
HTTP_CODE=$(echo "$AUTH_RESPONSE" | tail -n 1)

if [ "$HTTP_CODE" = "200" ]; then
  echo "✅ Success!"
else
  echo "❌ Failed with status $HTTP_CODE"
  echo "   Response: $AUTH_RESPONSE"
fi

# Test 4: Query endpoint with authentication
echo -n "4. Testing query endpoint with auth... "
QUERY_RESPONSE=$(curl -s -X POST $API_BASE/api/query \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query":"Hello, assistant!"}')

if echo "$QUERY_RESPONSE" | jq -e '.response' > /dev/null 2>&1; then
  echo "✅ Success!"
  echo "   Response: $(echo $QUERY_RESPONSE | jq -r '.response' | head -c 50)..."
else
  echo "❌ Failed!"
  echo "   Response: $QUERY_RESPONSE"
fi

echo ""
echo "✨ API Authentication test complete!"
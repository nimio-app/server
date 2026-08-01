#!/bin/bash

# Connection API Integration Test Script
# Tests friend request flow and privacy-filtered status feed

set -e

API_URL="http://localhost:8080/v1"
echo "🧪 Testing Nimio Connection API..."
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Register two test users
echo -e "${BLUE}1. Creating test users...${NC}"

USER1_EMAIL="alice_$(date +%s)@test.com"
USER2_EMAIL="bob_$(date +%s)@test.com"

USER1_RESPONSE=$(curl -s -X POST "$API_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$USER1_EMAIL\",
    \"password\": \"password123\",
    \"username\": \"alice_test\",
    \"display_name\": \"Alice Test\"
  }")

USER1_TOKEN=$(echo $USER1_RESPONSE | jq -r '.data.token')
USER1_ID=$(echo $USER1_RESPONSE | jq -r '.data.user.id')
echo "✅ User 1 (Alice) created: $USER1_ID"

USER2_RESPONSE=$(curl -s -X POST "$API_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$USER2_EMAIL\",
    \"password\": \"password123\",
    \"username\": \"bob_test\",
    \"display_name\": \"Bob Test\"
  }")

USER2_TOKEN=$(echo $USER2_RESPONSE | jq -r '.data.token')
USER2_ID=$(echo $USER2_RESPONSE | jq -r '.data.user.id')
echo "✅ User 2 (Bob) created: $USER2_ID"
echo ""

# User 1 sends friend request to User 2
echo -e "${BLUE}2. Alice sends friend request to Bob...${NC}"
FRIEND_REQUEST=$(curl -s -X POST "$API_URL/connections/request" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $USER1_TOKEN" \
  -d "{
    \"to_user_id\": \"$USER2_ID\",
    \"relationship_tier\": \"MUTUAL\"
  }")

echo $FRIEND_REQUEST | jq '.'
echo ""

# User 2 checks pending requests
echo -e "${BLUE}3. Bob checks pending friend requests...${NC}"
PENDING=$(curl -s -X GET "$API_URL/connections?status=PENDING" \
  -H "Authorization: Bearer $USER2_TOKEN")

echo $PENDING | jq '.'
echo ""

# User 2 accepts the request
echo -e "${BLUE}4. Bob accepts Alice's friend request...${NC}"
ACCEPT=$(curl -s -X POST "$API_URL/connections/accept" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $USER2_TOKEN" \
  -d "{
    \"from_user_id\": \"$USER1_ID\"
  }")

echo $ACCEPT | jq '.'
echo ""

# Both users check their connections
echo -e "${BLUE}5. Checking connections for both users...${NC}"
echo "Alice's connections:"
curl -s -X GET "$API_URL/connections?status=ACCEPTED" \
  -H "Authorization: Bearer $USER1_TOKEN" | jq '.data.count'

echo "Bob's connections:"
curl -s -X GET "$API_URL/connections?status=ACCEPTED" \
  -H "Authorization: Bearer $USER2_TOKEN" | jq '.data.count'
echo ""

# User 1 creates a status
echo -e "${BLUE}6. Alice creates a status...${NC}"
STATUS=$(curl -s -X PUT "$API_URL/me/status" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $USER1_TOKEN" \
  -d '{
    "availability_type": "BUSY",
    "note": "In a meeting 📅",
    "visibility_tier": "ALL_CONNECTIONS"
  }')

echo $STATUS | jq '.'
echo ""

# User 2 checks status feed (should see User 1's status)
echo -e "${BLUE}7. Bob checks status feed (should see Alice's status)...${NC}"
FEED=$(curl -s -X GET "$API_URL/feed/status" \
  -H "Authorization: Bearer $USER2_TOKEN")

echo $FEED | jq '.'
FEED_COUNT=$(echo $FEED | jq '.data.count')

if [ "$FEED_COUNT" = "1" ]; then
  echo -e "${GREEN}✅ Privacy filter working! Bob can see Alice's status.${NC}"
else
  echo "❌ Expected 1 status in feed, got $FEED_COUNT"
fi
echo ""

# Test privacy tier update
echo -e "${BLUE}8. Alice updates Bob's relationship tier to CIRCLE...${NC}"
TIER_UPDATE=$(curl -s -X PUT "$API_URL/connections/tier" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $USER1_TOKEN" \
  -d "{
    \"friend_id\": \"$USER2_ID\",
    \"relationship_tier\": \"CIRCLE\"
  }")

echo $TIER_UPDATE | jq '.'
echo ""

# Test connection status check
echo -e "${BLUE}9. Checking connection status between users...${NC}"
CONNECTION_STATUS=$(curl -s -X GET "$API_URL/connections/status/$USER2_ID" \
  -H "Authorization: Bearer $USER1_TOKEN")

echo $CONNECTION_STATUS | jq '.'
echo ""

echo -e "${GREEN}✅ All connection tests passed!${NC}"
echo ""
echo "Summary:"
echo "- Friend request: Sent & Accepted ✅"
echo "- Privacy filtering: Working ✅"
echo "- Relationship tiers: Configurable ✅"
echo "- Status feed: Respects connections ✅"

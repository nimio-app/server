# Nimio Backend - API Examples

This document contains practical examples for all available API endpoints.

## Prerequisites

1. Server running on `http://localhost:8080`
2. `curl` or a REST client (Postman, Insomnia, etc.)
3. `jq` for pretty JSON output (optional): `brew install jq`

---

## Authentication Endpoints

### 1. Register a New User

**Request:**
```bash
curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "password": "SecurePass123!",
    "username": "alice_wonder",
    "display_name": "Alice Wonderland"
  }' | jq
```

**Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "email": "alice@example.com",
      "created_at": "2026-07-25T10:30:00Z",
      "updated_at": "2026-07-25T10:30:00Z"
    },
    "profile": {
      "user_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "username": "alice_wonder",
      "display_name": "Alice Wonderland",
      "created_at": "2026-07-25T10:30:00Z",
      "updated_at": "2026-07-25T10:30:00Z"
    },
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MjE5MTM2MDAsImlhdCI6MTcyMTkxMjcwMCwic3ViIjoiZjQ3YWMxMGItNThjYy00MzcyLWE1NjctMGUwMmIyYzNkNDc5In0...."
  }
}
```

**Error Cases:**

Email already taken (409 Conflict):
```json
{
  "success": false,
  "error": {
    "code": "Conflict",
    "message": "email already taken"
  }
}
```

Username already taken (409 Conflict):
```json
{
  "success": false,
  "error": {
    "code": "Conflict",
    "message": "username already taken"
  }
}
```

Invalid input (400 Bad Request):
```json
{
  "success": false,
  "error": {
    "code": "Bad Request",
    "message": "invalid input"
  }
}
```

---

### 2. Login

**Request:**
```bash
curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "password": "SecurePass123!"
  }' | jq
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "email": "alice@example.com",
      "created_at": "2026-07-25T10:30:00Z",
      "updated_at": "2026-07-25T10:30:00Z"
    },
    "profile": {
      "user_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "username": "alice_wonder",
      "display_name": "Alice Wonderland",
      "created_at": "2026-07-25T10:30:00Z",
      "updated_at": "2026-07-25T10:30:00Z"
    },
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
}
```

**Save the token for authenticated requests:**
```bash
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

---

## Profile Endpoints

### 3. Get My Profile

**Request:**
```bash
curl http://localhost:8080/v1/me/profile \
  -H "Authorization: Bearer $TOKEN" | jq
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "email": "alice@example.com",
      "created_at": "2026-07-25T10:30:00Z",
      "updated_at": "2026-07-25T10:30:00Z"
    },
    "profile": {
      "user_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "username": "alice_wonder",
      "display_name": "Alice Wonderland",
      "avatar_url": null,
      "bio": null,
      "created_at": "2026-07-25T10:30:00Z",
      "updated_at": "2026-07-25T10:30:00Z"
    }
  }
}
```

---

## Status Endpoints

### 4. Create/Update Status

**Example 1: Set as BUSY with auto-expire**
```bash
curl -X PUT http://localhost:8080/v1/me/status \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "availability_type": "BUSY",
    "note": "In a client meeting",
    "visibility_tier": "ALL_CONNECTIONS",
    "expires_at": "2026-07-25T15:00:00Z"
  }' | jq
```

**Example 2: Set as WANT_TO_TALK (no expiry)**
```bash
curl -X PUT http://localhost:8080/v1/me/status \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "availability_type": "WANT_TO_TALK",
    "note": "Feeling chatty! Hit me up 😊",
    "visibility_tier": "ALL_CONNECTIONS"
  }' | jq
```

**Example 3: FOCUS mode (Circle only)**
```bash
curl -X PUT http://localhost:8080/v1/me/status \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "availability_type": "FOCUS",
    "note": "Deep work session - urgent only",
    "visibility_tier": "CIRCLE_ONLY",
    "expires_at": "2026-07-25T17:00:00Z"
  }' | jq
```

**Example 4: DRIVING**
```bash
curl -X PUT http://localhost:8080/v1/me/status \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "availability_type": "DRIVING",
    "note": "On my way home",
    "visibility_tier": "ALL_CONNECTIONS",
    "expires_at": "2026-07-25T12:30:00Z"
  }' | jq
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": "a1b2c3d4-e5f6-4a5b-8c9d-0e1f2a3b4c5d",
    "user_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "availability_type": "BUSY",
    "note": "In a client meeting",
    "visibility_tier": "ALL_CONNECTIONS",
    "expires_at": "2026-07-25T15:00:00Z",
    "created_at": "2026-07-25T11:00:00Z",
    "updated_at": "2026-07-25T11:00:00Z",
    "is_active": true
  }
}
```

**Validation Errors:**
```json
{
  "success": false,
  "error": {
    "code": "Bad Request",
    "message": "invalid availability_type"
  }
}
```

---

### 5. Get My Current Status

**Request:**
```bash
curl http://localhost:8080/v1/me/status \
  -H "Authorization: Bearer $TOKEN" | jq
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": "a1b2c3d4-e5f6-4a5b-8c9d-0e1f2a3b4c5d",
    "user_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "availability_type": "BUSY",
    "note": "In a client meeting",
    "visibility_tier": "ALL_CONNECTIONS",
    "expires_at": "2026-07-25T15:00:00Z",
    "created_at": "2026-07-25T11:00:00Z",
    "updated_at": "2026-07-25T11:00:00Z",
    "is_active": true
  }
}
```

**No Active Status (404 Not Found):**
```json
{
  "success": false,
  "error": {
    "code": "Not Found",
    "message": "no active status"
  }
}
```

---

### 6. Clear Status

**Request:**
```bash
curl -X DELETE http://localhost:8080/v1/me/status \
  -H "Authorization: Bearer $TOKEN" | jq
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "message": "status cleared successfully"
  }
}
```

---

### 7. Get Status Feed

Returns all statuses visible to the authenticated user based on connection relationships and privacy settings.

**Request:**
```bash
curl http://localhost:8080/v1/feed/status \
  -H "Authorization: Bearer $TOKEN" | jq
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "statuses": [
      {
        "status": {
          "id": "b2c3d4e5-f6a7-4b5c-8d9e-0f1a2b3c4d5e",
          "user_id": "c1d2e3f4-a5b6-4c7d-9e0f-1a2b3c4d5e6f",
          "availability_type": "WANT_TO_TALK",
          "note": "Free for the next hour!",
          "visibility_tier": "ALL_CONNECTIONS",
          "expires_at": "2026-07-25T14:00:00Z",
          "created_at": "2026-07-25T13:00:00Z",
          "updated_at": "2026-07-25T13:00:00Z",
          "is_active": true
        },
        "profile": {
          "user_id": "c1d2e3f4-a5b6-4c7d-9e0f-1a2b3c4d5e6f",
          "username": "bob_smith",
          "display_name": "Bob Smith",
          "avatar_url": "https://example.com/avatars/bob.jpg",
          "bio": "Software engineer",
          "created_at": "2026-07-20T10:00:00Z",
          "updated_at": "2026-07-20T10:00:00Z"
        }
      },
      {
        "status": {
          "id": "d4e5f6a7-b8c9-4d0e-1f2a-3b4c5d6e7f8a",
          "user_id": "e5f6a7b8-c9d0-4e1f-2a3b-4c5d6e7f8a9b",
          "availability_type": "FOCUS",
          "note": "Writing documentation",
          "visibility_tier": "CIRCLE_ONLY",
          "expires_at": null,
          "created_at": "2026-07-25T09:00:00Z",
          "updated_at": "2026-07-25T09:00:00Z",
          "is_active": true
        },
        "profile": {
          "user_id": "e5f6a7b8-c9d0-4e1f-2a3b-4c5d6e7f8a9b",
          "username": "charlie_dev",
          "display_name": "Charlie",
          "created_at": "2026-07-15T08:00:00Z",
          "updated_at": "2026-07-15T08:00:00Z"
        }
      }
    ],
    "count": 2
  }
}
```

**Empty Feed (200 OK):**
```json
{
  "success": true,
  "data": {
    "statuses": [],
    "count": 0
  }
}
```

---

## Availability Types Reference

| Type | Description | Icon | Use Case |
|------|-------------|------|----------|
| `FREE` | Available to chat | 🟢 | Open for conversation |
| `BUSY` | Occupied but can be interrupted | 🟡 | Working but not urgent |
| `FOCUS` | Deep work, don't disturb | 🔴 | Critical focus time |
| `DRIVING` | On the road | 🚗 | Cannot respond |
| `WANT_TO_TALK` | Actively seeking conversation | ❤️ | Feeling social |

---

## Visibility Tiers Reference

| Tier | Description | Who Can See |
|------|-------------|-------------|
| `ALL_CONNECTIONS` | Public to all friends | All accepted connections |
| `CIRCLE_ONLY` | Inner circle | Only CIRCLE tier connections |
| `CUSTOM_LIST` | Specific people | Users in custom visibility list |

---

## Complete Example Workflow

```bash
#!/bin/bash

# 1. Register
echo "=== Registering User ==="
REGISTER_RESPONSE=$(curl -s -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "demo@nimio.org",
    "password": "Demo123456!",
    "username": "demo_user",
    "display_name": "Demo User"
  }')

TOKEN=$(echo $REGISTER_RESPONSE | jq -r '.data.token')
echo "Token: $TOKEN"

# 2. Get Profile
echo -e "\n=== Getting Profile ==="
curl -s http://localhost:8080/v1/me/profile \
  -H "Authorization: Bearer $TOKEN" | jq

# 3. Set Status to BUSY
echo -e "\n=== Setting Status to BUSY ==="
curl -s -X PUT http://localhost:8080/v1/me/status \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "availability_type": "BUSY",
    "note": "In a meeting",
    "visibility_tier": "ALL_CONNECTIONS",
    "expires_at": "2026-07-25T15:00:00Z"
  }' | jq

# 4. Get Current Status
echo -e "\n=== Getting Current Status ==="
curl -s http://localhost:8080/v1/me/status \
  -H "Authorization: Bearer $TOKEN" | jq

# 5. Update to FREE
echo -e "\n=== Updating to FREE ==="
curl -s -X PUT http://localhost:8080/v1/me/status \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "availability_type": "FREE",
    "note": "Available for chat!",
    "visibility_tier": "ALL_CONNECTIONS"
  }' | jq

# 6. Get Status Feed
echo -e "\n=== Getting Status Feed ==="
curl -s http://localhost:8080/v1/feed/status \
  -H "Authorization: Bearer $TOKEN" | jq

# 7. Clear Status
echo -e "\n=== Clearing Status ==="
curl -s -X DELETE http://localhost:8080/v1/me/status \
  -H "Authorization: Bearer $TOKEN" | jq
```

---

## Error Handling

All errors follow this format:

```json
{
  "success": false,
  "error": {
    "code": "HTTP_STATUS_TEXT",
    "message": "human-readable error description"
  }
}
```

Common error codes:
- `400 Bad Request` - Invalid input or validation error
- `401 Unauthorized` - Missing or invalid JWT token
- `404 Not Found` - Resource not found
- `409 Conflict` - Resource already exists (email/username taken)
- `500 Internal Server Error` - Server-side error

---

## Testing with Postman

1. Import the collection (create a new collection)
2. Set base URL variable: `http://localhost:8080`
3. Create environment variable `token` and set it after login
4. Use `{{token}}` in Authorization header: `Bearer {{token}}`

---

## Notes

- All timestamps are in ISO 8601 format with timezone (RFC3339)
- UUIDs are version 4 (random)
- JWT tokens expire after 15 minutes by default (configurable via `JWT_ACCESS_EXPIRY`)
- Expired statuses are automatically deactivated
- Setting a new status automatically deactivates the previous one

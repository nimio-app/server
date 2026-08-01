# Nimio Backend API

> "Respect starts before the first message."

Nimio is an open-source intentional availability sharing app. This repository contains the production-ready Go backend API built with Clean Architecture principles.

## 🎯 Product Vision

Nimio tracks **intentional availability** (emotional readiness to talk), not passive device activity. It eliminates the anxiety of "Will I bother them if I text now?" while respecting everyone's attention and mental space.

### Key Features

- 🟢 **Availability Presets**: Free, Busy, Focus, Driving, Want to talk
- 📝 **Custom Notes**: Add context to your availability
- ⏱️ **Auto-Expiry**: Temporary statuses (15m, 30m, 1h, custom)
- 🔒 **Privacy Controls**: ALL_CONNECTIONS, CIRCLE_ONLY, CUSTOM_LIST
- 🤝 **Connection System**: Friend relationships with privacy tiers

## 🏗️ Architecture

This project follows **Clean Architecture** principles with clear separation of concerns:

```
cmd/api/               # Application entry point
internal/
  ├── config/          # Configuration management
  ├── domain/          # Business entities & errors
  ├── repository/      # Database layer (PostgreSQL)
  ├── service/         # Business logic
  ├── handler/         # HTTP controllers
  └── middleware/      # Auth & CORS middleware
migrations/            # SQL schema migrations
```

## 🚀 Tech Stack

- **Language**: Go 1.22+
- **Router**: Chi v5
- **Database**: PostgreSQL 16 with pgx/v5
- **Auth**: JWT with Argon2id password hashing
- **Email**: Resend for transactional emails
- **Storage**: Cloudflare R2 for avatar images
- **Development**: Docker Compose for local environment

## 📋 Prerequisites

- Go 1.22 or higher
- Docker & Docker Compose
- Make (optional, for convenience commands)
- [golang-migrate](https://github.com/golang-migrate/migrate) (for manual migrations)

## 🛠️ Quick Start

### 1. Clone and Setup

```bash
git clone https://github.com/nimio/server.git
cd server

# Copy environment variables
cp .env.example .env

# Edit .env and set your JWT_SECRET and DB_PASSWORD
```

### 2. Start Database

```bash
# Start PostgreSQL with Docker Compose
make docker-up

# Or manually:
docker-compose up -d
```

### 3. Run Migrations

```bash
# Install golang-migrate if you haven't already
# macOS:
brew install golang-migrate

# Linux:
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/local/bin/

# Run migrations
make migrate-up

# Or manually:
migrate -path migrations -database "postgres://nimio:nimio_dev_password@localhost:5432/nimio_db?sslmode=disable" up
```

### 4. Install Dependencies

```bash
go mod download
```

### 5. Run the Server

```bash
# Using Make
make run

# Or directly
go run cmd/api/main.go
```

The API will be available at `http://localhost:8080`

## 📚 API Documentation

### Authentication Endpoints

#### Register
```bash
POST /v1/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepassword123",
  "username": "johndoe",
  "display_name": "John Doe"
}
```

#### Login
```bash
POST /v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepassword123"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "uuid",
      "email": "user@example.com",
      "email_verified": false,
      "created_at": "2026-07-26T12:00:00Z",
      "verified_at": null
    },
    "profile": {
      "user_id": "uuid",
      "username": "johndoe",
      "display_name": "John Doe"
    },
    "token": "eyJhbGc..."
  }
}
```

#### Verify Email
```bash
POST /v1/auth/verify-email
Content-Type: application/json

{
  "token": "verification_token_from_email"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Email verified successfully! You can now use all features."
  }
}
```

#### Resend Verification Email
```bash
POST /v1/auth/resend-verification
Content-Type: application/json

{
  "email": "user@example.com"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Verification email sent! Please check your inbox."
  }
}
```

#### Google Sign-In
```bash
POST /v1/auth/google
Content-Type: application/json

{
  "id_token": "eyJhbGciOiJSUzI1NiIsImtpZCI6..."
}
```

**Description:**
Authenticate users via Google OAuth. The client must obtain an ID token from Google Sign-In and send it to this endpoint. The backend verifies the token with Google's public keys and either logs in an existing user or auto-registers a new user.

**Flow:**
1. Client initiates Google Sign-In and obtains an ID token
2. Client sends the ID token to `/v1/auth/google`
3. Backend verifies the token with Google
4. If user exists (by Google ID or email), return existing account
5. If user is new, auto-register with `email_verified: true` and Google profile data
6. Return JWT token for authenticated session

**Response:**
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "uuid",
      "email": "user@gmail.com",
      "email_verified": true,
      "google_id": "117234567890123456789",
      "auth_provider": "google",
      "verified_at": "2026-07-29T12:00:00Z",
      "created_at": "2026-07-29T12:00:00Z"
    },
    "profile": {
      "user_id": "uuid",
      "username": "johndoe_1234",
      "display_name": "John Doe",
      "avatar_url": "https://lh3.googleusercontent.com/..."
    },
    "token": "eyJhbGc..."
  }
}
```

**Notes:**
- Google accounts are pre-verified (no email verification needed)
- Usernames are auto-generated from email with a random suffix
- Avatar URLs from Google are stored in the profile
- Users authenticated via Google have `auth_provider: "google"` and no password hash

### Protected Endpoints (Requires JWT)

Add header: `Authorization: Bearer <token>`

#### Get My Profile
```bash
GET /v1/me/profile
Authorization: Bearer <token>
```

**Response:**
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "uuid",
      "email": "user@example.com",
      "email_verified": true,
      "created_at": "2026-08-01T12:00:00Z"
    },
    "profile": {
      "user_id": "uuid",
      "username": "johndoe",
      "display_name": "John Doe",
      "avatar_url": "https://...",
      "bio": "Product designer",
      "created_at": "2026-08-01T12:00:00Z",
      "updated_at": "2026-08-01T12:00:00Z"
    }
  }
}
```

#### Update My Profile
```bash
PUT /v1/me/profile
Content-Type: application/json
Authorization: Bearer <token>

{
  "username": "newusername",
  "display_name": "Jane Doe",
  "bio": "Software engineer who loves building things"
}
```

**Notes:**
- All fields are optional - only send the fields you want to update
- Username must be 3-50 characters, alphanumeric + underscore only
- Display name max 100 characters
- Bio max 500 characters
- Username must be unique across all users

**Response:**
```json
{
  "success": true,
  "data": {
    "profile": {
      "user_id": "uuid",
      "username": "newusername",
      "display_name": "Jane Doe",
      "bio": "Software engineer who loves building things",
      "avatar_url": "https://...",
      "created_at": "2026-08-01T12:00:00Z",
      "updated_at": "2026-08-01T14:30:00Z"
    },
    "message": "Profile updated successfully"
  }
}
```

#### Search Users
```bash
GET /v1/users/search?q=john&limit=20
Authorization: Bearer <token>
```

**Query Parameters:**
- `q` (required): Search query - matches username, email, or display name (min 2 characters)
- `limit` (optional): Max results to return (default: 20, max: 50)

**Search Behavior:**
- Case-insensitive search
- Matches username, email, or display name
- Results ordered by relevance (exact username match first)
- Perfect for typeahead/autocomplete in "add friend" UI

**Response:**
```json
{
  "success": true,
  "data": {
    "users": [
      {
        "user_id": "uuid",
        "username": "johndoe",
        "display_name": "John Doe",
        "avatar_url": "https://...",
        "bio": "Product designer",
        "created_at": "2026-08-01T12:00:00Z",
        "updated_at": "2026-08-01T12:00:00Z"
      },
      {
        "user_id": "uuid",
        "username": "johnny_test",
        "display_name": "Johnny Test",
        "avatar_url": null,
        "bio": null,
        "created_at": "2026-07-28T10:00:00Z",
        "updated_at": "2026-07-28T10:00:00Z"
      }
    ],
    "count": 2
  }
}
```

#### Update Status
```bash
PUT /v1/me/status
Content-Type: application/json

{
  "availability_type": "BUSY",
  "note": "In a meeting",
  "visibility_tier": "ALL_CONNECTIONS",
  "expires_at": "2026-07-25T15:00:00Z"
}
```

#### Get My Status
```bash
GET /v1/me/status
```

#### Clear Status
```bash
DELETE /v1/me/status
```

#### Upload Avatar
```bash
POST /v1/me/avatar
Content-Type: multipart/form-data

# Using curl:
curl -X POST https://api.nimio.org/v1/me/avatar \
  -H "Authorization: Bearer <token>" \
  -F "avatar=@/path/to/image.jpg"
```

**Requirements:**
- Max file size: 5MB
- Allowed formats: JPEG, PNG, GIF, WebP
- Field name: `avatar`

**Response:**
```json
{
  "success": true,
  "data": {
    "avatar_url": "https://pub-xxxxx.r2.dev/avatars/uuid.jpg",
    "message": "Avatar uploaded successfully"
  }
}
```

#### Delete Avatar
```bash
DELETE /v1/me/avatar
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Avatar deleted successfully"
  }
}
```

### Connection Endpoints

#### Send Friend Request
```bash
POST /v1/connections/request
Content-Type: application/json
Authorization: Bearer <token>

{
  "to_user_id": "uuid",
  "relationship_tier": "MUTUAL"
}
```

**Relationship Tiers:**
- `ALL` - Can see all your statuses (lowest privacy)
- `CIRCLE` - Can see statuses marked CIRCLE_ONLY or ALL_CONNECTIONS
- `MUTUAL` - Standard friend connection (default)

**Response:**
```json
{
  "success": true,
  "data": {
    "connection": {
      "id": "uuid",
      "user_id": "uuid",
      "friend_id": "uuid",
      "relationship_tier": "MUTUAL",
      "status": "PENDING",
      "created_at": "2026-08-01T12:00:00Z"
    },
    "message": "Friend request sent successfully"
  }
}
```

#### Accept Friend Request
```bash
POST /v1/connections/accept
Content-Type: application/json
Authorization: Bearer <token>

{
  "from_user_id": "uuid"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "connection": {
      "id": "uuid",
      "user_id": "uuid",
      "friend_id": "uuid",
      "relationship_tier": "MUTUAL",
      "status": "ACCEPTED",
      "updated_at": "2026-08-01T12:00:00Z"
    },
    "message": "Friend request accepted successfully"
  }
}
```

#### Reject Friend Request
```bash
POST /v1/connections/reject
Content-Type: application/json
Authorization: Bearer <token>

{
  "from_user_id": "uuid"
}
```

#### Block User
```bash
POST /v1/connections/block
Content-Type: application/json
Authorization: Bearer <token>

{
  "user_id": "uuid"
}
```

#### Update Relationship Tier
```bash
PUT /v1/connections/tier
Content-Type: application/json
Authorization: Bearer <token>

{
  "friend_id": "uuid",
  "relationship_tier": "CIRCLE"
}
```

#### Get My Connections

**Endpoint:** `GET /v1/connections?status=<status>`

**Auth:** Required

**Query Parameters:**
- `status` (optional): Filter by `PENDING`, `ACCEPTED`, or `BLOCKED`

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "connections": [
      {
        "connection": {
          "id": "uuid",
          "user_id": "uuid",
          "friend_id": "uuid",
          "relationship_tier": "MUTUAL",
          "status": "PENDING",
          "created_at": "2024-01-01T00:00:00Z",
          "updated_at": "2024-01-01T00:00:00Z"
        },
        "profile": {
          "user_id": "uuid",
          "username": "johndoe",
          "display_name": "John Doe",
          "avatar_url": "https://...",
          "bio": "Software engineer"
        },
        "initiated_by_me": false,
        "counterpart_user_id": "uuid",
        "pending_action_hint": "INCOMING"
      }
    ],
    "count": 1
  }
}
```

**Direction Metadata:**
- `initiated_by_me` (boolean): `true` if auth user sent the request, `false` if received
- `counterpart_user_id` (string): The other user's ID (not the auth user)
- `pending_action_hint` (string): Only present for PENDING status
  - `"INCOMING"` → Show Accept/Decline buttons
  - `"OUTGOING"` → Show Cancel button

**Android UX Guide:**
```kotlin
when (connection.status) {
    "PENDING" -> {
        if (connection.pending_action_hint == "INCOMING") {
            // Show: Accept | Decline
            showAcceptDeclineButtons(connection.counterpart_user_id)
        } else {
            // Show: Cancel Request
            showCancelButton(connection.counterpart_user_id)
        }
    }
    "ACCEPTED" -> {
        // Show: Remove Friend
        showRemoveFriendButton(connection.counterpart_user_id)
    }
    "BLOCKED" -> {
        if (connection.initiated_by_me) {
            // Show: Unblock
            showUnblockButton(connection.counterpart_user_id)
        }
    }
}
```

**Examples:**

*Get all pending requests:*
```bash
GET /v1/connections?status=PENDING
```

*Get all accepted friends:*
```bash
GET /v1/connections?status=ACCEPTED
```

*Get all connections (no filter):*
```bash
GET /v1/connections
```

---

#### Connection Direction Examples

**Example 1: Incoming Pending Request (Show Accept/Decline)**
```json
{
  "success": true,
  "data": {
    "connections": [
      {
        "connection": {
          "id": "550e8400-e29b-41d4-a716-446655440000",
          "user_id": "other-user-uuid",
          "friend_id": "auth-user-uuid",
          "relationship_tier": "MUTUAL",
          "status": "PENDING",
          "created_at": "2026-08-01T10:00:00Z",
          "updated_at": "2026-08-01T10:00:00Z"
        },
        "profile": {
          "user_id": "other-user-uuid",
          "username": "alice",
          "display_name": "Alice Smith",
          "avatar_url": "https://cdn.example.com/alice.jpg",
          "bio": "Product designer"
        },
        "initiated_by_me": false,
        "counterpart_user_id": "other-user-uuid",
        "pending_action_hint": "INCOMING"
      }
    ],
    "count": 1
  }
}
```

**Example 2: Outgoing Pending Request (Show Cancel)**
```json
{
  "success": true,
  "data": {
    "connections": [
      {
        "connection": {
          "id": "660e9511-f30c-52e5-b827-557766551111",
          "user_id": "auth-user-uuid",
          "friend_id": "other-user-uuid",
          "relationship_tier": "MUTUAL",
          "status": "PENDING",
          "created_at": "2026-08-01T09:30:00Z",
          "updated_at": "2026-08-01T09:30:00Z"
        },
        "profile": {
          "user_id": "other-user-uuid",
          "username": "bob",
          "display_name": "Bob Johnson",
          "avatar_url": "https://cdn.example.com/bob.jpg",
          "bio": "Software engineer"
        },
        "initiated_by_me": true,
        "counterpart_user_id": "other-user-uuid",
        "pending_action_hint": "OUTGOING"
      }
    ],
    "count": 1
  }
}
```

**Example 3: Accepted Connection**
```json
{
  "success": true,
  "data": {
    "connections": [
      {
        "connection": {
          "id": "770fa622-g41d-63f6-c938-668877662222",
          "user_id": "auth-user-uuid",
          "friend_id": "friend-user-uuid",
          "relationship_tier": "CIRCLE",
          "status": "ACCEPTED",
          "created_at": "2026-07-15T14:20:00Z",
          "updated_at": "2026-07-15T14:25:00Z"
        },
        "profile": {
          "user_id": "friend-user-uuid",
          "username": "charlie",
          "display_name": "Charlie Brown",
          "avatar_url": "https://cdn.example.com/charlie.jpg",
          "bio": "Designer & developer"
        },
        "initiated_by_me": true,
        "counterpart_user_id": "friend-user-uuid"
      }
    ],
    "count": 1
  }
}
```

---

#### Android/Kotlin Integration

**Data Classes:**
```kotlin
data class ConnectionResponse(
    val success: Boolean,
    val data: ConnectionData
)

data class ConnectionData(
    val connections: List<ConnectionWithProfile>,
    val count: Int
)

data class ConnectionWithProfile(
    val connection: Connection,
    val profile: Profile,
    val initiated_by_me: Boolean,
    val counterpart_user_id: String,
    val pending_action_hint: String? = null
)

data class Connection(
    val id: String,
    val user_id: String,
    val friend_id: String,
    val relationship_tier: String,
    val status: String,
    val created_at: String,
    val updated_at: String
)

data class Profile(
    val user_id: String,
    val username: String,
    val display_name: String,
    val avatar_url: String?,
    val bio: String?
)
```

**UI Rendering Logic:**
```kotlin
fun renderConnectionItem(conn: ConnectionWithProfile) {
    val profile = conn.profile
    
    when (conn.connection.status) {
        "PENDING" -> {
            when (conn.pending_action_hint) {
                "INCOMING" -> {
                    // Incoming request - show Accept/Decline
                    showUserInfo(profile)
                    showButton("Accept") { 
                        acceptRequest(conn.counterpart_user_id) 
                    }
                    showButton("Decline") { 
                        declineRequest(conn.counterpart_user_id) 
                    }
                }
                "OUTGOING" -> {
                    // Outgoing request - show Cancel
                    showUserInfo(profile)
                    showButton("Cancel Request") { 
                        cancelRequest(conn.counterpart_user_id) 
                    }
                    showLabel("Pending...")
                }
            }
        }
        "ACCEPTED" -> {
            // Accepted friend
            showUserInfo(profile)
            showButton("Message") { openChat(conn.counterpart_user_id) }
            showButton("Remove Friend") { 
                removeFriend(conn.counterpart_user_id) 
            }
        }
        "BLOCKED" -> {
            if (conn.initiated_by_me) {
                // You blocked this user
                showUserInfo(profile)
                showButton("Unblock") { 
                    unblock(conn.counterpart_user_id) 
                }
            }
        }
    }
}

// API call implementations
suspend fun acceptRequest(userId: String) {
    api.post("/v1/connections/accept") {
        body = json { "from_user_id" to userId }
    }
}

suspend fun declineRequest(userId: String) {
    api.post("/v1/connections/reject") {
        body = json { "from_user_id" to userId }
    }
}

suspend fun cancelRequest(userId: String) {
    api.delete("/v1/connections/$userId")
}

suspend fun removeFriend(userId: String) {
    api.delete("/v1/connections/$userId")
}
```

**Usage Scenarios:**

*Scenario 1: Alice sends you a friend request*
- Response: `initiated_by_me: false`, `pending_action_hint: "INCOMING"`
- UI: Show "Accept" and "Decline" buttons
- User sees: "Alice Smith wants to connect"

*Scenario 2: You sent Bob a friend request*
- Response: `initiated_by_me: true`, `pending_action_hint: "OUTGOING"`
- UI: Show "Cancel Request" button with "Pending..." indicator
- User sees: "Friend request sent to Bob Johnson"

*Scenario 3: Charlie is your friend*
- Response: `initiated_by_me: true` (or false), no `pending_action_hint`
- UI: Show "Message" and "Remove Friend" buttons
- User sees: Normal friend actions

---

#### Get Connection Status
```bash
GET /v1/connections/status/{userId}
Authorization: Bearer <token>
```

Check the connection status with a specific user.

**Response (Connected):**
```json
{
  "success": true,
  "data": {
    "connection": {
      "id": "uuid",
      "status": "ACCEPTED",
      "relationship_tier": "MUTUAL"
    },
    "status": "ACCEPTED"
  }
}
```

**Response (No Connection):**
```json
{
  "success": true,
  "data": {
    "connection": null,
    "status": "none"
  }
}
```

#### Remove Connection
```bash
DELETE /v1/connections/{friendId}
Authorization: Bearer <token>
```

Removes an accepted connection (unfriend).

#### Get Status Feed
```bash
GET /v1/feed/status
```

#### Get Status Feed
```bash
GET /v1/feed/status
Authorization: Bearer <token>
```

Returns statuses from all accepted connections, filtered by privacy settings.

**Privacy Rules:**
- `ALL_CONNECTIONS`: Visible to all accepted friends
- `CIRCLE_ONLY`: Only visible to friends with CIRCLE relationship tier
- `CUSTOM_LIST`: Only visible to specific users in the custom list

**Response:**
```json
{
  "success": true,
  "data": {
    "statuses": [
      {
        "status": {
          "id": "uuid",
          "user_id": "uuid",
          "availability_type": "BUSY",
          "note": "In a meeting",
          "visibility_tier": "ALL_CONNECTIONS",
          "expires_at": "2026-08-01T15:00:00Z",
          "created_at": "2026-08-01T14:00:00Z"
        },
        "profile": {
          "user_id": "uuid",
          "username": "johndoe",
          "display_name": "John Doe",
          "avatar_url": "https://..."
        }
      }
    ],
    "count": 1
  }
}
```

### Availability Types

- `FREE` - Available to chat
- `BUSY` - Occupied but can be interrupted
- `FOCUS` - Deep work, don't disturb
- `DRIVING` - On the road
- `WANT_TO_TALK` - Actively seeking conversation

### Visibility Tiers

- `ALL_CONNECTIONS` - Visible to all accepted friends
- `CIRCLE_ONLY` - Visible only to "Circle" tier connections
- `CUSTOM_LIST` - Visible to specific users only

## 🧪 Testing

```bash
# Run all tests
make test

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 🗃️ Database Schema

The database uses PostgreSQL with the following core tables:

- `users` - Authentication data
- `profiles` - User profiles
- `connections` - Friend relationships
- `statuses` - Availability states
- `status_visibility_lists` - Custom visibility controls
- `refresh_tokens` - JWT refresh token management

See [migrations/000001_init.up.sql](migrations/000001_init.up.sql) for complete schema.

## 🔧 Development Commands

```bash
# Build the application
make build

# Run the application
make run

# Run tests
make test

# Start Docker services
make docker-up

# Stop Docker services
make docker-down

# View logs
make docker-logs

# Run migrations up
make migrate-up

# Rollback last migration
make migrate-down

# Create new migration
make migrate-create name=add_feature_table

# Clean build artifacts
make clean
```

## 🌍 Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `ENV` | Environment (development/production) | `development` |
| `DB_HOST` | Database host | `localhost` |
| `DB_PORT` | Database port | `5432` |
| `DB_USER` | Database user | `nimio` |
| `DB_PASSWORD` | Database password | **required** |
| `DB_NAME` | Database name | `nimio_db` |
| `DB_SSLMODE` | SSL mode | `disable` |
| `JWT_SECRET` | JWT signing secret | **required** |
| `JWT_ACCESS_EXPIRY` | Access token expiry | `15m` |
| `JWT_REFRESH_EXPIRY` | Refresh token expiry | `168h` |
| `CORS_ALLOWED_ORIGINS` | CORS allowed origins | `http://localhost:3000` |
| `RESEND_API_KEY` | Resend API key for emails | **required** |
| `FROM_EMAIL` | Email sender address | `noreply@nimio.org` |
| `FROM_NAME` | Email sender name | `Nimio` |
| `APP_URL` | Frontend app URL for links | `http://localhost:3000` |
| `R2_ACCOUNT_ID` | Cloudflare R2 account ID | **required** |
| `R2_ACCESS_KEY_ID` | R2 access key ID | **required** |
| `R2_SECRET_ACCESS_KEY` | R2 secret access key | **required** |
| `R2_BUCKET_NAME` | R2 bucket name | `nimio` |
| `R2_PUBLIC_URL` | R2 public URL or CDN | `` |
| `GOOGLE_WEB_CLIENT_ID` | Google OAuth Web Client ID | **required** |

## 📧 Email Verification

Nimio uses [Resend](https://resend.com) for transactional emails. When users register:

1. A verification email is sent automatically with a 24-hour token
2. Users receive a branded HTML email with a verification link
3. The `email_verified` field remains `false` until verification
4. Frontend should check this field and display appropriate UI

**Email Flow:**
- Registration → Email sent with token → User clicks link → Call `/v1/auth/verify-email` → `email_verified: true`
- Token format: Base64-encoded 32-byte random string
- Expiry: 24 hours from generation
- Users can request resend via `/v1/auth/resend-verification`

**Setup:**
1. Create a [Resend account](https://resend.com)
2. Get your API key
3. Add `RESEND_API_KEY` to `.env`
4. Configure your domain for production (or use `onboarding@resend.dev` for testing)

## � Google OAuth Sign-In

Nimio supports Google OAuth for seamless authentication without passwords.

**Features:**
- One-tap sign-in with Google accounts
- Auto-registration for new users
- Pre-verified email addresses (no verification needed)
- Automatic profile data import (name, avatar)
- Secure token verification with Google's public keys

**Setup:**
1. Create a project in [Google Cloud Console](https://console.cloud.google.com)
2. Enable the "Google+ API" or "Google Identity Services"
3. Create OAuth 2.0 credentials (Web application)
4. Add authorized redirect URIs for your frontend
5. Copy your **Web Client ID**
6. Add `GOOGLE_WEB_CLIENT_ID` to `.env`
7. Configure your Android/iOS app with the same project credentials

**Client Implementation:**
```typescript
// Example with @react-oauth/google in React
import { GoogleOAuthProvider, useGoogleLogin } from '@react-oauth/google';

const handleGoogleSignIn = useGoogleLogin({
  onSuccess: async (credentialResponse) => {
    const response = await fetch('https://api.nimio.org/v1/auth/google', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id_token: credentialResponse.credential })
    });
    const { token, user, profile } = await response.json();
    // Store token and navigate to app
  }
});
```

**Database Schema:**
- Users have an optional `google_id` field (unique)
- `auth_provider` field indicates `email` or `google`
- Google users have `email_verified: true` by default
- Password hash is null for OAuth users

## �🖼️ Avatar Storage with Cloudflare R2

Nimio uses [Cloudflare R2](https://www.cloudflare.com/products/r2/) for avatar image storage.

**Features:**
- Direct multipart upload to R2 (S3-compatible)
- Max file size: 5MB
- Supported formats: JPEG, PNG, GIF, WebP
- Automatic old avatar deletion on new upload
- Publ**Connection/friend request system**
- [x] **Privacy tier controls (ALL, CIRCLE, MUTUAL)**
- [x] **Status visibility filtering by connections**
- [x] JWT middleware
- [x] Docker development environment
- [x] Clean Architecture structure

### 🚧 Phase 2 (Upcoming)

   ```env
   R2_ACCOUNT_ID=your_account_id
   R2_ACCESS_KEY_ID=your_access_key
   R2_SECRET_ACCESS_KEY=your_secret_key
   R2_BUCKET_NAME=nimio
   R2_PUBLIC_URL=https://pub-xxxxx.r2.dev  # or your custom domain
   ```

**API Usage:**
- `POST /v1/me/avatar` - Upload avatar (multipart form with `avatar` field)
- `DELETE /v1/me/avatar` - Delete avatar
- Avatar URLs are stored in `profiles.avatar_url`

## 🔒 Security Features

- **Argon2id** password hashing with random salts
- **JWT** authentication with configurable expiry
- **Email verification** with secure random tokens (24-hour expiry)
- **SQL injection** protection via prepared statements (pgx)
- **CORS** configuration for cross-origin requests
- **Request timeouts** and rate limiting ready
- **UUID** primary keys for unpredictability

## 📝 Project Status

### ✅ Phase 1 Complete

- [x] Database schema with migrations
- [x] User registration & authentication
- [x] Google OAuth sign-in
- [x] Email verification with Resend
- [x] Avatar uploads with Cloudflare R2
- [x] Profile management (view & update)
- [x] User search (by username/email)
- [x] Status creation, updates, and deletion
- [x] Privacy-aware status feed
- [x] Connection/friend request system
- [x] Privacy tier controls (ALL, CIRCLE, MUTUAL)
- [x] Status visibility filtering by connections
- [x] JWT middleware
- [x] Docker development environment
- [x] Clean Architecture structure

### 🚧 Phase 2 (Upcoming)

- [ ] "Ask if Free" nudge notifications
- [ ] Server-Sent Events (SSE) for real-time updates
- [ ] Status history & analytics
- [ ] Rate limiting & abuse prevention
- [ ] Comprehensive test coverage

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🔗 Links

- Website: [https://nimio.org](https://nimio.org)
- Issues: [GitHub Issues](https://github.com/nimio/server/issues)

## 💬 Philosophy

Nimio believes that **presence is a gift, not an obligation**. This API is built with respect for users' attention, mental space, and autonomy at its core.

---

Built with ❤️ by the Nimio community

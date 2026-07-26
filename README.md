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

### Protected Endpoints (Requires JWT)

Add header: `Authorization: Bearer <token>`

#### Get My Profile
```bash
GET /v1/me/profile
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

#### Get Status Feed
```bash
GET /v1/feed/status
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

## 🔒 Security Features

- **Argon2id** password hashing with random salts
- **JWT** authentication with configurable expiry
- **SQL injection** protection via prepared statements (pgx)
- **CORS** configuration for cross-origin requests
- **Request timeouts** and rate limiting ready
- **UUID** primary keys for unpredictability

## 📝 Project Status

### ✅ Phase 1 Complete

- [x] Database schema with migrations
- [x] User registration & authentication
- [x] Profile management
- [x] Status creation, updates, and deletion
- [x] Privacy-aware status feed
- [x] JWT middleware
- [x] Docker development environment
- [x] Clean Architecture structure

### 🚧 Phase 2 (Upcoming)

- [ ] Connection/friend request system
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

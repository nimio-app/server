# Getting Started with Nimio Backend

This guide will help you set up and run the Nimio backend API on your local machine.

## Prerequisites Installation

### 1. Install Go 1.22+

**macOS:**
```bash
brew install go@1.22
```

**Linux (Ubuntu/Debian):**
```bash
wget https://go.dev/dl/go1.22.5.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.22.5.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

Verify installation:
```bash
go version
# Should output: go version go1.22.x ...
```

### 2. Install Docker & Docker Compose

**macOS:**
```bash
brew install --cask docker
# Then open Docker Desktop from Applications
```

**Linux:**
```bash
# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Install Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
```

### 3. Install golang-migrate

**macOS:**
```bash
brew install golang-migrate
```

**Linux:**
```bash
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/local/bin/
sudo chmod +x /usr/local/bin/migrate
```

## Setup Steps

### 1. Configure Environment

The `.env` file has been created for you. Update these critical values:

```bash
# Edit .env file
nano .env  # or use your preferred editor
```

**Required changes:**
```env
# Generate a strong JWT secret (32+ characters)
JWT_SECRET=your-super-secret-jwt-key-change-in-production-min-32-chars

# Set a strong database password
DB_PASSWORD=your-secure-database-password-here
```

**Generate a secure JWT secret:**
```bash
# Using openssl
openssl rand -base64 32

# Or using Python
python3 -c "import secrets; print(secrets.token_urlsafe(32))"
```

### 2. Install Go Dependencies

```bash
go mod download
go mod tidy
```

### 3. Start PostgreSQL Database

```bash
# Start database with Docker Compose
make docker-up

# Or manually
docker-compose up -d

# Verify it's running
docker ps
# Should show nimio-postgres container running
```

Wait a few seconds for PostgreSQL to initialize, then verify connection:
```bash
docker exec -it nimio-postgres psql -U nimio -d nimio_db -c "SELECT version();"
```

### 4. Run Database Migrations

```bash
# Run migrations
make migrate-up

# Or manually
migrate -path migrations -database "postgres://nimio:YOUR_DB_PASSWORD@localhost:5432/nimio_db?sslmode=disable" up
```

You should see output like:
```
20260725/u init (123.456789ms)
```

### 5. Start the API Server

```bash
# Using Make
make run

# Or directly
go run cmd/api/main.go
```

You should see:
```
✓ Database connection established
🚀 Nimio API server starting on port 8080 (env: development)
```

### 6. Test the API

**Health Check:**
```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "success": true,
  "data": {
    "status": "ok",
    "app": "nimio-api"
  }
}
```

**Register a User:**
```bash
curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "securepassword123",
    "username": "testuser",
    "display_name": "Test User"
  }'
```

Expected response:
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "...",
      "email": "test@example.com",
      "created_at": "..."
    },
    "profile": {
      "username": "testuser",
      "display_name": "Test User",
      ...
    },
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
}
```

**Login:**
```bash
curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "securepassword123"
  }'
```

**Get Profile (Authenticated):**
```bash
# Save the token from login/register response
TOKEN="your-jwt-token-here"

curl http://localhost:8080/v1/me/profile \
  -H "Authorization: Bearer $TOKEN"
```

**Create a Status:**
```bash
curl -X PUT http://localhost:8080/v1/me/status \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "availability_type": "BUSY",
    "note": "In a meeting",
    "visibility_tier": "ALL_CONNECTIONS",
    "expires_at": "2026-07-25T15:00:00Z"
  }'
```

## Troubleshooting

### Database Connection Issues

**Error: "failed to connect to database"**
```bash
# Check if PostgreSQL is running
docker ps | grep nimio-postgres

# View PostgreSQL logs
docker logs nimio-postgres

# Restart the container
docker-compose restart postgres
```

### Migration Errors

**Error: "Dirty database version"**
```bash
# Force reset to version
migrate -path migrations -database "postgres://nimio:PASSWORD@localhost:5432/nimio_db?sslmode=disable" force VERSION

# Then try migrations again
make migrate-up
```

**Start fresh:**
```bash
# Drop and recreate database
docker exec -it nimio-postgres psql -U nimio -c "DROP DATABASE IF EXISTS nimio_db; CREATE DATABASE nimio_db;"

# Run migrations
make migrate-up
```

### Port Already in Use

**Error: "bind: address already in use"**
```bash
# Find process using port 8080
lsof -i :8080

# Kill the process
kill -9 <PID>

# Or change the port in .env
echo "PORT=8081" >> .env
```

### JWT Token Issues

**Error: "invalid or expired token"**
- Ensure `JWT_SECRET` in `.env` matches what was used to generate the token
- Check token expiry (default: 15 minutes)
- Re-login to get a fresh token

## Development Workflow

### Running Tests
```bash
make test
```

### Viewing Database
```bash
# Connect to PostgreSQL
docker exec -it nimio-postgres psql -U nimio -d nimio_db

# Useful SQL commands:
# \dt              - List all tables
# \d users         - Describe users table
# SELECT * FROM users;
# \q               - Quit
```

### Creating New Migrations
```bash
# Create a new migration
make migrate-create name=add_user_settings

# Edit the generated files in migrations/
# Then apply them
make migrate-up
```

### Stopping Services
```bash
# Stop API server: Ctrl+C

# Stop database
make docker-down

# Or keep data and just stop
docker-compose stop
```

## Next Steps

1. **Explore the API**: Try all endpoints in the [README.md](README.md#-api-documentation)
2. **Read the Code**: Start with [cmd/api/main.go](cmd/api/main.go) to understand the flow
3. **Add Features**: Implement Phase 2 features (connections, notifications, etc.)
4. **Write Tests**: Add test coverage for repositories and services
5. **Deploy**: Set up production environment with proper secrets management

## Additional Resources

- [Go Documentation](https://go.dev/doc/)
- [Chi Router Guide](https://go-chi.io/)
- [PostgreSQL Docs](https://www.postgresql.org/docs/)
- [JWT.io](https://jwt.io/) - Debug JWT tokens
- [Nimio Website](https://nimio.org)

## Need Help?

- Check existing [GitHub Issues](https://github.com/nimio/server/issues)
- Open a new issue with details about your problem
- Review the [Contributing Guide](README.md#-contributing)

Happy coding! 🚀

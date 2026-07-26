# Nimio Backend Architecture

This document explains the architectural decisions, design patterns, and code organization of the Nimio backend API.

## 🏛️ Architectural Principles

### Clean Architecture

The project follows **Clean Architecture** (also known as Hexagonal Architecture or Ports & Adapters) to ensure:

1. **Independence from frameworks** - Business logic doesn't depend on Chi, pgx, or any external library
2. **Testability** - Core logic can be tested without databases or HTTP
3. **Database independence** - Could swap PostgreSQL for another DB with minimal changes
4. **UI independence** - The API layer can be replaced with gRPC, GraphQL, etc.

### Dependency Rule

Dependencies point inward:
```
HTTP/External → Handler → Service → Repository → Domain
                  ↓          ↓          ↓
              (uses)    (uses)    (uses)
```

**Domain layer** has zero dependencies on anything else.

---

## 📂 Project Structure

```
server/
├── cmd/api/                    # Application entry point
│   └── main.go                 # Server initialization, routing, graceful shutdown
│
├── internal/                   # Private application code
│   ├── config/                 # Configuration management
│   │   └── config.go           # Environment variable loading, validation
│   │
│   ├── domain/                 # Core business entities (ZERO dependencies)
│   │   ├── user.go             # User, Profile entities
│   │   ├── status.go           # Status entity, enums, validation
│   │   ├── connection.go       # Connection entity, relationship types
│   │   └── errors.go           # Domain-specific error types
│   │
│   ├── repository/             # Data access layer (SQL queries)
│   │   ├── user_repository.go      # User & Profile CRUD
│   │   ├── status_repository.go    # Status CRUD + visibility queries
│   │   └── connection_repository.go # Connection management
│   │
│   ├── service/                # Business logic layer
│   │   ├── auth_service.go     # Registration, login, JWT generation, password hashing
│   │   └── status_service.go   # Status creation, validation, privacy rules
│   │
│   ├── handler/                # HTTP request/response layer
│   │   ├── response.go         # Standard JSON response utilities
│   │   ├── auth_handler.go     # POST /v1/auth/register, /login
│   │   ├── profile_handler.go  # GET /v1/me/profile
│   │   └── status_handler.go   # Status endpoints (GET/PUT/DELETE)
│   │
│   └── middleware/             # HTTP middleware
│       └── auth.go             # JWT validation, context injection
│
├── migrations/                 # SQL schema migrations
│   ├── 000001_init.up.sql      # Initial schema creation
│   └── 000001_init.down.sql    # Rollback script
│
├── docker-compose.yml          # Local PostgreSQL setup
├── Makefile                    # Development commands
├── go.mod                      # Go module definition
└── .env.example                # Environment variable template
```

---

## 🔄 Request Flow

### Example: Creating a Status

```
1. HTTP Request
   POST /v1/me/status
   Headers: Authorization: Bearer <jwt>
   Body: { "availability_type": "BUSY", ... }
   
   ↓

2. Middleware Layer (middleware/auth.go)
   - Extract JWT from Authorization header
   - Validate token signature & expiry
   - Extract user_id from token claims
   - Inject user_id into request context
   
   ↓

3. Handler Layer (handler/status_handler.go)
   - Parse JSON request body
   - Validate input format (required fields, types)
   - Extract user_id from context
   - Call service layer
   
   ↓

4. Service Layer (service/status_service.go)
   - Apply business rules (note length, expiry validation)
   - Deactivate any existing active status
   - Create new status entity
   - Call repository layer
   
   ↓

5. Repository Layer (repository/status_repository.go)
   - Execute SQL INSERT query
   - Handle database errors (constraints, connection)
   - Map database rows to domain entities
   
   ↓

6. Response
   - Service returns domain.Status
   - Handler wraps in standard JSON response
   - HTTP 200 OK with status data
```

---

## 🎯 Layer Responsibilities

### 1. Domain Layer (`internal/domain/`)

**Purpose:** Define core business entities and rules

**Contains:**
- Entities (User, Profile, Status, Connection)
- Value objects (AvailabilityType, VisibilityTier)
- Domain errors
- Entity validation logic

**Dependencies:** NONE

**Example:**
```go
type Status struct {
    ID               uuid.UUID
    UserID           uuid.UUID
    AvailabilityType AvailabilityType
    Note             *string
    VisibilityTier   VisibilityTier
    ExpiresAt        *time.Time
    CreatedAt        time.Time
    UpdatedAt        time.Time
    IsActive         bool
}

func (s *Status) IsExpired() bool {
    if s.ExpiresAt == nil {
        return false
    }
    return time.Now().After(*s.ExpiresAt)
}
```

### 2. Repository Layer (`internal/repository/`)

**Purpose:** Database access and persistence

**Contains:**
- Interface definitions (contract for data access)
- PostgreSQL implementations
- SQL query logic
- Result mapping

**Dependencies:** domain, pgx/v5

**Pattern:**
```go
type StatusRepository interface {
    Create(ctx context.Context, status *domain.Status) error
    GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*domain.Status, error)
    // ... other methods
}

type statusRepository struct {
    db *pgxpool.Pool
}

func NewStatusRepository(db *pgxpool.Pool) StatusRepository {
    return &statusRepository{db: db}
}
```

**Why interfaces?**
- Enables dependency injection
- Makes testing easier (mock repositories)
- Allows swapping implementations

### 3. Service Layer (`internal/service/`)

**Purpose:** Business logic and orchestration

**Contains:**
- Business rule validation
- Multi-step operations
- Privacy & visibility logic
- Password hashing, JWT generation

**Dependencies:** domain, repository, config

**Example:**
```go
func (s *statusService) CreateStatus(
    ctx context.Context,
    userID uuid.UUID,
    availabilityType domain.AvailabilityType,
    note *string,
    visibilityTier domain.VisibilityTier,
    expiresAt *time.Time,
) (*domain.Status, error) {
    // Business validation
    if note != nil && len(*note) > 500 {
        return nil, domain.ErrInvalidInput
    }
    
    // Deactivate old status (business rule: one active status per user)
    _ = s.statusRepo.DeactivateUserStatuses(ctx, userID)
    
    // Create new status
    status := &domain.Status{...}
    if err := s.statusRepo.Create(ctx, status); err != nil {
        return nil, err
    }
    
    return status, nil
}
```

### 4. Handler Layer (`internal/handler/`)

**Purpose:** HTTP request/response handling

**Contains:**
- Request parsing & validation
- Response formatting (JSON)
- HTTP status codes
- Error translation

**Dependencies:** domain, service, middleware

**Pattern:**
```go
func (h *StatusHandler) CreateStatus(w http.ResponseWriter, r *http.Request) {
    // 1. Get authenticated user from context
    userID, ok := middleware.GetUserID(r.Context())
    if !ok {
        ErrorResponse(w, http.StatusUnauthorized, "unauthorized")
        return
    }
    
    // 2. Parse request
    var req CreateStatusRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        ValidationErrorResponse(w, "invalid request body")
        return
    }
    
    // 3. Call service
    status, err := h.statusService.CreateStatus(...)
    if err != nil {
        ErrorResponse(w, http.StatusInternalServerError, "failed to create status")
        return
    }
    
    // 4. Return response
    SuccessResponse(w, http.StatusOK, status)
}
```

### 5. Middleware Layer (`internal/middleware/`)

**Purpose:** Cross-cutting concerns

**Contains:**
- JWT authentication
- Request context enrichment
- (Future: Rate limiting, logging, metrics)

**Pattern:**
```go
func Auth(authService service.AuthService) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Extract & validate token
            token := extractToken(r)
            userID, err := authService.ValidateToken(token)
            if err != nil {
                ErrorResponse(w, 401, "unauthorized")
                return
            }
            
            // Inject into context
            ctx := context.WithValue(r.Context(), UserIDKey, userID)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

---

## 🔐 Security Design

### Authentication Flow

1. **Registration/Login**
   - Password hashed with Argon2id (memory-hard, resistant to GPU attacks)
   - Random 16-byte salt per password
   - Stored format: `$argon2id$<salt>$<hash>`

2. **JWT Generation**
   ```go
   claims := jwt.MapClaims{
       "sub": userID.String(),      // Subject (user ID)
       "iat": time.Now().Unix(),     // Issued at
       "exp": time.Now().Add(15m),   // Expiry (15 minutes)
   }
   token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
   ```

3. **Token Validation**
   - Verify HMAC signature with secret
   - Check expiration time
   - Extract user ID from claims

### Password Security

- **Algorithm:** Argon2id (winner of Password Hashing Competition)
- **Parameters:** 
  - Time cost: 1 iteration
  - Memory cost: 64 MB
  - Parallelism: 4 threads
  - Output length: 32 bytes
- **Salt:** 16 random bytes per password
- **No plaintext storage:** Only hash + salt stored

### Database Security

- **Prepared statements:** All queries use parameterized statements (prevents SQL injection)
- **UUID primary keys:** Unpredictable IDs (vs. sequential integers)
- **Unique constraints:** Email, username uniqueness enforced at DB level
- **Password hash never exposed:** JSON tag `json:"-"` on User.PasswordHash

---

## 🗄️ Database Design

### Schema Highlights

1. **Enums for Type Safety**
   ```sql
   CREATE TYPE availability_type AS ENUM ('FREE', 'BUSY', 'FOCUS', 'DRIVING', 'WANT_TO_TALK');
   CREATE TYPE visibility_tier AS ENUM ('ALL_CONNECTIONS', 'CIRCLE_ONLY', 'CUSTOM_LIST');
   ```

2. **Constraints for Data Integrity**
   ```sql
   -- Email validation
   CONSTRAINT email_valid CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$')
   
   -- Username format
   CONSTRAINT username_valid CHECK (username ~* '^[a-z0-9_]{3,50}$')
   
   -- No self-connections
   CONSTRAINT no_self_connection CHECK (user_id != friend_id)
   
   -- One active status per user
   CREATE UNIQUE INDEX idx_statuses_user_active ON statuses(user_id) WHERE is_active = TRUE;
   ```

3. **Automatic Timestamp Updates**
   ```sql
   CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
       FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
   ```

4. **Privacy-Aware Queries**
   ```sql
   -- Get visible statuses based on connection tier and visibility settings
   SELECT s.*, p.*
   FROM statuses s
   INNER JOIN connections c ON (...)
   WHERE s.is_active = TRUE
     AND c.status = 'ACCEPTED'
     AND (
         s.visibility_tier = 'ALL_CONNECTIONS'
         OR (s.visibility_tier = 'CIRCLE_ONLY' AND c.relationship_tier = 'CIRCLE')
         OR (s.visibility_tier = 'CUSTOM_LIST' AND ...)
     )
   ```

---

## 🧪 Testing Strategy

### Unit Tests (Repositories)
```go
func TestUserRepository_Create(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer db.Close()
    
    repo := repository.NewUserRepository(db)
    
    user := &domain.User{...}
    profile := &domain.Profile{...}
    
    err := repo.Create(context.Background(), user, profile)
    assert.NoError(t, err)
    assert.NotEqual(t, uuid.Nil, user.ID)
}
```

### Integration Tests (Handlers)
```go
func TestAuthHandler_Register(t *testing.T) {
    // Setup test server
    srv := setupTestServer(t)
    
    body := `{"email":"test@example.com","password":"Test123!","username":"test","display_name":"Test"}`
    req := httptest.NewRequest("POST", "/v1/auth/register", strings.NewReader(body))
    w := httptest.NewRecorder()
    
    srv.ServeHTTP(w, req)
    
    assert.Equal(t, 201, w.Code)
    // ... assert response body
}
```

---

## 🚀 Performance Considerations

### Connection Pooling
```go
poolConfig.MaxConns = 25          // Max concurrent connections
poolConfig.MinConns = 5           // Keep warm connections
poolConfig.MaxConnLifetime = 1h   // Recycle old connections
poolConfig.MaxConnIdleTime = 30m  // Close idle connections
poolConfig.HealthCheckPeriod = 1m // Regular health checks
```

### Indexes
```sql
-- Fast user lookups
CREATE INDEX idx_users_email ON users(email);

-- Fast connection queries
CREATE INDEX idx_connections_user_status ON connections(user_id, status);

-- Fast status feed queries
CREATE INDEX idx_statuses_user_id ON statuses(user_id);
CREATE INDEX idx_statuses_expires_at ON statuses(expires_at) WHERE expires_at IS NOT NULL;
```

### Query Optimization
- Use `SELECT` with specific columns (not `SELECT *`)
- Filter inactive statuses at database level
- Use `INNER JOIN` for strong relationships
- Leverage PostgreSQL query planner

---

## 🔮 Future Enhancements

### Phase 2 Features

1. **Connection Management**
   - Send/accept/reject friend requests
   - Block/unblock users
   - Manage relationship tiers

2. **Real-time Updates**
   - Server-Sent Events (SSE) for status changes
   - WebSocket for bidirectional communication
   - Push notification integration

3. **Advanced Features**
   - "Ask if Free" nudges
   - Status history & analytics
   - Recurring availability schedules
   - Location-based statuses

### Technical Improvements

1. **Observability**
   - Structured logging (zerolog, zap)
   - Metrics (Prometheus)
   - Distributed tracing (OpenTelemetry)

2. **Resilience**
   - Rate limiting (per-user, per-IP)
   - Circuit breakers for external services
   - Retry logic with exponential backoff

3. **Testing**
   - Increase test coverage to 80%+
   - Add integration tests
   - Load testing with k6 or vegeta

4. **DevOps**
   - CI/CD pipeline (GitHub Actions)
   - Docker multi-stage builds
   - Kubernetes deployment manifests
   - Database migration automation

---

## 📚 Design Patterns Used

| Pattern | Where | Why |
|---------|-------|-----|
| **Repository** | repository/ | Abstract data access, enable testing |
| **Dependency Injection** | main.go | Loose coupling, testability |
| **Middleware Chain** | Chi router | Cross-cutting concerns (auth, CORS) |
| **Service Layer** | service/ | Business logic separation |
| **DTO Pattern** | handler/ | Decouple HTTP from domain |
| **Factory Pattern** | New*() functions | Consistent object creation |
| **Context Pattern** | All layers | Request-scoped values, cancellation |

---

## 🎓 Learning Resources

- [Clean Architecture (Uncle Bob)](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Go Chi Router](https://go-chi.io/)
- [PostgreSQL Performance](https://www.postgresql.org/docs/current/performance-tips.html)
- [JWT Best Practices](https://datatracker.ietf.org/doc/html/rfc8725)
- [Argon2 Specification](https://github.com/P-H-C/phc-winner-argon2)

---

Built with ❤️ following industry best practices

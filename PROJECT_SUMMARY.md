# Nimio Backend - Project Summary

Last updated: 2026-08-03

## Overview

Nimio backend is a Go API for intentional availability sharing. It is structured with Clean Architecture boundaries and currently ships auth, profile, status, avatar, user search, and social connection flows.

## Architecture and runtime

- Entry point: `cmd/api/main.go`
- Layers: `domain` -> `repository` -> `service` -> `handler` (+ `middleware`)
- Router: Chi (`/health`, `/v1/...`)
- Database: PostgreSQL via pgx pool
- Migrations: golang-migrate, executed on startup in `main.go`

## Implemented feature areas

- Authentication:
  - Email/password registration and login
  - JWT token issuance and validation
  - Google sign-in
  - Email verification and resend verification
- Profile:
  - Get and update my profile
  - Avatar upload/delete (Cloudflare R2-backed storage service)
  - User search
- Status:
  - Create/update current status
  - Get and clear current status
  - Visibility-aware status feed
- Social graph:
  - Send/accept/reject friend requests
  - Block user
  - Remove connection
  - Get connections and bilateral connection status
  - Update relationship tier

## Connection model note

Connections are in transition from a legacy single-tier model to directional tiers:

- Legacy column: `relationship_tier` (kept for compatibility)
- Directional columns: `user_tier`, `friend_tier`

Migration `000005_directional_relationship_tiers.up.sql` introduces directional tier fields and backfills existing rows.

## Main API surface

### Public

- `GET /health`
- `POST /v1/auth/register`
- `POST /v1/auth/login`
- `POST /v1/auth/google`
- `POST /v1/auth/verify-email`
- `POST /v1/auth/resend-verification`

### Protected (JWT)

- `GET /v1/me/profile`
- `PUT /v1/me/profile`
- `POST /v1/me/avatar`
- `DELETE /v1/me/avatar`
- `GET /v1/me/status`
- `PUT /v1/me/status`
- `DELETE /v1/me/status`
- `GET /v1/feed/status`
- `POST /v1/connections/request`
- `POST /v1/connections/accept`
- `POST /v1/connections/reject`
- `POST /v1/connections/block`
- `PUT /v1/connections/{connectionId}/tier`
- `GET /v1/connections`
- `GET /v1/connections/status/{userId}`
- `DELETE /v1/connections/{friendId}`
- `GET /v1/users/search`

## Operational notes

- Config loading is strict for production-critical integrations:
  - `DB_PASSWORD`
  - `JWT_SECRET`
  - `RESEND_API_KEY`
  - `GOOGLE_WEB_CLIENT_ID`
  - `R2_ACCOUNT_ID`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`
- CORS origins are configurable with `CORS_ALLOWED_ORIGINS`.
- Startup runs pending DB migrations before serving traffic.

## Documentation map

- `README.md`: product overview and endpoint examples
- `GETTING_STARTED.md`: local setup and troubleshooting
- `API_EXAMPLES.md`: request/response examples
- `ARCHITECTURE.md`: architecture rationale and layering details

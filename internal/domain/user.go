package domain

import (
	"time"

	"github.com/google/uuid"
)

// User represents the core authentication entity
type User struct {
	ID                         uuid.UUID  `json:"id"`
	Email                      string     `json:"email"`
	PasswordHash               string     `json:"-"` // Never expose in JSON
	GoogleID                   *string    `json:"-"`
	AuthProvider               string     `json:"auth_provider"`
	EmailVerified              bool       `json:"email_verified"`
	VerificationToken          *string    `json:"-"` // Never expose in JSON
	VerificationTokenExpiresAt *time.Time `json:"-"` // Never expose in JSON
	VerifiedAt                 *time.Time `json:"verified_at,omitempty"`
	CreatedAt                  time.Time  `json:"created_at"`
	UpdatedAt                  time.Time  `json:"updated_at"`
}

// Profile represents the public user profile
type Profile struct {
	UserID      uuid.UUID `json:"user_id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	Bio         *string   `json:"bio,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UserWithProfile combines user and profile data for convenience
type UserWithProfile struct {
	User    User    `json:"user"`
	Profile Profile `json:"profile"`
}

// RefreshToken tracks persisted refresh-token sessions.
type RefreshToken struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	TokenHash string     `json:"-"`
	ExpiresAt time.Time  `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

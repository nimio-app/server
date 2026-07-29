package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nimio/server/internal/domain"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_Create_EmailUser(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &userRepository{db: mock}

	userID := uuid.New()
	now := time.Now()
	verificationToken := "test-token"
	expiresAt := now.Add(24 * time.Hour)

	user := &domain.User{
		ID:                         userID,
		Email:                      "test@example.com",
		PasswordHash:               "$argon2id$salt$hash",
		GoogleID:                   nil,
		AuthProvider:               "email",
		EmailVerified:              false,
		VerificationToken:          &verificationToken,
		VerificationTokenExpiresAt: &expiresAt,
		CreatedAt:                  now,
		UpdatedAt:                  now,
	}

	profile := &domain.Profile{
		UserID:      userID,
		Username:    "testuser",
		DisplayName: "Test User",
		AvatarURL:   nil,
		Bio:         nil,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Expect transaction
	mock.ExpectBegin()

	// Expect user insert - verify all 10 placeholders match
	mock.ExpectQuery(`INSERT INTO users`).
		WithArgs(
			user.ID,                         // $1
			user.Email,                      // $2
			user.PasswordHash,               // $3
			user.GoogleID,                   // $4
			user.AuthProvider,               // $5
			user.EmailVerified,              // $6
			user.VerificationToken,          // $7
			user.VerificationTokenExpiresAt, // $8
			user.CreatedAt,                  // $9
			user.UpdatedAt,                  // $10
		).
		WillReturnRows(pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(userID, now, now))

	// Expect profile insert
	mock.ExpectQuery(`INSERT INTO profiles`).
		WithArgs(
			profile.UserID,
			profile.Username,
			profile.DisplayName,
			profile.AvatarURL,
			profile.Bio,
			profile.CreatedAt,
			profile.UpdatedAt,
		).
		WillReturnRows(pgxmock.NewRows([]string{"created_at", "updated_at"}).
			AddRow(now, now))

	mock.ExpectCommit()

	// Execute
	err = repo.Create(context.Background(), user, profile)

	// Assert
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Create_GoogleUser(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &userRepository{db: mock}

	userID := uuid.New()
	now := time.Now()
	googleID := "117234567890123456789"
	avatarURL := "https://lh3.googleusercontent.com/..."

	user := &domain.User{
		ID:                         userID,
		Email:                      "user@gmail.com",
		PasswordHash:               "", // No password for Google users
		GoogleID:                   &googleID,
		AuthProvider:               "google",
		EmailVerified:              true,
		VerificationToken:          nil,
		VerificationTokenExpiresAt: nil,
		VerifiedAt:                 &now,
		CreatedAt:                  now,
		UpdatedAt:                  now,
	}

	profile := &domain.Profile{
		UserID:      userID,
		Username:    "johndoe_1234",
		DisplayName: "John Doe",
		AvatarURL:   &avatarURL,
		Bio:         nil,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Expect transaction
	mock.ExpectBegin()

	// Expect user insert - CRITICAL: verify all 10 placeholders match args
	mock.ExpectQuery(`INSERT INTO users`).
		WithArgs(
			user.ID,                         // $1: id
			user.Email,                      // $2: email
			user.PasswordHash,               // $3: password_hash (empty for Google)
			user.GoogleID,                   // $4: google_id
			user.AuthProvider,               // $5: auth_provider
			user.EmailVerified,              // $6: email_verified
			user.VerificationToken,          // $7: verification_token
			user.VerificationTokenExpiresAt, // $8: verification_token_expires_at
			user.CreatedAt,                  // $9: created_at
			user.UpdatedAt,                  // $10: updated_at
		).
		WillReturnRows(pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(userID, now, now))

	// Expect profile insert
	mock.ExpectQuery(`INSERT INTO profiles`).
		WithArgs(
			profile.UserID,
			profile.Username,
			profile.DisplayName,
			profile.AvatarURL,
			profile.Bio,
			profile.CreatedAt,
			profile.UpdatedAt,
		).
		WillReturnRows(pgxmock.NewRows([]string{"created_at", "updated_at"}).
			AddRow(now, now))

	mock.ExpectCommit()

	// Execute
	err = repo.Create(context.Background(), user, profile)

	// Assert
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByGoogleID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &userRepository{db: mock}

	userID := uuid.New()
	now := time.Now()
	googleID := "117234567890123456789"
	email := "user@gmail.com"
	passwordHash := ""
	authProvider := "google"

	mock.ExpectQuery(`SELECT (.+) FROM users WHERE google_id`).
		WithArgs(googleID).
		WillReturnRows(pgxmock.NewRows(
			[]string{"id", "email", "password_hash", "google_id", "auth_provider",
				"email_verified", "verification_token", "verification_token_expires_at",
				"verified_at", "created_at", "updated_at"},
		).AddRow(userID, email, passwordHash, &googleID, authProvider, true, nil, nil, &now, now, now))

	user, err := repo.GetByGoogleID(context.Background(), googleID)

	require.NoError(t, err)
	assert.Equal(t, userID, user.ID)
	assert.Equal(t, email, user.Email)
	assert.NotNil(t, user.GoogleID)
	assert.Equal(t, googleID, *user.GoogleID)
	assert.Equal(t, authProvider, user.AuthProvider)
	assert.True(t, user.EmailVerified)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByEmail_GoogleUser(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &userRepository{db: mock}

	userID := uuid.New()
	now := time.Now()
	googleID := "117234567890123456789"
	email := "user@gmail.com"

	mock.ExpectQuery(`SELECT (.+) FROM users WHERE email`).
		WithArgs(email).
		WillReturnRows(pgxmock.NewRows(
			[]string{"id", "email", "password_hash", "google_id", "auth_provider",
				"email_verified", "verification_token", "verification_token_expires_at",
				"verified_at", "created_at", "updated_at"},
		).AddRow(userID, email, "", &googleID, "google", true, nil, nil, &now, now, now))

	user, err := repo.GetByEmail(context.Background(), email)

	require.NoError(t, err)
	assert.Equal(t, userID, user.ID)
	assert.NotNil(t, user.GoogleID)
	assert.Equal(t, googleID, *user.GoogleID)
	assert.Equal(t, "google", user.AuthProvider)
	assert.NoError(t, mock.ExpectationsWereMet())
}

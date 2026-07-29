package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nimio/server/internal/domain"
)

// UserRepository handles user-related database operations
type UserRepository interface {
	Create(ctx context.Context, user *domain.User, profile *domain.Profile) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByGoogleID(ctx context.Context, googleID string) (*domain.User, error)
	GetByVerificationToken(ctx context.Context, token string) (*domain.User, error)
	UpdateVerificationToken(ctx context.Context, userID uuid.UUID, token *string, expiresAt *time.Time) error
	MarkEmailAsVerified(ctx context.Context, userID uuid.UUID) error
	GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*domain.Profile, error)
	GetProfileByUsername(ctx context.Context, username string) (*domain.Profile, error)
	UpdateProfile(ctx context.Context, profile *domain.Profile) error
}

type userRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepository{db: db}
}

// Create creates a new user and profile in a transaction
func (r *userRepository) Create(ctx context.Context, user *domain.User, profile *domain.Profile) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert user
	userQuery := `
		INSERT INTO users (id, email, password_hash, google_id, auth_provider, email_verified, 
			verification_token, verification_token_expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at
	`
	err = tx.QueryRow(ctx, userQuery, user.ID, user.Email, user.PasswordHash, user.EmailVerified,
		user.VerificationToken, user.VerificationTokenExpiresAt, user.CreatedAt, user.UpdatedAt).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrEmailTaken
		}
		return fmt.Errorf("insert user: %w", err)
	}

	// Insert profile
	profileQuery := `
		INSERT INTO profiles (user_id, username, display_name, avatar_url, bio, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at
	`
	err = tx.QueryRow(ctx, profileQuery, profile.UserID, profile.Username, profile.DisplayName,
		profile.AvatarURL, profile.Bio, profile.CreatedAt, profile.UpdatedAt).
		Scan(&profile.CreatedAt, &profile.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrUsernameTaken
		}
		return fmt.Errorf("insert profile: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// GetByID retrieves a user by ID
func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, email_verified, verification_token, 
			verification_token_expires_at, verified_at, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	user := &domain.User{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.EmailVerified,
		&user.VerificationToken, &user.VerificationTokenExpiresAt,
		&user.VerifiedAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("query user: %w", err)
	}
	return user, nil
}

// GetByEmail retrieves a user by email
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, email_verified, verification_token, 
			verification_token_expires_at, verified_at, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	user := &domain.User{}
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.EmailVerified,
		&user.VerificationToken, &user.VerificationTokenExpiresAt,
		&user.VerifiedAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("query user: %w", err)
	}
	return user, nil
}

// GetByGoogleID retrieves a user by Google ID
func (r *userRepository) GetByGoogleID(ctx context.Context, googleID string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, google_id, auth_provider, email_verified, 
			verification_token, verification_token_expires_at, verified_at, created_at, updated_at
		FROM users
		WHERE google_id = $1
	`

	user := &domain.User{}
	err := r.db.QueryRow(ctx, query, googleID).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.GoogleID, &user.AuthProvider,
		&user.EmailVerified, &user.VerificationToken, &user.VerificationTokenExpiresAt,
		&user.VerifiedAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("query user by google ID: %w", err)
	}

	return user, nil
}

// GetByVerificationToken retrieves a user by verification token
func (r *userRepository) GetByVerificationToken(ctx context.Context, token string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, email_verified, verification_token, 
			verification_token_expires_at, verified_at, created_at, updated_at
		FROM users
		WHERE verification_token = $1
	`
	user := &domain.User{}
	err := r.db.QueryRow(ctx, query, token).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.EmailVerified,
		&user.VerificationToken, &user.VerificationTokenExpiresAt,
		&user.VerifiedAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("query user: %w", err)
	}
	return user, nil
}

// UpdateVerificationToken updates the verification token for a user
func (r *userRepository) UpdateVerificationToken(ctx context.Context, userID uuid.UUID, token *string, expiresAt *time.Time) error {
	query := `
		UPDATE users
		SET verification_token = $1, verification_token_expires_at = $2, updated_at = NOW()
		WHERE id = $3
	`
	result, err := r.db.Exec(ctx, query, token, expiresAt, userID)
	if err != nil {
		return fmt.Errorf("update verification token: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// MarkEmailAsVerified marks a user's email as verified
func (r *userRepository) MarkEmailAsVerified(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users
		SET email_verified = TRUE, 
			verification_token = NULL, 
			verification_token_expires_at = NULL,
			verified_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`
	result, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("mark email as verified: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// GetProfileByUserID retrieves a profile by user ID
func (r *userRepository) GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*domain.Profile, error) {
	query := `
		SELECT user_id, username, display_name, avatar_url, bio, created_at, updated_at
		FROM profiles
		WHERE user_id = $1
	`
	profile := &domain.Profile{}
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&profile.UserID, &profile.Username, &profile.DisplayName,
		&profile.AvatarURL, &profile.Bio, &profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("query profile: %w", err)
	}
	return profile, nil
}

// GetProfileByUsername retrieves a profile by username
func (r *userRepository) GetProfileByUsername(ctx context.Context, username string) (*domain.Profile, error) {
	query := `
		SELECT user_id, username, display_name, avatar_url, bio, created_at, updated_at
		FROM profiles
		WHERE username = $1
	`
	profile := &domain.Profile{}
	err := r.db.QueryRow(ctx, query, username).Scan(
		&profile.UserID, &profile.Username, &profile.DisplayName,
		&profile.AvatarURL, &profile.Bio, &profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("query profile: %w", err)
	}
	return profile, nil
}

// UpdateProfile updates a user's profile
func (r *userRepository) UpdateProfile(ctx context.Context, profile *domain.Profile) error {
	query := `
		UPDATE profiles
		SET display_name = $1, avatar_url = $2, bio = $3, updated_at = NOW()
		WHERE user_id = $4
		RETURNING updated_at
	`
	err := r.db.QueryRow(ctx, query, profile.DisplayName, profile.AvatarURL, profile.Bio, profile.UserID).
		Scan(&profile.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("update profile: %w", err)
	}
	return nil
}

// isUniqueViolation checks if the error is a unique constraint violation
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return pgxErrorCode(err) == "23505"
}

// pgxErrorCode extracts the PostgreSQL error code
func pgxErrorCode(err error) string {
	if err == nil {
		return ""
	}
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState()
	}
	return ""
}

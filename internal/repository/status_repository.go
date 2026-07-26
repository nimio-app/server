package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nimio/server/internal/domain"
)

// StatusRepository handles status-related database operations
type StatusRepository interface {
	Create(ctx context.Context, status *domain.Status) error
	GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*domain.Status, error)
	DeactivateUserStatuses(ctx context.Context, userID uuid.UUID) error
	GetVisibleStatuses(ctx context.Context, userID uuid.UUID) ([]*domain.StatusWithProfile, error)
	ExpireOldStatuses(ctx context.Context) error
}

type statusRepository struct {
	db *pgxpool.Pool
}

// NewStatusRepository creates a new status repository
func NewStatusRepository(db *pgxpool.Pool) StatusRepository {
	return &statusRepository{db: db}
}

// Create creates a new status
func (r *statusRepository) Create(ctx context.Context, status *domain.Status) error {
	query := `
		INSERT INTO statuses (id, user_id, availability_type, note, visibility_tier, expires_at, created_at, updated_at, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRow(ctx, query,
		status.ID, status.UserID, status.AvailabilityType, status.Note,
		status.VisibilityTier, status.ExpiresAt, status.CreatedAt, status.UpdatedAt, status.IsActive,
	).Scan(&status.ID, &status.CreatedAt, &status.UpdatedAt)
	
	if err != nil {
		// Unique constraint violation on active status
		if isUniqueViolation(err) {
			return fmt.Errorf("user already has an active status: %w", domain.ErrAlreadyExists)
		}
		return fmt.Errorf("insert status: %w", err)
	}
	return nil
}

// GetActiveByUserID retrieves the active status for a user
func (r *statusRepository) GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*domain.Status, error) {
	// First expire any old statuses
	_ = r.ExpireOldStatuses(ctx)

	query := `
		SELECT id, user_id, availability_type, note, visibility_tier, expires_at, created_at, updated_at, is_active
		FROM statuses
		WHERE user_id = $1 AND is_active = TRUE
	`
	status := &domain.Status{}
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&status.ID, &status.UserID, &status.AvailabilityType, &status.Note,
		&status.VisibilityTier, &status.ExpiresAt, &status.CreatedAt, &status.UpdatedAt, &status.IsActive,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNoActiveStatus
		}
		return nil, fmt.Errorf("query status: %w", err)
	}
	return status, nil
}

// DeactivateUserStatuses deactivates all active statuses for a user
func (r *statusRepository) DeactivateUserStatuses(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE statuses
		SET is_active = FALSE, updated_at = NOW()
		WHERE user_id = $1 AND is_active = TRUE
	`
	result, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("deactivate statuses: %w", err)
	}
	
	if result.RowsAffected() == 0 {
		return domain.ErrNoActiveStatus
	}
	
	return nil
}

// GetVisibleStatuses retrieves all statuses visible to the requesting user based on privacy rules
func (r *statusRepository) GetVisibleStatuses(ctx context.Context, userID uuid.UUID) ([]*domain.StatusWithProfile, error) {
	// First expire any old statuses
	_ = r.ExpireOldStatuses(ctx)

	query := `
		SELECT 
			s.id, s.user_id, s.availability_type, s.note, s.visibility_tier, 
			s.expires_at, s.created_at, s.updated_at, s.is_active,
			p.user_id, p.username, p.display_name, p.avatar_url, p.bio,
			p.created_at, p.updated_at
		FROM statuses s
		INNER JOIN profiles p ON s.user_id = p.user_id
		INNER JOIN connections c ON (
			(c.user_id = $1 AND c.friend_id = s.user_id) OR 
			(c.friend_id = $1 AND c.user_id = s.user_id)
		)
		WHERE s.is_active = TRUE
			AND c.status = 'ACCEPTED'
			AND (
				-- ALL_CONNECTIONS: visible to all accepted connections
				s.visibility_tier = 'ALL_CONNECTIONS'
				-- CIRCLE_ONLY: visible only to CIRCLE tier connections
				OR (s.visibility_tier = 'CIRCLE_ONLY' AND c.relationship_tier = 'CIRCLE')
				-- CUSTOM_LIST: check custom visibility list
				OR (s.visibility_tier = 'CUSTOM_LIST' AND EXISTS (
					SELECT 1 FROM status_visibility_lists svl 
					WHERE svl.status_id = s.id AND svl.visible_to_user_id = $1
				))
			)
		ORDER BY s.created_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query visible statuses: %w", err)
	}
	defer rows.Close()

	var statuses []*domain.StatusWithProfile
	for rows.Next() {
		sp := &domain.StatusWithProfile{}
		err := rows.Scan(
			&sp.Status.ID, &sp.Status.UserID, &sp.Status.AvailabilityType, &sp.Status.Note,
			&sp.Status.VisibilityTier, &sp.Status.ExpiresAt, &sp.Status.CreatedAt,
			&sp.Status.UpdatedAt, &sp.Status.IsActive,
			&sp.Profile.UserID, &sp.Profile.Username, &sp.Profile.DisplayName,
			&sp.Profile.AvatarURL, &sp.Profile.Bio, &sp.Profile.CreatedAt, &sp.Profile.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan status: %w", err)
		}
		statuses = append(statuses, sp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}

	return statuses, nil
}

// ExpireOldStatuses marks expired statuses as inactive
func (r *statusRepository) ExpireOldStatuses(ctx context.Context) error {
	query := `
		UPDATE statuses
		SET is_active = FALSE, updated_at = NOW()
		WHERE is_active = TRUE 
			AND expires_at IS NOT NULL 
			AND expires_at <= NOW()
	`
	_, err := r.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("expire statuses: %w", err)
	}
	return nil
}

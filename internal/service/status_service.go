package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nimio/server/internal/domain"
	"github.com/nimio/server/internal/repository"
)

// StatusService handles status-related business logic
type StatusService interface {
	CreateStatus(ctx context.Context, userID uuid.UUID, availabilityType domain.AvailabilityType, note *string, visibilityTier domain.VisibilityTier, expiresAt *time.Time) (*domain.Status, error)
	GetUserStatus(ctx context.Context, userID uuid.UUID) (*domain.Status, error)
	ClearUserStatus(ctx context.Context, userID uuid.UUID) error
	GetVisibleStatuses(ctx context.Context, userID uuid.UUID) ([]*domain.StatusWithProfile, error)
}

type statusService struct {
	statusRepo repository.StatusRepository
}

// NewStatusService creates a new status service
func NewStatusService(statusRepo repository.StatusRepository) StatusService {
	return &statusService{
		statusRepo: statusRepo,
	}
}

// CreateStatus creates or updates a user's status
func (s *statusService) CreateStatus(ctx context.Context, userID uuid.UUID, availabilityType domain.AvailabilityType, note *string, visibilityTier domain.VisibilityTier, expiresAt *time.Time) (*domain.Status, error) {
	// Validate note length
	if note != nil && len(*note) > 500 {
		return nil, fmt.Errorf("note exceeds maximum length of 500 characters: %w", domain.ErrInvalidInput)
	}

	// Validate expires_at is in the future
	if expiresAt != nil && expiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("expires_at must be in the future: %w", domain.ErrInvalidInput)
	}

	// Deactivate any existing active statuses
	_ = s.statusRepo.DeactivateUserStatuses(ctx, userID)

	// Create new status
	status := &domain.Status{
		ID:               uuid.New(),
		UserID:           userID,
		AvailabilityType: availabilityType,
		Note:             note,
		VisibilityTier:   visibilityTier,
		ExpiresAt:        expiresAt,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		IsActive:         true,
	}

	if err := s.statusRepo.Create(ctx, status); err != nil {
		return nil, fmt.Errorf("create status: %w", err)
	}

	return status, nil
}

// GetUserStatus retrieves the active status for a user
func (s *statusService) GetUserStatus(ctx context.Context, userID uuid.UUID) (*domain.Status, error) {
	status, err := s.statusRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Double-check expiration in case background job hasn't run
	if status.IsExpired() {
		_ = s.statusRepo.DeactivateUserStatuses(ctx, userID)
		return nil, domain.ErrNoActiveStatus
	}

	return status, nil
}

// ClearUserStatus deactivates the user's current status
func (s *statusService) ClearUserStatus(ctx context.Context, userID uuid.UUID) error {
	return s.statusRepo.DeactivateUserStatuses(ctx, userID)
}

// GetVisibleStatuses retrieves all statuses visible to the requesting user
func (s *statusService) GetVisibleStatuses(ctx context.Context, userID uuid.UUID) ([]*domain.StatusWithProfile, error) {
	statuses, err := s.statusRepo.GetVisibleStatuses(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get visible statuses: %w", err)
	}

	return statuses, nil
}

package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nimio/server/internal/domain"
	"github.com/nimio/server/internal/repository"
)

// ConnectionService handles connection-related business logic
type ConnectionService interface {
	SendFriendRequest(ctx context.Context, fromUserID, toUserID uuid.UUID, tier domain.RelationshipTier) (*domain.Connection, error)
	AcceptFriendRequest(ctx context.Context, userID, friendID uuid.UUID) (*domain.Connection, error)
	RejectFriendRequest(ctx context.Context, userID, friendID uuid.UUID) error
	BlockUser(ctx context.Context, userID, blockedUserID uuid.UUID) (*domain.Connection, error)
	RemoveConnection(ctx context.Context, userID, friendID uuid.UUID) error
	UpdateRelationshipTier(ctx context.Context, userID, friendID uuid.UUID, tier domain.RelationshipTier) (*domain.Connection, error)
	GetMyConnections(ctx context.Context, userID uuid.UUID, status domain.ConnectionStatus) ([]*domain.ConnectionWithProfile, error)
	GetConnectionStatus(ctx context.Context, userID, otherUserID uuid.UUID) (*domain.Connection, error)
}

type connectionService struct {
	connRepo repository.ConnectionRepository
	userRepo repository.UserRepository
}

// NewConnectionService creates a new connection service
func NewConnectionService(connRepo repository.ConnectionRepository, userRepo repository.UserRepository) ConnectionService {
	return &connectionService{
		connRepo: connRepo,
		userRepo: userRepo,
	}
}

// SendFriendRequest sends a friend request from one user to another
func (s *connectionService) SendFriendRequest(ctx context.Context, fromUserID, toUserID uuid.UUID, tier domain.RelationshipTier) (*domain.Connection, error) {
	// Reject self-request
	if fromUserID == toUserID {
		return nil, domain.ErrSelfConnection
	}

	// Check if target user exists
	targetUser, err := s.userRepo.GetByID(ctx, toUserID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("check target user: %w", err)
	}

	if targetUser == nil {
		return nil, domain.ErrNotFound
	}

	// Check if connection already exists (bidirectional)
	existingConn, err := s.connRepo.GetByUsers(ctx, fromUserID, toUserID)
	if err != nil && err != domain.ErrNotFound {
		return nil, fmt.Errorf("check existing connection: %w", err)
	}

	// Reject duplicate active relationship/request
	if existingConn != nil {
		if existingConn.Status == domain.ConnectionPending {
			return nil, domain.ErrDuplicatePending
		}
		if existingConn.Status == domain.ConnectionAccepted {
			return nil, domain.ErrAlreadyConnected
		}
		if existingConn.Status == domain.ConnectionBlocked {
			return nil, domain.ErrConnectionBlocked
		}
	}

	// Validate and normalize tier (map MUTUAL → ALL)
	tier = domain.NormalizeRelationshipTier(tier)
	if !domain.IsValidRelationshipTier(tier) {
		tier = domain.RelationshipAll // Default to ALL
	}

	// Create connection with directional tiers (both start as ALL by default)
	now := time.Now()
	connection := &domain.Connection{
		ID:        uuid.New(),
		UserID:    fromUserID,
		FriendID:  toUserID,
		UserTier:  domain.RelationshipAll, // Sender's tier for receiver
		FriendTier: domain.RelationshipAll, // Receiver's tier for sender (will be set after accept)
		RelationshipTier: domain.RelationshipAll, // Deprecated field
		Status:    domain.ConnectionPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Create with DB-level race condition protection
	if err := s.connRepo.Create(ctx, connection); err != nil {
		// Handle unique constraint violation from concurrent requests
		if err == domain.ErrAlreadyExists {
			// Re-check to determine specific error type
			existingConn, checkErr := s.connRepo.GetByUsers(ctx, fromUserID, toUserID)
			if checkErr == nil && existingConn != nil {
				switch existingConn.Status {
				case domain.ConnectionPending:
					return nil, domain.ErrDuplicatePending
				case domain.ConnectionAccepted:
					return nil, domain.ErrAlreadyConnected
				case domain.ConnectionBlocked:
					return nil, domain.ErrConnectionBlocked
				}
			}
			return nil, domain.ErrDuplicatePending // Fallback
		}
		return nil, fmt.Errorf("create connection: %w", err)
	}

	return connection, nil
}

// AcceptFriendRequest accepts a pending friend request
func (s *connectionService) AcceptFriendRequest(ctx context.Context, userID, friendID uuid.UUID) (*domain.Connection, error) {
	// Get the connection where userID is the receiver (friend_id)
	connection, err := s.connRepo.GetByUsers(ctx, friendID, userID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, fmt.Errorf("friend request not found: %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get connection: %w", err)
	}

	// Verify that userID is the recipient (friend_id in the original request)
	if connection.FriendID != userID {
		return nil, fmt.Errorf("cannot accept request you didn't receive: %w", domain.ErrForbidden)
	}

	// Check if already accepted
	if connection.Status == domain.ConnectionAccepted {
		return connection, nil // Already accepted, no-op
	}

	// Check if blocked
	if connection.Status == domain.ConnectionBlocked {
		return nil, fmt.Errorf("connection is blocked: %w", domain.ErrForbidden)
	}

	// Verify status is pending
	if connection.Status != domain.ConnectionPending {
		return nil, fmt.Errorf("connection is not pending: %w", domain.ErrInvalidInput)
	}

	// Update to accepted
	connection.Status = domain.ConnectionAccepted
	if err := s.connRepo.Update(ctx, connection); err != nil {
		return nil, fmt.Errorf("update connection: %w", err)
	}

	return connection, nil
}

// RejectFriendRequest rejects a pending friend request by deleting it
func (s *connectionService) RejectFriendRequest(ctx context.Context, userID, friendID uuid.UUID) error {
	connection, err := s.connRepo.GetByUsers(ctx, friendID, userID)
	if err != nil {
		if err == domain.ErrNotFound {
			return fmt.Errorf("friend request not found: %w", domain.ErrNotFound)
		}
		return fmt.Errorf("get connection: %w", err)
	}

	// Verify that userID is the recipient
	if connection.FriendID != userID {
		return fmt.Errorf("cannot reject request you didn't receive: %w", domain.ErrForbidden)
	}

	// Only pending requests can be rejected
	if connection.Status != domain.ConnectionPending {
		return fmt.Errorf("can only reject pending requests: %w", domain.ErrInvalidInput)
	}

	// Delete the connection
	if err := s.connRepo.Delete(ctx, connection.ID); err != nil {
		return fmt.Errorf("delete connection: %w", err)
	}

	return nil
}

// BlockUser blocks another user
func (s *connectionService) BlockUser(ctx context.Context, userID, blockedUserID uuid.UUID) (*domain.Connection, error) {
	if userID == blockedUserID {
		return nil, fmt.Errorf("cannot block yourself: %w", domain.ErrInvalidInput)
	}

	// Check if connection exists
	connection, err := s.connRepo.GetByUsers(ctx, userID, blockedUserID)
	if err != nil && err != domain.ErrNotFound {
		return nil, fmt.Errorf("check connection: %w", err)
	}

	now := time.Now()

	if connection == nil {
		// Create new blocked connection
		connection = &domain.Connection{
			ID:               uuid.New(),
			UserID:           userID,
			FriendID:         blockedUserID,
			RelationshipTier: domain.RelationshipMutual,
			Status:           domain.ConnectionBlocked,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		if err := s.connRepo.Create(ctx, connection); err != nil {
			return nil, fmt.Errorf("create blocked connection: %w", err)
		}
	} else {
		// Update existing connection to blocked
		connection.Status = domain.ConnectionBlocked
		if err := s.connRepo.Update(ctx, connection); err != nil {
			return nil, fmt.Errorf("update connection to blocked: %w", err)
		}
	}

	return connection, nil
}

// RemoveConnection removes a connection between two users
func (s *connectionService) RemoveConnection(ctx context.Context, userID, friendID uuid.UUID) error {
	connection, err := s.connRepo.GetByUsers(ctx, userID, friendID)
	if err != nil {
		if err == domain.ErrNotFound {
			return fmt.Errorf("connection not found: %w", domain.ErrNotFound)
		}
		return fmt.Errorf("get connection: %w", err)
	}

	// Can only remove accepted connections
	if connection.Status != domain.ConnectionAccepted {
		return fmt.Errorf("can only remove accepted connections: %w", domain.ErrInvalidInput)
	}

	// Delete the connection
	if err := s.connRepo.Delete(ctx, connection.ID); err != nil {
		return fmt.Errorf("delete connection: %w", err)
	}

	return nil
}

// UpdateRelationshipTier updates the privacy tier of a connection (directional)
// Only updates the requesting user's tier for the counterpart
func (s *connectionService) UpdateRelationshipTier(ctx context.Context, userID, friendID uuid.UUID, tier domain.RelationshipTier) (*domain.Connection, error) {
	connection, err := s.connRepo.GetByUsers(ctx, userID, friendID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, fmt.Errorf("connection not found: %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get connection: %w", err)
	}

	// Only update tier for accepted connections
	if connection.Status != domain.ConnectionAccepted {
		return nil, fmt.Errorf("can only update tier for accepted connections: %w", domain.ErrInvalidInput)
	}

	// Validate and normalize tier (map MUTUAL → ALL, reject invalid)
	tier = domain.NormalizeRelationshipTier(tier)
	if !domain.IsValidRelationshipTier(tier) {
		return nil, fmt.Errorf("invalid relationship tier (must be ALL or CIRCLE): %w", domain.ErrInvalidInput)
	}

	// Update only the requesting user's tier
	if connection.UserID == userID {
		connection.UserTier = tier
	} else if connection.FriendID == userID {
		connection.FriendTier = tier
	} else {
		return nil, fmt.Errorf("not authorized to update this connection: %w", domain.ErrForbidden)
	}

	if err := s.connRepo.Update(ctx, connection); err != nil {
		return nil, fmt.Errorf("update connection: %w", err)
	}

	return connection, nil
}

// GetMyConnections retrieves all connections for a user with optional status filter
func (s *connectionService) GetMyConnections(ctx context.Context, userID uuid.UUID, status domain.ConnectionStatus) ([]*domain.ConnectionWithProfile, error) {
	connections, err := s.connRepo.ListByUserID(ctx, userID, status)
	if err != nil {
		return nil, fmt.Errorf("list connections: %w", err)
	}

	// Enrich with profile data and direction metadata
	result := make([]*domain.ConnectionWithProfile, 0, len(connections))
	for _, conn := range connections {
		// Determine which user is the "other" user
		otherUserID := conn.FriendID
		if conn.FriendID == userID {
			otherUserID = conn.UserID
		}

		// Get profile
		profile, err := s.userRepo.GetProfileByUserID(ctx, otherUserID)
		if err != nil {
			// Skip if profile not found
			continue
		}

		// Compute direction metadata and directional tiers
		initiatedByMe := conn.UserID == userID
		
		// Get the tier that auth user has assigned to the counterpart
		myTierForThem := conn.GetTierFor(userID, otherUserID)
		// Get the tier that counterpart has assigned to auth user  
		theirTierForMe := conn.GetTierFor(otherUserID, userID)
		
		// Compute pending action hint
		var pendingHint domain.PendingActionHint
		if conn.Status == domain.ConnectionPending {
			if initiatedByMe {
				pendingHint = domain.PendingActionOutgoing
			} else {
				pendingHint = domain.PendingActionIncoming
			}
		}

		result = append(result, &domain.ConnectionWithProfile{
			Connection:        *conn,
			Profile:           *profile,
			InitiatedByMe:     initiatedByMe,
			CounterpartUserID: otherUserID.String(),
			MyTierForThem:     myTierForThem,
			TheirTierForMe:    theirTierForMe,
			PendingActionHint: pendingHint,
		})
	}

	return result, nil
}

// GetConnectionStatus retrieves the connection status between two users
func (s *connectionService) GetConnectionStatus(ctx context.Context, userID, otherUserID uuid.UUID) (*domain.Connection, error) {
	connection, err := s.connRepo.GetByUsers(ctx, userID, otherUserID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, nil // No connection exists, return nil without error
		}
		return nil, fmt.Errorf("get connection: %w", err)
	}

	return connection, nil
}

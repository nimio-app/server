package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/nimio/server/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestConnectionService_DirectionalTiers tests the new 2-tier directional system
func TestConnectionService_DirectionalTiers(t *testing.T) {
	ctx := context.Background()

	t.Run("new_connection_defaults_to_ALL", func(t *testing.T) {
		connRepo := new(MockConnectionRepository)
		userRepo := new(MockUserRepository)
		service := NewConnectionService(connRepo, userRepo)

		fromUserID := uuid.New()
		toUserID := uuid.New()

		// Mock user exists
		userRepo.On("GetByID", ctx, toUserID).Return(&domain.User{ID: toUserID}, nil)
		// Mock no existing connection
		connRepo.On("GetByUsers", ctx, fromUserID, toUserID).Return(nil, domain.ErrNotFound)
		// Mock successful create
		connRepo.On("Create", ctx, mock.AnythingOfType("*domain.Connection")).Return(nil)

		connection, err := service.SendFriendRequest(ctx, fromUserID, toUserID, domain.RelationshipAll)

		require.NoError(t, err)
		assert.NotNil(t, connection)
		assert.Equal(t, domain.RelationshipAll, connection.UserTier, "New connections should default to ALL for user")
		assert.Equal(t, domain.RelationshipAll, connection.FriendTier, "New connections should default to ALL for friend")
	})

	t.Run("MUTUAL_input_maps_to_ALL", func(t *testing.T) {
		connRepo := new(MockConnectionRepository)
		userRepo := new(MockUserRepository)
		service := NewConnectionService(connRepo, userRepo)

		fromUserID := uuid.New()
		toUserID := uuid.New()

		userRepo.On("GetByID", ctx, toUserID).Return(&domain.User{ID: toUserID}, nil)
		connRepo.On("GetByUsers", ctx, fromUserID, toUserID).Return(nil, domain.ErrNotFound)
		connRepo.On("Create", ctx, mock.AnythingOfType("*domain.Connection")).Return(nil)

		// Send request with deprecated MUTUAL tier
		connection, err := service.SendFriendRequest(ctx, fromUserID, toUserID, domain.RelationshipMutual)

		require.NoError(t, err)
		assert.Equal(t, domain.RelationshipAll, connection.UserTier, "MUTUAL should be normalized to ALL")
	})

	t.Run("tier_update_is_directional", func(t *testing.T) {
		connRepo := new(MockConnectionRepository)
		userRepo := new(MockUserRepository)
		service := NewConnectionService(connRepo, userRepo)

		userA := uuid.New()
		userB := uuid.New()

		// User A updates their tier for User B to CIRCLE
		connection := &domain.Connection{
			ID:         uuid.New(),
			UserID:     userA,
			FriendID:   userB,
			UserTier:   domain.RelationshipAll,
			FriendTier: domain.RelationshipAll,
			Status:     domain.ConnectionAccepted,
		}
		connRepo.On("GetByUsers", ctx, userA, userB).Return(connection, nil)
		connRepo.On("Update", ctx, mock.AnythingOfType("*domain.Connection")).Return(nil)

		updatedConn, err := service.UpdateRelationshipTier(ctx, userA, userB, domain.RelationshipCircle)

		require.NoError(t, err)
		assert.Equal(t, domain.RelationshipCircle, updatedConn.UserTier, "User A's tier for B should be CIRCLE")
		assert.Equal(t, domain.RelationshipAll, updatedConn.FriendTier, "User B's tier for A should remain ALL")
	})

	t.Run("asymmetric_tiers_work", func(t *testing.T) {
		connRepo := new(MockConnectionRepository)
		userRepo := new(MockUserRepository)
		service := NewConnectionService(connRepo, userRepo)

		userA := uuid.New()
		userB := uuid.New()

		// Connection where A has B in CIRCLE, but B has A as ALL
		connection := &domain.Connection{
			ID:         uuid.New(),
			UserID:     userA,
			FriendID:   userB,
			UserTier:   domain.RelationshipCircle, // A's tier for B
			FriendTier: domain.RelationshipAll,    // B's tier for A
			Status:     domain.ConnectionAccepted,
		}
		connRepo.On("ListByUserID", ctx, userA, domain.ConnectionAccepted).Return([]*domain.Connection{connection}, nil)

		// Mock profile
		profile := &domain.Profile{
			UserID:   userB,
			Username: "userB",
		}
		userRepo.On("GetProfileByUserID", ctx, userB).Return(profile, nil)

		// Get connections for User A
		connections, err := service.GetMyConnections(ctx, userA, domain.ConnectionAccepted)

		require.NoError(t, err)
		require.Len(t, connections, 1)
		assert.Equal(t, domain.RelationshipCircle, connections[0].MyTierForThem, "A categorizes B as CIRCLE")
		assert.Equal(t, domain.RelationshipAll, connections[0].TheirTierForMe, "B categorizes A as ALL")
	})

	t.Run("tier_update_rejects_invalid_values", func(t *testing.T) {
		connRepo := new(MockConnectionRepository)
		userRepo := new(MockUserRepository)
		service := NewConnectionService(connRepo, userRepo)

		userA := uuid.New()
		userB := uuid.New()

		connection := &domain.Connection{
			ID:       uuid.New(),
			UserID:   userA,
			FriendID: userB,
			Status:   domain.ConnectionAccepted,
		}
		connRepo.On("GetByUsers", ctx, userA, userB).Return(connection, nil)

		// Try to set an invalid tier (should be rejected by validation)
		_, err := service.UpdateRelationshipTier(ctx, userA, userB, "INVALID")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid relationship tier")
	})

	t.Run("tier_update_only_for_accepted_connections", func(t *testing.T) {
		connRepo := new(MockConnectionRepository)
		userRepo := new(MockUserRepository)
		service := NewConnectionService(connRepo, userRepo)

		userA := uuid.New()
		userB := uuid.New()

		// Try to update tier on a PENDING connection
		connection := &domain.Connection{
			ID:       uuid.New(),
			UserID:   userA,
			FriendID: userB,
			Status:   domain.ConnectionPending,
		}
		connRepo.On("GetByUsers", ctx, userA, userB).Return(connection, nil)

		_, err := service.UpdateRelationshipTier(ctx, userA, userB, domain.RelationshipCircle)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "can only update tier for accepted connections")
	})
}

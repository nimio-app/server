package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/nimio/server/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConnectionService_GetMyConnections_Direction tests direction metadata
func TestConnectionService_GetMyConnections_Direction(t *testing.T) {
	ctx := context.Background()
	authUserID := uuid.New()
	otherUserID := uuid.New()

	t.Run("pending_outgoing_has_initiated_by_me_true", func(t *testing.T) {
		connRepo := new(MockConnectionRepository)
		userRepo := new(MockUserRepository)
		service := NewConnectionService(connRepo, userRepo)

		// Mock outgoing pending request (auth user is sender)
		connection := &domain.Connection{
			ID:       uuid.New(),
			UserID:   authUserID,  // Auth user initiated
			FriendID: otherUserID,
			Status:   domain.ConnectionPending,
		}
		connRepo.On("ListByUserID", ctx, authUserID, domain.ConnectionPending).Return([]*domain.Connection{connection}, nil)

		// Mock profile
		profile := &domain.Profile{
			UserID:   otherUserID,
			Username: "otheruser",
		}
		userRepo.On("GetProfileByUserID", ctx, otherUserID).Return(profile, nil)

		// Get connections
		connections, err := service.GetMyConnections(ctx, authUserID, domain.ConnectionPending)

		require.NoError(t, err)
		require.Len(t, connections, 1)
		assert.True(t, connections[0].InitiatedByMe, "Outgoing request should have initiated_by_me=true")
		assert.Equal(t, otherUserID.String(), connections[0].CounterpartUserID)
		assert.Equal(t, domain.PendingActionOutgoing, connections[0].PendingActionHint)
	})

	t.Run("pending_incoming_has_initiated_by_me_false", func(t *testing.T) {
		connRepo := new(MockConnectionRepository)
		userRepo := new(MockUserRepository)
		service := NewConnectionService(connRepo, userRepo)

		// Mock incoming pending request (other user is sender)
		connection := &domain.Connection{
			ID:       uuid.New(),
			UserID:   otherUserID, // Other user initiated
			FriendID: authUserID,
			Status:   domain.ConnectionPending,
		}
		connRepo.On("ListByUserID", ctx, authUserID, domain.ConnectionPending).Return([]*domain.Connection{connection}, nil)

		// Mock profile
		profile := &domain.Profile{
			UserID:   otherUserID,
			Username: "otheruser",
		}
		userRepo.On("GetProfileByUserID", ctx, otherUserID).Return(profile, nil)

		// Get connections
		connections, err := service.GetMyConnections(ctx, authUserID, domain.ConnectionPending)

		require.NoError(t, err)
		require.Len(t, connections, 1)
		assert.False(t, connections[0].InitiatedByMe, "Incoming request should have initiated_by_me=false")
		assert.Equal(t, otherUserID.String(), connections[0].CounterpartUserID)
		assert.Equal(t, domain.PendingActionIncoming, connections[0].PendingActionHint)
	})

	t.Run("counterpart_user_id_is_other_party", func(t *testing.T) {
		connRepo := new(MockConnectionRepository)
		userRepo := new(MockUserRepository)
		service := NewConnectionService(connRepo, userRepo)

		user1 := uuid.New()
		user2 := uuid.New()

		// Mock connection where auth user is friend_id
		connection := &domain.Connection{
			ID:       uuid.New(),
			UserID:   user1,
			FriendID: user2, // Auth user
			Status:   domain.ConnectionAccepted,
		}
		connRepo.On("ListByUserID", ctx, user2, domain.ConnectionAccepted).Return([]*domain.Connection{connection}, nil)

		// Mock profile
		profile := &domain.Profile{
			UserID:   user1,
			Username: "user1",
		}
		userRepo.On("GetProfileByUserID", ctx, user1).Return(profile, nil)

		// Get connections
		connections, err := service.GetMyConnections(ctx, user2, domain.ConnectionAccepted)

		require.NoError(t, err)
		require.Len(t, connections, 1)
		assert.Equal(t, user1.String(), connections[0].CounterpartUserID, "Counterpart should be the other user")
		assert.Equal(t, "user1", connections[0].Profile.Username)
	})

	t.Run("accepted_preserves_correct_direction", func(t *testing.T) {
		connRepo := new(MockConnectionRepository)
		userRepo := new(MockUserRepository)
		service := NewConnectionService(connRepo, userRepo)

		// Mock accepted connection (auth user was initiator)
		connection := &domain.Connection{
			ID:       uuid.New(),
			UserID:   authUserID, // Auth user initiated
			FriendID: otherUserID,
			Status:   domain.ConnectionAccepted,
		}
		connRepo.On("ListByUserID", ctx, authUserID, domain.ConnectionAccepted).Return([]*domain.Connection{connection}, nil)

		// Mock profile
		profile := &domain.Profile{
			UserID:   otherUserID,
			Username: "otheruser",
		}
		userRepo.On("GetProfileByUserID", ctx, otherUserID).Return(profile, nil)

		// Get connections
		connections, err := service.GetMyConnections(ctx, authUserID, domain.ConnectionAccepted)

		require.NoError(t, err)
		require.Len(t, connections, 1)
		assert.True(t, connections[0].InitiatedByMe, "Should preserve direction even after acceptance")
		assert.Empty(t, connections[0].PendingActionHint, "Accepted connections should not have action hint")
	})

	t.Run("blocked_preserves_correct_direction", func(t *testing.T) {
		connRepo := new(MockConnectionRepository)
		userRepo := new(MockUserRepository)
		service := NewConnectionService(connRepo, userRepo)

		// Mock blocked connection (auth user blocked the other)
		connection := &domain.Connection{
			ID:       uuid.New(),
			UserID:   authUserID, // Auth user initiated block
			FriendID: otherUserID,
			Status:   domain.ConnectionBlocked,
		}
		connRepo.On("ListByUserID", ctx, authUserID, domain.ConnectionBlocked).Return([]*domain.Connection{connection}, nil)

		// Mock profile
		profile := &domain.Profile{
			UserID:   otherUserID,
			Username: "blockeduser",
		}
		userRepo.On("GetProfileByUserID", ctx, otherUserID).Return(profile, nil)

		// Get connections
		connections, err := service.GetMyConnections(ctx, authUserID, domain.ConnectionBlocked)

		require.NoError(t, err)
		require.Len(t, connections, 1)
		assert.True(t, connections[0].InitiatedByMe, "Should show who initiated the block")
		assert.Empty(t, connections[0].PendingActionHint, "Blocked connections should not have action hint")
	})
}

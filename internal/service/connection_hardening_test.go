package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/nimio/server/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestConnectionService_SendFriendRequest_SelfRequest tests that self-requests are rejected
func TestConnectionService_SendFriendRequest_SelfRequest(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	connRepo := new(MockConnectionRepository)
	userRepo := new(MockUserRepository)
	service := NewConnectionService(connRepo, userRepo)

	// Attempt to send request to self
	connection, err := service.SendFriendRequest(ctx, userID, userID, domain.RelationshipMutual)

	assert.Nil(t, connection)
	assert.ErrorIs(t, err, domain.ErrSelfConnection)
	connRepo.AssertNotCalled(t, "Create")
}

// TestConnectionService_SendFriendRequest_DuplicatePending tests duplicate pending request rejection
func TestConnectionService_SendFriendRequest_DuplicatePending(t *testing.T) {
	ctx := context.Background()
	fromUserID := uuid.New()
	toUserID := uuid.New()

	connRepo := new(MockConnectionRepository)
	userRepo := new(MockUserRepository)
	service := NewConnectionService(connRepo, userRepo)

	// Mock user exists
	userRepo.On("GetByID", ctx, toUserID).Return(&domain.User{ID: toUserID}, nil)

	// Mock existing pending connection
	existingConn := &domain.Connection{
		ID:       uuid.New(),
		UserID:   fromUserID,
		FriendID: toUserID,
		Status:   domain.ConnectionPending,
	}
	connRepo.On("GetByUsers", ctx, fromUserID, toUserID).Return(existingConn, nil)

	// Attempt to send duplicate request
	connection, err := service.SendFriendRequest(ctx, fromUserID, toUserID, domain.RelationshipMutual)

	assert.Nil(t, connection)
	assert.ErrorIs(t, err, domain.ErrDuplicatePending)
	connRepo.AssertNotCalled(t, "Create")
}

// TestConnectionService_SendFriendRequest_AlreadyConnected tests rejection when already connected
func TestConnectionService_SendFriendRequest_AlreadyConnected(t *testing.T) {
	ctx := context.Background()
	fromUserID := uuid.New()
	toUserID := uuid.New()

	connRepo := new(MockConnectionRepository)
	userRepo := new(MockUserRepository)
	service := NewConnectionService(connRepo, userRepo)

	// Mock user exists
	userRepo.On("GetByID", ctx, toUserID).Return(&domain.User{ID: toUserID}, nil)

	// Mock existing accepted connection
	existingConn := &domain.Connection{
		ID:       uuid.New(),
		UserID:   fromUserID,
		FriendID: toUserID,
		Status:   domain.ConnectionAccepted,
	}
	connRepo.On("GetByUsers", ctx, fromUserID, toUserID).Return(existingConn, nil)

	// Attempt to send request when already connected
	connection, err := service.SendFriendRequest(ctx, fromUserID, toUserID, domain.RelationshipMutual)

	assert.Nil(t, connection)
	assert.ErrorIs(t, err, domain.ErrAlreadyConnected)
	connRepo.AssertNotCalled(t, "Create")
}

// TestConnectionService_SendFriendRequest_Blocked tests rejection when connection is blocked
func TestConnectionService_SendFriendRequest_Blocked(t *testing.T) {
	ctx := context.Background()
	fromUserID := uuid.New()
	toUserID := uuid.New()

	connRepo := new(MockConnectionRepository)
	userRepo := new(MockUserRepository)
	service := NewConnectionService(connRepo, userRepo)

	// Mock user exists
	userRepo.On("GetByID", ctx, toUserID).Return(&domain.User{ID: toUserID}, nil)

	// Mock existing blocked connection
	existingConn := &domain.Connection{
		ID:       uuid.New(),
		UserID:   toUserID,
		FriendID: fromUserID,
		Status:   domain.ConnectionBlocked,
	}
	connRepo.On("GetByUsers", ctx, fromUserID, toUserID).Return(existingConn, nil)

	// Attempt to send request when blocked
	connection, err := service.SendFriendRequest(ctx, fromUserID, toUserID, domain.RelationshipMutual)

	assert.Nil(t, connection)
	assert.ErrorIs(t, err, domain.ErrConnectionBlocked)
	connRepo.AssertNotCalled(t, "Create")
}

// TestConnectionService_SendFriendRequest_ConcurrentDuplicate tests DB-level race protection
func TestConnectionService_SendFriendRequest_ConcurrentDuplicate(t *testing.T) {
	ctx := context.Background()
	fromUserID := uuid.New()
	toUserID := uuid.New()

	connRepo := new(MockConnectionRepository)
	userRepo := new(MockUserRepository)
	service := NewConnectionService(connRepo, userRepo)

	// Mock user exists
	userRepo.On("GetByID", ctx, toUserID).Return(&domain.User{ID: toUserID}, nil)

	// Mock no existing connection initially
	connRepo.On("GetByUsers", ctx, fromUserID, toUserID).Return(nil, domain.ErrNotFound).Once()

	// Mock unique constraint violation from concurrent insert
	connRepo.On("Create", ctx, mock.AnythingOfType("*domain.Connection")).Return(domain.ErrAlreadyExists).Once()

	// Mock re-check after unique constraint violation
	existingConn := &domain.Connection{
		ID:       uuid.New(),
		UserID:   toUserID,
		FriendID: fromUserID,
		Status:   domain.ConnectionPending,
	}
	connRepo.On("GetByUsers", ctx, fromUserID, toUserID).Return(existingConn, nil).Once()

	// Attempt to send request (simulating concurrent request scenario)
	connection, err := service.SendFriendRequest(ctx, fromUserID, toUserID, domain.RelationshipMutual)

	assert.Nil(t, connection)
	assert.ErrorIs(t, err, domain.ErrDuplicatePending)
	connRepo.AssertCalled(t, "Create", ctx, mock.AnythingOfType("*domain.Connection"))
}

// TestConnectionService_SendFriendRequest_Success tests successful request creation
func TestConnectionService_SendFriendRequest_Success(t *testing.T) {
	ctx := context.Background()
	fromUserID := uuid.New()
	toUserID := uuid.New()

	connRepo := new(MockConnectionRepository)
	userRepo := new(MockUserRepository)
	service := NewConnectionService(connRepo, userRepo)

	// Mock user exists
	userRepo.On("GetByID", ctx, toUserID).Return(&domain.User{ID: toUserID}, nil)

	// Mock no existing connection
	connRepo.On("GetByUsers", ctx, fromUserID, toUserID).Return(nil, domain.ErrNotFound)

	// Mock successful create
	connRepo.On("Create", ctx, mock.AnythingOfType("*domain.Connection")).Return(nil)

	// Send request
	connection, err := service.SendFriendRequest(ctx, fromUserID, toUserID, domain.RelationshipMutual)

	assert.NoError(t, err)
	assert.NotNil(t, connection)
	assert.Equal(t, fromUserID, connection.UserID)
	assert.Equal(t, toUserID, connection.FriendID)
	assert.Equal(t, domain.ConnectionPending, connection.Status)
	assert.Equal(t, domain.RelationshipMutual, connection.RelationshipTier)
}

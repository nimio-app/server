package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nimio/server/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock repositories
type MockConnectionRepository struct {
	mock.Mock
}

func (m *MockConnectionRepository) Create(ctx context.Context, connection *domain.Connection) error {
	args := m.Called(ctx, connection)
	return args.Error(0)
}

func (m *MockConnectionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Connection, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Connection), args.Error(1)
}

func (m *MockConnectionRepository) GetByUsers(ctx context.Context, userID, friendID uuid.UUID) (*domain.Connection, error) {
	args := m.Called(ctx, userID, friendID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Connection), args.Error(1)
}

func (m *MockConnectionRepository) Update(ctx context.Context, connection *domain.Connection) error {
	args := m.Called(ctx, connection)
	return args.Error(0)
}

func (m *MockConnectionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockConnectionRepository) ListByUserID(ctx context.Context, userID uuid.UUID, status domain.ConnectionStatus) ([]*domain.Connection, error) {
	args := m.Called(ctx, userID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Connection), args.Error(1)
}

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*domain.Profile, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Profile), args.Error(1)
}

// Implement other required methods with panics (won't be called in these tests)
func (m *MockUserRepository) Create(ctx context.Context, user *domain.User, profile *domain.Profile) error {
	panic("not implemented")
}
func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	panic("not implemented")
}
func (m *MockUserRepository) GetByGoogleID(ctx context.Context, googleID string) (*domain.User, error) {
	panic("not implemented")
}
func (m *MockUserRepository) GetByVerificationToken(ctx context.Context, token string) (*domain.User, error) {
	panic("not implemented")
}
func (m *MockUserRepository) UpdateVerificationToken(ctx context.Context, userID uuid.UUID, token *string, expiresAt *time.Time) error {
	panic("not implemented")
}
func (m *MockUserRepository) MarkEmailAsVerified(ctx context.Context, userID uuid.UUID) error {
	panic("not implemented")
}
func (m *MockUserRepository) GetProfileByUsername(ctx context.Context, username string) (*domain.Profile, error) {
	panic("not implemented")
}
func (m *MockUserRepository) UpdateProfile(ctx context.Context, profile *domain.Profile) error {
	panic("not implemented")
}
func (m *MockUserRepository) SearchUsers(ctx context.Context, query string, limit int) ([]*domain.Profile, error) {
	panic("not implemented")
}

func TestConnectionService_SendFriendRequest(t *testing.T) {
	ctx := context.Background()
	fromUserID := uuid.New()
	toUserID := uuid.New()

	t.Run("success - send friend request", func(t *testing.T) {
		connRepo := new(MockConnectionRepository)
		userRepo := new(MockUserRepository)
		service := NewConnectionService(connRepo, userRepo)

		// Mock user exists
		userRepo.On("GetByID", ctx, toUserID).Return(&domain.User{ID: toUserID}, nil)

		// Mock no existing connection
		connRepo.On("GetByUsers", ctx, fromUserID, toUserID).Return(nil, domain.ErrNotFound)

		// Mock create
		connRepo.On("Create", ctx, mock.AnythingOfType("*domain.Connection")).Return(nil)

		connection, err := service.SendFriendRequest(ctx, fromUserID, toUserID, domain.RelationshipMutual)

		require.NoError(t, err)
		assert.NotNil(t, connection)
		assert.Equal(t, fromUserID, connection.UserID)
		assert.Equal(t, toUserID, connection.FriendID)
		assert.Equal(t, domain.ConnectionPending, connection.Status)
		assert.Equal(t, domain.RelationshipMutual, connection.RelationshipTier)

		connRepo.AssertExpectations(t)
		userRepo.AssertExpectations(t)
	})

	t.Run("error - cannot send to self", func(t *testing.T) {
		connRepo := new(MockConnectionRepository)
		userRepo := new(MockUserRepository)
		service := NewConnectionService(connRepo, userRepo)

		_, err := service.SendFriendRequest(ctx, fromUserID, fromUserID, domain.RelationshipMutual)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot send friend request to yourself")
	})

	t.Run("error - user not found", func(t *testing.T) {
		connRepo := new(MockConnectionRepository)
		userRepo := new(MockUserRepository)
		service := NewConnectionService(connRepo, userRepo)

		userRepo.On("GetByID", ctx, toUserID).Return(nil, domain.ErrNotFound)

		_, err := service.SendFriendRequest(ctx, fromUserID, toUserID, domain.RelationshipMutual)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")

		userRepo.AssertExpectations(t)
	})

	t.Run("error - request already pending", func(t *testing.T) {
		connRepo := new(MockConnectionRepository)
		userRepo := new(MockUserRepository)
		service := NewConnectionService(connRepo, userRepo)

		userRepo.On("GetByID", ctx, toUserID).Return(&domain.User{ID: toUserID}, nil)
		connRepo.On("GetByUsers", ctx, fromUserID, toUserID).Return(&domain.Connection{
			Status: domain.ConnectionPending,
		}, nil)

		_, err := service.SendFriendRequest(ctx, fromUserID, toUserID, domain.RelationshipMutual)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "already pending")

		connRepo.AssertExpectations(t)
		userRepo.AssertExpectations(t)
	})
}

func TestConnectionService_AcceptFriendRequest(t *testing.T) {
	ctx := context.Background()
	requesterID := uuid.New()
	recipientID := uuid.New()

	t.Run("success - accept pending request", func(t *testing.T) {
		connRepo := new(MockConnectionRepository)
		userRepo := new(MockUserRepository)
		service := NewConnectionService(connRepo, userRepo)

		existingConn := &domain.Connection{
			ID:       uuid.New(),
			UserID:   requesterID,
			FriendID: recipientID,
			Status:   domain.ConnectionPending,
		}

		// Mock get connection
		connRepo.On("GetByUsers", ctx, requesterID, recipientID).Return(existingConn, nil)

		// Mock update
		connRepo.On("Update", ctx, mock.AnythingOfType("*domain.Connection")).Return(nil)

		connection, err := service.AcceptFriendRequest(ctx, recipientID, requesterID)

		require.NoError(t, err)
		assert.NotNil(t, connection)
		assert.Equal(t, domain.ConnectionAccepted, connection.Status)

		connRepo.AssertExpectations(t)
	})

	t.Run("error - request not found", func(t *testing.T) {
		connRepo := new(MockConnectionRepository)
		userRepo := new(MockUserRepository)
		service := NewConnectionService(connRepo, userRepo)

		connRepo.On("GetByUsers", ctx, requesterID, recipientID).Return(nil, domain.ErrNotFound)

		_, err := service.AcceptFriendRequest(ctx, recipientID, requesterID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		connRepo.AssertExpectations(t)
	})

	t.Run("error - cannot accept request you didn't receive", func(t *testing.T) {
		connRepo := new(MockConnectionRepository)
		userRepo := new(MockUserRepository)
		service := NewConnectionService(connRepo, userRepo)

		existingConn := &domain.Connection{
			ID:       uuid.New(),
			UserID:   recipientID, // Recipient is the requester
			FriendID: requesterID,
			Status:   domain.ConnectionPending,
		}

		connRepo.On("GetByUsers", ctx, requesterID, recipientID).Return(existingConn, nil)

		_, err := service.AcceptFriendRequest(ctx, recipientID, requesterID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot accept request you didn't receive")

		connRepo.AssertExpectations(t)
	})
}

func TestConnectionService_BlockUser(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	blockedUserID := uuid.New()

	t.Run("success - block user with no existing connection", func(t *testing.T) {
		connRepo := new(MockConnectionRepository)
		userRepo := new(MockUserRepository)
		service := NewConnectionService(connRepo, userRepo)

		// No existing connection
		connRepo.On("GetByUsers", ctx, userID, blockedUserID).Return(nil, domain.ErrNotFound)

		// Create blocked connection
		connRepo.On("Create", ctx, mock.AnythingOfType("*domain.Connection")).Return(nil)

		connection, err := service.BlockUser(ctx, userID, blockedUserID)

		require.NoError(t, err)
		assert.NotNil(t, connection)
		assert.Equal(t, domain.ConnectionBlocked, connection.Status)

		connRepo.AssertExpectations(t)
	})

	t.Run("success - block existing connection", func(t *testing.T) {
		connRepo := new(MockConnectionRepository)
		userRepo := new(MockUserRepository)
		service := NewConnectionService(connRepo, userRepo)

		existingConn := &domain.Connection{
			ID:     uuid.New(),
			Status: domain.ConnectionAccepted,
		}

		connRepo.On("GetByUsers", ctx, userID, blockedUserID).Return(existingConn, nil)
		connRepo.On("Update", ctx, mock.AnythingOfType("*domain.Connection")).Return(nil)

		connection, err := service.BlockUser(ctx, userID, blockedUserID)

		require.NoError(t, err)
		assert.NotNil(t, connection)
		assert.Equal(t, domain.ConnectionBlocked, connection.Status)

		connRepo.AssertExpectations(t)
	})

	t.Run("error - cannot block yourself", func(t *testing.T) {
		connRepo := new(MockConnectionRepository)
		userRepo := new(MockUserRepository)
		service := NewConnectionService(connRepo, userRepo)

		_, err := service.BlockUser(ctx, userID, userID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot block yourself")
	})
}

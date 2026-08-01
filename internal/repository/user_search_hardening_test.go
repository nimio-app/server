package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUserRepository_SearchUsers_ExcludesAuthUser tests that search excludes the authenticated user
func TestUserRepository_SearchUsers_ExcludesAuthUser(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &userRepository{db: mock}
	
	authUserID := uuid.New()
	otherUserID := uuid.New()
	query := "john"
	limit := 20
	now := time.Now()

	// Mock expects query with excludeUserID parameter
	rows := pgxmock.NewRows([]string{
		"user_id", "username", "display_name", "avatar_url", "bio", "created_at", "updated_at",
	}).AddRow(otherUserID, "johndoe", "John Doe", nil, nil, now, now)

	mock.ExpectQuery(`SELECT (.+) FROM profiles p`).
		WithArgs("%john%", "john", 20, authUserID).
		WillReturnRows(rows)

	profiles, err := repo.SearchUsers(context.Background(), query, limit, &authUserID)

	require.NoError(t, err)
	assert.Len(t, profiles, 1)
	assert.Equal(t, "johndoe", profiles[0].Username)
	assert.NotEqual(t, authUserID, profiles[0].UserID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUserRepository_SearchUsers_AllowsNilExclude tests that search works without exclusion
func TestUserRepository_SearchUsers_AllowsNilExclude(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &userRepository{db: mock}
	
	query := "test"
	limit := 10
	now := time.Now()

	rows := pgxmock.NewRows([]string{
		"user_id", "username", "display_name", "avatar_url", "bio", "created_at", "updated_at",
	}).AddRow(uuid.New(), "testuser", "Test User", nil, nil, now, now)

	// When excludeUserID is nil, query should not have the 4th parameter
	mock.ExpectQuery(`SELECT (.+) FROM profiles p`).
		WithArgs("%test%", "test", 10).
		WillReturnRows(rows)

	profiles, err := repo.SearchUsers(context.Background(), query, limit, nil)

	require.NoError(t, err)
	assert.Len(t, profiles, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

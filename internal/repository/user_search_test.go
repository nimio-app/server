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

func TestUserRepository_SearchUsers(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &userRepository{db: mock}
	now := time.Now()

	t.Run("search by username", func(t *testing.T) {
		query := "john"
		limit := 20
		avatarURL := "https://avatar.url"
		bio := "Bio"

		rows := pgxmock.NewRows([]string{
			"user_id", "username", "display_name", "avatar_url", "bio", "created_at", "updated_at",
		}).
			AddRow(uuid.New(), "johndoe", "John Doe", &avatarURL, &bio, now, now).
			AddRow(uuid.New(), "johnny_test", "Johnny Test", (*string)(nil), (*string)(nil), now, now)

		mock.ExpectQuery(`SELECT (.+) FROM profiles p`).
			WithArgs("%john%", "john", 20).
			WillReturnRows(rows)

		profiles, err := repo.SearchUsers(context.Background(), query, limit)

		require.NoError(t, err)
		assert.Len(t, profiles, 2)
		assert.Equal(t, "johndoe", profiles[0].Username)
		assert.Equal(t, "johnny_test", profiles[1].Username)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty results", func(t *testing.T) {
		query := "nonexistent"
		limit := 20

		rows := pgxmock.NewRows([]string{
			"user_id", "username", "display_name", "avatar_url", "bio", "created_at", "updated_at",
		})

		mock.ExpectQuery(`SELECT (.+) FROM profiles p`).
			WithArgs("%nonexistent%", "nonexistent", 20).
			WillReturnRows(rows)

		profiles, err := repo.SearchUsers(context.Background(), query, limit)

		require.NoError(t, err)
		assert.Empty(t, profiles)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_UpdateProfile(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &userRepository{db: mock}
	now := time.Now()

	t.Run("successful update with all fields", func(t *testing.T) {
		userID := uuid.New()
		username := "newusername"
		displayName := "New Name"
		bio := "New bio"
		avatarURL := "https://new-avatar.url"

		profile := &domain.Profile{
			UserID:      userID,
			Username:    username,
			DisplayName: displayName,
			AvatarURL:   &avatarURL,
			Bio:         &bio,
		}

		mock.ExpectQuery(`UPDATE profiles`).
			WithArgs(username, displayName, &avatarURL, &bio, userID).
			WillReturnRows(pgxmock.NewRows([]string{"updated_at"}).AddRow(now))

		err := repo.UpdateProfile(context.Background(), profile)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

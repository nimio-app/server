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

// ConnectionRepository handles connection-related database operations
type ConnectionRepository interface {
	Create(ctx context.Context, connection *domain.Connection) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Connection, error)
	GetByUsers(ctx context.Context, userID, friendID uuid.UUID) (*domain.Connection, error)
	Update(ctx context.Context, connection *domain.Connection) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByUserID(ctx context.Context, userID uuid.UUID, status domain.ConnectionStatus) ([]*domain.Connection, error)
}

type connectionRepository struct {
	db *pgxpool.Pool
}

// NewConnectionRepository creates a new connection repository
func NewConnectionRepository(db *pgxpool.Pool) ConnectionRepository {
	return &connectionRepository{db: db}
}

// Create creates a new connection
func (r *connectionRepository) Create(ctx context.Context, connection *domain.Connection) error {
	query := `
		INSERT INTO connections (id, user_id, friend_id, relationship_tier, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRow(ctx, query,
		connection.ID, connection.UserID, connection.FriendID,
		connection.RelationshipTier, connection.Status,
		connection.CreatedAt, connection.UpdatedAt,
	).Scan(&connection.ID, &connection.CreatedAt, &connection.UpdatedAt)
	
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrAlreadyExists
		}
		return fmt.Errorf("insert connection: %w", err)
	}
	return nil
}

// GetByID retrieves a connection by ID
func (r *connectionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Connection, error) {
	query := `
		SELECT id, user_id, friend_id, relationship_tier, status, created_at, updated_at
		FROM connections
		WHERE id = $1
	`
	connection := &domain.Connection{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&connection.ID, &connection.UserID, &connection.FriendID,
		&connection.RelationshipTier, &connection.Status,
		&connection.CreatedAt, &connection.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("query connection: %w", err)
	}
	return connection, nil
}

// GetByUsers retrieves a connection between two users (bidirectional)
func (r *connectionRepository) GetByUsers(ctx context.Context, userID, friendID uuid.UUID) (*domain.Connection, error) {
	query := `
		SELECT id, user_id, friend_id, relationship_tier, status, created_at, updated_at
		FROM connections
		WHERE (user_id = $1 AND friend_id = $2) OR (user_id = $2 AND friend_id = $1)
	`
	connection := &domain.Connection{}
	err := r.db.QueryRow(ctx, query, userID, friendID).Scan(
		&connection.ID, &connection.UserID, &connection.FriendID,
		&connection.RelationshipTier, &connection.Status,
		&connection.CreatedAt, &connection.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("query connection: %w", err)
	}
	return connection, nil
}

// Update updates a connection
func (r *connectionRepository) Update(ctx context.Context, connection *domain.Connection) error {
	query := `
		UPDATE connections
		SET relationship_tier = $1, status = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING updated_at
	`
	err := r.db.QueryRow(ctx, query,
		connection.RelationshipTier, connection.Status, connection.ID,
	).Scan(&connection.UpdatedAt)
	
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("update connection: %w", err)
	}
	return nil
}

// Delete deletes a connection
func (r *connectionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM connections WHERE id = $1`
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete connection: %w", err)
	}
	
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	
	return nil
}

// ListByUserID retrieves all connections for a user with optional status filter
func (r *connectionRepository) ListByUserID(ctx context.Context, userID uuid.UUID, status domain.ConnectionStatus) ([]*domain.Connection, error) {
	query := `
		SELECT id, user_id, friend_id, relationship_tier, status, created_at, updated_at
		FROM connections
		WHERE (user_id = $1 OR friend_id = $1)
	`
	args := []interface{}{userID}
	
	if status != "" {
		query += ` AND status = $2`
		args = append(args, status)
	}
	
	query += ` ORDER BY created_at DESC`
	
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query connections: %w", err)
	}
	defer rows.Close()

	var connections []*domain.Connection
	for rows.Next() {
		conn := &domain.Connection{}
		err := rows.Scan(
			&conn.ID, &conn.UserID, &conn.FriendID,
			&conn.RelationshipTier, &conn.Status,
			&conn.CreatedAt, &conn.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan connection: %w", err)
		}
		connections = append(connections, conn)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}

	return connections, nil
}

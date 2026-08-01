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
	// Normalize deprecated MUTUAL to ALL
	userTier := domain.NormalizeRelationshipTier(connection.UserTier)
	friendTier := domain.NormalizeRelationshipTier(connection.FriendTier)
	
	query := `
		INSERT INTO connections (id, user_id, friend_id, relationship_tier, user_tier, friend_tier, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRow(ctx, query,
		connection.ID, connection.UserID, connection.FriendID,
		userTier, userTier, friendTier, // Set relationship_tier to user_tier for backward compat
		connection.Status,
		connection.CreatedAt, connection.UpdatedAt,
	).Scan(&connection.ID, &connection.CreatedAt, &connection.UpdatedAt)
	
	if err != nil {
		// Catch unique constraint violation (both old and new bidirectional constraint)
		if isUniqueViolation(err) {
			return domain.ErrAlreadyExists
		}
		return fmt.Errorf("insert connection: %w", err)
	}
	
	// Update the struct with normalized values
	connection.UserTier = userTier
	connection.FriendTier = friendTier
	connection.RelationshipTier = userTier
	
	return nil
}

// GetByID retrieves a connection by ID
func (r *connectionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Connection, error) {
	query := `
		SELECT id, user_id, friend_id, relationship_tier, user_tier, friend_tier, status, created_at, updated_at
		FROM connections
		WHERE id = $1
	`
	connection := &domain.Connection{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&connection.ID, &connection.UserID, &connection.FriendID,
		&connection.RelationshipTier, &connection.UserTier, &connection.FriendTier,
		&connection.Status,
		&connection.CreatedAt, &connection.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("query connection: %w", err)
	}
	
	// Normalize deprecated values
	connection.UserTier = domain.NormalizeRelationshipTier(connection.UserTier)
	connection.FriendTier = domain.NormalizeRelationshipTier(connection.FriendTier)
	connection.RelationshipTier = domain.NormalizeRelationshipTier(connection.RelationshipTier)
	
	return connection, nil
}

// GetByUsers retrieves a connection between two users (bidirectional)
func (r *connectionRepository) GetByUsers(ctx context.Context, userID, friendID uuid.UUID) (*domain.Connection, error) {
	query := `
		SELECT id, user_id, friend_id, relationship_tier, user_tier, friend_tier, status, created_at, updated_at
		FROM connections
		WHERE (user_id = $1 AND friend_id = $2) OR (user_id = $2 AND friend_id = $1)
	`
	connection := &domain.Connection{}
	err := r.db.QueryRow(ctx, query, userID, friendID).Scan(
		&connection.ID, &connection.UserID, &connection.FriendID,
		&connection.RelationshipTier, &connection.UserTier, &connection.FriendTier,
		&connection.Status,
		&connection.CreatedAt, &connection.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("query connection: %w", err)
	}
	
	// Normalize deprecated values
	connection.UserTier = domain.NormalizeRelationshipTier(connection.UserTier)
	connection.FriendTier = domain.NormalizeRelationshipTier(connection.FriendTier)
	connection.RelationshipTier = domain.NormalizeRelationshipTier(connection.RelationshipTier)
	
	return connection, nil
}

// Update updates a connection
func (r *connectionRepository) Update(ctx context.Context, connection *domain.Connection) error {
	// Normalize deprecated MUTUAL to ALL
	userTier := domain.NormalizeRelationshipTier(connection.UserTier)
	friendTier := domain.NormalizeRelationshipTier(connection.FriendTier)
	
	query := `
		UPDATE connections
		SET user_tier = $1, friend_tier = $2, relationship_tier = $1, status = $3, updated_at = NOW()
		WHERE id = $4
		RETURNING updated_at
	`
	err := r.db.QueryRow(ctx, query,
		userTier, friendTier, connection.Status, connection.ID,
	).Scan(&connection.UpdatedAt)
	
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("update connection: %w", err)
	}
	
	// Update struct with normalized values
	connection.UserTier = userTier
	connection.FriendTier = friendTier
	connection.RelationshipTier = userTier
	
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
		SELECT id, user_id, friend_id, relationship_tier, user_tier, friend_tier, status, created_at, updated_at
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
			&conn.RelationshipTier, &conn.UserTier, &conn.FriendTier,
			&conn.Status,
			&conn.CreatedAt, &conn.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan connection: %w", err)
		}
		
		// Normalize deprecated values
		conn.UserTier = domain.NormalizeRelationshipTier(conn.UserTier)
		conn.FriendTier = domain.NormalizeRelationshipTier(conn.FriendTier)
		conn.RelationshipTier = domain.NormalizeRelationshipTier(conn.RelationshipTier)
		
		connections = append(connections, conn)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}

	return connections, nil
}

package domain

import (
	"time"

	"github.com/google/uuid"
)

// RelationshipTier defines the privacy tier of a connection
type RelationshipTier string

const (
	RelationshipAll    RelationshipTier = "ALL"
	RelationshipCircle RelationshipTier = "CIRCLE"
	RelationshipMutual RelationshipTier = "MUTUAL"
)

// ConnectionStatus defines the state of a connection
type ConnectionStatus string

const (
	ConnectionPending  ConnectionStatus = "PENDING"
	ConnectionAccepted ConnectionStatus = "ACCEPTED"
	ConnectionBlocked  ConnectionStatus = "BLOCKED"
)

// Connection represents a relationship between two users
type Connection struct {
	ID               uuid.UUID        `json:"id"`
	UserID           uuid.UUID        `json:"user_id"`
	FriendID         uuid.UUID        `json:"friend_id"`
	RelationshipTier RelationshipTier `json:"relationship_tier"`
	Status           ConnectionStatus `json:"status"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// PendingActionHint indicates the action available for a pending connection
type PendingActionHint string

const (
	PendingActionIncoming PendingActionHint = "INCOMING" // Can Accept/Decline
	PendingActionOutgoing PendingActionHint = "OUTGOING" // Can Cancel
)

// ConnectionWithProfile combines connection data with the friend's profile
type ConnectionWithProfile struct {
	Connection         Connection       `json:"connection"`
	Profile            Profile          `json:"profile"`
	InitiatedByMe      bool             `json:"initiated_by_me"`
	CounterpartUserID  string           `json:"counterpart_user_id"`
	PendingActionHint  PendingActionHint `json:"pending_action_hint,omitempty"`
}

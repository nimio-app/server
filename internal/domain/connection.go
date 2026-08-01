package domain

import (
	"time"

	"github.com/google/uuid"
)

// RelationshipTier defines the privacy tier of a connection
// Note: MUTUAL is deprecated and mapped to ALL for backward compatibility
type RelationshipTier string

const (
	RelationshipAll    RelationshipTier = "ALL"    // Default tier for all connections
	RelationshipCircle RelationshipTier = "CIRCLE" // Close friends/inner circle
	RelationshipMutual RelationshipTier = "MUTUAL" // DEPRECATED: Use ALL instead
)

// NormalizeRelationshipTier maps deprecated MUTUAL to ALL
func NormalizeRelationshipTier(tier RelationshipTier) RelationshipTier {
	if tier == RelationshipMutual {
		return RelationshipAll
	}
	return tier
}

// IsValidRelationshipTier checks if a tier is valid (ALL or CIRCLE)
func IsValidRelationshipTier(tier RelationshipTier) bool {
	normalized := NormalizeRelationshipTier(tier)
	return normalized == RelationshipAll || normalized == RelationshipCircle
}

// ConnectionStatus defines the state of a connection
type ConnectionStatus string

const (
	ConnectionPending  ConnectionStatus = "PENDING"
	ConnectionAccepted ConnectionStatus = "ACCEPTED"
	ConnectionBlocked  ConnectionStatus = "BLOCKED"
)

// Connection represents a relationship between two users
// Tiers are now directional: each user can assign a different tier to the other
type Connection struct {
	ID               uuid.UUID        `json:"id"`
	UserID           uuid.UUID        `json:"user_id"`
	FriendID         uuid.UUID        `json:"friend_id"`
	RelationshipTier RelationshipTier `json:"relationship_tier"` // DEPRECATED: Use UserTier/FriendTier
	UserTier         RelationshipTier `json:"user_tier"`         // Tier user_id assigns to friend_id
	FriendTier       RelationshipTier `json:"friend_tier"`       // Tier friend_id assigns to user_id
	Status           ConnectionStatus `json:"status"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// GetTierFor returns the tier that fromUser has assigned to toUser
func (c *Connection) GetTierFor(fromUser, toUser uuid.UUID) RelationshipTier {
	if c.UserID == fromUser && c.FriendID == toUser {
		return c.UserTier
	}
	if c.FriendID == fromUser && c.UserID == toUser {
		return c.FriendTier
	}
	return RelationshipAll // Default fallback
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
	MyTierForThem      RelationshipTier `json:"my_tier_for_them"`      // How I categorize them
	TheirTierForMe     RelationshipTier `json:"their_tier_for_me"`     // How they categorize me
	PendingActionHint  PendingActionHint `json:"pending_action_hint,omitempty"`
}

package domain

import (
	"time"

	"github.com/google/uuid"
)

// AvailabilityType represents the user's current availability state
type AvailabilityType string

const (
	AvailabilityFree       AvailabilityType = "FREE"
	AvailabilityBusy       AvailabilityType = "BUSY"
	AvailabilityFocus      AvailabilityType = "FOCUS"
	AvailabilityDriving    AvailabilityType = "DRIVING"
	AvailabilityWantToTalk AvailabilityType = "WANT_TO_TALK"
)

// VisibilityTier controls who can see a status
type VisibilityTier string

const (
	VisibilityAllConnections VisibilityTier = "ALL_CONNECTIONS"
	VisibilityCircleOnly     VisibilityTier = "CIRCLE_ONLY"
	VisibilityCustomList     VisibilityTier = "CUSTOM_LIST"
)

// Status represents a user's intentional availability state
type Status struct {
	ID               uuid.UUID        `json:"id"`
	UserID           uuid.UUID        `json:"user_id"`
	AvailabilityType AvailabilityType `json:"availability_type"`
	Note             *string          `json:"note,omitempty"`
	VisibilityTier   VisibilityTier   `json:"visibility_tier"`
	ExpiresAt        *time.Time       `json:"expires_at,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
	IsActive         bool             `json:"is_active"`
}

// StatusWithProfile includes the user's profile information with their status
type StatusWithProfile struct {
	Status  Status  `json:"status"`
	Profile Profile `json:"profile"`
}

// IsExpired checks if the status has expired
func (s *Status) IsExpired() bool {
	if s.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*s.ExpiresAt)
}

// IsValidAvailabilityType checks if the availability type is valid
func IsValidAvailabilityType(t string) bool {
	switch AvailabilityType(t) {
	case AvailabilityFree, AvailabilityBusy, AvailabilityFocus, AvailabilityDriving, AvailabilityWantToTalk:
		return true
	}
	return false
}

// IsValidVisibilityTier checks if the visibility tier is valid
func IsValidVisibilityTier(t string) bool {
	switch VisibilityTier(t) {
	case VisibilityAllConnections, VisibilityCircleOnly, VisibilityCustomList:
		return true
	}
	return false
}

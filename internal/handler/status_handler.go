package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/nimio/server/internal/domain"
	"github.com/nimio/server/internal/middleware"
	"github.com/nimio/server/internal/service"
)

// StatusHandler handles status-related endpoints
type StatusHandler struct {
	statusService service.StatusService
}

// NewStatusHandler creates a new status handler
func NewStatusHandler(statusService service.StatusService) *StatusHandler {
	return &StatusHandler{
		statusService: statusService,
	}
}

// CreateStatusRequest represents the create/update status request body
type CreateStatusRequest struct {
	AvailabilityType string     `json:"availability_type"`
	Note             *string    `json:"note,omitempty"`
	VisibilityTier   string     `json:"visibility_tier"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
}

// CreateStatus handles PUT /v1/me/status
func (h *StatusHandler) CreateStatus(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		ErrorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ValidationErrorResponse(w, "invalid request body")
		return
	}

	// Validate input
	if !domain.IsValidAvailabilityType(req.AvailabilityType) {
		ValidationErrorResponse(w, "invalid availability_type")
		return
	}

	if !domain.IsValidVisibilityTier(req.VisibilityTier) {
		ValidationErrorResponse(w, "invalid visibility_tier")
		return
	}

	// Create status
	status, err := h.statusService.CreateStatus(
		r.Context(),
		userID,
		domain.AvailabilityType(req.AvailabilityType),
		req.Note,
		domain.VisibilityTier(req.VisibilityTier),
		req.ExpiresAt,
	)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "failed to create status")
		return
	}

	SuccessResponse(w, http.StatusOK, status)
}

// GetMyStatus handles GET /v1/me/status
func (h *StatusHandler) GetMyStatus(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		ErrorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get status
	status, err := h.statusService.GetUserStatus(r.Context(), userID)
	if err != nil {
		if err == domain.ErrNoActiveStatus {
			ErrorResponse(w, http.StatusNotFound, "no active status")
		} else {
			ErrorResponse(w, http.StatusInternalServerError, "failed to get status")
		}
		return
	}

	SuccessResponse(w, http.StatusOK, status)
}

// ClearStatus handles DELETE /v1/me/status
func (h *StatusHandler) ClearStatus(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		ErrorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Clear status
	err := h.statusService.ClearUserStatus(r.Context(), userID)
	if err != nil {
		if err == domain.ErrNoActiveStatus {
			ErrorResponse(w, http.StatusNotFound, "no active status to clear")
		} else {
			ErrorResponse(w, http.StatusInternalServerError, "failed to clear status")
		}
		return
	}

	SuccessResponse(w, http.StatusOK, map[string]string{
		"message": "status cleared successfully",
	})
}

// GetStatusFeed handles GET /v1/feed/status
func (h *StatusHandler) GetStatusFeed(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		ErrorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get visible statuses
	statuses, err := h.statusService.GetVisibleStatuses(r.Context(), userID)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "failed to get status feed")
		return
	}

	SuccessResponse(w, http.StatusOK, map[string]interface{}{
		"statuses": statuses,
		"count":    len(statuses),
	})
}

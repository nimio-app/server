package handler

import (
	"net/http"

	"github.com/nimio/server/internal/domain"
	"github.com/nimio/server/internal/middleware"
	"github.com/nimio/server/internal/repository"
)

// ProfileHandler handles profile-related endpoints
type ProfileHandler struct {
	userRepo repository.UserRepository
}

// NewProfileHandler creates a new profile handler
func NewProfileHandler(userRepo repository.UserRepository) *ProfileHandler {
	return &ProfileHandler{
		userRepo: userRepo,
	}
}

// ProfileResponse represents the profile response
type ProfileResponse struct {
	User    domain.User    `json:"user"`
	Profile domain.Profile `json:"profile"`
}

// GetMyProfile handles GET /v1/me/profile
func (h *ProfileHandler) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		ErrorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get user
	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		if err == domain.ErrNotFound {
			ErrorResponse(w, http.StatusNotFound, "user not found")
		} else {
			ErrorResponse(w, http.StatusInternalServerError, "failed to get user")
		}
		return
	}

	// Get profile
	profile, err := h.userRepo.GetProfileByUserID(r.Context(), userID)
	if err != nil {
		if err == domain.ErrNotFound {
			ErrorResponse(w, http.StatusNotFound, "profile not found")
		} else {
			ErrorResponse(w, http.StatusInternalServerError, "failed to get profile")
		}
		return
	}

	SuccessResponse(w, http.StatusOK, ProfileResponse{
		User:    *user,
		Profile: *profile,
	})
}

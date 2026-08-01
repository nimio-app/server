package handler

import (
	"encoding/json"
	"fmt"
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

// UpdateProfileRequest represents the profile update request
type UpdateProfileRequest struct {
	Username    *string `json:"username,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	Bio         *string `json:"bio,omitempty"`
}

// UpdateMyProfile handles PUT /v1/me/profile
func (h *ProfileHandler) UpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		ErrorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ValidationErrorResponse(w, "invalid request body")
		return
	}

	// Get current profile
	profile, err := h.userRepo.GetProfileByUserID(r.Context(), userID)
	if err != nil {
		if err == domain.ErrNotFound {
			ErrorResponse(w, http.StatusNotFound, "profile not found")
		} else {
			ErrorResponse(w, http.StatusInternalServerError, "failed to get profile")
		}
		return
	}

	// Update fields if provided
	if req.Username != nil && *req.Username != "" {
		// Validate username format
		if len(*req.Username) < 3 || len(*req.Username) > 50 {
			ValidationErrorResponse(w, "username must be between 3 and 50 characters")
			return
		}
		profile.Username = *req.Username
	}

	if req.DisplayName != nil && *req.DisplayName != "" {
		if len(*req.DisplayName) > 100 {
			ValidationErrorResponse(w, "display_name must be 100 characters or less")
			return
		}
		profile.DisplayName = *req.DisplayName
	}

	if req.Bio != nil {
		if len(*req.Bio) > 500 {
			ValidationErrorResponse(w, "bio must be 500 characters or less")
			return
		}
		profile.Bio = req.Bio
	}

	// Update profile
	if err := h.userRepo.UpdateProfile(r.Context(), profile); err != nil {
		if err == domain.ErrUsernameTaken {
			ErrorResponse(w, http.StatusConflict, "username already taken")
		} else if err == domain.ErrNotFound {
			ErrorResponse(w, http.StatusNotFound, "profile not found")
		} else {
			ErrorResponse(w, http.StatusInternalServerError, "failed to update profile")
		}
		return
	}

	SuccessResponse(w, http.StatusOK, map[string]interface{}{
		"profile": profile,
		"message": "Profile updated successfully",
	})
}

// SearchUsers handles GET /v1/users/search
func (h *ProfileHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	_, ok := middleware.GetUserID(r.Context())
	if !ok {
		ErrorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get query parameter
	query := r.URL.Query().Get("q")
	if query == "" {
		ValidationErrorResponse(w, "query parameter 'q' is required")
		return
	}

	if len(query) < 2 {
		ValidationErrorResponse(w, "query must be at least 2 characters")
		return
	}

	// Get limit parameter (default 20, max 50)
	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if _, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil {
			limit = 20
		}
		if limit > 50 {
			limit = 50
		}
	}

	// Search users
	profiles, err := h.userRepo.SearchUsers(r.Context(), query, limit)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "failed to search users")
		return
	}

	SuccessResponse(w, http.StatusOK, map[string]interface{}{
		"users": profiles,
		"count": len(profiles),
	})
}

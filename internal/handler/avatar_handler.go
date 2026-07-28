package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/nimio/server/internal/domain"
	"github.com/nimio/server/internal/middleware"
	"github.com/nimio/server/internal/service"
)

const (
	maxAvatarSize = 5 << 20 // 5MB
	allowedTypes  = "image/jpeg,image/jpg,image/png,image/gif,image/webp"
)

// AvatarRepository interface for user profile operations
type AvatarRepository interface {
	GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*domain.Profile, error)
	UpdateProfile(ctx context.Context, profile *domain.Profile) error
}

// AvatarHandler handles avatar upload endpoints
type AvatarHandler struct {
	storageService service.StorageService
	userRepo       AvatarRepository
}

// NewAvatarHandler creates a new avatar handler
func NewAvatarHandler(storageService service.StorageService, userRepo AvatarRepository) *AvatarHandler {
	return &AvatarHandler{
		storageService: storageService,
		userRepo:       userRepo,
	}
}

// UploadAvatar handles POST /v1/me/avatar
func (h *AvatarHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userUUID, ok := middleware.GetUserID(r.Context())
	if !ok {
		ErrorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse multipart form with max size
	if err := r.ParseMultipartForm(maxAvatarSize); err != nil {
		fmt.Printf("Failed to parse multipart form: %v\n", err)
		ErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("failed to parse multipart form: %v", err))
		return
	}

	// Get the file from form
	file, header, err := r.FormFile("avatar")
	if err != nil {
		fmt.Printf("Failed to get 'avatar' file from form: %v\n", err)
		fmt.Printf("Available form fields: %v\n", r.MultipartForm.File)
		ValidationErrorResponse(w, fmt.Sprintf("avatar file is required (error: %v)", err))
		return
	}
	defer file.Close()

	// Validate file size
	if header.Size > maxAvatarSize {
		ErrorResponse(w, http.StatusBadRequest, "file too large (max 5MB)")
		return
	}

	// Validate content type
	contentType := header.Header.Get("Content-Type")
	fmt.Printf("Received file: %s, size: %d, content-type: %s\n", header.Filename, header.Size, contentType)
	if !isValidImageType(contentType) {
		ErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("invalid file type '%s' (allowed: %s)", contentType, allowedTypes))
		return
	}

	// Read file data
	fileData, err := io.ReadAll(file)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "failed to read file")
		return
	}

	// Upload to R2
	avatarURL, err := h.storageService.UploadAvatar(r.Context(), userUUID, fileData, contentType)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "failed to upload avatar")
		return
	}

	// Get current profile
	profile, err := h.userRepo.GetProfileByUserID(r.Context(), userUUID)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "failed to get profile")
		return
	}

	// Delete old avatar if exists
	if profile.AvatarURL != nil && *profile.AvatarURL != "" {
		_ = h.storageService.DeleteAvatar(r.Context(), *profile.AvatarURL)
	}

	// Update profile with new avatar URL
	profile.AvatarURL = &avatarURL
	if err := h.userRepo.UpdateProfile(r.Context(), profile); err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "failed to update profile")
		return
	}

	SuccessResponse(w, http.StatusOK, map[string]string{
		"avatar_url": avatarURL,
		"message":    "Avatar uploaded successfully",
	})
}

// DeleteAvatar handles DELETE /v1/me/avatar
func (h *AvatarHandler) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userUUID, ok := middleware.GetUserID(r.Context())
	if !ok {
		ErrorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get current profile
	profile, err := h.userRepo.GetProfileByUserID(r.Context(), userUUID)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "failed to get profile")
		return
	}

	// Delete from R2 if exists
	if profile.AvatarURL != nil && *profile.AvatarURL != "" {
		if err := h.storageService.DeleteAvatar(r.Context(), *profile.AvatarURL); err != nil {
			// Log but don't fail - continue to clear DB
			fmt.Printf("Failed to delete avatar from R2: %v\n", err)
		}
	}

	// Clear avatar URL in database
	profile.AvatarURL = nil
	if err := h.userRepo.UpdateProfile(r.Context(), profile); err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "failed to update profile")
		return
	}

	SuccessResponse(w, http.StatusOK, map[string]string{
		"message": "Avatar deleted successfully",
	})
}

// isValidImageType checks if the content type is an allowed image format
func isValidImageType(contentType string) bool {
	switch contentType {
	case "image/jpeg", "image/jpg", "image/png", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}

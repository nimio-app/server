package handler

import (
	"encoding/json"
	"net/http"

	"github.com/nimio/server/internal/service"
)

// GoogleAuthHandler handles Google OAuth endpoints
type GoogleAuthHandler struct {
	authService service.AuthService
}

// NewGoogleAuthHandler creates a new Google auth handler
func NewGoogleAuthHandler(authService service.AuthService) *GoogleAuthHandler {
	return &GoogleAuthHandler{
		authService: authService,
	}
}

// GoogleSignInRequest represents the Google sign-in request
type GoogleSignInRequest struct {
	IDToken string `json:"id_token"`
}

// GoogleSignIn handles POST /v1/auth/google
func (h *GoogleAuthHandler) GoogleSignIn(w http.ResponseWriter, r *http.Request) {
	var req GoogleSignInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ValidationErrorResponse(w, "invalid request body")
		return
	}

	if req.IDToken == "" {
		ValidationErrorResponse(w, "id_token is required")
		return
	}

	user, profile, token, err := h.authService.GoogleSignIn(r.Context(), req.IDToken)
	if err != nil {
		ErrorResponse(w, http.StatusUnauthorized, err.Error())
		return
	}

	SuccessResponse(w, http.StatusOK, map[string]interface{}{
		"user":    user,
		"profile": profile,
		"token":   token,
	})
}

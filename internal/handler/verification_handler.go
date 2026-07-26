package handler

import (
	"encoding/json"
	"net/http"

	"github.com/nimio/server/internal/service"
)

// VerificationHandler handles email verification endpoints
type VerificationHandler struct {
	authService service.AuthService
}

// NewVerificationHandler creates a new verification handler
func NewVerificationHandler(authService service.AuthService) *VerificationHandler {
	return &VerificationHandler{
		authService: authService,
	}
}

// VerifyEmailRequest represents the email verification request body
type VerifyEmailRequest struct {
	Token string `json:"token"`
}

// ResendVerificationRequest represents the resend verification email request
type ResendVerificationRequest struct {
	Email string `json:"email"`
}

// VerifyEmail handles GET /v1/auth/verify-email
func (h *VerificationHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req VerifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ValidationErrorResponse(w, "invalid request body")
		return
	}

	if req.Token == "" {
		ValidationErrorResponse(w, "token is required")
		return
	}

	err := h.authService.VerifyEmail(r.Context(), req.Token)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	SuccessResponse(w, http.StatusOK, map[string]string{
		"message": "Email verified successfully! You can now use all features.",
	})
}

// ResendVerification handles POST /v1/auth/resend-verification
func (h *VerificationHandler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	var req ResendVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ValidationErrorResponse(w, "invalid request body")
		return
	}

	if req.Email == "" {
		ValidationErrorResponse(w, "email is required")
		return
	}

	err := h.authService.ResendVerificationEmail(r.Context(), req.Email)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	SuccessResponse(w, http.StatusOK, map[string]string{
		"message": "Verification email sent! Please check your inbox.",
	})
}

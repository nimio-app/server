package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/nimio/server/internal/domain"
	"github.com/nimio/server/internal/service"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	authService service.AuthService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// RegisterRequest represents the registration request body
type RegisterRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse represents the authentication response
type AuthResponse struct {
	User    domain.User    `json:"user"`
	Profile domain.Profile `json:"profile"`
	Token   string         `json:"token"`
}

// Register handles POST /v1/auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ValidationErrorResponse(w, "invalid request body")
		return
	}

	// Validate input
	if err := validateRegisterRequest(&req); err != nil {
		ValidationErrorResponse(w, err.Error())
		return
	}

	// Register user
	user, profile, token, err := h.authService.Register(r.Context(), req.Email, req.Password, req.Username, req.DisplayName)
	if err != nil {
		switch err {
		case domain.ErrEmailTaken:
			ErrorResponse(w, http.StatusConflict, "email already taken")
		case domain.ErrUsernameTaken:
			ErrorResponse(w, http.StatusConflict, "username already taken")
		default:
			ErrorResponse(w, http.StatusInternalServerError, "failed to register user")
		}
		return
	}

	SuccessResponse(w, http.StatusCreated, AuthResponse{
		User:    *user,
		Profile: *profile,
		Token:   token,
	})
}

// Login handles POST /v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ValidationErrorResponse(w, "invalid request body")
		return
	}

	// Validate input
	if err := validateLoginRequest(&req); err != nil {
		ValidationErrorResponse(w, err.Error())
		return
	}

	// Login user
	user, profile, token, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		switch err {
		case domain.ErrInvalidCredentials:
			ErrorResponse(w, http.StatusUnauthorized, "invalid email or password")
		default:
			ErrorResponse(w, http.StatusInternalServerError, "failed to login")
		}
		return
	}

	SuccessResponse(w, http.StatusOK, AuthResponse{
		User:    *user,
		Profile: *profile,
		Token:   token,
	})
}

// validateRegisterRequest validates the registration request
func validateRegisterRequest(req *RegisterRequest) error {
	if req.Email == "" {
		return domain.ErrInvalidInput
	}
	if req.Password == "" || len(req.Password) < 8 {
		return domain.ErrInvalidInput
	}
	if req.Username == "" || len(req.Username) < 3 || len(req.Username) > 50 {
		return domain.ErrInvalidInput
	}
	if req.DisplayName == "" || len(req.DisplayName) > 100 {
		return domain.ErrInvalidInput
	}

	// Validate email format
	if !strings.Contains(req.Email, "@") {
		return domain.ErrInvalidInput
	}

	// Validate username format (alphanumeric and underscore only)
	for _, char := range req.Username {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '_') {
			return domain.ErrInvalidInput
		}
	}

	return nil
}

// validateLoginRequest validates the login request
func validateLoginRequest(req *LoginRequest) error {
	if req.Email == "" {
		return domain.ErrInvalidInput
	}
	if req.Password == "" {
		return domain.ErrInvalidInput
	}
	return nil
}

package service

import (
	"context"
	"fmt"

	"google.golang.org/api/idtoken"
)

// GoogleAuthService handles Google OAuth token verification
type GoogleAuthService interface {
	VerifyIDToken(ctx context.Context, idToken string) (*GoogleClaims, error)
}

type googleAuthService struct {
	webClientID string
}

// GoogleClaims represents the claims from a Google ID token
type GoogleClaims struct {
	Sub           string `json:"sub"`            // Google User ID
	Email         string `json:"email"`          // User email
	EmailVerified bool   `json:"email_verified"` // Email verification status
	Name          string `json:"name"`           // Full name
	Picture       string `json:"picture"`        // Profile picture URL
	GivenName     string `json:"given_name"`     // First name
	FamilyName    string `json:"family_name"`    // Last name
}

// NewGoogleAuthService creates a new Google auth service
func NewGoogleAuthService(webClientID string) GoogleAuthService {
	return &googleAuthService{
		webClientID: webClientID,
	}
}

// VerifyIDToken validates a Google ID token and extracts claims
func (s *googleAuthService) VerifyIDToken(ctx context.Context, idToken string) (*GoogleClaims, error) {
	// Validate the ID token with Google's public keys
	payload, err := idtoken.Validate(ctx, idToken, s.webClientID)
	if err != nil {
		return nil, fmt.Errorf("invalid ID token: %w", err)
	}

	// Extract claims
	claims := &GoogleClaims{
		Sub:           payload.Subject,
		Email:         payload.Claims["email"].(string),
		EmailVerified: payload.Claims["email_verified"].(bool),
	}

	// Optional fields
	if name, ok := payload.Claims["name"].(string); ok {
		claims.Name = name
	}
	if picture, ok := payload.Claims["picture"].(string); ok {
		claims.Picture = picture
	}
	if givenName, ok := payload.Claims["given_name"].(string); ok {
		claims.GivenName = givenName
	}
	if familyName, ok := payload.Claims["family_name"].(string); ok {
		claims.FamilyName = familyName
	}

	return claims, nil
}

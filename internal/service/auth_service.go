package service

import (
	"context"
	"crypto/sha256"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"

	"github.com/nimio/server/internal/config"
	"github.com/nimio/server/internal/domain"
	"github.com/nimio/server/internal/repository"
)

// AuthService handles authentication logic
type AuthService interface {
	Register(ctx context.Context, email, password, username, displayName string) (*domain.User, *domain.Profile, string, string, error)
	Login(ctx context.Context, email, password string) (*domain.User, *domain.Profile, string, string, error)
	GoogleSignIn(ctx context.Context, idToken string) (*domain.User, *domain.Profile, string, string, error)
	RefreshAccessToken(ctx context.Context, refreshToken string) (string, string, error)
	ValidateToken(tokenString string) (uuid.UUID, error)
	VerifyEmail(ctx context.Context, token string) error
	ResendVerificationEmail(ctx context.Context, email string) error
}

type authService struct {
	userRepo      repository.UserRepository
	emailService  EmailService
	googleAuth    GoogleAuthService
	jwtSecret     string
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

// NewAuthService creates a new auth service
func NewAuthService(userRepo repository.UserRepository, emailService EmailService, googleAuth GoogleAuthService, cfg *config.Config) AuthService {
	return &authService{
		userRepo:      userRepo,
		emailService:  emailService,
		googleAuth:    googleAuth,
		jwtSecret:     cfg.JWT.Secret,
		accessExpiry:  cfg.JWT.AccessExpiry,
		refreshExpiry: cfg.JWT.RefreshExpiry,
	}
}

// Register creates a new user account
func (s *authService) Register(ctx context.Context, email, password, username, displayName string) (*domain.User, *domain.Profile, string, string, error) {
	// Check if email exists
	existingUser, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil && existingUser != nil {
		return nil, nil, "", "", domain.ErrEmailTaken
	}
	if err != nil && err != domain.ErrNotFound {
		return nil, nil, "", "", fmt.Errorf("check email: %w", err)
	}

	// Check if username exists
	existingProfile, err := s.userRepo.GetProfileByUsername(ctx, username)
	if err == nil && existingProfile != nil {
		return nil, nil, "", "", domain.ErrUsernameTaken
	}
	if err != nil && err != domain.ErrNotFound {
		return nil, nil, "", "", fmt.Errorf("check username: %w", err)
	}

	// Hash password
	passwordHash, err := hashPassword(password)
	if err != nil {
		return nil, nil, "", "", fmt.Errorf("hash password: %w", err)
	}

	// Generate verification token
	verificationToken, err := generateVerificationToken()
	if err != nil {
		return nil, nil, "", "", fmt.Errorf("generate verification token: %w", err)
	}

	expiresAt := time.Now().Add(24 * time.Hour) // 24 hour expiry

	// Create user and profile
	user := &domain.User{
		ID:                         uuid.New(),
		Email:                      email,
		PasswordHash:               passwordHash,
		AuthProvider:               "email",
		EmailVerified:              false,
		VerificationToken:          &verificationToken,
		VerificationTokenExpiresAt: &expiresAt,
		CreatedAt:                  time.Now(),
		UpdatedAt:                  time.Now(),
	}

	profile := &domain.Profile{
		UserID:      user.ID,
		Username:    username,
		DisplayName: displayName,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.userRepo.Create(ctx, user, profile); err != nil {
		return nil, nil, "", "", fmt.Errorf("create user: %w", err)
	}

	// Send verification email (don't fail registration if email fails)
	if err := s.emailService.SendVerificationEmail(email, username, verificationToken); err != nil {
		// Log error but don't fail registration
		fmt.Printf("Failed to send verification email: %v\n", err)
	}

	// Generate JWT token
	token, refreshToken, err := s.generateSessionTokens(ctx, user.ID)
	if err != nil {
		return nil, nil, "", "", fmt.Errorf("generate session tokens: %w", err)
	}

	return user, profile, token, refreshToken, nil
}

// Login authenticates a user
func (s *authService) Login(ctx context.Context, email, password string) (*domain.User, *domain.Profile, string, string, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, nil, "", "", domain.ErrInvalidCredentials
		}
		return nil, nil, "", "", fmt.Errorf("get user: %w", err)
	}

	// Verify password
	if !verifyPassword(password, user.PasswordHash) {
		return nil, nil, "", "", domain.ErrInvalidCredentials
	}

	// Get profile
	profile, err := s.userRepo.GetProfileByUserID(ctx, user.ID)
	if err != nil {
		return nil, nil, "", "", fmt.Errorf("get profile: %w", err)
	}

	// Generate session tokens
	token, refreshToken, err := s.generateSessionTokens(ctx, user.ID)
	if err != nil {
		return nil, nil, "", "", fmt.Errorf("generate session tokens: %w", err)
	}

	return user, profile, token, refreshToken, nil
}

// ValidateToken validates a JWT token and returns the user ID
func (s *authService) ValidateToken(tokenString string) (uuid.UUID, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return uuid.Nil, domain.ErrInvalidToken
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userIDStr, ok := claims["sub"].(string)
		if !ok {
			return uuid.Nil, domain.ErrInvalidToken
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return uuid.Nil, domain.ErrInvalidToken
		}

		// Check expiration
		exp, ok := claims["exp"].(float64)
		if !ok {
			return uuid.Nil, domain.ErrInvalidToken
		}
		if time.Now().Unix() > int64(exp) {
			return uuid.Nil, domain.ErrTokenExpired
		}

		return userID, nil
	}

	return uuid.Nil, domain.ErrInvalidToken
}

// generateToken generates a JWT token for a user
func (s *authService) generateToken(userID uuid.UUID) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(s.accessExpiry).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// generateSessionTokens creates and persists a refresh token plus short-lived access token.
func (s *authService) generateSessionTokens(ctx context.Context, userID uuid.UUID) (string, string, error) {
	accessToken, err := s.generateToken(userID)
	if err != nil {
		return "", "", err
	}

	rawRefreshToken, err := generateRefreshToken()
	if err != nil {
		return "", "", err
	}

	hashed := hashToken(rawRefreshToken)
	refresh := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: hashed,
		ExpiresAt: time.Now().Add(s.refreshExpiry),
		CreatedAt: time.Now(),
	}

	if err := s.userRepo.CreateRefreshToken(ctx, refresh); err != nil {
		return "", "", err
	}

	return accessToken, rawRefreshToken, nil
}

// RefreshAccessToken rotates a valid refresh token and returns a fresh access+refresh pair.
func (s *authService) RefreshAccessToken(ctx context.Context, refreshToken string) (string, string, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return "", "", domain.ErrInvalidToken
	}

	hashed := hashToken(refreshToken)
	stored, err := s.userRepo.GetValidRefreshTokenByHash(ctx, hashed)
	if err != nil {
		if err == domain.ErrNotFound {
			return "", "", domain.ErrInvalidToken
		}
		return "", "", fmt.Errorf("find refresh token: %w", err)
	}

	if err := s.userRepo.RevokeRefreshToken(ctx, stored.ID); err != nil {
		if err == domain.ErrNotFound {
			return "", "", domain.ErrInvalidToken
		}
		return "", "", fmt.Errorf("revoke refresh token: %w", err)
	}

	accessToken, newRefreshToken, err := s.generateSessionTokens(ctx, stored.UserID)
	if err != nil {
		return "", "", fmt.Errorf("generate refreshed session tokens: %w", err)
	}

	return accessToken, newRefreshToken, nil
}

// hashPassword hashes a password using Argon2id
func hashPassword(password string) (string, error) {
	// Generate a random salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// Hash the password
	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	// Encode salt and hash to base64
	saltEncoded := base64.RawStdEncoding.EncodeToString(salt)
	hashEncoded := base64.RawStdEncoding.EncodeToString(hash)

	// Return formatted string: $argon2id$salt$hash
	return fmt.Sprintf("$argon2id$%s$%s", saltEncoded, hashEncoded), nil
}

// verifyPassword verifies a password against a hash
func verifyPassword(password, encodedHash string) bool {
	// Parse the encoded hash format: $argon2id$salt$hash
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 4 || parts[0] != "" || parts[1] != "argon2id" {
		return false
	}

	saltEncoded := parts[2]
	hashEncoded := parts[3]

	// Decode salt and hash
	salt, err := base64.RawStdEncoding.DecodeString(saltEncoded)
	if err != nil {
		return false
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(hashEncoded)
	if err != nil {
		return false
	}

	// Hash the input password with the same salt
	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	// Compare hashes
	if len(hash) != len(expectedHash) {
		return false
	}

	for i := range hash {
		if hash[i] != expectedHash[i] {
			return false
		}
	}

	return true
}

// VerifyEmail verifies a user's email with the provided token
func (s *authService) VerifyEmail(ctx context.Context, token string) error {
	// Get user by verification token
	user, err := s.userRepo.GetByVerificationToken(ctx, token)
	if err != nil {
		if err == domain.ErrNotFound {
			return fmt.Errorf("invalid or expired verification token")
		}
		return fmt.Errorf("get user by token: %w", err)
	}

	// Check if already verified
	if user.EmailVerified {
		return fmt.Errorf("email already verified")
	}

	// Check if token is expired
	if user.VerificationTokenExpiresAt != nil && time.Now().After(*user.VerificationTokenExpiresAt) {
		return fmt.Errorf("verification token has expired")
	}

	// Mark email as verified
	if err := s.userRepo.MarkEmailAsVerified(ctx, user.ID); err != nil {
		return fmt.Errorf("mark email as verified: %w", err)
	}

	return nil
}

// ResendVerificationEmail resends the verification email
func (s *authService) ResendVerificationEmail(ctx context.Context, email string) error {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if err == domain.ErrNotFound {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("get user: %w", err)
	}

	// Check if already verified
	if user.EmailVerified {
		return fmt.Errorf("email already verified")
	}

	// Generate new verification token
	verificationToken, err := generateVerificationToken()
	if err != nil {
		return fmt.Errorf("generate verification token: %w", err)
	}

	expiresAt := time.Now().Add(24 * time.Hour)

	// Update verification token
	if err := s.userRepo.UpdateVerificationToken(ctx, user.ID, &verificationToken, &expiresAt); err != nil {
		return fmt.Errorf("update verification token: %w", err)
	}

	// Get profile for username
	profile, err := s.userRepo.GetProfileByUserID(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("get profile: %w", err)
	}

	// Send verification email
	if err := s.emailService.SendVerificationEmail(email, profile.Username, verificationToken); err != nil {
		return fmt.Errorf("send verification email: %w", err)
	}

	return nil
}

// generateVerificationToken generates a secure random token for email verification
func generateVerificationToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GoogleSignIn handles Google OAuth sign-in
func (s *authService) GoogleSignIn(ctx context.Context, idToken string) (*domain.User, *domain.Profile, string, string, error) {
	// Verify the Google ID token
	claims, err := s.googleAuth.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, nil, "", "", fmt.Errorf("invalid Google ID token: %w", err)
	}

	// Check if user exists by Google ID
	user, err := s.userRepo.GetByGoogleID(ctx, claims.Sub)
	if err != nil && err != domain.ErrNotFound {
		return nil, nil, "", "", fmt.Errorf("check google user: %w", err)
	}

	// If user doesn't exist by Google ID, check by email
	if user == nil {
		user, err = s.userRepo.GetByEmail(ctx, claims.Email)
		if err != nil && err != domain.ErrNotFound {
			return nil, nil, "", "", fmt.Errorf("check email: %w", err)
		}
	}

	// If user exists, generate token and return
	if user != nil {
		// Get profile
		profile, err := s.userRepo.GetProfileByUserID(ctx, user.ID)
		if err != nil {
			return nil, nil, "", "", fmt.Errorf("get profile: %w", err)
		}

		// Generate session tokens
		token, refreshToken, err := s.generateSessionTokens(ctx, user.ID)
		if err != nil {
			return nil, nil, "", "", fmt.Errorf("generate session tokens: %w", err)
		}

		return user, profile, token, refreshToken, nil
	}

	// User doesn't exist, create new account
	// Generate username from email or name
	username := generateUsernameFromEmail(claims.Email)
	displayName := claims.Name
	if displayName == "" {
		displayName = claims.GivenName + " " + claims.FamilyName
	}
	if displayName == "" {
		displayName = username
	}

	// Create new user
	now := time.Now()
	newUser := &domain.User{
		ID:            uuid.New(),
		Email:         claims.Email,
		PasswordHash:  "", // No password for OAuth users
		GoogleID:      &claims.Sub,
		AuthProvider:  "google",
		EmailVerified: true, // Google emails are pre-verified
		VerifiedAt:    &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Create profile with avatar from Google
	var avatarURL *string
	if claims.Picture != "" {
		avatarURL = &claims.Picture
	}

	profile := &domain.Profile{
		UserID:      newUser.ID,
		Username:    username,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Save to database
	if err := s.userRepo.Create(ctx, newUser, profile); err != nil {
		return nil, nil, "", "", fmt.Errorf("create user: %w", err)
	}

	// Generate session tokens
	token, refreshToken, err := s.generateSessionTokens(ctx, newUser.ID)
	if err != nil {
		return nil, nil, "", "", fmt.Errorf("generate session tokens: %w", err)
	}

	return newUser, profile, token, refreshToken, nil
}

func generateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return base64.RawStdEncoding.EncodeToString(sum[:])
}

// generateUsernameFromEmail creates a username from email
func generateUsernameFromEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) == 0 {
		return "user" + fmt.Sprintf("%d", time.Now().Unix())
	}
	username := strings.ToLower(parts[0])
	// Remove non-alphanumeric characters except underscore
	username = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return -1
	}, username)
	
	// Add random suffix to avoid conflicts
	username = username + "_" + fmt.Sprintf("%d", time.Now().Unix()%10000)
	
	return username
}


package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
	"crypto/rand"
	"encoding/base64"
	
	"github.com/nimio/server/internal/config"
	"github.com/nimio/server/internal/domain"
	"github.com/nimio/server/internal/repository"
)

// AuthService handles authentication logic
type AuthService interface {
	Register(ctx context.Context, email, password, username, displayName string) (*domain.User, *domain.Profile, string, error)
	Login(ctx context.Context, email, password string) (*domain.User, *domain.Profile, string, error)
	ValidateToken(tokenString string) (uuid.UUID, error)
}

type authService struct {
	userRepo repository.UserRepository
	cfg      *config.Config
}

// NewAuthService creates a new auth service
func NewAuthService(userRepo repository.UserRepository, cfg *config.Config) AuthService {
	return &authService{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

// Register creates a new user account
func (s *authService) Register(ctx context.Context, email, password, username, displayName string) (*domain.User, *domain.Profile, string, error) {
	// Check if email exists
	existingUser, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil && existingUser != nil {
		return nil, nil, "", domain.ErrEmailTaken
	}
	if err != nil && err != domain.ErrNotFound {
		return nil, nil, "", fmt.Errorf("check email: %w", err)
	}

	// Check if username exists
	existingProfile, err := s.userRepo.GetProfileByUsername(ctx, username)
	if err == nil && existingProfile != nil {
		return nil, nil, "", domain.ErrUsernameTaken
	}
	if err != nil && err != domain.ErrNotFound {
		return nil, nil, "", fmt.Errorf("check username: %w", err)
	}

	// Hash password
	passwordHash, err := hashPassword(password)
	if err != nil {
		return nil, nil, "", fmt.Errorf("hash password: %w", err)
	}

	// Create user and profile
	user := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	profile := &domain.Profile{
		UserID:      user.ID,
		Username:    username,
		DisplayName: displayName,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.userRepo.Create(ctx, user, profile); err != nil {
		return nil, nil, "", fmt.Errorf("create user: %w", err)
	}

	// Generate JWT token
	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil, nil, "", fmt.Errorf("generate token: %w", err)
	}

	return user, profile, token, nil
}

// Login authenticates a user
func (s *authService) Login(ctx context.Context, email, password string) (*domain.User, *domain.Profile, string, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, nil, "", domain.ErrInvalidCredentials
		}
		return nil, nil, "", fmt.Errorf("get user: %w", err)
	}

	// Verify password
	if !verifyPassword(password, user.PasswordHash) {
		return nil, nil, "", domain.ErrInvalidCredentials
	}

	// Get profile
	profile, err := s.userRepo.GetProfileByUserID(ctx, user.ID)
	if err != nil {
		return nil, nil, "", fmt.Errorf("get profile: %w", err)
	}

	// Generate JWT token
	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil, nil, "", fmt.Errorf("generate token: %w", err)
	}

	return user, profile, token, nil
}

// ValidateToken validates a JWT token and returns the user ID
func (s *authService) ValidateToken(tokenString string) (uuid.UUID, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.JWT.Secret), nil
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
		"exp": time.Now().Add(s.cfg.JWT.AccessExpiry).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.cfg.JWT.Secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
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
	// Parse the encoded hash
	var saltEncoded, hashEncoded string
	_, err := fmt.Sscanf(encodedHash, "$argon2id$%s$%s", &saltEncoded, &hashEncoded)
	if err != nil {
		return false
	}

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

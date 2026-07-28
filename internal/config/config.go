package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	CORS     CORSConfig
	Email    EmailConfig
	R2       R2Config
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Port string
	Env  string
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

// JWTConfig holds JWT token configuration
type JWTConfig struct {
	Secret         string
	AccessExpiry   time.Duration
	RefreshExpiry  time.Duration
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins []string
}

// EmailConfig holds email/Resend configuration
type EmailConfig struct {
	ResendAPIKey string
	FromEmail    string
	FromName     string
	AppURL       string
}

// R2Config holds Cloudflare R2 storage configuration
type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	PublicURL       string // CDN URL for serving images
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists (for local development)
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
			Env:  getEnv("ENV", "development"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "nimio"),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "nimio_db"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			Secret:        getEnv("JWT_SECRET", ""),
			AccessExpiry:  parseDuration(getEnv("JWT_ACCESS_EXPIRY", "15m"), 15*time.Minute),
			RefreshExpiry: parseDuration(getEnv("JWT_REFRESH_EXPIRY", "168h"), 168*time.Hour),
		},
		CORS: CORSConfig{
			AllowedOrigins: parseSlice(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")),
		},
		Email: EmailConfig{
			ResendAPIKey: getEnv("RESEND_API_KEY", ""),
			FromEmail:    getEnv("FROM_EMAIL", "noreply@nimio.org"),
			FromName:     getEnv("FROM_NAME", "Nimio"),
			AppURL:       getEnv("APP_URL", "http://localhost:3000"),
		},
		R2: R2Config{
			AccountID:       os.Getenv("R2_ACCOUNT_ID"),
			AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
			BucketName:      getEnv("R2_BUCKET_NAME", "nimio"),
			PublicURL:       getEnv("R2_PUBLIC_URL", ""),
		},
	}

	// Validate required fields
	if cfg.Database.Password == "" {
		return nil, fmt.Errorf("DB_PASSWORD is required")
	}
	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if cfg.Email.ResendAPIKey == "" {
		return nil, fmt.Errorf("RESEND_API_KEY is required")
	}
	if cfg.R2.AccountID == "" || cfg.R2.AccessKeyID == "" || cfg.R2.SecretAccessKey == "" {
		return nil, fmt.Errorf("R2_ACCOUNT_ID, R2_ACCESS_KEY_ID, and R2_SECRET_ACCESS_KEY are required")
	}

	return cfg, nil
}

// DSN returns the PostgreSQL connection string
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// parseDuration parses a duration string or returns a default value
func parseDuration(value string, defaultValue time.Duration) time.Duration {
	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return duration
}

// parseSlice parses a comma-separated string into a slice
func parseSlice(value string) []string {
	if value == "" {
		return []string{}
	}
	result := []string{}
	for i := 0; i < len(value); {
		end := i
		for end < len(value) && value[end] != ',' {
			end++
		}
		item := value[i:end]
		if len(item) > 0 {
			result = append(result, item)
		}
		i = end + 1
	}
	return result
}

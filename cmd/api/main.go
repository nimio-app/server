package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/nimio/server/internal/config"
	"github.com/nimio/server/internal/handler"
	"github.com/nimio/server/internal/middleware"
	"github.com/nimio/server/internal/repository"
	"github.com/nimio/server/internal/service"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database connection pool
	dbPool, err := initDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	log.Println("✓ Database connection established")

	// Run database migrations
	if err := runMigrations(cfg); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("✓ Database migrations completed")

	// Initialize repositories
	userRepo := repository.NewUserRepository(dbPool)
	statusRepo := repository.NewStatusRepository(dbPool)
	connRepo := repository.NewConnectionRepository(dbPool)

	// Initialize email service
	emailService := service.NewEmailService(
		cfg.Email.ResendAPIKey,
		cfg.Email.FromEmail,
		cfg.Email.FromName,
		cfg.Email.AppURL,
	)

	// Initialize Google auth service
	googleAuthService := service.NewGoogleAuthService(cfg.Google.WebClientID)

	// Initialize storage service
	storageService, err := service.NewStorageService(
		cfg.R2.AccountID,
		cfg.R2.AccessKeyID,
		cfg.R2.SecretAccessKey,
		cfg.R2.BucketName,
		cfg.R2.PublicURL,
	)
	if err != nil {
		log.Fatalf("Failed to initialize storage service: %v", err)
	}

	// Initialize services
	authService := service.NewAuthService(userRepo, emailService, googleAuthService, cfg)
	statusService := service.NewStatusService(statusRepo)
	connectionService := service.NewConnectionService(connRepo, userRepo)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	googleAuthHandler := handler.NewGoogleAuthHandler(authService)
	verificationHandler := handler.NewVerificationHandler(authService)
	profileHandler := handler.NewProfileHandler(userRepo)
	statusHandler := handler.NewStatusHandler(statusService)
	avatarHandler := handler.NewAvatarHandler(storageService, userRepo)
	connectionHandler := handler.NewConnectionHandler(connectionService)

	// Setup router
	r := setupRouter(cfg, authService, authHandler, googleAuthHandler, verificationHandler, profileHandler, statusHandler, avatarHandler, connectionHandler)

	// Create server
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("🚀 Nimio API server starting on port %s (env: %s)", cfg.Server.Port, cfg.Server.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("✓ Server stopped gracefully")
}

// initDB initializes the database connection pool
func initDB(cfg *config.Config) (*pgxpool.Pool, error) {
	ctx := context.Background()

	poolConfig, err := pgxpool.ParseConfig(cfg.Database.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	// Connection pool settings
	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}

// runMigrations runs all pending database migrations
func runMigrations(cfg *config.Config) error {
	// Build migration source path
	migrationsPath := "file://migrations"
	
	// Build postgres:// URL for migrate library
	// migrate library requires postgres:// scheme, not key=value format
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)
	
	// Create migrator instance
	m, err := migrate.New(migrationsPath, dbURL)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}

	// Get current version
	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("get migration version: %w", err)
	}

	if dirty {
		log.Printf("⚠️  Migration version %d is dirty - manual intervention may be required", version)
	} else if err == migrate.ErrNilVersion {
		log.Println("ℹ️  No migrations have been run yet")
	} else {
		log.Printf("ℹ️  Current migration version: %d", version)
	}

	return nil
}

// setupRouter configures the HTTP router with all routes and middleware
func setupRouter(
	cfg *config.Config,
	authService service.AuthService,
	authHandler *handler.AuthHandler,
	googleAuthHandler *handler.GoogleAuthHandler,
	verificationHandler *handler.VerificationHandler,
	profileHandler *handler.ProfileHandler,
	statusHandler *handler.StatusHandler,
	avatarHandler *handler.AvatarHandler,
	connectionHandler *handler.ConnectionHandler,
) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(60 * time.Second))

	// CORS middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		handler.SuccessResponse(w, http.StatusOK, map[string]string{
			"status": "ok",
			"app":    "nimio-api",
		})
	})

	// API v1 routes
	r.Route("/v1", func(r chi.Router) {
		// Public auth routes
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/verify-email", verificationHandler.VerifyEmail)
			r.Post("/resend-verification", verificationHandler.ResendVerification)
			r.Post("/refresh", authHandler.RefreshToken)
			r.Post("/google", googleAuthHandler.GoogleSignIn)
			r.Post("/login", authHandler.Login)
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(authService))

			// Profile routes
			r.Route("/me", func(r chi.Router) {
				r.Get("/profile", profileHandler.GetMyProfile)
				r.Put("/profile", profileHandler.UpdateMyProfile)
				
				// Avatar routes
				r.Post("/avatar", avatarHandler.UploadAvatar)
				r.Delete("/avatar", avatarHandler.DeleteAvatar)
				
				// Status routes
				r.Route("/status", func(r chi.Router) {
					r.Get("/", statusHandler.GetMyStatus)
					r.Put("/", statusHandler.CreateStatus)
					r.Delete("/", statusHandler.ClearStatus)
				})
			})

			// Connection routes
			r.Route("/connections", func(r chi.Router) {
				r.Post("/request", connectionHandler.SendFriendRequest)
				r.Post("/accept", connectionHandler.AcceptFriendRequest)
				r.Post("/reject", connectionHandler.RejectFriendRequest)
				r.Post("/block", connectionHandler.BlockUser)
				r.Put("/{connectionId}/tier", connectionHandler.UpdateRelationshipTier)
				r.Get("/", connectionHandler.GetMyConnections)
				r.Get("/status/{userId}", connectionHandler.GetConnectionStatus)
				r.Delete("/{friendId}", connectionHandler.RemoveConnection)
			})

			// User search routes
			r.Route("/users", func(r chi.Router) {
				r.Get("/search", profileHandler.SearchUsers)
			})

			// Feed routes
			r.Route("/feed", func(r chi.Router) {
				r.Get("/status", statusHandler.GetStatusFeed)
			})
		})
	})

	return r
}

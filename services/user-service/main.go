package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/clementhaon/sandbox-api-go/pkg/auth"
	"github.com/clementhaon/sandbox-api-go/pkg/config"
	"github.com/clementhaon/sandbox-api-go/pkg/database"
	"github.com/clementhaon/sandbox-api-go/pkg/errors"
	"github.com/clementhaon/sandbox-api-go/pkg/logger"
	"github.com/clementhaon/sandbox-api-go/pkg/metrics"
	"github.com/clementhaon/sandbox-api-go/pkg/middleware"
	"github.com/clementhaon/sandbox-api-go/services/user-service/handlers"
	"github.com/clementhaon/sandbox-api-go/services/user-service/services"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Initialize logger
	logger.Initialize()
	logger.Info("Starting user-service")

	// Initialize metrics
	metrics.InitAppInfo("1.0.0", "dev", time.Now().Format("2006-01-02"), runtime.Version())

	// Initialize the database
	db, err := database.InitDB()
	if err != nil {
		logger.Fatal("Failed to initialize database", err)
	}
	defer db.Close()

	// Run migrations
	dbURL := config.GetEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/userservice?sslmode=disable")
	m, err := migrate.New("file://database/migrations", dbURL)
	if err != nil {
		logger.Fatal("Failed to create migrate instance", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		logger.Fatal("Failed to run migrations", err)
	}
	logger.Info("Database migrations completed")

	// Initialize JWT manager
	jwtSecret := config.GetEnv("JWT_SECRET", "default-secret-change-me-in-prod")
	jwtManager, err := auth.NewJWTManager(jwtSecret)
	if err != nil {
		logger.Fatal("Failed to initialize JWT manager", fmt.Errorf("%s", err.Error()))
	}

	// Auth middleware with injected JWT manager
	authMW := middleware.NewAuthMiddleware(jwtManager)

	// Initialize services
	authSvc := services.NewAuthService(db, jwtManager)
	userSvc := services.NewUserService(db)
	profileSvc := services.NewProfileService(db)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authSvc)
	userHandler := handlers.NewUserHandler(userSvc)
	profileHandler := handlers.NewProfileHandler(profileSvc)
	internalHandler := handlers.NewInternalHandler(db)

	// Create the HTTP server with routes
	mux := createMux(authMW, authHandler, userHandler, profileHandler, internalHandler)
	server := &http.Server{
		Addr:    ":8081",
		Handler: middleware.PanicRecoveryMiddleware(middleware.RequestLoggingMiddleware(mux)),
	}

	// Start the server in a goroutine
	go func() {
		logger.Info("user-service listening on :8081")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", err)
		}
	}()

	// Wait for interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutdown signal received")

	// Gracefully shutdown the server
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Failed to gracefully shutdown server", err)
	}

	logger.Info("user-service shutdown completed")
}

func createMux(
	authMW func(middleware.ErrorHandler) http.HandlerFunc,
	authHandler *handlers.AuthHandler,
	userHandler *handlers.UserHandler,
	profileHandler *handlers.ProfileHandler,
	internalHandler *handlers.InternalHandler,
) http.Handler {
	mux := http.NewServeMux()

	// Public routes (no authentication required)
	mux.HandleFunc("/", middleware.ErrorMiddleware(handleHome))
	mux.HandleFunc("POST /auth/register", middleware.ErrorMiddleware(authHandler.HandleRegister))
	mux.HandleFunc("POST /auth/login", middleware.ErrorMiddleware(authHandler.HandleLogin))
	mux.HandleFunc("POST /auth/logout", middleware.ErrorMiddleware(authHandler.HandleLogout))

	// Prometheus metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// Auth protected routes
	mux.HandleFunc("GET /auth/user", authMW(authHandler.HandleGetUser))

	// Users Management Routes
	mux.HandleFunc("GET /users", authMW(userHandler.ListUsers))
	mux.HandleFunc("GET /users/{id}", authMW(userHandler.GetUser))
	mux.HandleFunc("POST /users", authMW(userHandler.CreateUser))
	mux.HandleFunc("PUT /users/{id}", authMW(userHandler.UpdateUser))
	mux.HandleFunc("PATCH /users/{id}/status", authMW(userHandler.UpdateUserStatus))
	mux.HandleFunc("DELETE /users/{id}", authMW(userHandler.DeleteUser))

	// Profile Routes
	mux.HandleFunc("GET /profile", authMW(profileHandler.HandleGetProfile))
	mux.HandleFunc("PUT /profile", authMW(profileHandler.HandleUpdateProfile))

	// Internal API (no auth - internal network only)
	mux.HandleFunc("GET /internal/users/{id}/brief", middleware.ErrorMiddleware(internalHandler.GetUserBrief))

	return mux
}

func handleHome(w http.ResponseWriter, r *http.Request) error {
	if r.URL.Path != "/" {
		return errors.NewNotFoundError("Page")
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"service": "user-service",
		"version": "1.0.0",
		"message": "User Service API",
	}

	logger.DebugContext(r.Context(), "Home endpoint accessed")
	json.NewEncoder(w).Encode(response)
	return nil
}

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

	"github.com/clementhaon/sandbox-api-go/auth"
	"github.com/clementhaon/sandbox-api-go/config"
	"github.com/clementhaon/sandbox-api-go/database"
	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/handlers"
	"github.com/clementhaon/sandbox-api-go/logger"
	"github.com/clementhaon/sandbox-api-go/metrics"
	"github.com/clementhaon/sandbox-api-go/middleware"
	"github.com/clementhaon/sandbox-api-go/services"
	"github.com/clementhaon/sandbox-api-go/storage"
	"github.com/clementhaon/sandbox-api-go/websocket"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Initialize logger first
	logger.Initialize()
	logger.Info("Starting sandbox-api-go application")

	// Initialize metrics
	metrics.InitAppInfo("2.0.0", "dev", time.Now().Format("2006-01-02"), runtime.Version())

	// Initialize the database
	if err := database.InitDB(); err != nil {
		logger.Fatal("Failed to initialize database", err)
	}
	defer database.CloseDB()
	db := database.DB

	// Initialize JWT manager
	jwtSecret, err := config.RequireEnv("JWT_SECRET")
	if err != nil {
		logger.Fatal("JWT_SECRET environment variable is required", err)
	}
	jwtManager, err := auth.NewJWTManager(jwtSecret)
	if err != nil {
		logger.Fatal("Failed to initialize JWT manager", fmt.Errorf("%s", err.Error()))
	}

	// Initialize MinIO storage
	minioStorage, err := storage.NewStorage(
		config.GetEnv("MINIO_ENDPOINT", "minio:9000"),
		config.GetEnv("MINIO_ROOT_USER", "minioadmin"),
		config.GetEnv("MINIO_ROOT_PASSWORD", "minioadmin123"),
		config.GetEnv("MINIO_BUCKET", "user-uploads"),
		config.GetEnv("MINIO_USE_SSL", "false") == "true",
	)
	if err != nil {
		logger.Fatal("Failed to initialize MinIO storage", err)
	}

	// Initialize WebSocket manager
	wsManager := websocket.NewManager()
	logger.Info("WebSocket manager initialized")

	// Auth middleware with injected JWT manager
	authMW := middleware.NewAuthMiddleware(jwtManager)

	// Initialize services
	authSvc := services.NewAuthService(db, jwtManager)
	userSvc := services.NewUserService(db)
	profileSvc := services.NewProfileService(db)
	columnSvc := services.NewColumnService(db)
	taskSvc := services.NewTaskService(db)
	timeEntrySvc := services.NewTimeEntryService(db)
	notificationSvc := services.NewNotificationService(db, wsManager)
	mediaSvc := services.NewMediaService(db, minioStorage)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authSvc)
	userHandler := handlers.NewUserHandler(userSvc)
	profileHandler := handlers.NewProfileHandler(profileSvc)
	columnHandler := handlers.NewColumnHandler(columnSvc)
	taskHandler := handlers.NewTaskHandler(taskSvc)
	timeEntryHandler := handlers.NewTimeEntryHandler(timeEntrySvc)
	notificationHandler := handlers.NewNotificationHandler(notificationSvc)
	mediaHandler := handlers.NewMediaHandler(mediaSvc)
	wsHandler := handlers.NewWebSocketHandler(wsManager, jwtManager)

	// Initialize rate limiter for auth routes
	rateLimiter := middleware.NewRateLimiter(10, time.Minute)

	// Create the HTTP server with error handling middleware
	mux := createMux(authMW, rateLimiter, authHandler, userHandler, profileHandler, columnHandler, taskHandler, timeEntryHandler, notificationHandler, mediaHandler, wsHandler)
	server := &http.Server{
		Addr:         ":8080",
		Handler:      middleware.PanicRecoveryMiddleware(middleware.RequestLoggingMiddleware(mux)),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start the server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", err)
		}
	}()

	// Wait for interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutdown signal received")
	fmt.Println("\n🛑 Shutting down server...")

	// Gracefully shutdown the server
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Failed to gracefully shutdown server", err)
	}

	logger.Info("Server shutdown completed")
	fmt.Println("✅ Server shut down cleanly")
}

func createMux(
	authMW func(middleware.ErrorHandler) http.HandlerFunc,
	rateLimiter *middleware.RateLimiter,
	authHandler *handlers.AuthHandler,
	userHandler *handlers.UserHandler,
	profileHandler *handlers.ProfileHandler,
	columnHandler *handlers.ColumnHandler,
	taskHandler *handlers.TaskHandler,
	timeEntryHandler *handlers.TimeEntryHandler,
	notificationHandler *handlers.NotificationHandler,
	mediaHandler *handlers.MediaHandler,
	wsHandler *handlers.WebSocketHandler,
) http.Handler {
	mux := http.NewServeMux()

	// Public routes (no authentication required)
	mux.HandleFunc("/", middleware.ErrorMiddleware(handleHome))
	mux.HandleFunc("POST /auth/register", rateLimiter.Limit(middleware.ErrorMiddleware(authHandler.HandleRegister)))
	mux.HandleFunc("POST /auth/login", rateLimiter.Limit(middleware.ErrorMiddleware(authHandler.HandleLogin)))
	mux.HandleFunc("POST /auth/logout", middleware.ErrorMiddleware(authHandler.HandleLogout))

	// Prometheus metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// WebSocket endpoint (auth via query param)
	mux.HandleFunc("/ws", wsHandler.HandleWebSocket)

	// ============================================
	// Users Management Routes
	// ============================================
	mux.HandleFunc("GET /users", authMW(userHandler.ListUsers))
	mux.HandleFunc("GET /users/{id}", authMW(userHandler.GetUser))
	mux.HandleFunc("POST /users", authMW(userHandler.CreateUser))
	mux.HandleFunc("PUT /users/{id}", authMW(userHandler.UpdateUser))
	mux.HandleFunc("PATCH /users/{id}/status", authMW(userHandler.UpdateUserStatus))
	mux.HandleFunc("DELETE /users/{id}", authMW(userHandler.DeleteUser))

	// ============================================
	// Columns Management Routes
	// ============================================
	mux.HandleFunc("GET /columns", authMW(columnHandler.ListColumns))
	mux.HandleFunc("POST /columns", authMW(columnHandler.CreateColumn))
	mux.HandleFunc("PUT /columns/{id}", authMW(columnHandler.UpdateColumn))
	mux.HandleFunc("DELETE /columns/{id}", authMW(columnHandler.DeleteColumn))
	mux.HandleFunc("PATCH /columns/reorder", authMW(columnHandler.ReorderColumns))

	// ============================================
	// Tasks Management Routes (Board)
	// ============================================
	mux.HandleFunc("GET /tasks/board", authMW(taskHandler.GetBoard))
	mux.HandleFunc("GET /tasks", authMW(taskHandler.ListTasks))
	mux.HandleFunc("GET /tasks/{id}", authMW(taskHandler.GetTask))
	mux.HandleFunc("POST /tasks", authMW(taskHandler.CreateTask))
	mux.HandleFunc("PUT /tasks/{id}", authMW(taskHandler.UpdateTask))
	mux.HandleFunc("PATCH /tasks/{id}/move", authMW(taskHandler.MoveTask))
	mux.HandleFunc("PATCH /tasks/reorder", authMW(taskHandler.ReorderTasks))
	mux.HandleFunc("DELETE /tasks/{id}", authMW(taskHandler.DeleteTask))

	// ============================================
	// Time Entries Routes
	// ============================================
	mux.HandleFunc("GET /time-entries", authMW(timeEntryHandler.ListTimeEntries))
	mux.HandleFunc("POST /time-entries", authMW(timeEntryHandler.CreateTimeEntry))
	mux.HandleFunc("DELETE /time-entries/{id}", authMW(timeEntryHandler.DeleteTimeEntry))

	// ============================================
	// Notifications Routes
	// ============================================
	mux.HandleFunc("GET /notifications", authMW(notificationHandler.ListNotifications))
	mux.HandleFunc("PATCH /notifications/read", authMW(notificationHandler.MarkNotificationsRead))
	mux.HandleFunc("PATCH /notifications/read-all", authMW(notificationHandler.MarkAllNotificationsRead))
	mux.HandleFunc("DELETE /notifications/{id}", authMW(notificationHandler.DeleteNotification))

	// ============================================
	// Auth & Profile Routes
	// ============================================
	mux.HandleFunc("GET /auth/user", authMW(authHandler.HandleGetUser))
	mux.HandleFunc("GET /profile", authMW(profileHandler.HandleGetProfile))
	mux.HandleFunc("PUT /profile", authMW(profileHandler.HandleUpdateProfile))

	// ============================================
	// Media Routes
	// ============================================
	mux.HandleFunc("POST /media/upload", authMW(mediaHandler.HandleGetPresignedUploadURL))
	mux.HandleFunc("POST /media/confirm", authMW(mediaHandler.HandleConfirmUpload))
	mux.HandleFunc("GET /media", authMW(mediaHandler.HandleGetUserMedia))
	mux.HandleFunc("GET /media/{id}", authMW(mediaHandler.HandleGetMediaByID))
	mux.HandleFunc("GET /media/{id}/download", authMW(mediaHandler.HandleGetPresignedDownloadURL))
	mux.HandleFunc("DELETE /media/{id}", authMW(mediaHandler.HandleDeleteMedia))

	return mux
}

func handleHome(w http.ResponseWriter, r *http.Request) error {
	if r.URL.Path != "/" {
		return errors.NewNotFoundError("Page")
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"message": "Welcome to the Go REST API with authentication! 🎉",
		"version": "2.0.0",
	}

	logger.DebugContext(r.Context(), "Home endpoint accessed")
	json.NewEncoder(w).Encode(response)
	return nil
}

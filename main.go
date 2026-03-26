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

	// Initialize MinIO
	if err := storage.InitMinIO(); err != nil {
		logger.Fatal("Failed to initialize MinIO", err)
	}

	// Initialize WebSocket manager
	websocket.Init()
	wsManager := websocket.GlobalManager
	logger.Info("WebSocket manager initialized")

	// Initialize services
	authSvc := services.NewAuthService(db)
	userSvc := services.NewUserService(db)
	profileSvc := services.NewProfileService(db)
	columnSvc := services.NewColumnService(db)
	taskSvc := services.NewTaskService(db)
	timeEntrySvc := services.NewTimeEntryService(db)
	notificationSvc := services.NewNotificationService(db, wsManager)
	mediaSvc := services.NewMediaService(db)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authSvc)
	userHandler := handlers.NewUserHandler(userSvc)
	profileHandler := handlers.NewProfileHandler(profileSvc)
	columnHandler := handlers.NewColumnHandler(columnSvc)
	taskHandler := handlers.NewTaskHandler(taskSvc)
	timeEntryHandler := handlers.NewTimeEntryHandler(timeEntrySvc)
	notificationHandler := handlers.NewNotificationHandler(notificationSvc)
	mediaHandler := handlers.NewMediaHandler(mediaSvc)
	wsHandler := handlers.NewWebSocketHandler(wsManager)

	// Create the HTTP server with error handling middleware
	mux := createMux(authHandler, userHandler, profileHandler, columnHandler, taskHandler, timeEntryHandler, notificationHandler, mediaHandler, wsHandler)
	server := &http.Server{
		Addr:    ":8080",
		Handler: middleware.PanicRecoveryMiddleware(middleware.RequestLoggingMiddleware(mux)),
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
	mux.HandleFunc("/auth/register", middleware.ErrorMiddleware(authHandler.HandleRegister))
	mux.HandleFunc("/auth/login", middleware.ErrorMiddleware(authHandler.HandleLogin))
	mux.HandleFunc("/auth/logout", middleware.ErrorMiddleware(authHandler.HandleLogout))

	// Prometheus metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// WebSocket endpoint (auth via query param)
	mux.HandleFunc("/ws", wsHandler.HandleWebSocket)

	// ============================================
	// Users Management Routes
	// ============================================
	mux.HandleFunc("GET /users", middleware.AuthMiddleware(userHandler.ListUsers))
	mux.HandleFunc("GET /users/{id}", middleware.AuthMiddleware(userHandler.GetUser))
	mux.HandleFunc("POST /users", middleware.AuthMiddleware(userHandler.CreateUser))
	mux.HandleFunc("PUT /users/{id}", middleware.AuthMiddleware(userHandler.UpdateUser))
	mux.HandleFunc("PATCH /users/{id}/status", middleware.AuthMiddleware(userHandler.UpdateUserStatus))
	mux.HandleFunc("DELETE /users/{id}", middleware.AuthMiddleware(userHandler.DeleteUser))

	// ============================================
	// Columns Management Routes
	// ============================================
	mux.HandleFunc("GET /columns", middleware.AuthMiddleware(columnHandler.ListColumns))
	mux.HandleFunc("POST /columns", middleware.AuthMiddleware(columnHandler.CreateColumn))
	mux.HandleFunc("PUT /columns/{id}", middleware.AuthMiddleware(columnHandler.UpdateColumn))
	mux.HandleFunc("DELETE /columns/{id}", middleware.AuthMiddleware(columnHandler.DeleteColumn))
	mux.HandleFunc("PATCH /columns/reorder", middleware.AuthMiddleware(columnHandler.ReorderColumns))

	// ============================================
	// Tasks Management Routes (Board)
	// ============================================
	mux.HandleFunc("GET /tasks/board", middleware.AuthMiddleware(taskHandler.GetBoard))
	mux.HandleFunc("GET /tasks", middleware.AuthMiddleware(taskHandler.ListTasks))
	mux.HandleFunc("GET /tasks/{id}", middleware.AuthMiddleware(taskHandler.GetTask))
	mux.HandleFunc("POST /tasks", middleware.AuthMiddleware(taskHandler.CreateTask))
	mux.HandleFunc("PUT /tasks/{id}", middleware.AuthMiddleware(taskHandler.UpdateTask))
	mux.HandleFunc("PATCH /tasks/{id}/move", middleware.AuthMiddleware(taskHandler.MoveTask))
	mux.HandleFunc("PATCH /tasks/reorder", middleware.AuthMiddleware(taskHandler.ReorderTasks))
	mux.HandleFunc("DELETE /tasks/{id}", middleware.AuthMiddleware(taskHandler.DeleteTask))

	// ============================================
	// Time Entries Routes
	// ============================================
	mux.HandleFunc("GET /time-entries", middleware.AuthMiddleware(timeEntryHandler.ListTimeEntries))
	mux.HandleFunc("POST /time-entries", middleware.AuthMiddleware(timeEntryHandler.CreateTimeEntry))
	mux.HandleFunc("DELETE /time-entries/{id}", middleware.AuthMiddleware(timeEntryHandler.DeleteTimeEntry))

	// ============================================
	// Notifications Routes
	// ============================================
	mux.HandleFunc("GET /notifications", middleware.AuthMiddleware(notificationHandler.ListNotifications))
	mux.HandleFunc("PATCH /notifications/read", middleware.AuthMiddleware(notificationHandler.MarkNotificationsRead))
	mux.HandleFunc("PATCH /notifications/read-all", middleware.AuthMiddleware(notificationHandler.MarkAllNotificationsRead))
	mux.HandleFunc("DELETE /notifications/{id}", middleware.AuthMiddleware(notificationHandler.DeleteNotification))

	// ============================================
	// Auth & Profile Routes
	// ============================================
	mux.HandleFunc("/auth/user", middleware.AuthMiddleware(authHandler.HandleGetUser))
	mux.HandleFunc("/profile", middleware.AuthMiddleware(handleProfile(profileHandler)))

	// ============================================
	// Media Routes
	// ============================================
	mux.HandleFunc("/media/upload", middleware.AuthMiddleware(mediaHandler.HandleGetPresignedUploadURL))
	mux.HandleFunc("/media/confirm", middleware.AuthMiddleware(mediaHandler.HandleConfirmUpload))
	mux.HandleFunc("/media", middleware.AuthMiddleware(mediaHandler.HandleGetUserMedia))
	mux.HandleFunc("/media/", middleware.AuthMiddleware(handleMediaRoutes(mediaHandler)))

	return mux
}

// handleProfile dispatches profile requests based on the HTTP method
func handleProfile(h *handlers.ProfileHandler) middleware.ErrorHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		switch r.Method {
		case http.MethodGet:
			return h.HandleGetProfile(w, r)
		case http.MethodPut:
			return h.HandleUpdateProfile(w, r)
		default:
			return errors.NewMethodNotAllowedError()
		}
	}
}

// handleMediaRoutes dispatches media requests based on path and HTTP method
func handleMediaRoutes(h *handlers.MediaHandler) middleware.ErrorHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		path := r.URL.Path

		// /media/{id}/download - Get a presigned download URL
		if len(path) > 7 && path[len(path)-9:] == "/download" {
			return h.HandleGetPresignedDownloadURL(w, r)
		}

		// /media/{id} - Get a media by ID or delete it
		switch r.Method {
		case http.MethodGet:
			return h.HandleGetMediaByID(w, r)
		case http.MethodDelete:
			return h.HandleDeleteMedia(w, r)
		default:
			return errors.NewMethodNotAllowedError()
		}
	}
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

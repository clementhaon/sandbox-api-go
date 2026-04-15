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
	"github.com/clementhaon/sandbox-api-go/repository"
	"github.com/clementhaon/sandbox-api-go/services"
	"github.com/clementhaon/sandbox-api-go/storage"
	"github.com/clementhaon/sandbox-api-go/websocket"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type app struct {
	config      *config.Config
	authMW      func(middleware.ErrorHandler) http.HandlerFunc
	rateLimiter *middleware.RateLimiter

	authHandler         *handlers.AuthHandler
	userHandler         *handlers.UserHandler
	profileHandler      *handlers.ProfileHandler
	columnHandler       *handlers.ColumnHandler
	taskHandler         *handlers.TaskHandler
	timeEntryHandler    *handlers.TimeEntryHandler
	notificationHandler *handlers.NotificationHandler
	mediaHandler        *handlers.MediaHandler
	wsHandler           *handlers.WebSocketHandler
}

func (a *app) routes() http.Handler {
	mux := http.NewServeMux()

	// Public routes (no authentication required)
	mux.HandleFunc("/", middleware.ErrorMiddleware(handleHome))
	mux.HandleFunc("POST /auth/register", a.rateLimiter.Limit(middleware.ErrorMiddleware(a.authHandler.HandleRegister)))
	mux.HandleFunc("POST /auth/login", a.rateLimiter.Limit(middleware.ErrorMiddleware(a.authHandler.HandleLogin)))
	mux.HandleFunc("POST /auth/logout", middleware.ErrorMiddleware(a.authHandler.HandleLogout))

	// Prometheus metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// WebSocket endpoint (auth via query param)
	mux.HandleFunc("/ws", a.wsHandler.HandleWebSocket)

	// Users Management Routes
	mux.HandleFunc("GET /users", a.authMW(a.userHandler.ListUsers))
	mux.HandleFunc("GET /users/{id}", a.authMW(a.userHandler.GetUser))
	mux.HandleFunc("POST /users", a.authMW(a.userHandler.CreateUser))
	mux.HandleFunc("PUT /users/{id}", a.authMW(a.userHandler.UpdateUser))
	mux.HandleFunc("PATCH /users/{id}/status", a.authMW(a.userHandler.UpdateUserStatus))
	mux.HandleFunc("DELETE /users/{id}", a.authMW(a.userHandler.DeleteUser))

	// Columns Management Routes
	mux.HandleFunc("GET /columns", a.authMW(a.columnHandler.ListColumns))
	mux.HandleFunc("POST /columns", a.authMW(a.columnHandler.CreateColumn))
	mux.HandleFunc("PUT /columns/{id}", a.authMW(a.columnHandler.UpdateColumn))
	mux.HandleFunc("DELETE /columns/{id}", a.authMW(a.columnHandler.DeleteColumn))
	mux.HandleFunc("PATCH /columns/reorder", a.authMW(a.columnHandler.ReorderColumns))

	// Tasks Management Routes (Board)
	mux.HandleFunc("GET /tasks/board", a.authMW(a.taskHandler.GetBoard))
	mux.HandleFunc("GET /tasks", a.authMW(a.taskHandler.ListTasks))
	mux.HandleFunc("GET /tasks/{id}", a.authMW(a.taskHandler.GetTask))
	mux.HandleFunc("POST /tasks", a.authMW(a.taskHandler.CreateTask))
	mux.HandleFunc("PUT /tasks/{id}", a.authMW(a.taskHandler.UpdateTask))
	mux.HandleFunc("PATCH /tasks/{id}/move", a.authMW(a.taskHandler.MoveTask))
	mux.HandleFunc("PATCH /tasks/reorder", a.authMW(a.taskHandler.ReorderTasks))
	mux.HandleFunc("DELETE /tasks/{id}", a.authMW(a.taskHandler.DeleteTask))

	// Time Entries Routes
	mux.HandleFunc("GET /time-entries", a.authMW(a.timeEntryHandler.ListTimeEntries))
	mux.HandleFunc("POST /time-entries", a.authMW(a.timeEntryHandler.CreateTimeEntry))
	mux.HandleFunc("DELETE /time-entries/{id}", a.authMW(a.timeEntryHandler.DeleteTimeEntry))

	// Notifications Routes
	mux.HandleFunc("GET /notifications", a.authMW(a.notificationHandler.ListNotifications))
	mux.HandleFunc("PATCH /notifications/read", a.authMW(a.notificationHandler.MarkNotificationsRead))
	mux.HandleFunc("PATCH /notifications/read-all", a.authMW(a.notificationHandler.MarkAllNotificationsRead))
	mux.HandleFunc("DELETE /notifications/{id}", a.authMW(a.notificationHandler.DeleteNotification))

	// Auth & Profile Routes
	mux.HandleFunc("GET /auth/user", a.authMW(a.authHandler.HandleGetUser))
	mux.HandleFunc("GET /profile", a.authMW(a.profileHandler.HandleGetProfile))
	mux.HandleFunc("PUT /profile", a.authMW(a.profileHandler.HandleUpdateProfile))

	// Media Routes
	mux.HandleFunc("POST /media/upload", a.authMW(a.mediaHandler.HandleGetPresignedUploadURL))
	mux.HandleFunc("POST /media/confirm", a.authMW(a.mediaHandler.HandleConfirmUpload))
	mux.HandleFunc("GET /media", a.authMW(a.mediaHandler.HandleGetUserMedia))
	mux.HandleFunc("GET /media/{id}", a.authMW(a.mediaHandler.HandleGetMediaByID))
	mux.HandleFunc("GET /media/{id}/download", a.authMW(a.mediaHandler.HandleGetPresignedDownloadURL))
	mux.HandleFunc("DELETE /media/{id}", a.authMW(a.mediaHandler.HandleDeleteMedia))

	return mux
}

func main() {
	// Initialize logger first
	logger.Initialize()
	logger.Info("Starting sandbox-api-go application")

	// Initialize metrics
	metrics.InitAppInfo("2.0.0", "dev", time.Now().Format("2006-01-02"), runtime.Version())

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", fmt.Errorf("%s", err.Error()))
	}

	// Initialize the database
	if err := database.InitDB(); err != nil {
		logger.Fatal("Failed to initialize database", err)
	}
	defer database.CloseDB()
	db := database.DB

	// Initialize JWT manager
	jwtManager, err := auth.NewJWTManager(cfg.JWTSecret)
	if err != nil {
		logger.Fatal("Failed to initialize JWT manager", fmt.Errorf("%s", err.Error()))
	}

	// Initialize MinIO storage
	minioStorage, err := storage.NewStorage(
		cfg.MinioEndpoint,
		cfg.MinioUser,
		cfg.MinioPassword,
		cfg.MinioBucket,
		cfg.MinioUseSSL,
	)
	if err != nil {
		logger.Fatal("Failed to initialize MinIO storage", err)
	}

	// Initialize WebSocket manager
	wsManager := websocket.NewManager()
	logger.Info("WebSocket manager initialized")

	// Initialize token blacklist
	blacklist := auth.NewTokenBlacklist()
	defer blacklist.Stop()

	// Auth middleware with injected JWT manager and blacklist
	authMW := middleware.NewAuthMiddleware(jwtManager, blacklist)

	// Initialize transaction manager
	txManager := database.NewTxManager(db)

	// Initialize repositories
	userRepo := repository.NewPostgresUserRepository(db)
	taskRepo := repository.NewPostgresTaskRepository(db)
	columnRepo := repository.NewPostgresColumnRepository(db)
	timeEntryRepo := repository.NewPostgresTimeEntryRepository(db)
	notifRepo := repository.NewPostgresNotificationRepository(db)
	mediaRepo := repository.NewPostgresMediaRepository(db)

	// Initialize services
	authSvc := services.NewAuthService(userRepo, jwtManager)
	userSvc := services.NewUserService(userRepo)
	profileSvc := services.NewProfileService(userRepo)
	columnSvc := services.NewColumnService(columnRepo, txManager)
	taskSvc := services.NewTaskService(taskRepo, columnRepo)
	timeEntrySvc := services.NewTimeEntryService(timeEntryRepo, txManager)
	notificationSvc := services.NewNotificationService(notifRepo, wsManager)
	mediaSvc := services.NewMediaService(mediaRepo, minioStorage)

	// Initialize rate limiter
	rateLimiter := middleware.NewRateLimiter(cfg.RateLimitRequests, cfg.RateLimitWindow)
	defer rateLimiter.Stop()

	// Build application
	a := &app{
		config:              cfg,
		authMW:              authMW,
		rateLimiter:         rateLimiter,
		authHandler:         handlers.NewAuthHandler(authSvc, jwtManager, blacklist),
		userHandler:         handlers.NewUserHandler(userSvc),
		profileHandler:      handlers.NewProfileHandler(profileSvc),
		columnHandler:       handlers.NewColumnHandler(columnSvc),
		taskHandler:         handlers.NewTaskHandler(taskSvc),
		timeEntryHandler:    handlers.NewTimeEntryHandler(timeEntrySvc),
		notificationHandler: handlers.NewNotificationHandler(notificationSvc),
		mediaHandler:        handlers.NewMediaHandler(mediaSvc),
		wsHandler:           handlers.NewWebSocketHandler(wsManager, jwtManager),
	}

	// Create the HTTP server
	handler := middleware.CSRFMiddleware(middleware.MaxBytesMiddleware(cfg.MaxBodySize)(a.routes()))
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      middleware.PanicRecoveryMiddleware(middleware.RequestLoggingMiddleware(handler)),
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

package main

import (
	"context"
	"database/sql"
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
	"github.com/clementhaon/sandbox-api-go/services/notification-service/handlers"
	"github.com/clementhaon/sandbox-api-go/services/notification-service/services"
	"github.com/clementhaon/sandbox-api-go/services/notification-service/websocket"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Initialize logger first
	logger.Initialize()
	logger.Info("Starting notification-service")

	// Initialize metrics
	metrics.InitAppInfo("1.0.0", "dev", time.Now().Format("2006-01-02"), runtime.Version())

	// Initialize the database
	db, err := database.InitDB()
	if err != nil {
		logger.Fatal("Failed to initialize database", err)
	}
	defer db.Close()

	// Run migrations
	runMigrations(db)

	// Initialize JWT manager
	jwtSecret := config.GetEnv("JWT_SECRET", "default-secret-change-me-in-prod")
	jwtManager, err := auth.NewJWTManager(jwtSecret)
	if err != nil {
		logger.Fatal("Failed to initialize JWT manager", fmt.Errorf("%s", err.Error()))
	}

	// Initialize WebSocket manager
	wsManager := websocket.NewManager()
	logger.Info("WebSocket manager initialized")

	// Auth middleware with injected JWT manager
	authMW := middleware.NewAuthMiddleware(jwtManager)

	// Initialize services
	notificationSvc := services.NewNotificationService(db, wsManager)

	// Initialize handlers
	notificationHandler := handlers.NewNotificationHandler(notificationSvc)
	wsHandler := handlers.NewWebSocketHandler(wsManager, jwtManager)

	// Create the HTTP server with error handling middleware
	mux := createMux(authMW, notificationHandler, wsHandler)
	server := &http.Server{
		Addr:    ":8083",
		Handler: middleware.PanicRecoveryMiddleware(middleware.RequestLoggingMiddleware(mux)),
	}

	// Start the server in a goroutine
	go func() {
		logger.Info("Notification service listening on :8083")
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

	logger.Info("Server shutdown completed")
}

func runMigrations(db *sql.DB) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		logger.Fatal("Failed to create migration driver", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://database/migrations", "postgres", driver)
	if err != nil {
		logger.Fatal("Failed to create migrate instance", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		logger.Fatal("Failed to run migrations", err)
	}

	logger.Info("Database migrations completed")
}

func createMux(
	authMW func(middleware.ErrorHandler) http.HandlerFunc,
	notificationHandler *handlers.NotificationHandler,
	wsHandler *handlers.WebSocketHandler,
) http.Handler {
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/", middleware.ErrorMiddleware(handleHome))

	// Prometheus metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// WebSocket endpoint (auth via query param)
	mux.HandleFunc("/ws", wsHandler.HandleWebSocket)

	// Notifications Routes (auth protected)
	mux.HandleFunc("GET /notifications", authMW(notificationHandler.ListNotifications))
	mux.HandleFunc("PATCH /notifications/read", authMW(notificationHandler.MarkNotificationsRead))
	mux.HandleFunc("PATCH /notifications/read-all", authMW(notificationHandler.MarkAllNotificationsRead))
	mux.HandleFunc("DELETE /notifications/{id}", authMW(notificationHandler.DeleteNotification))

	return mux
}

func handleHome(w http.ResponseWriter, r *http.Request) error {
	if r.URL.Path != "/" {
		return errors.NewNotFoundError("Page")
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"service": "notification-service",
		"version": "1.0.0",
	}

	logger.DebugContext(r.Context(), "Home endpoint accessed")
	json.NewEncoder(w).Encode(response)
	return nil
}

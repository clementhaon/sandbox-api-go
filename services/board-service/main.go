package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
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
	"github.com/clementhaon/sandbox-api-go/pkg/httpclient"
	"github.com/clementhaon/sandbox-api-go/pkg/logger"
	"github.com/clementhaon/sandbox-api-go/pkg/metrics"
	"github.com/clementhaon/sandbox-api-go/pkg/middleware"
	"github.com/clementhaon/sandbox-api-go/services/board-service/handlers"
	"github.com/clementhaon/sandbox-api-go/services/board-service/services"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Initialize logger
	logger.Initialize()
	logger.Info("Starting board-service")

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

	// Initialize User Service client
	userServiceURL := config.GetEnv("USER_SERVICE_URL", "http://user-service:8081")
	userClient := httpclient.NewUserServiceClient(userServiceURL)

	// Auth middleware
	authMW := middleware.NewAuthMiddleware(jwtManager)

	// Initialize services
	taskSvc := services.NewTaskService(db, userClient)
	columnSvc := services.NewColumnService(db)
	timeEntrySvc := services.NewTimeEntryService(db)

	// Initialize handlers
	taskHandler := handlers.NewTaskHandler(taskSvc)
	columnHandler := handlers.NewColumnHandler(columnSvc)
	timeEntryHandler := handlers.NewTimeEntryHandler(timeEntrySvc)

	// Create router
	mux := createMux(authMW, taskHandler, columnHandler, timeEntryHandler)
	server := &http.Server{
		Addr:    ":8082",
		Handler: middleware.PanicRecoveryMiddleware(middleware.RequestLoggingMiddleware(mux)),
	}

	// Start the server in a goroutine
	go func() {
		logger.Info("Board service listening on :8082")
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

	logger.Info("Board service shutdown completed")
}

func runMigrations(db *sql.DB) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		logger.Fatal("Failed to create migration driver", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://database/migrations", "postgres", driver)
	if err != nil {
		logger.Fatal("Failed to create migration instance", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		logger.Fatal("Failed to run migrations", err)
	}

	log.Println("Database migrations completed successfully")
}

func createMux(
	authMW func(middleware.ErrorHandler) http.HandlerFunc,
	taskHandler *handlers.TaskHandler,
	columnHandler *handlers.ColumnHandler,
	timeEntryHandler *handlers.TimeEntryHandler,
) http.Handler {
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/", middleware.ErrorMiddleware(handleHome))
	mux.Handle("/metrics", promhttp.Handler())

	// Tasks routes (auth protected)
	mux.HandleFunc("GET /tasks/board", authMW(taskHandler.GetBoard))
	mux.HandleFunc("GET /tasks", authMW(taskHandler.ListTasks))
	mux.HandleFunc("GET /tasks/{id}", authMW(taskHandler.GetTask))
	mux.HandleFunc("POST /tasks", authMW(taskHandler.CreateTask))
	mux.HandleFunc("PUT /tasks/{id}", authMW(taskHandler.UpdateTask))
	mux.HandleFunc("PATCH /tasks/{id}/move", authMW(taskHandler.MoveTask))
	mux.HandleFunc("PATCH /tasks/reorder", authMW(taskHandler.ReorderTasks))
	mux.HandleFunc("DELETE /tasks/{id}", authMW(taskHandler.DeleteTask))

	// Columns routes (auth protected)
	mux.HandleFunc("GET /columns", authMW(columnHandler.ListColumns))
	mux.HandleFunc("POST /columns", authMW(columnHandler.CreateColumn))
	mux.HandleFunc("PUT /columns/{id}", authMW(columnHandler.UpdateColumn))
	mux.HandleFunc("DELETE /columns/{id}", authMW(columnHandler.DeleteColumn))
	mux.HandleFunc("PATCH /columns/reorder", authMW(columnHandler.ReorderColumns))

	// Time entries routes (auth protected)
	mux.HandleFunc("GET /time-entries", authMW(timeEntryHandler.ListTimeEntries))
	mux.HandleFunc("POST /time-entries", authMW(timeEntryHandler.CreateTimeEntry))
	mux.HandleFunc("DELETE /time-entries/{id}", authMW(timeEntryHandler.DeleteTimeEntry))

	return mux
}

func handleHome(w http.ResponseWriter, r *http.Request) error {
	if r.URL.Path != "/" {
		return errors.NewNotFoundError("Page")
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"service": "board-service",
		"version": "1.0.0",
		"message": "Board Service API",
	}

	logger.DebugContext(r.Context(), "Home endpoint accessed")
	json.NewEncoder(w).Encode(response)
	return nil
}

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
	"github.com/clementhaon/sandbox-api-go/services/media-service/handlers"
	"github.com/clementhaon/sandbox-api-go/services/media-service/services"
	"github.com/clementhaon/sandbox-api-go/services/media-service/storage"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Initialize logger
	logger.Initialize()
	logger.Info("Starting media-service")

	// Initialize metrics
	metrics.InitAppInfo("1.0.0", "dev", time.Now().Format("2006-01-02"), runtime.Version())

	// Initialize database
	db, err := database.InitDB()
	if err != nil {
		logger.Fatal("Failed to initialize database", err)
	}
	defer db.Close()

	// Run migrations
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
	logger.Info("Database migrations completed")

	// Initialize JWT manager
	jwtSecret := config.GetEnv("JWT_SECRET", "default-secret-change-me-in-prod")
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

	// Auth middleware
	authMW := middleware.NewAuthMiddleware(jwtManager)

	// Initialize service and handler
	mediaSvc := services.NewMediaService(db, minioStorage)
	mediaHandler := handlers.NewMediaHandler(mediaSvc)

	// Create router
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/", middleware.ErrorMiddleware(handleHome))
	mux.Handle("/metrics", promhttp.Handler())

	// Media routes (auth protected)
	mux.HandleFunc("POST /media/upload", authMW(mediaHandler.HandleGetPresignedUploadURL))
	mux.HandleFunc("POST /media/confirm", authMW(mediaHandler.HandleConfirmUpload))
	mux.HandleFunc("GET /media", authMW(mediaHandler.HandleGetUserMedia))
	mux.HandleFunc("GET /media/{id}", authMW(mediaHandler.HandleGetMediaByID))
	mux.HandleFunc("GET /media/{id}/download", authMW(mediaHandler.HandleGetPresignedDownloadURL))
	mux.HandleFunc("DELETE /media/{id}", authMW(mediaHandler.HandleDeleteMedia))

	// Create server with middleware
	server := &http.Server{
		Addr:    ":8084",
		Handler: middleware.PanicRecoveryMiddleware(middleware.RequestLoggingMiddleware(mux)),
	}

	// Start server
	go func() {
		logger.Info("Media service listening on :8084")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Failed to gracefully shutdown server", err)
	}

	logger.Info("Media service shutdown completed")
}

func handleHome(w http.ResponseWriter, r *http.Request) error {
	if r.URL.Path != "/" {
		return errors.NewNotFoundError("Page")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service": "media-service",
		"version": "1.0.0",
	})
	return nil
}

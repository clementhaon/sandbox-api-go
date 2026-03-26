package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/clementhaon/sandbox-api-go/database"
	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/handlers"
	"github.com/clementhaon/sandbox-api-go/logger"
	"github.com/clementhaon/sandbox-api-go/metrics"
	"github.com/clementhaon/sandbox-api-go/middleware"
	"github.com/clementhaon/sandbox-api-go/storage"
	"github.com/clementhaon/sandbox-api-go/websocket"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Initialize logger first
	logger.Initialize()
	logger.Info("Starting sandbox-api-go application")

	// Initialize metrics
	metrics.InitAppInfo("2.0.0", "dev", time.Now().Format("2006-01-02"), runtime.Version())

	// Initialisation de la base de données
	if err := database.InitDB(); err != nil {
		logger.Fatal("Failed to initialize database", err)
	}
	defer database.CloseDB()

	// Initialisation de MinIO
	if err := storage.InitMinIO(); err != nil {
		logger.Fatal("Failed to initialize MinIO", err)
	}

	// Initialize WebSocket manager
	websocket.Init()
	logger.Info("WebSocket manager initialized")

	// Création du serveur HTTP avec middleware de gestion d'erreurs
	server := &http.Server{
		Addr:    ":8080",
		Handler: middleware.PanicRecoveryMiddleware(middleware.RequestLoggingMiddleware(createMux())),
	}

	// Démarrage du serveur dans une goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", err)
		}
	}()

	// Attente des signaux d'interruption
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutdown signal received")
	fmt.Println("\n🛑 Arrêt du serveur...")

	// Arrêt gracieux du serveur
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Failed to gracefully shutdown server", err)
	}

	logger.Info("Server shutdown completed")
	fmt.Println("✅ Serveur arrêté proprement")
}

// createMux crée et configure le routeur HTTP
func createMux() http.Handler {
	mux := http.NewServeMux()

	// Routes publiques (pas d'authentification requise)
	mux.HandleFunc("/", middleware.ErrorMiddleware(handleHome))
	mux.HandleFunc("/auth/register", middleware.ErrorMiddleware(handlers.HandleRegister))
	mux.HandleFunc("/auth/login", middleware.ErrorMiddleware(handlers.HandleLogin))
	mux.HandleFunc("/auth/logout", middleware.ErrorMiddleware(handlers.HandleLogout))

	// Prometheus metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// WebSocket endpoint (auth via query param)
	mux.HandleFunc("/ws", handlers.HandleWebSocket)

	// ============================================
	// Users Management Routes
	// ============================================
	mux.HandleFunc("GET /users", middleware.AuthMiddleware(handlers.ListUsers))
	mux.HandleFunc("GET /users/{id}", middleware.AuthMiddleware(handlers.GetUser))
	mux.HandleFunc("POST /users", middleware.AuthMiddleware(handlers.CreateUser))
	mux.HandleFunc("PUT /users/{id}", middleware.AuthMiddleware(handlers.UpdateUser))
	mux.HandleFunc("PATCH /users/{id}/status", middleware.AuthMiddleware(handlers.UpdateUserStatus))
	mux.HandleFunc("DELETE /users/{id}", middleware.AuthMiddleware(handlers.DeleteUser))

	// ============================================
	// Columns Management Routes
	// ============================================
	mux.HandleFunc("GET /columns", middleware.AuthMiddleware(handlers.ListColumns))
	mux.HandleFunc("POST /columns", middleware.AuthMiddleware(handlers.CreateColumn))
	mux.HandleFunc("PUT /columns/{id}", middleware.AuthMiddleware(handlers.UpdateColumn))
	mux.HandleFunc("DELETE /columns/{id}", middleware.AuthMiddleware(handlers.DeleteColumn))
	mux.HandleFunc("PATCH /columns/reorder", middleware.AuthMiddleware(handlers.ReorderColumns))

	// ============================================
	// Tasks Management Routes (Board)
	// ============================================
	mux.HandleFunc("GET /tasks/board", middleware.AuthMiddleware(handlers.GetBoard))
	mux.HandleFunc("GET /tasks", middleware.AuthMiddleware(handlers.ListTasks))
	mux.HandleFunc("GET /tasks/{id}", middleware.AuthMiddleware(handlers.GetTask))
	mux.HandleFunc("POST /tasks", middleware.AuthMiddleware(handlers.CreateTask))
	mux.HandleFunc("PUT /tasks/{id}", middleware.AuthMiddleware(handlers.UpdateTask))
	mux.HandleFunc("PATCH /tasks/{id}/move", middleware.AuthMiddleware(handlers.MoveTask))
	mux.HandleFunc("PATCH /tasks/reorder", middleware.AuthMiddleware(handlers.ReorderTasks))
	mux.HandleFunc("DELETE /tasks/{id}", middleware.AuthMiddleware(handlers.DeleteTask))

	// ============================================
	// Time Entries Routes
	// ============================================
	mux.HandleFunc("GET /time-entries", middleware.AuthMiddleware(handlers.ListTimeEntries))
	mux.HandleFunc("POST /time-entries", middleware.AuthMiddleware(handlers.CreateTimeEntry))
	mux.HandleFunc("DELETE /time-entries/{id}", middleware.AuthMiddleware(handlers.DeleteTimeEntry))

	// ============================================
	// Notifications Routes
	// ============================================
	mux.HandleFunc("GET /notifications", middleware.AuthMiddleware(handlers.ListNotifications))
	mux.HandleFunc("PATCH /notifications/read", middleware.AuthMiddleware(handlers.MarkNotificationsRead))
	mux.HandleFunc("PATCH /notifications/read-all", middleware.AuthMiddleware(handlers.MarkAllNotificationsRead))
	mux.HandleFunc("DELETE /notifications/{id}", middleware.AuthMiddleware(handlers.DeleteNotification))

	// ============================================
	// Auth & Profile Routes
	// ============================================
	mux.HandleFunc("/auth/user", middleware.AuthMiddleware(handlers.HandleGetUser))
	mux.HandleFunc("/profile", middleware.AuthMiddleware(handleProfile))

	// ============================================
	// Media Routes
	// ============================================
	mux.HandleFunc("/media/upload", middleware.AuthMiddleware(handlers.HandleGetPresignedUploadURL))
	mux.HandleFunc("/media/confirm", middleware.AuthMiddleware(handlers.HandleConfirmUpload))
	mux.HandleFunc("/media", middleware.AuthMiddleware(handlers.HandleGetUserMedia))
	mux.HandleFunc("/media/", middleware.AuthMiddleware(handleMediaRoutes))

	return mux
}

// handleProfile dispatche les requêtes de profil selon la méthode HTTP
func handleProfile(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case http.MethodGet:
		return handlers.HandleGetProfile(w, r)
	case http.MethodPut:
		return handlers.HandleUpdateProfile(w, r)
	default:
		return errors.NewMethodNotAllowedError()
	}
}

// handleMediaRoutes dispatche les requêtes média selon le path et la méthode HTTP
func handleMediaRoutes(w http.ResponseWriter, r *http.Request) error {
	path := r.URL.Path

	// /media/{id}/download - Obtenir une URL de téléchargement présignée
	if len(path) > 7 && path[len(path)-9:] == "/download" {
		return handlers.HandleGetPresignedDownloadURL(w, r)
	}

	// /media/{id} - Obtenir un média par ID ou le supprimer
	switch r.Method {
	case http.MethodGet:
		return handlers.HandleGetMediaByID(w, r)
	case http.MethodDelete:
		return handlers.HandleDeleteMedia(w, r)
	default:
		return errors.NewMethodNotAllowedError()
	}
}

func handleHome(w http.ResponseWriter, r *http.Request) error {
	if r.URL.Path != "/" {
		return errors.NewNotFoundError("Page")
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"message": "Bienvenue dans l'API REST Go avec authentification! 🎉",
		"version": "2.0.0",
	}

	logger.DebugContext(r.Context(), "Home endpoint accessed")
	json.NewEncoder(w).Encode(response)
	return nil
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sandbox-api-go/database"
	"sandbox-api-go/errors"
	"sandbox-api-go/handlers"
	"sandbox-api-go/logger"
	"sandbox-api-go/metrics"
	"sandbox-api-go/middleware"
	"sandbox-api-go/storage"
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

	// Routes protégées (authentification requise)
	mux.HandleFunc("/tasks", middleware.AuthMiddleware(handlers.HandleTasks))
	mux.HandleFunc("/tasks/", middleware.AuthMiddleware(handlers.HandleTaskByID))
	mux.HandleFunc("/auth/user", middleware.AuthMiddleware(handlers.HandleGetUser))
	mux.HandleFunc("/profile", middleware.AuthMiddleware(handleProfile))

	// Routes pour la gestion des médias (authentification requise)
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

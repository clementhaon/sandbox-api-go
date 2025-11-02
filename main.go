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
	"sandbox-api-go/handlers"
	"sandbox-api-go/middleware"
	"sandbox-api-go/logger"
	"sandbox-api-go/errors"
	"sandbox-api-go/metrics"
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

	// Création du serveur HTTP avec middleware de gestion d'erreurs
	server := &http.Server{
		Addr:    ":8080",
		Handler: middleware.PanicRecoveryMiddleware(middleware.RequestLoggingMiddleware(createMux())),
	}

	// Démarrage du serveur dans une goroutine
	go func() {
		logger.Info("Server starting", map[string]interface{}{
			"port": "8080",
			"endpoints": map[string]interface{}{
				"public": []string{
					"POST /auth/register",
					"POST /auth/login",
					"POST /auth/logout",
				},
				"authenticated": []string{
					"GET /auth/user",
					"GET /api/profile",
					"PUT /api/profile",
					"GET /api/tasks",
					"POST /api/tasks",
					"GET /api/tasks/{id}",
					"PUT /api/tasks/{id}",
					"DELETE /api/tasks/{id}",
				},
			},
		})
		
		fmt.Println("🚀 Serveur API REST avec authentification démarré sur http://localhost:8080")
		fmt.Println("📋 Endpoints disponibles:")
		fmt.Println("  Authentification (publique):")
		fmt.Println("    POST /auth/register      - S'inscrire")
		fmt.Println("    POST /auth/login         - Se connecter")
		fmt.Println("    POST /auth/logout        - Se déconnecter")
		fmt.Println("  Profil utilisateur (authentification requise):")
		fmt.Println("    GET    /auth/user        - Obtenir les informations JWT de l'utilisateur")
		fmt.Println("    GET    /api/profile      - Obtenir le profil complet")
		fmt.Println("    PUT    /api/profile      - Modifier le profil (first_name, last_name, avatar_url)")
		fmt.Println("  Tâches (authentification requise):")
		fmt.Println("    GET    /api/tasks        - Lister vos tâches")
		fmt.Println("    POST   /api/tasks        - Créer une tâche")
		fmt.Println("    GET    /api/tasks/{id}   - Obtenir une tâche")
		fmt.Println("    PUT    /api/tasks/{id}   - Mettre à jour une tâche")
		fmt.Println("    DELETE /api/tasks/{id}   - Supprimer une tâche")
		fmt.Println("🗄️  Base de données PostgreSQL connectée")

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
	mux.HandleFunc("/api/tasks", middleware.AuthMiddleware(handlers.HandleTasks))
	mux.HandleFunc("/api/tasks/", middleware.AuthMiddleware(handlers.HandleTaskByID))
	mux.HandleFunc("/auth/user", middleware.AuthMiddleware(handlers.HandleGetUser))
	mux.HandleFunc("/api/profile", middleware.AuthMiddleware(handleProfile))

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

func handleHome(w http.ResponseWriter, r *http.Request) error {
	if r.URL.Path != "/" {
		return errors.NewNotFoundError("Page")
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"message": "Bienvenue dans l'API REST Go avec authentification! 🎉",
		"version": "2.0.0",
		"features": []string{
			"Advanced error handling",
			"Structured logging",
			"Input validation",
			"Request tracking",
		},
		"endpoints": map[string]interface{}{
			"auth": map[string]string{
				"register": "POST /auth/register",
				"login":    "POST /auth/login",
				"logout":   "POST /auth/logout",
			},
			"profile": map[string]string{
				"get":    "GET /api/profile (with Authorization header)",
				"update": "PUT /api/profile (with Authorization header)",
			},
			"tasks": map[string]string{
				"list":   "GET /api/tasks (with Authorization header)",
				"create": "POST /api/tasks (with Authorization header)",
				"get":    "GET /api/tasks/{id} (with Authorization header)",
				"update": "PUT /api/tasks/{id} (with Authorization header)",
				"delete": "DELETE /api/tasks/{id} (with Authorization header)",
			},
		},
		"example": map[string]interface{}{
			"login": map[string]interface{}{
				"url":  "/auth/login",
				"body": map[string]string{"email": "user@example.com", "password": "YourPassword123"},
			},
			"usage": "Utilisez le token reçu avec 'Authorization: Bearer <token>'",
		},
	}

	logger.DebugContext(r.Context(), "Home endpoint accessed")
	json.NewEncoder(w).Encode(response)
	return nil
} 
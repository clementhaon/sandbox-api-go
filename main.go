package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sandbox-api-go/database"
	"sandbox-api-go/handlers"
	"sandbox-api-go/middleware"
	"syscall"
	"time"
)

func main() {
	// Initialisation de la base de données
	if err := database.InitDB(); err != nil {
		log.Fatalf("❌ Erreur lors de l'initialisation de la base de données: %v", err)
	}
	defer database.CloseDB()

	// Création du serveur HTTP
	server := &http.Server{
		Addr:    ":8080",
		Handler: createMux(),
	}

	// Démarrage du serveur dans une goroutine
	go func() {
		fmt.Println("🚀 Serveur API REST avec authentification démarré sur http://localhost:8080")
		fmt.Println("📋 Endpoints disponibles:")
		fmt.Println("  Authentification (publique):")
		fmt.Println("    POST /auth/register      - S'inscrire")
		fmt.Println("    POST /auth/login         - Se connecter")
		fmt.Println("    POST /auth/logout        - Se déconnecter")
		fmt.Println("  Tâches (authentification requise):")
		fmt.Println("    GET    /auth/user        - Obtenir les informations de l'utilisateur")
		fmt.Println("    GET    /api/tasks       - Lister vos tâches")
		fmt.Println("    POST   /api/tasks       - Créer une tâche")
		fmt.Println("    GET    /api/tasks/{id}  - Obtenir une tâche")
		fmt.Println("    PUT    /api/tasks/{id}  - Mettre à jour une tâche")
		fmt.Println("    DELETE /api/tasks/{id}  - Supprimer une tâche")
		fmt.Println("🗄️  Base de données PostgreSQL connectée")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Erreur lors du démarrage du serveur: %v", err)
		}
	}()

	// Attente des signaux d'interruption
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n🛑 Arrêt du serveur...")

	// Arrêt gracieux du serveur
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("❌ Erreur lors de l'arrêt du serveur: %v", err)
	}

	fmt.Println("✅ Serveur arrêté proprement")
}

// createMux crée et configure le routeur HTTP
func createMux() http.Handler {
	mux := http.NewServeMux()

	// Routes publiques (pas d'authentification requise)
	mux.HandleFunc("/", handleHome)
	mux.HandleFunc("/auth/register", handlers.HandleRegister)
	mux.HandleFunc("/auth/login", handlers.HandleLogin)
	mux.HandleFunc("/auth/logout", handlers.HandleLogout)

	// Routes protégées (authentification requise)
	mux.HandleFunc("/api/tasks", middleware.AuthMiddleware(handlers.HandleTasks))
	mux.HandleFunc("/api/tasks/", middleware.AuthMiddleware(handlers.HandleTaskByID))
	mux.HandleFunc("/auth/user", middleware.AuthMiddleware(handlers.HandleGetUser))

	return mux
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"message": "Bienvenue dans l'API REST Go avec authentification! 🎉",
		"endpoints": map[string]interface{}{
			"auth": map[string]string{
				"register": "POST /auth/register",
				"login":    "POST /auth/login",
				"logout":   "POST /auth/logout",
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
				"body": map[string]string{"username": "admin", "password": "password123"},
			},
			"usage": "Utilisez le token reçu avec 'Authorization: Bearer <token>'",
		},
	}
	json.NewEncoder(w).Encode(response)
} 
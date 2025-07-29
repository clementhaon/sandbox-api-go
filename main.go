package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sandbox-api-go/handlers"
	"sandbox-api-go/middleware"
)

func main() {
	// Routes publiques (pas d'authentification requise)
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/auth/register", handlers.HandleRegister)
	http.HandleFunc("/auth/login", handlers.HandleLogin)

	// Routes protégées (authentification requise)
	http.HandleFunc("/api/tasks", middleware.AuthMiddleware(handlers.HandleTasks))
	http.HandleFunc("/api/tasks/", middleware.AuthMiddleware(handlers.HandleTaskByID))

	fmt.Println("🚀 Serveur API REST avec authentification démarré sur http://localhost:8080")
	fmt.Println("📋 Endpoints disponibles:")
	fmt.Println("  Authentification (publique):")
	fmt.Println("    POST /auth/register      - S'inscrire")
	fmt.Println("    POST /auth/login         - Se connecter")
	fmt.Println("  Tâches (authentification requise):")
	fmt.Println("    GET    /api/tasks       - Lister vos tâches")
	fmt.Println("    POST   /api/tasks       - Créer une tâche")
	fmt.Println("    GET    /api/tasks/{id}  - Obtenir une tâche")
	fmt.Println("    PUT    /api/tasks/{id}  - Mettre à jour une tâche")
	fmt.Println("    DELETE /api/tasks/{id}  - Supprimer une tâche")
	fmt.Println("💡 Utilisez 'Authorization: Bearer <token>' pour les endpoints protégés")

	log.Fatal(http.ListenAndServe(":8080", nil))
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
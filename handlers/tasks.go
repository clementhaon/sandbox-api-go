package handlers

import (
	"encoding/json"
	"net/http"
	"sandbox-api-go/middleware"
	"sandbox-api-go/models"
	"strconv"
	"strings"
)

// Stockage en mémoire pour les tâches (en prod, utilisez une vraie DB)
var tasks []models.Task
var taskNextID = 1

func init() {
	// Tâches de test pour l'utilisateur admin (ID: 1)
	tasks = []models.Task{
		{ID: 1, Title: "Apprendre Go", Description: "Créer une API REST", Completed: false, UserID: 1},
		{ID: 2, Title: "Tester l'API", Description: "Faire des requêtes HTTP", Completed: false, UserID: 1},
	}
	taskNextID = 3
}

// HandleTasks gère les requêtes GET et POST sur /api/tasks
func HandleTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		getAllUserTasks(w, r)
	case http.MethodPost:
		createTask(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Méthode non autorisée"})
	}
}

// HandleTaskByID gère les requêtes sur /api/tasks/{id}
func HandleTaskByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extraire l'ID depuis l'URL
	path := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	if path == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "ID requis"})
		return
	}

	id, err := strconv.Atoi(path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "ID invalide"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		getTaskByID(w, r, id)
	case http.MethodPut:
		updateTask(w, r, id)
	case http.MethodDelete:
		deleteTask(w, r, id)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Méthode non autorisée"})
	}
}

// getAllUserTasks retourne toutes les tâches de l'utilisateur connecté
func getAllUserTasks(w http.ResponseWriter, r *http.Request) {
	// Récupérer l'utilisateur depuis le contexte (ajouté par le middleware)
	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Erreur d'authentification"})
		return
	}

	// Filtrer les tâches pour cet utilisateur
	var userTasks []models.Task
	for _, task := range tasks {
		if task.UserID == claims.UserID {
			userTasks = append(userTasks, task)
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"tasks":    userTasks,
		"count":    len(userTasks),
		"username": claims.Username,
	})
}

// createTask crée une nouvelle tâche pour l'utilisateur connecté
func createTask(w http.ResponseWriter, r *http.Request) {
	// Récupérer l'utilisateur depuis le contexte
	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Erreur d'authentification"})
		return
	}

	var newTask models.Task
	if err := json.NewDecoder(r.Body).Decode(&newTask); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "JSON invalide"})
		return
	}

	// Validation
	if newTask.Title == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Le titre est requis"})
		return
	}

	// Assigner l'ID et l'utilisateur
	newTask.ID = taskNextID
	newTask.UserID = claims.UserID
	taskNextID++
	tasks = append(tasks, newTask)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newTask)
}

// getTaskByID retourne une tâche spécifique si elle appartient à l'utilisateur
func getTaskByID(w http.ResponseWriter, r *http.Request, id int) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Erreur d'authentification"})
		return
	}

	for _, task := range tasks {
		if task.ID == id && task.UserID == claims.UserID {
			json.NewEncoder(w).Encode(task)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "Tâche non trouvée"})
}

// updateTask met à jour une tâche si elle appartient à l'utilisateur
func updateTask(w http.ResponseWriter, r *http.Request, id int) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Erreur d'authentification"})
		return
	}

	for i, task := range tasks {
		if task.ID == id && task.UserID == claims.UserID {
			var updatedTask models.Task
			if err := json.NewDecoder(r.Body).Decode(&updatedTask); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "JSON invalide"})
				return
			}

			// Conserver l'ID et l'UserID
			updatedTask.ID = id
			updatedTask.UserID = claims.UserID
			tasks[i] = updatedTask

			json.NewEncoder(w).Encode(updatedTask)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "Tâche non trouvée"})
}

// deleteTask supprime une tâche si elle appartient à l'utilisateur
func deleteTask(w http.ResponseWriter, r *http.Request, id int) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Erreur d'authentification"})
		return
	}

	for i, task := range tasks {
		if task.ID == id && task.UserID == claims.UserID {
			// Supprimer la tâche du slice
			tasks = append(tasks[:i], tasks[i+1:]...)
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "Tâche non trouvée"})
} 
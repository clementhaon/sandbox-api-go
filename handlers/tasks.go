package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sandbox-api-go/database"
	"sandbox-api-go/middleware"
	"sandbox-api-go/models"
	"strconv"
	"strings"
)

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

	// Récupérer les tâches depuis la base de données
	rows, err := database.DB.Query("SELECT id, title, description, completed, user_id FROM tasks WHERE user_id = $1 ORDER BY created_at DESC", claims.UserID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Erreur lors de la récupération des tâches"})
		return
	}
	defer rows.Close()

	var userTasks []models.Task
	for rows.Next() {
		var task models.Task
		if err := rows.Scan(&task.ID, &task.Title, &task.Description, &task.Completed, &task.UserID); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Erreur lors de la lecture des tâches"})
			return
		}
		userTasks = append(userTasks, task)
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

	// Créer la tâche dans la base de données
	var createdTask models.Task
	err := database.DB.QueryRow(
		"INSERT INTO tasks (title, description, completed, user_id) VALUES ($1, $2, $3, $4) RETURNING id, title, description, completed, user_id",
		newTask.Title, newTask.Description, newTask.Completed, claims.UserID,
	).Scan(&createdTask.ID, &createdTask.Title, &createdTask.Description, &createdTask.Completed, &createdTask.UserID)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Erreur lors de la création de la tâche"})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdTask)
}

// getTaskByID retourne une tâche spécifique si elle appartient à l'utilisateur
func getTaskByID(w http.ResponseWriter, r *http.Request, id int) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Erreur d'authentification"})
		return
	}

	var task models.Task
	err := database.DB.QueryRow(
		"SELECT id, title, description, completed, user_id FROM tasks WHERE id = $1 AND user_id = $2",
		id, claims.UserID,
	).Scan(&task.ID, &task.Title, &task.Description, &task.Completed, &task.UserID)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Tâche non trouvée"})
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Erreur lors de la récupération de la tâche"})
		return
	}

	json.NewEncoder(w).Encode(task)
}

// updateTask met à jour une tâche si elle appartient à l'utilisateur
func updateTask(w http.ResponseWriter, r *http.Request, id int) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Erreur d'authentification"})
		return
	}

	var updatedTask models.Task
	if err := json.NewDecoder(r.Body).Decode(&updatedTask); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "JSON invalide"})
		return
	}

	// Mettre à jour la tâche dans la base de données
	var result models.Task
	err := database.DB.QueryRow(
		"UPDATE tasks SET title = $1, description = $2, completed = $3, updated_at = CURRENT_TIMESTAMP WHERE id = $4 AND user_id = $5 RETURNING id, title, description, completed, user_id",
		updatedTask.Title, updatedTask.Description, updatedTask.Completed, id, claims.UserID,
	).Scan(&result.ID, &result.Title, &result.Description, &result.Completed, &result.UserID)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Tâche non trouvée"})
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Erreur lors de la mise à jour de la tâche"})
		return
	}

	json.NewEncoder(w).Encode(result)
}

// deleteTask supprime une tâche si elle appartient à l'utilisateur
func deleteTask(w http.ResponseWriter, r *http.Request, id int) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Erreur d'authentification"})
		return
	}

	// Supprimer la tâche de la base de données
	result, err := database.DB.Exec("DELETE FROM tasks WHERE id = $1 AND user_id = $2", id, claims.UserID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Erreur lors de la suppression de la tâche"})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Erreur lors de la vérification de la suppression"})
		return
	}

	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Tâche non trouvée"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
} 
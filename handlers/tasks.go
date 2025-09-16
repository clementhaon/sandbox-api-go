package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sandbox-api-go/database"
	"sandbox-api-go/middleware"
	"sandbox-api-go/models"
	"sandbox-api-go/errors"
	"sandbox-api-go/logger"
	"sandbox-api-go/metrics"
	"sandbox-api-go/validation"
	"strconv"
	"strings"
	"time"
)

// HandleTasks gère les requêtes GET et POST sur /api/tasks
func HandleTasks(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		return getAllUserTasks(w, r)
	case http.MethodPost:
		return createTask(w, r)
	default:
		return errors.NewMethodNotAllowedError()
	}
}

// HandleTaskByID gère les requêtes sur /api/tasks/{id}
func HandleTaskByID(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	// Extraire l'ID depuis l'URL
	path := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	if path == "" {
		return errors.NewMissingFieldError("task_id")
	}

	id, err := strconv.Atoi(path)
	if err != nil {
		logger.WarnContext(r.Context(), "Invalid task ID format", map[string]interface{}{
			"provided_id": path,
			"error":       err.Error(),
		})
		return errors.NewInvalidFormatError("task_id", "integer")
	}

	switch r.Method {
	case http.MethodGet:
		return getTaskByID(w, r, id)
	case http.MethodPut:
		return updateTask(w, r, id)
	case http.MethodDelete:
		return deleteTask(w, r, id)
	default:
		return errors.NewMethodNotAllowedError()
	}
}

// getAllUserTasks retourne toutes les tâches de l'utilisateur connecté
func getAllUserTasks(w http.ResponseWriter, r *http.Request) error {
	// Récupérer l'utilisateur depuis le contexte (ajouté par le middleware)
	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		logger.ErrorContext(r.Context(), "Missing user context in authenticated request", nil)
		return errors.NewInternalError().WithDetails(map[string]interface{}{
			"issue": "user_context_missing",
		})
	}

	// Récupérer les tâches depuis la base de données
	startTime := time.Now()
	rows, err := database.DB.Query("SELECT id, title, description, completed, user_id FROM tasks WHERE user_id = $1 ORDER BY created_at DESC", claims.UserID)
	logger.LogDatabaseOperation(r.Context(), "SELECT", "tasks", time.Since(startTime), err)
	metrics.RecordDatabaseOperation("SELECT", "tasks", time.Since(startTime), err)
	
	if err != nil {
		logger.ErrorContext(r.Context(), "Error fetching user tasks", err, map[string]interface{}{
			"user_id": claims.UserID,
		})
		return errors.NewDatabaseError().WithCause(err)
	}
	defer rows.Close()

	var userTasks []models.Task
	for rows.Next() {
		var task models.Task
		if err := rows.Scan(&task.ID, &task.Title, &task.Description, &task.Completed, &task.UserID); err != nil {
			logger.ErrorContext(r.Context(), "Error scanning task row", err)
			return errors.NewDatabaseError().WithCause(err)
		}
		userTasks = append(userTasks, task)
	}

	logger.DebugContext(r.Context(), "Retrieved user tasks", map[string]interface{}{
		"user_id":    claims.UserID,
		"task_count": len(userTasks),
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"tasks":    userTasks,
		"count":    len(userTasks),
		"username": claims.Username,
	})
	return nil
}

// createTask crée une nouvelle tâche pour l'utilisateur connecté
func createTask(w http.ResponseWriter, r *http.Request) error {
	// Récupérer l'utilisateur depuis le contexte
	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		logger.ErrorContext(r.Context(), "Missing user context in authenticated request", nil)
		return errors.NewInternalError().WithDetails(map[string]interface{}{
			"issue": "user_context_missing",
		})
	}

	var newTask models.Task
	if err := json.NewDecoder(r.Body).Decode(&newTask); err != nil {
		logger.WarnContext(r.Context(), "Invalid JSON in create task request", map[string]interface{}{
			"error": err.Error(),
		})
		return errors.NewInvalidJSONError()
	}

	// Validation
	if validationErr := validation.ValidateTaskInput(newTask.Title, newTask.Description); validationErr != nil {
		return validationErr
	}

	// Créer la tâche dans la base de données
	var createdTask models.Task
	startTime := time.Now()
	err := database.DB.QueryRow(
		"INSERT INTO tasks (title, description, completed, user_id) VALUES ($1, $2, $3, $4) RETURNING id, title, description, completed, user_id",
		newTask.Title, newTask.Description, newTask.Completed, claims.UserID,
	).Scan(&createdTask.ID, &createdTask.Title, &createdTask.Description, &createdTask.Completed, &createdTask.UserID)
	logger.LogDatabaseOperation(r.Context(), "INSERT", "tasks", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(r.Context(), "Error creating task", err, map[string]interface{}{
			"user_id": claims.UserID,
			"title":   newTask.Title,
		})
		return errors.NewDatabaseError().WithCause(err)
	}

	logger.InfoContext(r.Context(), "Task created successfully", map[string]interface{}{
		"task_id": createdTask.ID,
		"user_id": claims.UserID,
		"title":   createdTask.Title,
	})

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdTask)
	return nil
}

// getTaskByID retourne une tâche spécifique si elle appartient à l'utilisateur
func getTaskByID(w http.ResponseWriter, r *http.Request, id int) error {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		logger.ErrorContext(r.Context(), "Missing user context in authenticated request", nil)
		return errors.NewInternalError().WithDetails(map[string]interface{}{
			"issue": "user_context_missing",
		})
	}

	var task models.Task
	startTime := time.Now()
	err := database.DB.QueryRow(
		"SELECT id, title, description, completed, user_id FROM tasks WHERE id = $1 AND user_id = $2",
		id, claims.UserID,
	).Scan(&task.ID, &task.Title, &task.Description, &task.Completed, &task.UserID)
	logger.LogDatabaseOperation(r.Context(), "SELECT", "tasks", time.Since(startTime), err)
	metrics.RecordDatabaseOperation("SELECT", "tasks", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		logger.WarnContext(r.Context(), "Task not found or access denied", map[string]interface{}{
			"task_id": id,
			"user_id": claims.UserID,
		})
		return errors.NewNotFoundError("Task")
	} else if err != nil {
		logger.ErrorContext(r.Context(), "Error fetching task", err, map[string]interface{}{
			"task_id": id,
			"user_id": claims.UserID,
		})
		return errors.NewDatabaseError().WithCause(err)
	}

	json.NewEncoder(w).Encode(task)
	return nil
}

// updateTask met à jour une tâche si elle appartient à l'utilisateur
func updateTask(w http.ResponseWriter, r *http.Request, id int) error {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		logger.ErrorContext(r.Context(), "Missing user context in authenticated request", nil)
		return errors.NewInternalError().WithDetails(map[string]interface{}{
			"issue": "user_context_missing",
		})
	}

	var updatedTask models.Task
	if err := json.NewDecoder(r.Body).Decode(&updatedTask); err != nil {
		logger.WarnContext(r.Context(), "Invalid JSON in update task request", map[string]interface{}{
			"error": err.Error(),
		})
		return errors.NewInvalidJSONError()
	}

	// Validation
	if validationErr := validation.ValidateTaskInput(updatedTask.Title, updatedTask.Description); validationErr != nil {
		return validationErr
	}

	// Mettre à jour la tâche dans la base de données
	var result models.Task
	startTime := time.Now()
	err := database.DB.QueryRow(
		"UPDATE tasks SET title = $1, description = $2, completed = $3, updated_at = CURRENT_TIMESTAMP WHERE id = $4 AND user_id = $5 RETURNING id, title, description, completed, user_id",
		updatedTask.Title, updatedTask.Description, updatedTask.Completed, id, claims.UserID,
	).Scan(&result.ID, &result.Title, &result.Description, &result.Completed, &result.UserID)
	logger.LogDatabaseOperation(r.Context(), "UPDATE", "tasks", time.Since(startTime), err)
	metrics.RecordDatabaseOperation("UPDATE", "tasks", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		logger.WarnContext(r.Context(), "Task not found for update or access denied", map[string]interface{}{
			"task_id": id,
			"user_id": claims.UserID,
		})
		return errors.NewNotFoundError("Task")
	} else if err != nil {
		logger.ErrorContext(r.Context(), "Error updating task", err, map[string]interface{}{
			"task_id": id,
			"user_id": claims.UserID,
		})
		return errors.NewDatabaseError().WithCause(err)
	}

	logger.InfoContext(r.Context(), "Task updated successfully", map[string]interface{}{
		"task_id": result.ID,
		"user_id": claims.UserID,
		"title":   result.Title,
	})

	json.NewEncoder(w).Encode(result)
	return nil
}

// deleteTask supprime une tâche si elle appartient à l'utilisateur
func deleteTask(w http.ResponseWriter, r *http.Request, id int) error {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		logger.ErrorContext(r.Context(), "Missing user context in authenticated request", nil)
		return errors.NewInternalError().WithDetails(map[string]interface{}{
			"issue": "user_context_missing",
		})
	}

	// Supprimer la tâche de la base de données
	startTime := time.Now()
	result, err := database.DB.Exec("DELETE FROM tasks WHERE id = $1 AND user_id = $2", id, claims.UserID)
	logger.LogDatabaseOperation(r.Context(), "DELETE", "tasks", time.Since(startTime), err)
	metrics.RecordDatabaseOperation("DELETE", "tasks", time.Since(startTime), err)
	
	if err != nil {
		logger.ErrorContext(r.Context(), "Error deleting task", err, map[string]interface{}{
			"task_id": id,
			"user_id": claims.UserID,
		})
		return errors.NewDatabaseError().WithCause(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		logger.ErrorContext(r.Context(), "Error checking deletion result", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	if rowsAffected == 0 {
		logger.WarnContext(r.Context(), "Task not found for deletion or access denied", map[string]interface{}{
			"task_id": id,
			"user_id": claims.UserID,
		})
		return errors.NewNotFoundError("Task")
	}

	logger.InfoContext(r.Context(), "Task deleted successfully", map[string]interface{}{
		"task_id": id,
		"user_id": claims.UserID,
	})

	w.WriteHeader(http.StatusNoContent)
	return nil
} 
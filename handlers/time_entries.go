package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/clementhaon/sandbox-api-go/database"
	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/logger"
	"github.com/clementhaon/sandbox-api-go/middleware"
	"github.com/clementhaon/sandbox-api-go/models"
)

// ListTimeEntries handles GET /time-entries?taskId=
func ListTimeEntries(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	taskIDStr := r.URL.Query().Get("taskId")
	if taskIDStr == "" {
		return errors.NewBadRequestError("taskId query parameter is required")
	}

	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil {
		return errors.NewBadRequestError("Invalid taskId")
	}

	startTime := time.Now()
	rows, err := database.DB.Query(`
		SELECT id, task_id, user_id, start_time, end_time, duration, description, created_at
		FROM time_entries
		WHERE task_id = $1
		ORDER BY start_time DESC
	`, taskID)
	logger.LogDatabaseOperation(r.Context(), "SELECT", "time_entries", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(r.Context(), "Error querying time entries", err)
		return errors.NewDatabaseError().WithCause(err)
	}
	defer rows.Close()

	entries := []models.TimeEntry{}
	for rows.Next() {
		var e models.TimeEntry
		err := rows.Scan(&e.ID, &e.TaskID, &e.UserID, &e.StartTime, &e.EndTime, &e.Duration, &e.Description, &e.CreatedAt)
		if err != nil {
			logger.ErrorContext(r.Context(), "Error scanning time entry row", err)
			return errors.NewDatabaseError().WithCause(err)
		}
		entries = append(entries, e)
	}

	json.NewEncoder(w).Encode(entries)
	return nil
}

// CreateTimeEntry handles POST /time-entries
func CreateTimeEntry(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		return errors.NewInternalError().WithDetails(map[string]interface{}{
			"issue": "user_context_missing",
		})
	}

	var req models.CreateTimeEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewInvalidJSONError()
	}

	if req.TaskID == 0 {
		return errors.NewBadRequestError("taskId is required")
	}
	if req.StartTime.IsZero() {
		return errors.NewBadRequestError("startTime is required")
	}
	if req.Duration <= 0 {
		return errors.NewBadRequestError("duration must be positive")
	}

	// Check if task exists
	var existingTaskID int
	startTime := time.Now()
	err := database.DB.QueryRow("SELECT id FROM tasks WHERE id = $1", req.TaskID).Scan(&existingTaskID)
	logger.LogDatabaseOperation(r.Context(), "SELECT", "tasks", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return errors.NewNotFoundError("Task not found")
	} else if err != nil {
		logger.ErrorContext(r.Context(), "Error checking task", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	// Insert time entry
	var e models.TimeEntry
	startTime = time.Now()
	err = database.DB.QueryRow(`
		INSERT INTO time_entries (task_id, user_id, start_time, end_time, duration, description)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, task_id, user_id, start_time, end_time, duration, description, created_at
	`, req.TaskID, claims.UserID, req.StartTime, req.EndTime, req.Duration, req.Description).
		Scan(&e.ID, &e.TaskID, &e.UserID, &e.StartTime, &e.EndTime, &e.Duration, &e.Description, &e.CreatedAt)
	logger.LogDatabaseOperation(r.Context(), "INSERT", "time_entries", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(r.Context(), "Error creating time entry", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	// Update task's tracked_time (convert seconds to minutes and add)
	durationMinutes := req.Duration / 60
	startTime = time.Now()
	_, err = database.DB.Exec(`UPDATE tasks SET tracked_time = tracked_time + $1, updated_at = NOW() WHERE id = $2`, durationMinutes, req.TaskID)
	logger.LogDatabaseOperation(r.Context(), "UPDATE", "tasks", time.Since(startTime), err)

	if err != nil {
		logger.WarnContext(r.Context(), "Error updating task tracked_time", map[string]interface{}{
			"error":   err.Error(),
			"task_id": req.TaskID,
		})
	}

	logger.InfoContext(r.Context(), "Time entry created", map[string]interface{}{
		"entry_id": e.ID,
		"task_id":  e.TaskID,
		"duration": e.Duration,
		"user_id":  claims.UserID,
	})

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(e)
	return nil
}

// DeleteTimeEntry handles DELETE /time-entries/{id}
func DeleteTimeEntry(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return errors.NewBadRequestError("Invalid time entry ID")
	}

	// Get the time entry to know duration and task_id for updating tracked_time
	var taskID, duration int
	startTime := time.Now()
	err = database.DB.QueryRow("SELECT task_id, duration FROM time_entries WHERE id = $1", id).Scan(&taskID, &duration)
	logger.LogDatabaseOperation(r.Context(), "SELECT", "time_entries", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return errors.NewNotFoundError("Time entry not found")
	} else if err != nil {
		logger.ErrorContext(r.Context(), "Error fetching time entry", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	// Delete the time entry
	startTime = time.Now()
	result, err := database.DB.Exec("DELETE FROM time_entries WHERE id = $1", id)
	logger.LogDatabaseOperation(r.Context(), "DELETE", "time_entries", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(r.Context(), "Error deleting time entry", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.NewNotFoundError("Time entry not found")
	}

	// Update task's tracked_time (subtract the duration in minutes)
	durationMinutes := duration / 60
	startTime = time.Now()
	_, err = database.DB.Exec(`UPDATE tasks SET tracked_time = GREATEST(0, tracked_time - $1), updated_at = NOW() WHERE id = $2`, durationMinutes, taskID)
	logger.LogDatabaseOperation(r.Context(), "UPDATE", "tasks", time.Since(startTime), err)

	if err != nil {
		logger.WarnContext(r.Context(), "Error updating task tracked_time after delete", map[string]interface{}{
			"error":   err.Error(),
			"task_id": taskID,
		})
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

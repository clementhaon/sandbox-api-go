package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"sandbox-api-go/database"
	"sandbox-api-go/errors"
	"sandbox-api-go/logger"
	"sandbox-api-go/middleware"
	"sandbox-api-go/models"

	"github.com/lib/pq"
)

// GetBoard handles GET /tasks/board - returns all columns and tasks for the board
func GetBoard(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	// Get columns
	startTime := time.Now()
	colRows, err := database.DB.Query(`SELECT id, title, "order", color, created_at, updated_at FROM columns ORDER BY "order" ASC`)
	logger.LogDatabaseOperation(r.Context(), "SELECT", "columns", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(r.Context(), "Error querying columns", err)
		return errors.NewDatabaseError().WithCause(err)
	}
	defer colRows.Close()

	columns := []models.Column{}
	for colRows.Next() {
		var c models.Column
		if err := colRows.Scan(&c.ID, &c.Title, &c.Order, &c.Color, &c.CreatedAt, &c.UpdatedAt); err != nil {
			logger.ErrorContext(r.Context(), "Error scanning column row", err)
			return errors.NewDatabaseError().WithCause(err)
		}
		columns = append(columns, c)
	}

	// Get tasks with assignee info
	startTime = time.Now()
	taskRows, err := database.DB.Query(`
		SELECT t.id, t.title, t.description, t.column_id, t."order", t.priority,
			t.assignee_id, t.deadline, t.estimated_time, t.tracked_time, t.tags,
			t.created_by, t.user_id, t.created_at, t.updated_at,
			u.id, u.username, u.avatar_url
		FROM tasks t
		LEFT JOIN users u ON t.assignee_id = u.id
		ORDER BY t.column_id, t."order" ASC
	`)
	logger.LogDatabaseOperation(r.Context(), "SELECT", "tasks", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(r.Context(), "Error querying tasks", err)
		return errors.NewDatabaseError().WithCause(err)
	}
	defer taskRows.Close()

	tasks := []models.Task{}
	for taskRows.Next() {
		var t models.TaskDB
		var assigneeID, assigneeUsername sql.NullInt64
		var assigneeUsernameStr, assigneeAvatarURL sql.NullString

		err := taskRows.Scan(
			&t.ID, &t.Title, &t.Description, &t.ColumnID, &t.Order, &t.Priority,
			&t.AssigneeID, &t.Deadline, &t.EstimatedTime, &t.TrackedTime, &t.Tags,
			&t.CreatedBy, &t.UserID, &t.CreatedAt, &t.UpdatedAt,
			&assigneeID, &assigneeUsernameStr, &assigneeAvatarURL,
		)
		if err != nil {
			logger.ErrorContext(r.Context(), "Error scanning task row", err)
			return errors.NewDatabaseError().WithCause(err)
		}

		task := t.ToTask()
		if assigneeID.Valid {
			task.Assignee = &models.UserBrief{
				ID:       int(assigneeID.Int64),
				Username: assigneeUsernameStr.String,
			}
			if assigneeAvatarURL.Valid {
				task.Assignee.AvatarURL = assigneeAvatarURL.String
			}
		}
		_ = assigneeUsername // suppress unused warning
		tasks = append(tasks, task)
	}

	response := models.BoardResponse{
		Columns: columns,
		Tasks:   tasks,
	}

	json.NewEncoder(w).Encode(response)
	return nil
}

// ListTasks handles GET /tasks - list tasks optionally filtered by column
func ListTasks(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	columnIDStr := r.URL.Query().Get("columnId")

	var query string
	var args []interface{}

	if columnIDStr != "" {
		columnID, err := strconv.Atoi(columnIDStr)
		if err != nil {
			return errors.NewBadRequestError("Invalid columnId")
		}
		query = `
			SELECT t.id, t.title, t.description, t.column_id, t."order", t.priority,
				t.assignee_id, t.deadline, t.estimated_time, t.tracked_time, t.tags,
				t.created_by, t.user_id, t.created_at, t.updated_at,
				u.id, u.username, u.avatar_url
			FROM tasks t
			LEFT JOIN users u ON t.assignee_id = u.id
			WHERE t.column_id = $1
			ORDER BY t."order" ASC
		`
		args = append(args, columnID)
	} else {
		query = `
			SELECT t.id, t.title, t.description, t.column_id, t."order", t.priority,
				t.assignee_id, t.deadline, t.estimated_time, t.tracked_time, t.tags,
				t.created_by, t.user_id, t.created_at, t.updated_at,
				u.id, u.username, u.avatar_url
			FROM tasks t
			LEFT JOIN users u ON t.assignee_id = u.id
			ORDER BY t.column_id, t."order" ASC
		`
	}

	startTime := time.Now()
	rows, err := database.DB.Query(query, args...)
	logger.LogDatabaseOperation(r.Context(), "SELECT", "tasks", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(r.Context(), "Error querying tasks", err)
		return errors.NewDatabaseError().WithCause(err)
	}
	defer rows.Close()

	tasks := []models.Task{}
	for rows.Next() {
		var t models.TaskDB
		var assigneeID sql.NullInt64
		var assigneeUsername, assigneeAvatarURL sql.NullString

		err := rows.Scan(
			&t.ID, &t.Title, &t.Description, &t.ColumnID, &t.Order, &t.Priority,
			&t.AssigneeID, &t.Deadline, &t.EstimatedTime, &t.TrackedTime, &t.Tags,
			&t.CreatedBy, &t.UserID, &t.CreatedAt, &t.UpdatedAt,
			&assigneeID, &assigneeUsername, &assigneeAvatarURL,
		)
		if err != nil {
			logger.ErrorContext(r.Context(), "Error scanning task row", err)
			return errors.NewDatabaseError().WithCause(err)
		}

		task := t.ToTask()
		if assigneeID.Valid {
			task.Assignee = &models.UserBrief{
				ID:       int(assigneeID.Int64),
				Username: assigneeUsername.String,
			}
			if assigneeAvatarURL.Valid {
				task.Assignee.AvatarURL = assigneeAvatarURL.String
			}
		}
		tasks = append(tasks, task)
	}

	json.NewEncoder(w).Encode(tasks)
	return nil
}

// GetTask handles GET /tasks/{id}
func GetTask(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return errors.NewBadRequestError("Invalid task ID")
	}

	var t models.TaskDB
	var assigneeID sql.NullInt64
	var assigneeUsername, assigneeAvatarURL sql.NullString

	startTime := time.Now()
	err = database.DB.QueryRow(`
		SELECT t.id, t.title, t.description, t.column_id, t."order", t.priority,
			t.assignee_id, t.deadline, t.estimated_time, t.tracked_time, t.tags,
			t.created_by, t.user_id, t.created_at, t.updated_at,
			u.id, u.username, u.avatar_url
		FROM tasks t
		LEFT JOIN users u ON t.assignee_id = u.id
		WHERE t.id = $1
	`, id).Scan(
		&t.ID, &t.Title, &t.Description, &t.ColumnID, &t.Order, &t.Priority,
		&t.AssigneeID, &t.Deadline, &t.EstimatedTime, &t.TrackedTime, &t.Tags,
		&t.CreatedBy, &t.UserID, &t.CreatedAt, &t.UpdatedAt,
		&assigneeID, &assigneeUsername, &assigneeAvatarURL,
	)
	logger.LogDatabaseOperation(r.Context(), "SELECT", "tasks", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return errors.NewNotFoundError("Task not found")
	} else if err != nil {
		logger.ErrorContext(r.Context(), "Error fetching task", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	task := t.ToTask()
	if assigneeID.Valid {
		task.Assignee = &models.UserBrief{
			ID:       int(assigneeID.Int64),
			Username: assigneeUsername.String,
		}
		if assigneeAvatarURL.Valid {
			task.Assignee.AvatarURL = assigneeAvatarURL.String
		}
	}

	json.NewEncoder(w).Encode(task)
	return nil
}

// CreateTask handles POST /tasks
func CreateTask(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		return errors.NewInternalError().WithDetails(map[string]interface{}{
			"issue": "user_context_missing",
		})
	}

	var req models.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewInvalidJSONError()
	}

	if req.Title == "" {
		return errors.NewBadRequestError("Title is required")
	}
	if req.ColumnID == 0 {
		return errors.NewBadRequestError("ColumnID is required")
	}

	// Set defaults
	if req.Priority == "" {
		req.Priority = models.PriorityMedium
	}
	if req.Tags == nil {
		req.Tags = []string{}
	}

	// Get max order in column
	var maxOrder int
	startTime := time.Now()
	err := database.DB.QueryRow(`SELECT COALESCE(MAX("order"), -1) FROM tasks WHERE column_id = $1`, req.ColumnID).Scan(&maxOrder)
	logger.LogDatabaseOperation(r.Context(), "SELECT MAX", "tasks", time.Since(startTime), err)
	if err != nil {
		logger.ErrorContext(r.Context(), "Error getting max order", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	// Insert task
	var t models.TaskDB
	startTime = time.Now()
	err = database.DB.QueryRow(`
		INSERT INTO tasks (title, description, column_id, "order", priority, assignee_id, deadline, estimated_time, tags, created_by, user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
		RETURNING id, title, description, column_id, "order", priority, assignee_id, deadline, estimated_time, tracked_time, tags, created_by, user_id, created_at, updated_at
	`,
		req.Title, req.Description, req.ColumnID, maxOrder+1, req.Priority,
		req.AssigneeID, req.Deadline, req.EstimatedTime, pq.Array(req.Tags), claims.UserID,
	).Scan(
		&t.ID, &t.Title, &t.Description, &t.ColumnID, &t.Order, &t.Priority,
		&t.AssigneeID, &t.Deadline, &t.EstimatedTime, &t.TrackedTime, &t.Tags,
		&t.CreatedBy, &t.UserID, &t.CreatedAt, &t.UpdatedAt,
	)
	logger.LogDatabaseOperation(r.Context(), "INSERT", "tasks", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(r.Context(), "Error creating task", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	task := t.ToTask()

	// Fetch assignee if set
	if t.AssigneeID != nil {
		var assignee models.UserBrief
		var avatarURL sql.NullString
		err = database.DB.QueryRow(`SELECT id, username, avatar_url FROM users WHERE id = $1`, *t.AssigneeID).
			Scan(&assignee.ID, &assignee.Username, &avatarURL)
		if err == nil {
			if avatarURL.Valid {
				assignee.AvatarURL = avatarURL.String
			}
			task.Assignee = &assignee
		}
	}

	logger.InfoContext(r.Context(), "Task created", map[string]interface{}{
		"task_id":   task.ID,
		"column_id": task.ColumnID,
		"user_id":   claims.UserID,
	})

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
	return nil
}

// UpdateTask handles PUT /tasks/{id}
func UpdateTask(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return errors.NewBadRequestError("Invalid task ID")
	}

	var req models.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewInvalidJSONError()
	}

	// Check if task exists
	var existingID int
	startTime := time.Now()
	err = database.DB.QueryRow("SELECT id FROM tasks WHERE id = $1", id).Scan(&existingID)
	logger.LogDatabaseOperation(r.Context(), "SELECT", "tasks", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return errors.NewNotFoundError("Task not found")
	} else if err != nil {
		logger.ErrorContext(r.Context(), "Error checking task", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	// Update task
	var t models.TaskDB
	startTime = time.Now()
	err = database.DB.QueryRow(`
		UPDATE tasks SET
			title = COALESCE(NULLIF($1, ''), title),
			description = COALESCE($2, description),
			column_id = CASE WHEN $3 > 0 THEN $3 ELSE column_id END,
			priority = COALESCE(NULLIF($4, ''), priority),
			assignee_id = $5,
			deadline = $6,
			estimated_time = CASE WHEN $7 > 0 THEN $7 ELSE estimated_time END,
			tags = COALESCE($8, tags),
			updated_at = NOW()
		WHERE id = $9
		RETURNING id, title, description, column_id, "order", priority, assignee_id, deadline, estimated_time, tracked_time, tags, created_by, user_id, created_at, updated_at
	`,
		req.Title, req.Description, req.ColumnID, req.Priority,
		req.AssigneeID, req.Deadline, req.EstimatedTime, pq.Array(req.Tags), id,
	).Scan(
		&t.ID, &t.Title, &t.Description, &t.ColumnID, &t.Order, &t.Priority,
		&t.AssigneeID, &t.Deadline, &t.EstimatedTime, &t.TrackedTime, &t.Tags,
		&t.CreatedBy, &t.UserID, &t.CreatedAt, &t.UpdatedAt,
	)
	logger.LogDatabaseOperation(r.Context(), "UPDATE", "tasks", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(r.Context(), "Error updating task", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	task := t.ToTask()

	// Fetch assignee if set
	if t.AssigneeID != nil {
		var assignee models.UserBrief
		var avatarURL sql.NullString
		err = database.DB.QueryRow(`SELECT id, username, avatar_url FROM users WHERE id = $1`, *t.AssigneeID).
			Scan(&assignee.ID, &assignee.Username, &avatarURL)
		if err == nil {
			if avatarURL.Valid {
				assignee.AvatarURL = avatarURL.String
			}
			task.Assignee = &assignee
		}
	}

	json.NewEncoder(w).Encode(task)
	return nil
}

// MoveTask handles PATCH /tasks/{id}/move
func MoveTask(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return errors.NewBadRequestError("Invalid task ID")
	}

	var req models.MoveTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewInvalidJSONError()
	}

	// Update task position
	var t models.TaskDB
	startTime := time.Now()
	err = database.DB.QueryRow(`
		UPDATE tasks SET column_id = $1, "order" = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING id, title, description, column_id, "order", priority, assignee_id, deadline, estimated_time, tracked_time, tags, created_by, user_id, created_at, updated_at
	`, req.ColumnID, req.Order, id).Scan(
		&t.ID, &t.Title, &t.Description, &t.ColumnID, &t.Order, &t.Priority,
		&t.AssigneeID, &t.Deadline, &t.EstimatedTime, &t.TrackedTime, &t.Tags,
		&t.CreatedBy, &t.UserID, &t.CreatedAt, &t.UpdatedAt,
	)
	logger.LogDatabaseOperation(r.Context(), "UPDATE", "tasks", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return errors.NewNotFoundError("Task not found")
	} else if err != nil {
		logger.ErrorContext(r.Context(), "Error moving task", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	task := t.ToTask()

	// Fetch assignee if set
	if t.AssigneeID != nil {
		var assignee models.UserBrief
		var avatarURL sql.NullString
		err = database.DB.QueryRow(`SELECT id, username, avatar_url FROM users WHERE id = $1`, *t.AssigneeID).
			Scan(&assignee.ID, &assignee.Username, &avatarURL)
		if err == nil {
			if avatarURL.Valid {
				assignee.AvatarURL = avatarURL.String
			}
			task.Assignee = &assignee
		}
	}

	json.NewEncoder(w).Encode(task)
	return nil
}

// ReorderTasks handles PATCH /tasks/reorder
func ReorderTasks(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	var req models.ReorderTasksRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewInvalidJSONError()
	}

	if req.ColumnID == 0 {
		return errors.NewBadRequestError("columnId is required")
	}
	if len(req.TaskIDs) == 0 {
		return errors.NewBadRequestError("taskIds is required")
	}

	// Update each task's order
	for i, taskID := range req.TaskIDs {
		startTime := time.Now()
		result, err := database.DB.Exec(`UPDATE tasks SET "order" = $1, updated_at = NOW() WHERE id = $2 AND column_id = $3`, i, taskID, req.ColumnID)
		logger.LogDatabaseOperation(r.Context(), "UPDATE", "tasks", time.Since(startTime), err)

		if err != nil {
			logger.ErrorContext(r.Context(), "Error updating task order", err)
			return errors.NewDatabaseError().WithCause(err)
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			return errors.NewNotFoundError("Task not found in column: " + strconv.Itoa(taskID))
		}
	}

	// Return updated tasks for the column
	startTime := time.Now()
	rows, err := database.DB.Query(`
		SELECT t.id, t.title, t.description, t.column_id, t."order", t.priority,
			t.assignee_id, t.deadline, t.estimated_time, t.tracked_time, t.tags,
			t.created_by, t.user_id, t.created_at, t.updated_at,
			u.id, u.username, u.avatar_url
		FROM tasks t
		LEFT JOIN users u ON t.assignee_id = u.id
		WHERE t.column_id = $1
		ORDER BY t."order" ASC
	`, req.ColumnID)
	logger.LogDatabaseOperation(r.Context(), "SELECT", "tasks", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(r.Context(), "Error querying tasks", err)
		return errors.NewDatabaseError().WithCause(err)
	}
	defer rows.Close()

	tasks := []models.Task{}
	for rows.Next() {
		var t models.TaskDB
		var assigneeID sql.NullInt64
		var assigneeUsername, assigneeAvatarURL sql.NullString

		err := rows.Scan(
			&t.ID, &t.Title, &t.Description, &t.ColumnID, &t.Order, &t.Priority,
			&t.AssigneeID, &t.Deadline, &t.EstimatedTime, &t.TrackedTime, &t.Tags,
			&t.CreatedBy, &t.UserID, &t.CreatedAt, &t.UpdatedAt,
			&assigneeID, &assigneeUsername, &assigneeAvatarURL,
		)
		if err != nil {
			logger.ErrorContext(r.Context(), "Error scanning task row", err)
			return errors.NewDatabaseError().WithCause(err)
		}

		task := t.ToTask()
		if assigneeID.Valid {
			task.Assignee = &models.UserBrief{
				ID:       int(assigneeID.Int64),
				Username: assigneeUsername.String,
			}
			if assigneeAvatarURL.Valid {
				task.Assignee.AvatarURL = assigneeAvatarURL.String
			}
		}
		tasks = append(tasks, task)
	}

	json.NewEncoder(w).Encode(tasks)
	return nil
}

// DeleteTask handles DELETE /tasks/{id}
func DeleteTask(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return errors.NewBadRequestError("Invalid task ID")
	}

	startTime := time.Now()
	result, err := database.DB.Exec("DELETE FROM tasks WHERE id = $1", id)
	logger.LogDatabaseOperation(r.Context(), "DELETE", "tasks", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(r.Context(), "Error deleting task", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.NewNotFoundError("Task not found")
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

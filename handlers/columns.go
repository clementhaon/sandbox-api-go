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
	"github.com/clementhaon/sandbox-api-go/models"
)

// ListColumns handles GET /columns - returns ordered list of columns
func ListColumns(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	startTime := time.Now()
	rows, err := database.DB.Query(`SELECT id, title, "order", color, created_at, updated_at FROM columns ORDER BY "order" ASC`)
	logger.LogDatabaseOperation(r.Context(), "SELECT", "columns", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(r.Context(), "Error querying columns", err)
		return errors.NewDatabaseError().WithCause(err)
	}
	defer rows.Close()

	columns := []models.Column{}
	for rows.Next() {
		var c models.Column
		err := rows.Scan(&c.ID, &c.Title, &c.Order, &c.Color, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			logger.ErrorContext(r.Context(), "Error scanning column row", err)
			return errors.NewDatabaseError().WithCause(err)
		}
		columns = append(columns, c)
	}

	json.NewEncoder(w).Encode(columns)
	return nil
}

// CreateColumn handles POST /columns
func CreateColumn(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	var req models.CreateColumnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewInvalidJSONError()
	}

	if req.Title == "" {
		return errors.NewBadRequestError("Title is required")
	}

	if req.Color == "" {
		req.Color = "#2196F3"
	}

	// Get max order
	var maxOrder int
	startTime := time.Now()
	err := database.DB.QueryRow(`SELECT COALESCE(MAX("order"), -1) FROM columns`).Scan(&maxOrder)
	logger.LogDatabaseOperation(r.Context(), "SELECT MAX", "columns", time.Since(startTime), err)
	if err != nil {
		logger.ErrorContext(r.Context(), "Error getting max order", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	// Insert column
	var c models.Column
	startTime = time.Now()
	err = database.DB.QueryRow(
		`INSERT INTO columns (title, "order", color) VALUES ($1, $2, $3)
		RETURNING id, title, "order", color, created_at, updated_at`,
		req.Title, maxOrder+1, req.Color,
	).Scan(&c.ID, &c.Title, &c.Order, &c.Color, &c.CreatedAt, &c.UpdatedAt)
	logger.LogDatabaseOperation(r.Context(), "INSERT", "columns", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(r.Context(), "Error creating column", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(c)
	return nil
}

// UpdateColumn handles PUT /columns/{id}
func UpdateColumn(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return errors.NewBadRequestError("Invalid column ID")
	}

	var req models.UpdateColumnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewInvalidJSONError()
	}

	// Check if column exists and get current values
	var existing models.Column
	startTime := time.Now()
	err = database.DB.QueryRow(`SELECT id, title, "order", color, created_at, updated_at FROM columns WHERE id = $1`, id).
		Scan(&existing.ID, &existing.Title, &existing.Order, &existing.Color, &existing.CreatedAt, &existing.UpdatedAt)
	logger.LogDatabaseOperation(r.Context(), "SELECT", "columns", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return errors.NewNotFoundError("Column not found")
	} else if err != nil {
		logger.ErrorContext(r.Context(), "Error fetching column", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	// Apply updates
	if req.Title != "" {
		existing.Title = req.Title
	}
	if req.Color != "" {
		existing.Color = req.Color
	}

	// Update column
	var c models.Column
	startTime = time.Now()
	err = database.DB.QueryRow(
		`UPDATE columns SET title = $1, color = $2, updated_at = NOW() WHERE id = $3
		RETURNING id, title, "order", color, created_at, updated_at`,
		existing.Title, existing.Color, id,
	).Scan(&c.ID, &c.Title, &c.Order, &c.Color, &c.CreatedAt, &c.UpdatedAt)
	logger.LogDatabaseOperation(r.Context(), "UPDATE", "columns", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(r.Context(), "Error updating column", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	json.NewEncoder(w).Encode(c)
	return nil
}

// DeleteColumn handles DELETE /columns/{id}
// Moves all tasks to the first column before deletion
func DeleteColumn(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return errors.NewBadRequestError("Invalid column ID")
	}

	// Get the first column (lowest order) that is not the one being deleted
	var firstColumnID int
	startTime := time.Now()
	err = database.DB.QueryRow(`SELECT id FROM columns WHERE id != $1 ORDER BY "order" ASC LIMIT 1`, id).Scan(&firstColumnID)
	logger.LogDatabaseOperation(r.Context(), "SELECT", "columns", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return errors.NewBadRequestError("Cannot delete the last column")
	} else if err != nil {
		logger.ErrorContext(r.Context(), "Error finding first column", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	// Move all tasks from the column being deleted to the first column
	startTime = time.Now()
	_, err = database.DB.Exec(`UPDATE tasks SET column_id = $1, updated_at = NOW() WHERE column_id = $2`, firstColumnID, id)
	logger.LogDatabaseOperation(r.Context(), "UPDATE", "tasks", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(r.Context(), "Error moving tasks", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	// Delete the column
	startTime = time.Now()
	result, err := database.DB.Exec(`DELETE FROM columns WHERE id = $1`, id)
	logger.LogDatabaseOperation(r.Context(), "DELETE", "columns", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(r.Context(), "Error deleting column", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.NewNotFoundError("Column not found")
	}

	// Reorder remaining columns to fill the gap
	startTime = time.Now()
	_, err = database.DB.Exec(`
		WITH ordered AS (
			SELECT id, ROW_NUMBER() OVER (ORDER BY "order") - 1 as new_order
			FROM columns
		)
		UPDATE columns SET "order" = ordered.new_order
		FROM ordered WHERE columns.id = ordered.id
	`)
	logger.LogDatabaseOperation(r.Context(), "UPDATE", "columns", time.Since(startTime), err)

	if err != nil {
		logger.WarnContext(r.Context(), "Error reordering columns after delete", map[string]interface{}{
			"error": err.Error(),
		})
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

// ReorderColumns handles PATCH /columns/reorder
func ReorderColumns(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	var req models.ReorderColumnsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewInvalidJSONError()
	}

	if len(req.ColumnIDs) == 0 {
		return errors.NewBadRequestError("columnIds is required")
	}

	// Update each column's order
	for i, columnID := range req.ColumnIDs {
		startTime := time.Now()
		result, err := database.DB.Exec(`UPDATE columns SET "order" = $1, updated_at = NOW() WHERE id = $2`, i, columnID)
		logger.LogDatabaseOperation(r.Context(), "UPDATE", "columns", time.Since(startTime), err)

		if err != nil {
			logger.ErrorContext(r.Context(), "Error updating column order", err)
			return errors.NewDatabaseError().WithCause(err)
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			return errors.NewNotFoundError("Column not found: " + strconv.Itoa(columnID))
		}
	}

	// Return updated columns
	startTime := time.Now()
	rows, err := database.DB.Query(`SELECT id, title, "order", color, created_at, updated_at FROM columns ORDER BY "order" ASC`)
	logger.LogDatabaseOperation(r.Context(), "SELECT", "columns", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(r.Context(), "Error querying columns", err)
		return errors.NewDatabaseError().WithCause(err)
	}
	defer rows.Close()

	columns := []models.Column{}
	for rows.Next() {
		var c models.Column
		err := rows.Scan(&c.ID, &c.Title, &c.Order, &c.Color, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			logger.ErrorContext(r.Context(), "Error scanning column row", err)
			return errors.NewDatabaseError().WithCause(err)
		}
		columns = append(columns, c)
	}

	json.NewEncoder(w).Encode(columns)
	return nil
}

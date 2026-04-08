package services

import (
	"context"
	"database/sql"
	"time"

	"github.com/clementhaon/sandbox-api-go/pkg/errors"
	"github.com/clementhaon/sandbox-api-go/pkg/logger"
	"github.com/clementhaon/sandbox-api-go/pkg/models"
)

type TimeEntryService interface {
	List(ctx context.Context, taskID int) ([]models.TimeEntry, error)
	Create(ctx context.Context, userID int, req models.CreateTimeEntryRequest) (models.TimeEntry, error)
	Delete(ctx context.Context, id int) error
}

type timeEntryService struct {
	db *sql.DB
}

func NewTimeEntryService(db *sql.DB) TimeEntryService {
	return &timeEntryService{db: db}
}

func (s *timeEntryService) List(ctx context.Context, taskID int) ([]models.TimeEntry, error) {
	startTime := time.Now()
	rows, err := s.db.Query(`
		SELECT id, task_id, user_id, start_time, end_time, duration, description, created_at
		FROM time_entries
		WHERE task_id = $1
		ORDER BY start_time DESC
	`, taskID)
	logger.LogDatabaseOperation(ctx, "SELECT", "time_entries", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error querying time entries", err)
		return nil, errors.NewDatabaseError().WithCause(err)
	}
	defer rows.Close()

	entries := []models.TimeEntry{}
	for rows.Next() {
		var e models.TimeEntry
		err := rows.Scan(&e.ID, &e.TaskID, &e.UserID, &e.StartTime, &e.EndTime, &e.Duration, &e.Description, &e.CreatedAt)
		if err != nil {
			logger.ErrorContext(ctx, "Error scanning time entry row", err)
			return nil, errors.NewDatabaseError().WithCause(err)
		}
		entries = append(entries, e)
	}

	return entries, nil
}

func (s *timeEntryService) Create(ctx context.Context, userID int, req models.CreateTimeEntryRequest) (models.TimeEntry, error) {
	if req.TaskID == 0 {
		return models.TimeEntry{}, errors.NewBadRequestError("taskId is required")
	}
	if req.StartTime.IsZero() {
		return models.TimeEntry{}, errors.NewBadRequestError("startTime is required")
	}
	if req.Duration <= 0 {
		return models.TimeEntry{}, errors.NewBadRequestError("duration must be positive")
	}

	var existingTaskID int
	startTime := time.Now()
	err := s.db.QueryRow("SELECT id FROM tasks WHERE id = $1", req.TaskID).Scan(&existingTaskID)
	logger.LogDatabaseOperation(ctx, "SELECT", "tasks", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return models.TimeEntry{}, errors.NewNotFoundError("Task not found")
	} else if err != nil {
		logger.ErrorContext(ctx, "Error checking task", err)
		return models.TimeEntry{}, errors.NewDatabaseError().WithCause(err)
	}

	var e models.TimeEntry
	startTime = time.Now()
	err = s.db.QueryRow(`
		INSERT INTO time_entries (task_id, user_id, start_time, end_time, duration, description)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, task_id, user_id, start_time, end_time, duration, description, created_at
	`, req.TaskID, userID, req.StartTime, req.EndTime, req.Duration, req.Description).
		Scan(&e.ID, &e.TaskID, &e.UserID, &e.StartTime, &e.EndTime, &e.Duration, &e.Description, &e.CreatedAt)
	logger.LogDatabaseOperation(ctx, "INSERT", "time_entries", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error creating time entry", err)
		return models.TimeEntry{}, errors.NewDatabaseError().WithCause(err)
	}

	durationMinutes := req.Duration / 60
	startTime = time.Now()
	_, err = s.db.Exec(`UPDATE tasks SET tracked_time = tracked_time + $1, updated_at = NOW() WHERE id = $2`, durationMinutes, req.TaskID)
	logger.LogDatabaseOperation(ctx, "UPDATE", "tasks", time.Since(startTime), err)
	if err != nil {
		logger.WarnContext(ctx, "Error updating task tracked_time", map[string]interface{}{
			"error":   err.Error(),
			"task_id": req.TaskID,
		})
	}

	logger.InfoContext(ctx, "Time entry created", map[string]interface{}{
		"entry_id": e.ID,
		"task_id":  e.TaskID,
		"duration": e.Duration,
		"user_id":  userID,
	})

	return e, nil
}

func (s *timeEntryService) Delete(ctx context.Context, id int) error {
	var taskID, duration int
	startTime := time.Now()
	err := s.db.QueryRow("SELECT task_id, duration FROM time_entries WHERE id = $1", id).Scan(&taskID, &duration)
	logger.LogDatabaseOperation(ctx, "SELECT", "time_entries", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return errors.NewNotFoundError("Time entry not found")
	} else if err != nil {
		logger.ErrorContext(ctx, "Error fetching time entry", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	startTime = time.Now()
	result, err := s.db.Exec("DELETE FROM time_entries WHERE id = $1", id)
	logger.LogDatabaseOperation(ctx, "DELETE", "time_entries", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error deleting time entry", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.NewNotFoundError("Time entry not found")
	}

	durationMinutes := duration / 60
	startTime = time.Now()
	_, err = s.db.Exec(`UPDATE tasks SET tracked_time = GREATEST(0, tracked_time - $1), updated_at = NOW() WHERE id = $2`, durationMinutes, taskID)
	logger.LogDatabaseOperation(ctx, "UPDATE", "tasks", time.Since(startTime), err)
	if err != nil {
		logger.WarnContext(ctx, "Error updating task tracked_time after delete", map[string]interface{}{
			"error":   err.Error(),
			"task_id": taskID,
		})
	}

	return nil
}

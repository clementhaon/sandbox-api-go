package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/logger"
	"github.com/clementhaon/sandbox-api-go/models"
)

type TimeEntryRepository interface {
	List(ctx context.Context, taskID int) ([]models.TimeEntry, error)
	TaskExists(ctx context.Context, taskID int) (bool, error)
	Create(ctx context.Context, userID int, req models.CreateTimeEntryRequest) (models.TimeEntry, error)
	AddTrackedTime(ctx context.Context, taskID int, durationMinutes int) error
	GetTaskIDAndDuration(ctx context.Context, id int) (taskID int, duration int, err error)
	Delete(ctx context.Context, id int) error
	SubtractTrackedTime(ctx context.Context, taskID int, durationMinutes int) error
}

type postgresTimeEntryRepo struct {
	db *sql.DB
}

func NewPostgresTimeEntryRepository(db *sql.DB) TimeEntryRepository {
	return &postgresTimeEntryRepo{db: db}
}

func (r *postgresTimeEntryRepo) List(ctx context.Context, taskID int) ([]models.TimeEntry, error) {
	startTime := time.Now()
	rows, err := r.db.QueryContext(ctx, `
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
		if err := rows.Scan(&e.ID, &e.TaskID, &e.UserID, &e.StartTime, &e.EndTime, &e.Duration, &e.Description, &e.CreatedAt); err != nil {
			logger.ErrorContext(ctx, "Error scanning time entry row", err)
			return nil, errors.NewDatabaseError().WithCause(err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (r *postgresTimeEntryRepo) TaskExists(ctx context.Context, taskID int) (bool, error) {
	var id int
	startTime := time.Now()
	err := r.db.QueryRowContext(ctx, "SELECT id FROM tasks WHERE id = $1", taskID).Scan(&id)
	logger.LogDatabaseOperation(ctx, "SELECT", "tasks", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		logger.ErrorContext(ctx, "Error checking task", err)
		return false, errors.NewDatabaseError().WithCause(err)
	}
	return true, nil
}

func (r *postgresTimeEntryRepo) Create(ctx context.Context, userID int, req models.CreateTimeEntryRequest) (models.TimeEntry, error) {
	var e models.TimeEntry
	startTime := time.Now()
	err := r.db.QueryRowContext(ctx, `
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
	return e, nil
}

func (r *postgresTimeEntryRepo) AddTrackedTime(ctx context.Context, taskID int, durationMinutes int) error {
	startTime := time.Now()
	_, err := r.db.ExecContext(ctx, `UPDATE tasks SET tracked_time = tracked_time + $1, updated_at = NOW() WHERE id = $2`, durationMinutes, taskID)
	logger.LogDatabaseOperation(ctx, "UPDATE", "tasks", time.Since(startTime), err)
	return err
}

func (r *postgresTimeEntryRepo) GetTaskIDAndDuration(ctx context.Context, id int) (int, int, error) {
	var taskID, duration int
	startTime := time.Now()
	err := r.db.QueryRowContext(ctx, "SELECT task_id, duration FROM time_entries WHERE id = $1", id).Scan(&taskID, &duration)
	logger.LogDatabaseOperation(ctx, "SELECT", "time_entries", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return 0, 0, errors.NewNotFoundError("Time entry not found")
	}
	if err != nil {
		logger.ErrorContext(ctx, "Error fetching time entry", err)
		return 0, 0, errors.NewDatabaseError().WithCause(err)
	}
	return taskID, duration, nil
}

func (r *postgresTimeEntryRepo) Delete(ctx context.Context, id int) error {
	startTime := time.Now()
	result, err := r.db.ExecContext(ctx, "DELETE FROM time_entries WHERE id = $1", id)
	logger.LogDatabaseOperation(ctx, "DELETE", "time_entries", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error deleting time entry", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.NewNotFoundError("Time entry not found")
	}
	return nil
}

func (r *postgresTimeEntryRepo) SubtractTrackedTime(ctx context.Context, taskID int, durationMinutes int) error {
	startTime := time.Now()
	_, err := r.db.ExecContext(ctx, `UPDATE tasks SET tracked_time = GREATEST(0, tracked_time - $1), updated_at = NOW() WHERE id = $2`, durationMinutes, taskID)
	logger.LogDatabaseOperation(ctx, "UPDATE", "tasks", time.Since(startTime), err)
	return err
}

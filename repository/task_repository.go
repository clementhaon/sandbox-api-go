package repository

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/clementhaon/sandbox-api-go/database"
	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/logger"
	"github.com/clementhaon/sandbox-api-go/models"

	"github.com/lib/pq"
)

type TaskRepository interface {
	ListWithAssignee(ctx context.Context, columnID *int) ([]models.Task, error)
	GetByID(ctx context.Context, id int) (models.Task, error)
	GetMaxOrder(ctx context.Context, columnID int) (int, error)
	Create(ctx context.Context, req models.CreateTaskRequest, order int, userID int) (models.Task, error)
	Exists(ctx context.Context, id int) (bool, error)
	Update(ctx context.Context, id int, req models.UpdateTaskRequest) (models.Task, error)
	Move(ctx context.Context, id int, columnID int, order int) (models.Task, error)
	Reorder(ctx context.Context, columnID int, taskIDs []int) error
	Delete(ctx context.Context, id int) error
	WithQuerier(q database.Querier) TaskRepository
}

type postgresTaskRepo struct {
	db database.Querier
}

func NewPostgresTaskRepository(db *sql.DB) TaskRepository {
	return &postgresTaskRepo{db: db}
}

func (r *postgresTaskRepo) WithQuerier(q database.Querier) TaskRepository {
	return &postgresTaskRepo{db: q}
}

func scanTaskRow(row interface{ Scan(...any) error }) (models.Task, error) {
	var t models.TaskDB
	var assigneeID sql.NullInt64
	var assigneeUsername, assigneeAvatarURL sql.NullString

	err := row.Scan(
		&t.ID, &t.Title, &t.Description, &t.ColumnID, &t.Order, &t.Priority,
		&t.AssigneeID, &t.Deadline, &t.EstimatedTime, &t.TrackedTime, &t.Tags,
		&t.CreatedBy, &t.UserID, &t.CreatedAt, &t.UpdatedAt,
		&assigneeID, &assigneeUsername, &assigneeAvatarURL,
	)
	if err != nil {
		return models.Task{}, err
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
	return task, nil
}

func scanTaskRows(ctx context.Context, rows *sql.Rows) ([]models.Task, error) {
	tasks := []models.Task{}
	for rows.Next() {
		task, err := scanTaskRow(rows)
		if err != nil {
			logger.ErrorContext(ctx, "Error scanning task row", err)
			return nil, errors.NewDatabaseError().WithCause(err)
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

const taskSelectWithAssignee = `
	SELECT t.id, t.title, t.description, t.column_id, t."order", t.priority,
		t.assignee_id, t.deadline, t.estimated_time, t.tracked_time, t.tags,
		t.created_by, t.user_id, t.created_at, t.updated_at,
		u.id, u.username, u.avatar_url
	FROM tasks t
	LEFT JOIN users u ON t.assignee_id = u.id`

func (r *postgresTaskRepo) ListWithAssignee(ctx context.Context, columnID *int) ([]models.Task, error) {
	var query string
	var args []interface{}

	if columnID != nil {
		query = taskSelectWithAssignee + ` WHERE t.column_id = $1 ORDER BY t."order" ASC`
		args = append(args, *columnID)
	} else {
		query = taskSelectWithAssignee + ` ORDER BY t.column_id, t."order" ASC`
	}

	startTime := time.Now()
	rows, err := r.db.QueryContext(ctx, query, args...)
	logger.LogDatabaseOperation(ctx, "SELECT", "tasks", time.Since(startTime), err)
	if err != nil {
		logger.ErrorContext(ctx, "Error querying tasks", err)
		return nil, errors.NewDatabaseError().WithCause(err)
	}
	defer rows.Close()

	return scanTaskRows(ctx, rows)
}

func (r *postgresTaskRepo) GetByID(ctx context.Context, id int) (models.Task, error) {
	startTime := time.Now()
	task, err := scanTaskRow(r.db.QueryRowContext(ctx, taskSelectWithAssignee+` WHERE t.id = $1`, id))
	logger.LogDatabaseOperation(ctx, "SELECT", "tasks", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return models.Task{}, errors.NewNotFoundError("Task not found")
	}
	if err != nil {
		logger.ErrorContext(ctx, "Error fetching task", err)
		return models.Task{}, errors.NewDatabaseError().WithCause(err)
	}
	return task, nil
}

func (r *postgresTaskRepo) GetMaxOrder(ctx context.Context, columnID int) (int, error) {
	var maxOrder int
	startTime := time.Now()
	err := r.db.QueryRowContext(ctx, `SELECT COALESCE(MAX("order"), -1) FROM tasks WHERE column_id = $1`, columnID).Scan(&maxOrder)
	logger.LogDatabaseOperation(ctx, "SELECT MAX", "tasks", time.Since(startTime), err)
	if err != nil {
		logger.ErrorContext(ctx, "Error getting max order", err)
		return 0, errors.NewDatabaseError().WithCause(err)
	}
	return maxOrder, nil
}

func (r *postgresTaskRepo) Create(ctx context.Context, req models.CreateTaskRequest, order int, userID int) (models.Task, error) {
	startTime := time.Now()
	task, err := scanTaskRow(r.db.QueryRowContext(ctx, `
		WITH inserted AS (
			INSERT INTO tasks (title, description, column_id, "order", priority, assignee_id, deadline, estimated_time, tags, created_by, user_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
			RETURNING *
		)
		SELECT i.id, i.title, i.description, i.column_id, i."order", i.priority,
			i.assignee_id, i.deadline, i.estimated_time, i.tracked_time, i.tags,
			i.created_by, i.user_id, i.created_at, i.updated_at,
			u.id, u.username, u.avatar_url
		FROM inserted i
		LEFT JOIN users u ON i.assignee_id = u.id`,
		req.Title, req.Description, req.ColumnID, order, req.Priority,
		req.AssigneeID, req.Deadline, req.EstimatedTime, pq.Array(req.Tags), userID,
	))
	logger.LogDatabaseOperation(ctx, "INSERT", "tasks", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error creating task", err)
		return models.Task{}, errors.NewDatabaseError().WithCause(err)
	}
	return task, nil
}

func (r *postgresTaskRepo) Exists(ctx context.Context, id int) (bool, error) {
	var existingID int
	startTime := time.Now()
	err := r.db.QueryRowContext(ctx, "SELECT id FROM tasks WHERE id = $1", id).Scan(&existingID)
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

func (r *postgresTaskRepo) Update(ctx context.Context, id int, req models.UpdateTaskRequest) (models.Task, error) {
	startTime := time.Now()
	task, err := scanTaskRow(r.db.QueryRowContext(ctx, `
		WITH updated AS (
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
			RETURNING *
		)
		SELECT u2.id, u2.title, u2.description, u2.column_id, u2."order", u2.priority,
			u2.assignee_id, u2.deadline, u2.estimated_time, u2.tracked_time, u2.tags,
			u2.created_by, u2.user_id, u2.created_at, u2.updated_at,
			usr.id, usr.username, usr.avatar_url
		FROM updated u2
		LEFT JOIN users usr ON u2.assignee_id = usr.id`,
		req.Title, req.Description, req.ColumnID, req.Priority,
		req.AssigneeID, req.Deadline, req.EstimatedTime, pq.Array(req.Tags), id,
	))
	logger.LogDatabaseOperation(ctx, "UPDATE", "tasks", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error updating task", err)
		return models.Task{}, errors.NewDatabaseError().WithCause(err)
	}
	return task, nil
}

func (r *postgresTaskRepo) Move(ctx context.Context, id int, columnID int, order int) (models.Task, error) {
	startTime := time.Now()
	task, err := scanTaskRow(r.db.QueryRowContext(ctx, `
		WITH moved AS (
			UPDATE tasks SET column_id = $1, "order" = $2, updated_at = NOW()
			WHERE id = $3
			RETURNING *
		)
		SELECT m.id, m.title, m.description, m.column_id, m."order", m.priority,
			m.assignee_id, m.deadline, m.estimated_time, m.tracked_time, m.tags,
			m.created_by, m.user_id, m.created_at, m.updated_at,
			u.id, u.username, u.avatar_url
		FROM moved m
		LEFT JOIN users u ON m.assignee_id = u.id`,
		columnID, order, id,
	))
	logger.LogDatabaseOperation(ctx, "UPDATE", "tasks", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return models.Task{}, errors.NewNotFoundError("Task not found")
	}
	if err != nil {
		logger.ErrorContext(ctx, "Error moving task", err)
		return models.Task{}, errors.NewDatabaseError().WithCause(err)
	}
	return task, nil
}

func (r *postgresTaskRepo) Reorder(ctx context.Context, columnID int, taskIDs []int) error {
	// If the querier is already a transaction, use it directly.
	// Otherwise, start a new transaction.
	var querier database.Querier
	var commitFn func() error

	if db, ok := r.db.(*sql.DB); ok {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			logger.ErrorContext(ctx, "Error starting transaction for reorder", err)
			return errors.NewDatabaseError().WithCause(err)
		}
		defer tx.Rollback()
		querier = tx
		commitFn = tx.Commit
	} else {
		querier = r.db
		commitFn = func() error { return nil }
	}

	for i, taskID := range taskIDs {
		startTime := time.Now()
		result, err := querier.ExecContext(ctx, `UPDATE tasks SET "order" = $1, updated_at = NOW() WHERE id = $2 AND column_id = $3`, i, taskID, columnID)
		logger.LogDatabaseOperation(ctx, "UPDATE", "tasks", time.Since(startTime), err)

		if err != nil {
			logger.ErrorContext(ctx, "Error updating task order", err)
			return errors.NewDatabaseError().WithCause(err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return errors.NewDatabaseError().WithCause(err)
		}
		if rowsAffected == 0 {
			return errors.NewNotFoundError("Task not found in column: " + strconv.Itoa(taskID))
		}
	}

	if err := commitFn(); err != nil {
		logger.ErrorContext(ctx, "Error committing reorder transaction", err)
		return errors.NewDatabaseError().WithCause(err)
	}
	return nil
}

func (r *postgresTaskRepo) Delete(ctx context.Context, id int) error {
	startTime := time.Now()
	result, err := r.db.ExecContext(ctx, "DELETE FROM tasks WHERE id = $1", id)
	logger.LogDatabaseOperation(ctx, "DELETE", "tasks", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error deleting task", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.NewDatabaseError().WithCause(err)
	}
	if rowsAffected == 0 {
		return errors.NewNotFoundError("Task not found")
	}
	return nil
}

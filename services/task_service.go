package services

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/logger"
	"github.com/clementhaon/sandbox-api-go/models"
	"github.com/clementhaon/sandbox-api-go/validation"

	"github.com/lib/pq"
)

type TaskService interface {
	GetBoard(ctx context.Context) (models.BoardResponse, error)
	List(ctx context.Context, columnID *int) ([]models.Task, error)
	GetByID(ctx context.Context, id int) (models.Task, error)
	Create(ctx context.Context, userID int, req models.CreateTaskRequest) (models.Task, error)
	Update(ctx context.Context, id int, req models.UpdateTaskRequest) (models.Task, error)
	Move(ctx context.Context, id int, req models.MoveTaskRequest) (models.Task, error)
	Reorder(ctx context.Context, columnID int, taskIDs []int) ([]models.Task, error)
	Delete(ctx context.Context, id int) error
}

type taskService struct {
	db *sql.DB
}

func NewTaskService(db *sql.DB) TaskService {
	return &taskService{db: db}
}

func (s *taskService) scanTaskRows(ctx context.Context, rows *sql.Rows) ([]models.Task, error) {
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
			logger.ErrorContext(ctx, "Error scanning task row", err)
			return nil, errors.NewDatabaseError().WithCause(err)
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
	return tasks, nil
}

func (s *taskService) fetchAssignee(t models.TaskDB) *models.UserBrief {
	if t.AssigneeID == nil {
		return nil
	}
	var assignee models.UserBrief
	var avatarURL sql.NullString
	err := s.db.QueryRow(`SELECT id, username, avatar_url FROM users WHERE id = $1`, *t.AssigneeID).
		Scan(&assignee.ID, &assignee.Username, &avatarURL)
	if err != nil {
		return nil
	}
	if avatarURL.Valid {
		assignee.AvatarURL = avatarURL.String
	}
	return &assignee
}

func (s *taskService) GetBoard(ctx context.Context) (models.BoardResponse, error) {
	startTime := time.Now()
	colRows, err := s.db.Query(`SELECT id, title, "order", color, created_at, updated_at FROM columns ORDER BY "order" ASC`)
	logger.LogDatabaseOperation(ctx, "SELECT", "columns", time.Since(startTime), err)
	if err != nil {
		logger.ErrorContext(ctx, "Error querying columns", err)
		return models.BoardResponse{}, errors.NewDatabaseError().WithCause(err)
	}
	defer colRows.Close()

	columns := []models.Column{}
	for colRows.Next() {
		var c models.Column
		if err := colRows.Scan(&c.ID, &c.Title, &c.Order, &c.Color, &c.CreatedAt, &c.UpdatedAt); err != nil {
			logger.ErrorContext(ctx, "Error scanning column row", err)
			return models.BoardResponse{}, errors.NewDatabaseError().WithCause(err)
		}
		columns = append(columns, c)
	}

	startTime = time.Now()
	taskRows, err := s.db.Query(`
		SELECT t.id, t.title, t.description, t.column_id, t."order", t.priority,
			t.assignee_id, t.deadline, t.estimated_time, t.tracked_time, t.tags,
			t.created_by, t.user_id, t.created_at, t.updated_at,
			u.id, u.username, u.avatar_url
		FROM tasks t
		LEFT JOIN users u ON t.assignee_id = u.id
		ORDER BY t.column_id, t."order" ASC
	`)
	logger.LogDatabaseOperation(ctx, "SELECT", "tasks", time.Since(startTime), err)
	if err != nil {
		logger.ErrorContext(ctx, "Error querying tasks", err)
		return models.BoardResponse{}, errors.NewDatabaseError().WithCause(err)
	}
	defer taskRows.Close()

	tasks, err := s.scanTaskRows(ctx, taskRows)
	if err != nil {
		return models.BoardResponse{}, err
	}

	return models.BoardResponse{Columns: columns, Tasks: tasks}, nil
}

func (s *taskService) List(ctx context.Context, columnID *int) ([]models.Task, error) {
	var query string
	var args []interface{}

	if columnID != nil {
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
		args = append(args, *columnID)
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
	rows, err := s.db.Query(query, args...)
	logger.LogDatabaseOperation(ctx, "SELECT", "tasks", time.Since(startTime), err)
	if err != nil {
		logger.ErrorContext(ctx, "Error querying tasks", err)
		return nil, errors.NewDatabaseError().WithCause(err)
	}
	defer rows.Close()

	return s.scanTaskRows(ctx, rows)
}

func (s *taskService) GetByID(ctx context.Context, id int) (models.Task, error) {
	var t models.TaskDB
	var assigneeID sql.NullInt64
	var assigneeUsername, assigneeAvatarURL sql.NullString

	startTime := time.Now()
	err := s.db.QueryRow(`
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
	logger.LogDatabaseOperation(ctx, "SELECT", "tasks", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return models.Task{}, errors.NewNotFoundError("Task not found")
	} else if err != nil {
		logger.ErrorContext(ctx, "Error fetching task", err)
		return models.Task{}, errors.NewDatabaseError().WithCause(err)
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

func (s *taskService) Create(ctx context.Context, userID int, req models.CreateTaskRequest) (models.Task, error) {
	if err := validation.ValidateTaskInput(req.Title, req.Description); err != nil {
		return models.Task{}, err
	}
	if req.ColumnID == 0 {
		return models.Task{}, errors.NewBadRequestError("ColumnID is required")
	}
	if req.Priority == "" {
		req.Priority = models.PriorityMedium
	}
	if req.Tags == nil {
		req.Tags = []string{}
	}

	var maxOrder int
	startTime := time.Now()
	err := s.db.QueryRow(`SELECT COALESCE(MAX("order"), -1) FROM tasks WHERE column_id = $1`, req.ColumnID).Scan(&maxOrder)
	logger.LogDatabaseOperation(ctx, "SELECT MAX", "tasks", time.Since(startTime), err)
	if err != nil {
		logger.ErrorContext(ctx, "Error getting max order", err)
		return models.Task{}, errors.NewDatabaseError().WithCause(err)
	}

	var t models.TaskDB
	var assigneeID sql.NullInt64
	var assigneeUsername, assigneeAvatarURL sql.NullString

	startTime = time.Now()
	err = s.db.QueryRow(`
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
		LEFT JOIN users u ON i.assignee_id = u.id
	`,
		req.Title, req.Description, req.ColumnID, maxOrder+1, req.Priority,
		req.AssigneeID, req.Deadline, req.EstimatedTime, pq.Array(req.Tags), userID,
	).Scan(
		&t.ID, &t.Title, &t.Description, &t.ColumnID, &t.Order, &t.Priority,
		&t.AssigneeID, &t.Deadline, &t.EstimatedTime, &t.TrackedTime, &t.Tags,
		&t.CreatedBy, &t.UserID, &t.CreatedAt, &t.UpdatedAt,
		&assigneeID, &assigneeUsername, &assigneeAvatarURL,
	)
	logger.LogDatabaseOperation(ctx, "INSERT", "tasks", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error creating task", err)
		return models.Task{}, errors.NewDatabaseError().WithCause(err)
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

	logger.InfoContext(ctx, "Task created", map[string]interface{}{
		"task_id":   task.ID,
		"column_id": task.ColumnID,
		"user_id":   userID,
	})

	return task, nil
}

func (s *taskService) Update(ctx context.Context, id int, req models.UpdateTaskRequest) (models.Task, error) {
	var existingID int
	startTime := time.Now()
	err := s.db.QueryRow("SELECT id FROM tasks WHERE id = $1", id).Scan(&existingID)
	logger.LogDatabaseOperation(ctx, "SELECT", "tasks", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return models.Task{}, errors.NewNotFoundError("Task not found")
	} else if err != nil {
		logger.ErrorContext(ctx, "Error checking task", err)
		return models.Task{}, errors.NewDatabaseError().WithCause(err)
	}

	var t models.TaskDB
	var assigneeID sql.NullInt64
	var assigneeUsername, assigneeAvatarURL sql.NullString

	startTime = time.Now()
	err = s.db.QueryRow(`
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
		LEFT JOIN users usr ON u2.assignee_id = usr.id
	`,
		req.Title, req.Description, req.ColumnID, req.Priority,
		req.AssigneeID, req.Deadline, req.EstimatedTime, pq.Array(req.Tags), id,
	).Scan(
		&t.ID, &t.Title, &t.Description, &t.ColumnID, &t.Order, &t.Priority,
		&t.AssigneeID, &t.Deadline, &t.EstimatedTime, &t.TrackedTime, &t.Tags,
		&t.CreatedBy, &t.UserID, &t.CreatedAt, &t.UpdatedAt,
		&assigneeID, &assigneeUsername, &assigneeAvatarURL,
	)
	logger.LogDatabaseOperation(ctx, "UPDATE", "tasks", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error updating task", err)
		return models.Task{}, errors.NewDatabaseError().WithCause(err)
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

func (s *taskService) Move(ctx context.Context, id int, req models.MoveTaskRequest) (models.Task, error) {
	var t models.TaskDB
	startTime := time.Now()
	err := s.db.QueryRow(`
		UPDATE tasks SET column_id = $1, "order" = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING id, title, description, column_id, "order", priority, assignee_id, deadline, estimated_time, tracked_time, tags, created_by, user_id, created_at, updated_at
	`, req.ColumnID, req.Order, id).Scan(
		&t.ID, &t.Title, &t.Description, &t.ColumnID, &t.Order, &t.Priority,
		&t.AssigneeID, &t.Deadline, &t.EstimatedTime, &t.TrackedTime, &t.Tags,
		&t.CreatedBy, &t.UserID, &t.CreatedAt, &t.UpdatedAt,
	)
	logger.LogDatabaseOperation(ctx, "UPDATE", "tasks", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return models.Task{}, errors.NewNotFoundError("Task not found")
	} else if err != nil {
		logger.ErrorContext(ctx, "Error moving task", err)
		return models.Task{}, errors.NewDatabaseError().WithCause(err)
	}

	task := t.ToTask()
	task.Assignee = s.fetchAssignee(t)

	return task, nil
}

func (s *taskService) Reorder(ctx context.Context, columnID int, taskIDs []int) ([]models.Task, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		logger.ErrorContext(ctx, "Error starting transaction for reorder", err)
		return nil, errors.NewDatabaseError().WithCause(err)
	}
	defer tx.Rollback()

	for i, taskID := range taskIDs {
		startTime := time.Now()
		result, err := tx.ExecContext(ctx, `UPDATE tasks SET "order" = $1, updated_at = NOW() WHERE id = $2 AND column_id = $3`, i, taskID, columnID)
		logger.LogDatabaseOperation(ctx, "UPDATE", "tasks", time.Since(startTime), err)

		if err != nil {
			logger.ErrorContext(ctx, "Error updating task order", err)
			return nil, errors.NewDatabaseError().WithCause(err)
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			return nil, errors.NewNotFoundError("Task not found in column: " + strconv.Itoa(taskID))
		}
	}

	if err := tx.Commit(); err != nil {
		logger.ErrorContext(ctx, "Error committing reorder transaction", err)
		return nil, errors.NewDatabaseError().WithCause(err)
	}

	return s.List(ctx, &columnID)
}

func (s *taskService) Delete(ctx context.Context, id int) error {
	startTime := time.Now()
	result, err := s.db.Exec("DELETE FROM tasks WHERE id = $1", id)
	logger.LogDatabaseOperation(ctx, "DELETE", "tasks", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error deleting task", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.NewNotFoundError("Task not found")
	}

	return nil
}

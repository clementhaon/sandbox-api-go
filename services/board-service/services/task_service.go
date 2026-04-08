package services

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/clementhaon/sandbox-api-go/pkg/errors"
	"github.com/clementhaon/sandbox-api-go/pkg/httpclient"
	"github.com/clementhaon/sandbox-api-go/pkg/logger"
	"github.com/clementhaon/sandbox-api-go/pkg/models"

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
	db         *sql.DB
	userClient *httpclient.UserServiceClient
}

func NewTaskService(db *sql.DB, userClient *httpclient.UserServiceClient) TaskService {
	return &taskService{db: db, userClient: userClient}
}

func (s *taskService) scanTaskRows(ctx context.Context, rows *sql.Rows) ([]models.Task, error) {
	tasks := []models.Task{}
	for rows.Next() {
		var t models.TaskDB
		err := rows.Scan(
			&t.ID, &t.Title, &t.Description, &t.ColumnID, &t.Order, &t.Priority,
			&t.AssigneeID, &t.Deadline, &t.EstimatedTime, &t.TrackedTime, &t.Tags,
			&t.CreatedBy, &t.UserID, &t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			logger.ErrorContext(ctx, "Error scanning task row", err)
			return nil, errors.NewDatabaseError().WithCause(err)
		}
		task := t.ToTask()
		tasks = append(tasks, task)
	}
	// Resolve assignees via User Service
	s.resolveAssignees(tasks)
	return tasks, nil
}

func (s *taskService) resolveAssignees(tasks []models.Task) {
	cache := map[int]*models.UserBrief{}
	for i := range tasks {
		if tasks[i].AssigneeID == nil {
			continue
		}
		id := *tasks[i].AssigneeID
		if brief, ok := cache[id]; ok {
			tasks[i].Assignee = brief
			continue
		}
		brief, err := s.userClient.GetUserBrief(id)
		if err == nil && brief != nil {
			cache[id] = brief
			tasks[i].Assignee = brief
		}
	}
}

func (s *taskService) fetchAssignee(t models.TaskDB) *models.UserBrief {
	if t.AssigneeID == nil {
		return nil
	}
	brief, err := s.userClient.GetUserBrief(*t.AssigneeID)
	if err != nil {
		return nil
	}
	return brief
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
			t.created_by, t.user_id, t.created_at, t.updated_at
		FROM tasks t
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
				t.created_by, t.user_id, t.created_at, t.updated_at
			FROM tasks t
			WHERE t.column_id = $1
			ORDER BY t."order" ASC
		`
		args = append(args, *columnID)
	} else {
		query = `
			SELECT t.id, t.title, t.description, t.column_id, t."order", t.priority,
				t.assignee_id, t.deadline, t.estimated_time, t.tracked_time, t.tags,
				t.created_by, t.user_id, t.created_at, t.updated_at
			FROM tasks t
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

	startTime := time.Now()
	err := s.db.QueryRow(`
		SELECT t.id, t.title, t.description, t.column_id, t."order", t.priority,
			t.assignee_id, t.deadline, t.estimated_time, t.tracked_time, t.tags,
			t.created_by, t.user_id, t.created_at, t.updated_at
		FROM tasks t
		WHERE t.id = $1
	`, id).Scan(
		&t.ID, &t.Title, &t.Description, &t.ColumnID, &t.Order, &t.Priority,
		&t.AssigneeID, &t.Deadline, &t.EstimatedTime, &t.TrackedTime, &t.Tags,
		&t.CreatedBy, &t.UserID, &t.CreatedAt, &t.UpdatedAt,
	)
	logger.LogDatabaseOperation(ctx, "SELECT", "tasks", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return models.Task{}, errors.NewNotFoundError("Task not found")
	} else if err != nil {
		logger.ErrorContext(ctx, "Error fetching task", err)
		return models.Task{}, errors.NewDatabaseError().WithCause(err)
	}

	task := t.ToTask()
	task.Assignee = s.fetchAssignee(t)

	return task, nil
}

func (s *taskService) Create(ctx context.Context, userID int, req models.CreateTaskRequest) (models.Task, error) {
	if req.Title == "" {
		return models.Task{}, errors.NewBadRequestError("Title is required")
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
	startTime = time.Now()
	err = s.db.QueryRow(`
		INSERT INTO tasks (title, description, column_id, "order", priority, assignee_id, deadline, estimated_time, tags, created_by, user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
		RETURNING id, title, description, column_id, "order", priority, assignee_id, deadline, estimated_time, tracked_time, tags, created_by, user_id, created_at, updated_at
	`,
		req.Title, req.Description, req.ColumnID, maxOrder+1, req.Priority,
		req.AssigneeID, req.Deadline, req.EstimatedTime, pq.Array(req.Tags), userID,
	).Scan(
		&t.ID, &t.Title, &t.Description, &t.ColumnID, &t.Order, &t.Priority,
		&t.AssigneeID, &t.Deadline, &t.EstimatedTime, &t.TrackedTime, &t.Tags,
		&t.CreatedBy, &t.UserID, &t.CreatedAt, &t.UpdatedAt,
	)
	logger.LogDatabaseOperation(ctx, "INSERT", "tasks", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error creating task", err)
		return models.Task{}, errors.NewDatabaseError().WithCause(err)
	}

	task := t.ToTask()
	task.Assignee = s.fetchAssignee(t)

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
	startTime = time.Now()
	err = s.db.QueryRow(`
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
	logger.LogDatabaseOperation(ctx, "UPDATE", "tasks", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error updating task", err)
		return models.Task{}, errors.NewDatabaseError().WithCause(err)
	}

	task := t.ToTask()
	task.Assignee = s.fetchAssignee(t)

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
	for i, taskID := range taskIDs {
		startTime := time.Now()
		result, err := s.db.Exec(`UPDATE tasks SET "order" = $1, updated_at = NOW() WHERE id = $2 AND column_id = $3`, i, taskID, columnID)
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

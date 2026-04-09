package services

import (
	"context"

	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/logger"
	"github.com/clementhaon/sandbox-api-go/models"
	"github.com/clementhaon/sandbox-api-go/repository"
	"github.com/clementhaon/sandbox-api-go/validation"
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
	taskRepo   repository.TaskRepository
	columnRepo repository.ColumnRepository
}

func NewTaskService(taskRepo repository.TaskRepository, columnRepo repository.ColumnRepository) TaskService {
	return &taskService{taskRepo: taskRepo, columnRepo: columnRepo}
}

func (s *taskService) GetBoard(ctx context.Context) (models.BoardResponse, error) {
	columns, err := s.columnRepo.List(ctx)
	if err != nil {
		return models.BoardResponse{}, err
	}

	tasks, err := s.taskRepo.ListWithAssignee(ctx, nil)
	if err != nil {
		return models.BoardResponse{}, err
	}

	return models.BoardResponse{Columns: columns, Tasks: tasks}, nil
}

func (s *taskService) List(ctx context.Context, columnID *int) ([]models.Task, error) {
	return s.taskRepo.ListWithAssignee(ctx, columnID)
}

func (s *taskService) GetByID(ctx context.Context, id int) (models.Task, error) {
	return s.taskRepo.GetByID(ctx, id)
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

	maxOrder, err := s.taskRepo.GetMaxOrder(ctx, req.ColumnID)
	if err != nil {
		return models.Task{}, err
	}

	task, err := s.taskRepo.Create(ctx, req, maxOrder+1, userID)
	if err != nil {
		return models.Task{}, err
	}

	logger.InfoContext(ctx, "Task created", map[string]interface{}{
		"task_id":   task.ID,
		"column_id": task.ColumnID,
		"user_id":   userID,
	})

	return task, nil
}

func (s *taskService) Update(ctx context.Context, id int, req models.UpdateTaskRequest) (models.Task, error) {
	exists, err := s.taskRepo.Exists(ctx, id)
	if err != nil {
		return models.Task{}, err
	}
	if !exists {
		return models.Task{}, errors.NewNotFoundError("Task not found")
	}

	return s.taskRepo.Update(ctx, id, req)
}

func (s *taskService) Move(ctx context.Context, id int, req models.MoveTaskRequest) (models.Task, error) {
	return s.taskRepo.Move(ctx, id, req.ColumnID, req.Order)
}

func (s *taskService) Reorder(ctx context.Context, columnID int, taskIDs []int) ([]models.Task, error) {
	if err := s.taskRepo.Reorder(ctx, columnID, taskIDs); err != nil {
		return nil, err
	}
	return s.taskRepo.ListWithAssignee(ctx, &columnID)
}

func (s *taskService) Delete(ctx context.Context, id int) error {
	return s.taskRepo.Delete(ctx, id)
}

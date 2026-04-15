package services

import (
	"context"
	"testing"
	"time"

	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/mocks"
	"github.com/clementhaon/sandbox-api-go/models"
)

func newTestTaskService(taskRepo *mocks.MockTaskRepository, columnRepo *mocks.MockColumnRepository) TaskService {
	return NewTaskService(taskRepo, columnRepo)
}

func TestTaskService_Create_Success(t *testing.T) {
	taskRepo := &mocks.MockTaskRepository{
		GetMaxOrderFn: func(ctx context.Context, columnID int) (int, error) {
			return 2, nil
		},
		CreateFn: func(ctx context.Context, req models.CreateTaskRequest, order int, userID int) (models.Task, error) {
			if order != 3 {
				t.Errorf("expected order 3, got %d", order)
			}
			if userID != 42 {
				t.Errorf("expected userID 42, got %d", userID)
			}
			return models.Task{
				ID:       1,
				Title:    req.Title,
				ColumnID: req.ColumnID,
				Priority: req.Priority,
				Order:    order,
			}, nil
		},
	}
	columnRepo := &mocks.MockColumnRepository{}

	svc := newTestTaskService(taskRepo, columnRepo)

	task, err := svc.Create(context.Background(), 42, models.CreateTaskRequest{
		Title:    "Test Task",
		ColumnID: 1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Title != "Test Task" {
		t.Errorf("expected title 'Test Task', got '%s'", task.Title)
	}
	if task.Priority != "medium" {
		t.Errorf("expected default priority 'medium', got '%s'", task.Priority)
	}
}

func TestTaskService_Create_ValidationError(t *testing.T) {
	taskRepo := &mocks.MockTaskRepository{}
	columnRepo := &mocks.MockColumnRepository{}
	svc := newTestTaskService(taskRepo, columnRepo)

	_, err := svc.Create(context.Background(), 1, models.CreateTaskRequest{
		Title:    "",
		ColumnID: 1,
	})
	if err == nil {
		t.Fatal("expected validation error for empty title")
	}
}

func TestTaskService_Create_MissingColumnID(t *testing.T) {
	taskRepo := &mocks.MockTaskRepository{}
	columnRepo := &mocks.MockColumnRepository{}
	svc := newTestTaskService(taskRepo, columnRepo)

	_, err := svc.Create(context.Background(), 1, models.CreateTaskRequest{
		Title:    "Valid Title",
		ColumnID: 0,
	})
	if err == nil {
		t.Fatal("expected error for missing ColumnID")
	}
	appErr, ok := errors.IsAppError(err)
	if !ok {
		t.Fatal("expected AppError")
	}
	if appErr.StatusCode != 400 {
		t.Errorf("expected status 400, got %d", appErr.StatusCode)
	}
}

func TestTaskService_Update_NotFound(t *testing.T) {
	taskRepo := &mocks.MockTaskRepository{
		ExistsFn: func(ctx context.Context, id int) (bool, error) {
			return false, nil
		},
	}
	columnRepo := &mocks.MockColumnRepository{}
	svc := newTestTaskService(taskRepo, columnRepo)

	_, err := svc.Update(context.Background(), 999, models.UpdateTaskRequest{Title: "New"})
	if err == nil {
		t.Fatal("expected not found error")
	}
	appErr, ok := errors.IsAppError(err)
	if !ok {
		t.Fatal("expected AppError")
	}
	if appErr.Code != errors.ErrNotFound {
		t.Errorf("expected NOT_FOUND, got %s", appErr.Code)
	}
}

func TestTaskService_Update_Success(t *testing.T) {
	taskRepo := &mocks.MockTaskRepository{
		ExistsFn: func(ctx context.Context, id int) (bool, error) {
			return true, nil
		},
		UpdateFn: func(ctx context.Context, id int, req models.UpdateTaskRequest) (models.Task, error) {
			return models.Task{ID: id, Title: req.Title}, nil
		},
	}
	columnRepo := &mocks.MockColumnRepository{}
	svc := newTestTaskService(taskRepo, columnRepo)

	task, err := svc.Update(context.Background(), 1, models.UpdateTaskRequest{Title: "Updated"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Title != "Updated" {
		t.Errorf("expected title 'Updated', got '%s'", task.Title)
	}
}

func TestTaskService_GetBoard(t *testing.T) {
	columns := []models.Column{
		{ID: 1, Title: "To Do", Order: 0},
		{ID: 2, Title: "Done", Order: 1},
	}
	tasks := []models.Task{
		{ID: 1, Title: "Task 1", ColumnID: 1},
		{ID: 2, Title: "Task 2", ColumnID: 2},
	}

	taskRepo := &mocks.MockTaskRepository{
		ListWithAssigneeFn: func(ctx context.Context, columnID *int) ([]models.Task, error) {
			return tasks, nil
		},
	}
	columnRepo := &mocks.MockColumnRepository{
		ListFn: func(ctx context.Context) ([]models.Column, error) {
			return columns, nil
		},
	}
	svc := newTestTaskService(taskRepo, columnRepo)

	board, err := svc.GetBoard(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(board.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(board.Columns))
	}
	if len(board.Tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(board.Tasks))
	}
}

func TestTaskService_Move(t *testing.T) {
	taskRepo := &mocks.MockTaskRepository{
		MoveFn: func(ctx context.Context, id int, columnID int, order int) (models.Task, error) {
			return models.Task{ID: id, ColumnID: columnID, Order: order}, nil
		},
	}
	columnRepo := &mocks.MockColumnRepository{}
	svc := newTestTaskService(taskRepo, columnRepo)

	task, err := svc.Move(context.Background(), 1, models.MoveTaskRequest{ColumnID: 2, Order: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.ColumnID != 2 {
		t.Errorf("expected column 2, got %d", task.ColumnID)
	}
}

func TestTaskService_Reorder(t *testing.T) {
	reorderCalled := false
	taskRepo := &mocks.MockTaskRepository{
		ReorderFn: func(ctx context.Context, columnID int, taskIDs []int) error {
			reorderCalled = true
			return nil
		},
		ListWithAssigneeFn: func(ctx context.Context, columnID *int) ([]models.Task, error) {
			return []models.Task{{ID: 2}, {ID: 1}}, nil
		},
	}
	columnRepo := &mocks.MockColumnRepository{}
	svc := newTestTaskService(taskRepo, columnRepo)

	tasks, err := svc.Reorder(context.Background(), 1, []int{2, 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reorderCalled {
		t.Error("expected Reorder to be called on repo")
	}
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}
}

func TestTaskService_Delete(t *testing.T) {
	deletedID := 0
	taskRepo := &mocks.MockTaskRepository{
		DeleteFn: func(ctx context.Context, id int) error {
			deletedID = id
			return nil
		},
	}
	columnRepo := &mocks.MockColumnRepository{}
	svc := newTestTaskService(taskRepo, columnRepo)

	err := svc.Delete(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedID != 5 {
		t.Errorf("expected delete ID 5, got %d", deletedID)
	}
}

func TestTaskService_Create_DescriptionTooLong(t *testing.T) {
	taskRepo := &mocks.MockTaskRepository{}
	columnRepo := &mocks.MockColumnRepository{}
	svc := newTestTaskService(taskRepo, columnRepo)

	_, err := svc.Create(context.Background(), 1, models.CreateTaskRequest{
		Title:       "Valid",
		ColumnID:    1,
		Description: string(make([]byte, 1001)),
	})
	if err == nil {
		t.Fatal("expected validation error for description too long")
	}
}

func TestTaskService_Create_WithAssignee(t *testing.T) {
	assigneeID := 10
	taskRepo := &mocks.MockTaskRepository{
		GetMaxOrderFn: func(ctx context.Context, columnID int) (int, error) {
			return 0, nil
		},
		CreateFn: func(ctx context.Context, req models.CreateTaskRequest, order int, userID int) (models.Task, error) {
			return models.Task{
				ID:         1,
				Title:      req.Title,
				ColumnID:   req.ColumnID,
				AssigneeID: req.AssigneeID,
				Assignee:   &models.UserBrief{ID: *req.AssigneeID, Username: "bob"},
			}, nil
		},
	}
	columnRepo := &mocks.MockColumnRepository{}
	svc := newTestTaskService(taskRepo, columnRepo)

	task, err := svc.Create(context.Background(), 1, models.CreateTaskRequest{
		Title:      "With Assignee",
		ColumnID:   1,
		AssigneeID: &assigneeID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Assignee == nil {
		t.Fatal("expected assignee to be set")
	}
	if task.Assignee.ID != 10 {
		t.Errorf("expected assignee ID 10, got %d", task.Assignee.ID)
	}
}

func TestTaskService_Create_WithDeadline(t *testing.T) {
	deadline := time.Now().Add(24 * time.Hour)
	taskRepo := &mocks.MockTaskRepository{
		GetMaxOrderFn: func(ctx context.Context, columnID int) (int, error) {
			return 0, nil
		},
		CreateFn: func(ctx context.Context, req models.CreateTaskRequest, order int, userID int) (models.Task, error) {
			return models.Task{
				ID:       1,
				Title:    req.Title,
				ColumnID: req.ColumnID,
				Deadline: req.Deadline,
			}, nil
		},
	}
	columnRepo := &mocks.MockColumnRepository{}
	svc := newTestTaskService(taskRepo, columnRepo)

	task, err := svc.Create(context.Background(), 1, models.CreateTaskRequest{
		Title:    "With Deadline",
		ColumnID: 1,
		Deadline: &deadline,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Deadline == nil {
		t.Fatal("expected deadline to be set")
	}
}

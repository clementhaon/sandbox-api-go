package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/middleware"
	"github.com/clementhaon/sandbox-api-go/mocks"
	"github.com/clementhaon/sandbox-api-go/models"
)

func withUserContext(r *http.Request, userID int) *http.Request {
	claims := &models.Claims{UserID: userID, Username: "testuser"}
	ctx := context.WithValue(r.Context(), middleware.UserContextKey, claims)
	return r.WithContext(ctx)
}

func TestTaskHandler_GetBoard(t *testing.T) {
	svc := &mocks.MockTaskService{
		GetBoardFn: func(ctx context.Context) (models.BoardResponse, error) {
			return models.BoardResponse{
				Columns: []models.Column{{ID: 1, Title: "To Do"}},
				Tasks:   []models.Task{{ID: 1, Title: "Task 1", ColumnID: 1}},
			}, nil
		},
	}

	handler := NewTaskHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/tasks/board", nil)
	w := httptest.NewRecorder()

	err := handler.GetBoard(w, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var board models.BoardResponse
	json.NewDecoder(w.Body).Decode(&board)
	if len(board.Columns) != 1 {
		t.Errorf("expected 1 column, got %d", len(board.Columns))
	}
	if len(board.Tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(board.Tasks))
	}
}

func TestTaskHandler_ListTasks(t *testing.T) {
	svc := &mocks.MockTaskService{
		ListFn: func(ctx context.Context, columnID *int) ([]models.Task, error) {
			return []models.Task{
				{ID: 1, Title: "Task 1"},
				{ID: 2, Title: "Task 2"},
			}, nil
		},
	}

	handler := NewTaskHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	w := httptest.NewRecorder()

	err := handler.ListTasks(w, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var tasks []models.Task
	json.NewDecoder(w.Body).Decode(&tasks)
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}
}

func TestTaskHandler_ListTasks_WithColumnFilter(t *testing.T) {
	var receivedColumnID *int
	svc := &mocks.MockTaskService{
		ListFn: func(ctx context.Context, columnID *int) ([]models.Task, error) {
			receivedColumnID = columnID
			return []models.Task{}, nil
		},
	}

	handler := NewTaskHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/tasks?columnId=3", nil)
	w := httptest.NewRecorder()

	err := handler.ListTasks(w, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedColumnID == nil || *receivedColumnID != 3 {
		t.Error("expected columnID to be 3")
	}
}

func TestTaskHandler_ListTasks_InvalidColumnId(t *testing.T) {
	svc := &mocks.MockTaskService{}
	handler := NewTaskHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/tasks?columnId=abc", nil)
	w := httptest.NewRecorder()

	err := handler.ListTasks(w, req)
	if err == nil {
		t.Fatal("expected error for invalid columnId")
	}
}

func TestTaskHandler_CreateTask(t *testing.T) {
	svc := &mocks.MockTaskService{
		CreateFn: func(ctx context.Context, userID int, req models.CreateTaskRequest) (models.Task, error) {
			return models.Task{ID: 1, Title: req.Title, ColumnID: req.ColumnID}, nil
		},
	}

	handler := NewTaskHandler(svc)
	body, _ := json.Marshal(models.CreateTaskRequest{
		Title:    "New Task",
		ColumnID: 1,
	})
	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
	req = withUserContext(req, 42)
	w := httptest.NewRecorder()

	err := handler.CreateTask(w, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	var task models.Task
	json.NewDecoder(w.Body).Decode(&task)
	if task.Title != "New Task" {
		t.Errorf("expected title 'New Task', got '%s'", task.Title)
	}
}

func TestTaskHandler_CreateTask_NoUserContext(t *testing.T) {
	svc := &mocks.MockTaskService{}
	handler := NewTaskHandler(svc)

	body, _ := json.Marshal(models.CreateTaskRequest{Title: "Task", ColumnID: 1})
	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
	// No user context set
	w := httptest.NewRecorder()

	err := handler.CreateTask(w, req)
	if err == nil {
		t.Fatal("expected error for missing user context")
	}
}

func TestTaskHandler_CreateTask_InvalidJSON(t *testing.T) {
	svc := &mocks.MockTaskService{}
	handler := NewTaskHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader([]byte("invalid json")))
	req = withUserContext(req, 1)
	w := httptest.NewRecorder()

	err := handler.CreateTask(w, req)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestTaskHandler_DeleteTask(t *testing.T) {
	deletedID := 0
	svc := &mocks.MockTaskService{
		DeleteFn: func(ctx context.Context, id int) error {
			deletedID = id
			return nil
		},
	}

	handler := NewTaskHandler(svc)
	req := httptest.NewRequest(http.MethodDelete, "/tasks/5", nil)
	req.SetPathValue("id", "5")
	w := httptest.NewRecorder()

	err := handler.DeleteTask(w, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}
	if deletedID != 5 {
		t.Errorf("expected delete ID 5, got %d", deletedID)
	}
}

func TestTaskHandler_DeleteTask_InvalidID(t *testing.T) {
	svc := &mocks.MockTaskService{}
	handler := NewTaskHandler(svc)

	req := httptest.NewRequest(http.MethodDelete, "/tasks/abc", nil)
	req.SetPathValue("id", "abc")
	w := httptest.NewRecorder()

	err := handler.DeleteTask(w, req)
	if err == nil {
		t.Fatal("expected error for invalid ID")
	}
}

func TestTaskHandler_GetTask_NotFound(t *testing.T) {
	svc := &mocks.MockTaskService{
		GetByIDFn: func(ctx context.Context, id int) (models.Task, error) {
			return models.Task{}, errors.NewNotFoundError("Task not found")
		},
	}

	handler := NewTaskHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/tasks/999", nil)
	req.SetPathValue("id", "999")
	w := httptest.NewRecorder()

	err := handler.GetTask(w, req)
	if err == nil {
		t.Fatal("expected error for not found task")
	}
	appErr, ok := errors.IsAppError(err)
	if !ok {
		t.Fatal("expected AppError")
	}
	if appErr.Code != errors.ErrNotFound {
		t.Errorf("expected NOT_FOUND, got %s", appErr.Code)
	}
}

func TestTaskHandler_ReorderTasks(t *testing.T) {
	svc := &mocks.MockTaskService{
		ReorderFn: func(ctx context.Context, columnID int, taskIDs []int) ([]models.Task, error) {
			return []models.Task{{ID: 2}, {ID: 1}}, nil
		},
	}

	handler := NewTaskHandler(svc)
	body, _ := json.Marshal(models.ReorderTasksRequest{ColumnID: 1, TaskIDs: []int{2, 1}})
	req := httptest.NewRequest(http.MethodPatch, "/tasks/reorder", bytes.NewReader(body))
	w := httptest.NewRecorder()

	err := handler.ReorderTasks(w, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTaskHandler_ReorderTasks_MissingColumnID(t *testing.T) {
	svc := &mocks.MockTaskService{}
	handler := NewTaskHandler(svc)

	body, _ := json.Marshal(models.ReorderTasksRequest{ColumnID: 0, TaskIDs: []int{1}})
	req := httptest.NewRequest(http.MethodPatch, "/tasks/reorder", bytes.NewReader(body))
	w := httptest.NewRecorder()

	err := handler.ReorderTasks(w, req)
	if err == nil {
		t.Fatal("expected error for missing columnId")
	}
}

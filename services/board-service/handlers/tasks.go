package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/clementhaon/sandbox-api-go/pkg/errors"
	"github.com/clementhaon/sandbox-api-go/pkg/middleware"
	"github.com/clementhaon/sandbox-api-go/pkg/models"
	"github.com/clementhaon/sandbox-api-go/services/board-service/services"
)

type TaskHandler struct {
	taskService services.TaskService
}

func NewTaskHandler(s services.TaskService) *TaskHandler {
	return &TaskHandler{taskService: s}
}

func (h *TaskHandler) GetBoard(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	board, err := h.taskService.GetBoard(r.Context())
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(board)
	return nil
}

func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	var columnID *int
	if columnIDStr := r.URL.Query().Get("columnId"); columnIDStr != "" {
		id, err := strconv.Atoi(columnIDStr)
		if err != nil {
			return errors.NewBadRequestError("Invalid columnId")
		}
		columnID = &id
	}

	tasks, err := h.taskService.List(r.Context(), columnID)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(tasks)
	return nil
}

func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return errors.NewBadRequestError("Invalid task ID")
	}

	task, err := h.taskService.GetByID(r.Context(), id)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(task)
	return nil
}

func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) error {
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

	task, err := h.taskService.Create(r.Context(), claims.UserID, req)
	if err != nil {
		return err
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
	return nil
}

func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return errors.NewBadRequestError("Invalid task ID")
	}

	var req models.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewInvalidJSONError()
	}

	task, err := h.taskService.Update(r.Context(), id, req)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(task)
	return nil
}

func (h *TaskHandler) MoveTask(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return errors.NewBadRequestError("Invalid task ID")
	}

	var req models.MoveTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewInvalidJSONError()
	}

	task, err := h.taskService.Move(r.Context(), id, req)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(task)
	return nil
}

func (h *TaskHandler) ReorderTasks(w http.ResponseWriter, r *http.Request) error {
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

	tasks, err := h.taskService.Reorder(r.Context(), req.ColumnID, req.TaskIDs)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(tasks)
	return nil
}

func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return errors.NewBadRequestError("Invalid task ID")
	}

	if err := h.taskService.Delete(r.Context(), id); err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

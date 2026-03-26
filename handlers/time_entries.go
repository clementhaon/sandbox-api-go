package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/middleware"
	"github.com/clementhaon/sandbox-api-go/models"
	"github.com/clementhaon/sandbox-api-go/services"
)

type TimeEntryHandler struct {
	timeEntryService services.TimeEntryService
}

func NewTimeEntryHandler(s services.TimeEntryService) *TimeEntryHandler {
	return &TimeEntryHandler{timeEntryService: s}
}

func (h *TimeEntryHandler) ListTimeEntries(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	taskIDStr := r.URL.Query().Get("taskId")
	if taskIDStr == "" {
		return errors.NewBadRequestError("taskId query parameter is required")
	}

	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil {
		return errors.NewBadRequestError("Invalid taskId")
	}

	entries, err := h.timeEntryService.List(r.Context(), taskID)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(entries)
	return nil
}

func (h *TimeEntryHandler) CreateTimeEntry(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		return errors.NewInternalError().WithDetails(map[string]interface{}{
			"issue": "user_context_missing",
		})
	}

	var req models.CreateTimeEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewInvalidJSONError()
	}

	entry, err := h.timeEntryService.Create(r.Context(), claims.UserID, req)
	if err != nil {
		return err
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(entry)
	return nil
}

func (h *TimeEntryHandler) DeleteTimeEntry(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return errors.NewBadRequestError("Invalid time entry ID")
	}

	if err := h.timeEntryService.Delete(r.Context(), id); err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/models"
	"github.com/clementhaon/sandbox-api-go/services"
)

type ColumnHandler struct {
	columnService services.ColumnService
}

func NewColumnHandler(s services.ColumnService) *ColumnHandler {
	return &ColumnHandler{columnService: s}
}

func (h *ColumnHandler) ListColumns(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	columns, err := h.columnService.List(r.Context())
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(columns)
	return nil
}

func (h *ColumnHandler) CreateColumn(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	var req models.CreateColumnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewInvalidJSONError()
	}

	if req.Title == "" {
		return errors.NewBadRequestError("Title is required")
	}

	column, err := h.columnService.Create(r.Context(), req)
	if err != nil {
		return err
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(column)
	return nil
}

func (h *ColumnHandler) UpdateColumn(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return errors.NewBadRequestError("Invalid column ID")
	}

	var req models.UpdateColumnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewInvalidJSONError()
	}

	column, err := h.columnService.Update(r.Context(), id, req)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(column)
	return nil
}

func (h *ColumnHandler) DeleteColumn(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return errors.NewBadRequestError("Invalid column ID")
	}

	if err := h.columnService.Delete(r.Context(), id); err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (h *ColumnHandler) ReorderColumns(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	var req models.ReorderColumnsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewInvalidJSONError()
	}

	if len(req.ColumnIDs) == 0 {
		return errors.NewBadRequestError("columnIds is required")
	}

	columns, err := h.columnService.Reorder(r.Context(), req.ColumnIDs)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(columns)
	return nil
}

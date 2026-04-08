package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/clementhaon/sandbox-api-go/pkg/errors"
	"github.com/clementhaon/sandbox-api-go/pkg/models"
	"github.com/clementhaon/sandbox-api-go/services/user-service/services"
)

type UserHandler struct {
	userService services.UserService
}

func NewUserHandler(s services.UserService) *UserHandler {
	return &UserHandler{userService: s}
}

func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))

	params := services.UserListParams{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    r.URL.Query().Get("sortBy"),
		SortOrder: r.URL.Query().Get("sortOrder"),
		Search:    r.URL.Query().Get("search"),
		Role:      r.URL.Query().Get("role"),
		Status:    r.URL.Query().Get("status"),
	}

	response, err := h.userService.List(r.Context(), params)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(response)
	return nil
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return errors.NewBadRequestError("Invalid user ID")
	}

	user, err := h.userService.GetByID(r.Context(), id)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(user)
	return nil
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewInvalidJSONError()
	}

	user, err := h.userService.Create(r.Context(), req)
	if err != nil {
		return err
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
	return nil
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return errors.NewBadRequestError("Invalid user ID")
	}

	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewInvalidJSONError()
	}

	user, err := h.userService.Update(r.Context(), id, req)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(user)
	return nil
}

func (h *UserHandler) UpdateUserStatus(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return errors.NewBadRequestError("Invalid user ID")
	}

	var req models.UpdateUserStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewInvalidJSONError()
	}

	user, err := h.userService.UpdateStatus(r.Context(), id, req.Status)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(user)
	return nil
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return errors.NewBadRequestError("Invalid user ID")
	}

	if err := h.userService.Delete(r.Context(), id); err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

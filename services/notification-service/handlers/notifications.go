package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/clementhaon/sandbox-api-go/pkg/errors"
	"github.com/clementhaon/sandbox-api-go/pkg/middleware"
	"github.com/clementhaon/sandbox-api-go/pkg/models"
	"github.com/clementhaon/sandbox-api-go/services/notification-service/services"
)

type NotificationHandler struct {
	notificationService services.NotificationService
}

func NewNotificationHandler(s services.NotificationService) *NotificationHandler {
	return &NotificationHandler{notificationService: s}
}

func (h *NotificationHandler) ListNotifications(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		return errors.NewInternalError().WithDetails(map[string]interface{}{
			"issue": "user_context_missing",
		})
	}

	notifications, err := h.notificationService.List(r.Context(), claims.UserID)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(notifications)
	return nil
}

func (h *NotificationHandler) MarkNotificationsRead(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		return errors.NewInternalError().WithDetails(map[string]interface{}{
			"issue": "user_context_missing",
		})
	}

	var req models.MarkNotificationsReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewInvalidJSONError()
	}

	if len(req.NotificationIDs) == 0 {
		return errors.NewBadRequestError("notificationIds is required")
	}

	marked, err := h.notificationService.MarkRead(r.Context(), claims.UserID, req.NotificationIDs)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"marked":  marked,
	})
	return nil
}

func (h *NotificationHandler) MarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		return errors.NewInternalError().WithDetails(map[string]interface{}{
			"issue": "user_context_missing",
		})
	}

	marked, err := h.notificationService.MarkAllRead(r.Context(), claims.UserID)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"marked":  marked,
	})
	return nil
}

func (h *NotificationHandler) DeleteNotification(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		return errors.NewInternalError().WithDetails(map[string]interface{}{
			"issue": "user_context_missing",
		})
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return errors.NewBadRequestError("Invalid notification ID")
	}

	if err := h.notificationService.Delete(r.Context(), claims.UserID, id); err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

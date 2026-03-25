package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/clementhaon/sandbox-api-go/database"
	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/logger"
	"github.com/clementhaon/sandbox-api-go/middleware"
	"github.com/clementhaon/sandbox-api-go/models"
)

// ListNotifications handles GET /notifications - returns notifications for current user
func ListNotifications(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		return errors.NewInternalError().WithDetails(map[string]interface{}{
			"issue": "user_context_missing",
		})
	}

	startTime := time.Now()
	rows, err := database.DB.Query(`
		SELECT id, type, title, message, read, data, created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, claims.UserID)
	logger.LogDatabaseOperation(r.Context(), "SELECT", "notifications", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(r.Context(), "Error querying notifications", err)
		return errors.NewDatabaseError().WithCause(err)
	}
	defer rows.Close()

	notifications := []models.Notification{}
	for rows.Next() {
		var n models.Notification
		err := rows.Scan(&n.ID, &n.Type, &n.Title, &n.Message, &n.Read, &n.Data, &n.CreatedAt)
		if err != nil {
			logger.ErrorContext(r.Context(), "Error scanning notification row", err)
			return errors.NewDatabaseError().WithCause(err)
		}
		notifications = append(notifications, n)
	}

	json.NewEncoder(w).Encode(notifications)
	return nil
}

// MarkNotificationsRead handles PATCH /notifications/read
func MarkNotificationsRead(w http.ResponseWriter, r *http.Request) error {
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

	// Update notifications - only mark as read if they belong to the user
	for _, notifID := range req.NotificationIDs {
		startTime := time.Now()
		_, err := database.DB.Exec(`UPDATE notifications SET read = true WHERE id = $1 AND user_id = $2`, notifID, claims.UserID)
		logger.LogDatabaseOperation(r.Context(), "UPDATE", "notifications", time.Since(startTime), err)

		if err != nil {
			logger.ErrorContext(r.Context(), "Error marking notification as read", err)
			return errors.NewDatabaseError().WithCause(err)
		}
	}

	// Return success
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"marked":  len(req.NotificationIDs),
	})
	return nil
}

// MarkAllNotificationsRead handles PATCH /notifications/read-all
func MarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		return errors.NewInternalError().WithDetails(map[string]interface{}{
			"issue": "user_context_missing",
		})
	}

	startTime := time.Now()
	result, err := database.DB.Exec(`UPDATE notifications SET read = true WHERE user_id = $1 AND read = false`, claims.UserID)
	logger.LogDatabaseOperation(r.Context(), "UPDATE", "notifications", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(r.Context(), "Error marking all notifications as read", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	rowsAffected, _ := result.RowsAffected()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"marked":  rowsAffected,
	})
	return nil
}

// DeleteNotification handles DELETE /notifications/{id}
func DeleteNotification(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		return errors.NewInternalError().WithDetails(map[string]interface{}{
			"issue": "user_context_missing",
		})
	}

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return errors.NewBadRequestError("Invalid notification ID")
	}

	// Delete notification - only if it belongs to the user
	startTime := time.Now()
	result, err := database.DB.Exec("DELETE FROM notifications WHERE id = $1 AND user_id = $2", id, claims.UserID)
	logger.LogDatabaseOperation(r.Context(), "DELETE", "notifications", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(r.Context(), "Error deleting notification", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.NewNotFoundError("Notification not found")
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

// CreateNotification is a helper function to create notifications (used internally)
func CreateNotification(userID int, notifType, title, message string, data models.NotificationData) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = database.DB.Exec(`
		INSERT INTO notifications (user_id, type, title, message, data)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, notifType, title, message, dataJSON)

	return err
}

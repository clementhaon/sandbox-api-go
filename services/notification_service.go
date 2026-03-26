package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/logger"
	"github.com/clementhaon/sandbox-api-go/models"
	"github.com/clementhaon/sandbox-api-go/websocket"
)

type NotificationService interface {
	List(ctx context.Context, userID int) ([]models.Notification, error)
	MarkRead(ctx context.Context, userID int, notificationIDs []int) (int, error)
	MarkAllRead(ctx context.Context, userID int) (int64, error)
	Delete(ctx context.Context, userID int, id int) error
	Create(ctx context.Context, userID int, notifType, title, message string, data models.NotificationData) error
}

type notificationService struct {
	db        *sql.DB
	wsManager *websocket.Manager
}

func NewNotificationService(db *sql.DB, wsManager *websocket.Manager) NotificationService {
	return &notificationService{db: db, wsManager: wsManager}
}

func (s *notificationService) List(ctx context.Context, userID int) ([]models.Notification, error) {
	startTime := time.Now()
	rows, err := s.db.Query(`
		SELECT id, type, title, message, read, data, created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	logger.LogDatabaseOperation(ctx, "SELECT", "notifications", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error querying notifications", err)
		return nil, errors.NewDatabaseError().WithCause(err)
	}
	defer rows.Close()

	notifications := []models.Notification{}
	for rows.Next() {
		var n models.Notification
		err := rows.Scan(&n.ID, &n.Type, &n.Title, &n.Message, &n.Read, &n.Data, &n.CreatedAt)
		if err != nil {
			logger.ErrorContext(ctx, "Error scanning notification row", err)
			return nil, errors.NewDatabaseError().WithCause(err)
		}
		notifications = append(notifications, n)
	}

	return notifications, nil
}

func (s *notificationService) MarkRead(ctx context.Context, userID int, notificationIDs []int) (int, error) {
	for _, notifID := range notificationIDs {
		startTime := time.Now()
		_, err := s.db.Exec(`UPDATE notifications SET read = true WHERE id = $1 AND user_id = $2`, notifID, userID)
		logger.LogDatabaseOperation(ctx, "UPDATE", "notifications", time.Since(startTime), err)

		if err != nil {
			logger.ErrorContext(ctx, "Error marking notification as read", err)
			return 0, errors.NewDatabaseError().WithCause(err)
		}
	}

	return len(notificationIDs), nil
}

func (s *notificationService) MarkAllRead(ctx context.Context, userID int) (int64, error) {
	startTime := time.Now()
	result, err := s.db.Exec(`UPDATE notifications SET read = true WHERE user_id = $1 AND read = false`, userID)
	logger.LogDatabaseOperation(ctx, "UPDATE", "notifications", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error marking all notifications as read", err)
		return 0, errors.NewDatabaseError().WithCause(err)
	}

	rowsAffected, _ := result.RowsAffected()
	return rowsAffected, nil
}

func (s *notificationService) Delete(ctx context.Context, userID int, id int) error {
	startTime := time.Now()
	result, err := s.db.Exec("DELETE FROM notifications WHERE id = $1 AND user_id = $2", id, userID)
	logger.LogDatabaseOperation(ctx, "DELETE", "notifications", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error deleting notification", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.NewNotFoundError("Notification not found")
	}

	return nil
}

func (s *notificationService) Create(ctx context.Context, userID int, notifType, title, message string, data models.NotificationData) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		INSERT INTO notifications (user_id, type, title, message, data)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, notifType, title, message, dataJSON)

	if err != nil {
		return err
	}

	if s.wsManager != nil {
		s.wsManager.SendToUser(userID, &websocket.Message{
			Type: "notification",
			Payload: map[string]interface{}{
				"type":    notifType,
				"title":   title,
				"message": message,
				"data":    data,
			},
		})
	}

	return nil
}

package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/clementhaon/sandbox-api-go/database"
	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/logger"
	"github.com/clementhaon/sandbox-api-go/models"
	"github.com/lib/pq"
)

type NotificationRepository interface {
	List(ctx context.Context, userID int) ([]models.Notification, error)
	MarkRead(ctx context.Context, userID int, notificationIDs []int) error
	MarkAllRead(ctx context.Context, userID int) (int64, error)
	Delete(ctx context.Context, userID int, id int) error
	Create(ctx context.Context, userID int, notifType, title, message string, dataJSON []byte) error
	WithQuerier(q database.Querier) NotificationRepository
}

type postgresNotificationRepo struct {
	db database.Querier
}

func NewPostgresNotificationRepository(db *sql.DB) NotificationRepository {
	return &postgresNotificationRepo{db: db}
}

func (r *postgresNotificationRepo) WithQuerier(q database.Querier) NotificationRepository {
	return &postgresNotificationRepo{db: q}
}

func (r *postgresNotificationRepo) List(ctx context.Context, userID int) ([]models.Notification, error) {
	startTime := time.Now()
	rows, err := r.db.QueryContext(ctx, `
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
		if err := rows.Scan(&n.ID, &n.Type, &n.Title, &n.Message, &n.Read, &n.Data, &n.CreatedAt); err != nil {
			logger.ErrorContext(ctx, "Error scanning notification row", err)
			return nil, errors.NewDatabaseError().WithCause(err)
		}
		notifications = append(notifications, n)
	}
	return notifications, nil
}

func (r *postgresNotificationRepo) MarkRead(ctx context.Context, userID int, notificationIDs []int) error {
	startTime := time.Now()
	_, err := r.db.ExecContext(ctx, `UPDATE notifications SET read = true WHERE id = ANY($1) AND user_id = $2`, pq.Array(notificationIDs), userID)
	logger.LogDatabaseOperation(ctx, "UPDATE", "notifications", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error marking notifications as read", err)
		return errors.NewDatabaseError().WithCause(err)
	}
	return nil
}

func (r *postgresNotificationRepo) MarkAllRead(ctx context.Context, userID int) (int64, error) {
	startTime := time.Now()
	result, err := r.db.ExecContext(ctx, `UPDATE notifications SET read = true WHERE user_id = $1 AND read = false`, userID)
	logger.LogDatabaseOperation(ctx, "UPDATE", "notifications", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error marking all notifications as read", err)
		return 0, errors.NewDatabaseError().WithCause(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, errors.NewDatabaseError().WithCause(err)
	}
	return rowsAffected, nil
}

func (r *postgresNotificationRepo) Delete(ctx context.Context, userID int, id int) error {
	startTime := time.Now()
	result, err := r.db.ExecContext(ctx, "DELETE FROM notifications WHERE id = $1 AND user_id = $2", id, userID)
	logger.LogDatabaseOperation(ctx, "DELETE", "notifications", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error deleting notification", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.NewDatabaseError().WithCause(err)
	}
	if rowsAffected == 0 {
		return errors.NewNotFoundError("Notification not found")
	}
	return nil
}

func (r *postgresNotificationRepo) Create(ctx context.Context, userID int, notifType, title, message string, dataJSON []byte) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO notifications (user_id, type, title, message, data)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, notifType, title, message, dataJSON)
	return err
}

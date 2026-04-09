package services

import (
	"context"
	"encoding/json"

	"github.com/clementhaon/sandbox-api-go/models"
	"github.com/clementhaon/sandbox-api-go/repository"
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
	notifRepo repository.NotificationRepository
	wsManager *websocket.Manager
}

func NewNotificationService(notifRepo repository.NotificationRepository, wsManager *websocket.Manager) NotificationService {
	return &notificationService{notifRepo: notifRepo, wsManager: wsManager}
}

func (s *notificationService) List(ctx context.Context, userID int) ([]models.Notification, error) {
	return s.notifRepo.List(ctx, userID)
}

func (s *notificationService) MarkRead(ctx context.Context, userID int, notificationIDs []int) (int, error) {
	if err := s.notifRepo.MarkRead(ctx, userID, notificationIDs); err != nil {
		return 0, err
	}
	return len(notificationIDs), nil
}

func (s *notificationService) MarkAllRead(ctx context.Context, userID int) (int64, error) {
	return s.notifRepo.MarkAllRead(ctx, userID)
}

func (s *notificationService) Delete(ctx context.Context, userID int, id int) error {
	return s.notifRepo.Delete(ctx, userID, id)
}

func (s *notificationService) Create(ctx context.Context, userID int, notifType, title, message string, data models.NotificationData) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if err := s.notifRepo.Create(ctx, userID, notifType, title, message, dataJSON); err != nil {
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

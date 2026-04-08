package models

import (
	"encoding/json"
	"time"
)

type Notification struct {
	ID        int             `json:"id"`
	UserID    int             `json:"-"`
	Type      string          `json:"type"`
	Title     string          `json:"title"`
	Message   string          `json:"message"`
	Read      bool            `json:"read"`
	Data      json.RawMessage `json:"data,omitempty"`
	CreatedAt time.Time       `json:"createdAt"`
}

type NotificationData struct {
	TaskID    int    `json:"taskId,omitempty"`
	TaskTitle string `json:"taskTitle,omitempty"`
	UserID    int    `json:"userId,omitempty"`
	UserName  string `json:"userName,omitempty"`
}

type MarkNotificationsReadRequest struct {
	NotificationIDs []int `json:"notificationIds"`
}

type CreateNotificationRequest struct {
	UserID  int              `json:"userId"`
	Type    string           `json:"type"`
	Title   string           `json:"title"`
	Message string           `json:"message"`
	Data    NotificationData `json:"data,omitempty"`
}

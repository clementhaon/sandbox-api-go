package models

import (
	"encoding/json"
	"time"
)

// Notification represents a user notification
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

// NotificationData represents the data field content for notifications
type NotificationData struct {
	TaskID    int    `json:"taskId,omitempty"`
	TaskTitle string `json:"taskTitle,omitempty"`
	UserID    int    `json:"userId,omitempty"`
	UserName  string `json:"userName,omitempty"`
}

// MarkNotificationsReadRequest represents the request to mark notifications as read
type MarkNotificationsReadRequest struct {
	NotificationIDs []int `json:"notificationIds"`
}

// CreateNotificationRequest represents the request to create a notification (internal use)
type CreateNotificationRequest struct {
	UserID  int              `json:"userId"`
	Type    string           `json:"type"`
	Title   string           `json:"title"`
	Message string           `json:"message"`
	Data    NotificationData `json:"data,omitempty"`
}

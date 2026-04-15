package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/mocks"
	"github.com/clementhaon/sandbox-api-go/models"
)

func TestNotificationHandler_ListNotifications(t *testing.T) {
	tests := []struct {
		name       string
		userID     int
		withCtx    bool
		listFn     func(ctx context.Context, userID int) ([]models.Notification, error)
		wantStatus int
		wantErr    bool
	}{
		{
			name:    "success",
			userID:  1,
			withCtx: true,
			listFn: func(ctx context.Context, userID int) ([]models.Notification, error) {
				return []models.Notification{{ID: 1, Title: "Hello"}}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:    "no user context",
			withCtx: false,
			wantErr: true,
		},
		{
			name:    "service error",
			userID:  1,
			withCtx: true,
			listFn: func(ctx context.Context, userID int) ([]models.Notification, error) {
				return nil, errors.NewInternalError()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mocks.MockNotificationService{ListFn: tt.listFn}
			handler := NewNotificationHandler(svc)

			req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
			if tt.withCtx {
				req = withUserContext(req, tt.userID)
			}
			w := httptest.NewRecorder()

			err := handler.ListNotifications(w, req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestNotificationHandler_MarkNotificationsRead(t *testing.T) {
	tests := []struct {
		name       string
		userID     int
		withCtx    bool
		body       interface{}
		markReadFn func(ctx context.Context, userID int, notificationIDs []int) (int, error)
		wantStatus int
		wantErr    bool
	}{
		{
			name:    "success",
			userID:  1,
			withCtx: true,
			body:    models.MarkNotificationsReadRequest{NotificationIDs: []int{1, 2}},
			markReadFn: func(ctx context.Context, userID int, notificationIDs []int) (int, error) {
				return len(notificationIDs), nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:    "no user context",
			withCtx: false,
			body:    models.MarkNotificationsReadRequest{NotificationIDs: []int{1}},
			wantErr: true,
		},
		{
			name:    "invalid json",
			userID:  1,
			withCtx: true,
			body:    "bad",
			wantErr: true,
		},
		{
			name:    "empty notification ids",
			userID:  1,
			withCtx: true,
			body:    models.MarkNotificationsReadRequest{NotificationIDs: []int{}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mocks.MockNotificationService{MarkReadFn: tt.markReadFn}
			handler := NewNotificationHandler(svc)

			var bodyBytes []byte
			switch v := tt.body.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				bodyBytes, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPatch, "/notifications/read", bytes.NewReader(bodyBytes))
			if tt.withCtx {
				req = withUserContext(req, tt.userID)
			}
			w := httptest.NewRecorder()

			err := handler.MarkNotificationsRead(w, req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestNotificationHandler_MarkAllNotificationsRead(t *testing.T) {
	tests := []struct {
		name          string
		userID        int
		withCtx       bool
		markAllReadFn func(ctx context.Context, userID int) (int64, error)
		wantStatus    int
		wantErr       bool
	}{
		{
			name:    "success",
			userID:  1,
			withCtx: true,
			markAllReadFn: func(ctx context.Context, userID int) (int64, error) {
				return 5, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:    "no user context",
			withCtx: false,
			wantErr: true,
		},
		{
			name:    "service error",
			userID:  1,
			withCtx: true,
			markAllReadFn: func(ctx context.Context, userID int) (int64, error) {
				return 0, errors.NewInternalError()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mocks.MockNotificationService{MarkAllReadFn: tt.markAllReadFn}
			handler := NewNotificationHandler(svc)

			req := httptest.NewRequest(http.MethodPatch, "/notifications/read-all", nil)
			if tt.withCtx {
				req = withUserContext(req, tt.userID)
			}
			w := httptest.NewRecorder()

			err := handler.MarkAllNotificationsRead(w, req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestNotificationHandler_DeleteNotification(t *testing.T) {
	tests := []struct {
		name       string
		userID     int
		withCtx    bool
		pathID     string
		deleteFn   func(ctx context.Context, userID int, id int) error
		wantStatus int
		wantErr    bool
	}{
		{
			name:    "success",
			userID:  1,
			withCtx: true,
			pathID:  "5",
			deleteFn: func(ctx context.Context, userID int, id int) error {
				return nil
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:    "no user context",
			withCtx: false,
			pathID:  "5",
			wantErr: true,
		},
		{
			name:    "invalid id",
			userID:  1,
			withCtx: true,
			pathID:  "abc",
			wantErr: true,
		},
		{
			name:    "not found",
			userID:  1,
			withCtx: true,
			pathID:  "999",
			deleteFn: func(ctx context.Context, userID int, id int) error {
				return errors.NewNotFoundError("Notification")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mocks.MockNotificationService{DeleteFn: tt.deleteFn}
			handler := NewNotificationHandler(svc)

			req := httptest.NewRequest(http.MethodDelete, "/notifications/"+tt.pathID, nil)
			req.SetPathValue("id", tt.pathID)
			if tt.withCtx {
				req = withUserContext(req, tt.userID)
			}
			w := httptest.NewRecorder()

			err := handler.DeleteNotification(w, req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

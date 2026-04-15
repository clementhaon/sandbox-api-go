package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/clementhaon/sandbox-api-go/mocks"
	"github.com/clementhaon/sandbox-api-go/models"
)

func TestNotificationService_List(t *testing.T) {
	tests := []struct {
		name    string
		userID  int
		listFn  func(ctx context.Context, userID int) ([]models.Notification, error)
		wantLen int
		wantErr bool
	}{
		{
			name:   "success",
			userID: 1,
			listFn: func(ctx context.Context, userID int) ([]models.Notification, error) {
				return []models.Notification{{ID: 1, Title: "Hello"}, {ID: 2, Title: "World"}}, nil
			},
			wantLen: 2,
		},
		{
			name:   "error",
			userID: 1,
			listFn: func(ctx context.Context, userID int) ([]models.Notification, error) {
				return nil, fmt.Errorf("db error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mocks.MockNotificationRepository{ListFn: tt.listFn}
			svc := NewNotificationService(repo, nil)

			notifs, err := svc.List(context.Background(), tt.userID)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(notifs) != tt.wantLen {
				t.Errorf("expected %d notifications, got %d", tt.wantLen, len(notifs))
			}
		})
	}
}

func TestNotificationService_MarkRead(t *testing.T) {
	tests := []struct {
		name       string
		userID     int
		ids        []int
		markReadFn func(ctx context.Context, userID int, notificationIDs []int) error
		wantCount  int
		wantErr    bool
	}{
		{
			name:   "success",
			userID: 1,
			ids:    []int{1, 2, 3},
			markReadFn: func(ctx context.Context, userID int, notificationIDs []int) error {
				return nil
			},
			wantCount: 3,
		},
		{
			name:   "repo error",
			userID: 1,
			ids:    []int{1},
			markReadFn: func(ctx context.Context, userID int, notificationIDs []int) error {
				return fmt.Errorf("db error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mocks.MockNotificationRepository{MarkReadFn: tt.markReadFn}
			svc := NewNotificationService(repo, nil)

			count, err := svc.MarkRead(context.Background(), tt.userID, tt.ids)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if count != tt.wantCount {
				t.Errorf("expected count %d, got %d", tt.wantCount, count)
			}
		})
	}
}

func TestNotificationService_MarkAllRead(t *testing.T) {
	tests := []struct {
		name          string
		userID        int
		markAllReadFn func(ctx context.Context, userID int) (int64, error)
		wantCount     int64
		wantErr       bool
	}{
		{
			name:   "success",
			userID: 1,
			markAllReadFn: func(ctx context.Context, userID int) (int64, error) {
				return 5, nil
			},
			wantCount: 5,
		},
		{
			name:   "error",
			userID: 1,
			markAllReadFn: func(ctx context.Context, userID int) (int64, error) {
				return 0, fmt.Errorf("db error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mocks.MockNotificationRepository{MarkAllReadFn: tt.markAllReadFn}
			svc := NewNotificationService(repo, nil)

			count, err := svc.MarkAllRead(context.Background(), tt.userID)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if count != tt.wantCount {
				t.Errorf("expected count %d, got %d", tt.wantCount, count)
			}
		})
	}
}

func TestNotificationService_Delete(t *testing.T) {
	tests := []struct {
		name     string
		userID   int
		id       int
		deleteFn func(ctx context.Context, userID int, id int) error
		wantErr  bool
	}{
		{
			name:   "success",
			userID: 1,
			id:     5,
			deleteFn: func(ctx context.Context, userID int, id int) error {
				return nil
			},
		},
		{
			name:   "error",
			userID: 1,
			id:     5,
			deleteFn: func(ctx context.Context, userID int, id int) error {
				return fmt.Errorf("not found")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mocks.MockNotificationRepository{DeleteFn: tt.deleteFn}
			svc := NewNotificationService(repo, nil)

			err := svc.Delete(context.Background(), tt.userID, tt.id)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestNotificationService_Create(t *testing.T) {
	tests := []struct {
		name     string
		userID   int
		createFn func(ctx context.Context, userID int, notifType, title, message string, dataJSON []byte) error
		wantErr  bool
	}{
		{
			name:   "success",
			userID: 1,
			createFn: func(ctx context.Context, userID int, notifType, title, message string, dataJSON []byte) error {
				return nil
			},
		},
		{
			name:   "repo error",
			userID: 1,
			createFn: func(ctx context.Context, userID int, notifType, title, message string, dataJSON []byte) error {
				return fmt.Errorf("db error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mocks.MockNotificationRepository{CreateFn: tt.createFn}
			svc := NewNotificationService(repo, nil)

			data := models.NotificationData{TaskID: 1, TaskTitle: "Test Task"}
			err := svc.Create(context.Background(), tt.userID, "task_assigned", "New Task", "You were assigned", data)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

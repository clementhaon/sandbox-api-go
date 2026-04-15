package services

import (
	"context"
	"testing"
	"time"

	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/mocks"
	"github.com/clementhaon/sandbox-api-go/models"
)

func TestTimeEntryService_Create_Success(t *testing.T) {
	var addedMinutes int
	repo := &mocks.MockTimeEntryRepository{
		TaskExistsFn: func(ctx context.Context, taskID int) (bool, error) {
			return true, nil
		},
		CreateFn: func(ctx context.Context, userID int, req models.CreateTimeEntryRequest) (models.TimeEntry, error) {
			return models.TimeEntry{
				ID:       1,
				TaskID:   req.TaskID,
				UserID:   userID,
				Duration: req.Duration,
			}, nil
		},
		AddTrackedTimeFn: func(ctx context.Context, taskID int, durationMinutes int) error {
			addedMinutes = durationMinutes
			return nil
		},
	}

	svc := NewTimeEntryService(repo, &mocks.MockTransactor{})
	entry, err := svc.Create(context.Background(), 42, models.CreateTimeEntryRequest{
		TaskID:    1,
		StartTime: time.Now(),
		Duration:  3600, // 1 hour in seconds
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.UserID != 42 {
		t.Errorf("expected userID 42, got %d", entry.UserID)
	}
	if addedMinutes != 60 {
		t.Errorf("expected 60 minutes added, got %d", addedMinutes)
	}
}

func TestTimeEntryService_Create_TaskNotFound(t *testing.T) {
	repo := &mocks.MockTimeEntryRepository{
		TaskExistsFn: func(ctx context.Context, taskID int) (bool, error) {
			return false, nil
		},
	}

	svc := NewTimeEntryService(repo, &mocks.MockTransactor{})
	_, err := svc.Create(context.Background(), 1, models.CreateTimeEntryRequest{
		TaskID:    999,
		StartTime: time.Now(),
		Duration:  60,
	})
	if err == nil {
		t.Fatal("expected not found error")
	}
	appErr, ok := errors.IsAppError(err)
	if !ok {
		t.Fatal("expected AppError")
	}
	if appErr.Code != errors.ErrNotFound {
		t.Errorf("expected NOT_FOUND, got %s", appErr.Code)
	}
}

func TestTimeEntryService_Create_ValidationErrors(t *testing.T) {
	repo := &mocks.MockTimeEntryRepository{}
	svc := NewTimeEntryService(repo, &mocks.MockTransactor{})

	tests := []struct {
		name string
		req  models.CreateTimeEntryRequest
	}{
		{"missing taskId", models.CreateTimeEntryRequest{TaskID: 0, StartTime: time.Now(), Duration: 60}},
		{"missing startTime", models.CreateTimeEntryRequest{TaskID: 1, Duration: 60}},
		{"zero duration", models.CreateTimeEntryRequest{TaskID: 1, StartTime: time.Now(), Duration: 0}},
		{"negative duration", models.CreateTimeEntryRequest{TaskID: 1, StartTime: time.Now(), Duration: -1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Create(context.Background(), 1, tt.req)
			if err == nil {
				t.Error("expected validation error")
			}
		})
	}
}

func TestTimeEntryService_Delete_Success(t *testing.T) {
	var subtractedMinutes int
	repo := &mocks.MockTimeEntryRepository{
		GetTaskIDAndDurationFn: func(ctx context.Context, id int) (int, int, error) {
			return 5, 1800, nil // task 5, 1800 seconds = 30 minutes
		},
		DeleteFn: func(ctx context.Context, id int) error {
			return nil
		},
		SubtractTrackedTimeFn: func(ctx context.Context, taskID int, durationMinutes int) error {
			subtractedMinutes = durationMinutes
			return nil
		},
	}

	svc := NewTimeEntryService(repo, &mocks.MockTransactor{})
	err := svc.Delete(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subtractedMinutes != 30 {
		t.Errorf("expected 30 minutes subtracted, got %d", subtractedMinutes)
	}
}

func TestTimeEntryService_Delete_NotFound(t *testing.T) {
	repo := &mocks.MockTimeEntryRepository{
		GetTaskIDAndDurationFn: func(ctx context.Context, id int) (int, int, error) {
			return 0, 0, errors.NewNotFoundError("Time entry not found")
		},
	}

	svc := NewTimeEntryService(repo, &mocks.MockTransactor{})
	err := svc.Delete(context.Background(), 999)
	if err == nil {
		t.Fatal("expected not found error")
	}
}

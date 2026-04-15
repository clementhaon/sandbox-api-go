package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/mocks"
	"github.com/clementhaon/sandbox-api-go/models"
)

func TestTimeEntryHandler_ListTimeEntries(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		listFn     func(ctx context.Context, taskID int) ([]models.TimeEntry, error)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "success",
			url:  "/time-entries?taskId=1",
			listFn: func(ctx context.Context, taskID int) ([]models.TimeEntry, error) {
				return []models.TimeEntry{{ID: 1, TaskID: taskID}}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:    "missing taskId",
			url:     "/time-entries",
			wantErr: true,
		},
		{
			name:    "invalid taskId",
			url:     "/time-entries?taskId=abc",
			wantErr: true,
		},
		{
			name: "service error",
			url:  "/time-entries?taskId=1",
			listFn: func(ctx context.Context, taskID int) ([]models.TimeEntry, error) {
				return nil, errors.NewInternalError()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mocks.MockTimeEntryService{ListFn: tt.listFn}
			handler := NewTimeEntryHandler(svc)

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			err := handler.ListTimeEntries(w, req)
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

func TestTimeEntryHandler_CreateTimeEntry(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name       string
		userID     int
		withCtx    bool
		body       interface{}
		createFn   func(ctx context.Context, userID int, req models.CreateTimeEntryRequest) (models.TimeEntry, error)
		wantStatus int
		wantErr    bool
	}{
		{
			name:    "success",
			userID:  42,
			withCtx: true,
			body:    models.CreateTimeEntryRequest{TaskID: 1, StartTime: now, Duration: 3600},
			createFn: func(ctx context.Context, userID int, req models.CreateTimeEntryRequest) (models.TimeEntry, error) {
				return models.TimeEntry{ID: 1, TaskID: req.TaskID, UserID: userID, Duration: req.Duration}, nil
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:    "no user context",
			withCtx: false,
			body:    models.CreateTimeEntryRequest{TaskID: 1},
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
			name:    "service error",
			userID:  1,
			withCtx: true,
			body:    models.CreateTimeEntryRequest{TaskID: 1, StartTime: now, Duration: 60},
			createFn: func(ctx context.Context, userID int, req models.CreateTimeEntryRequest) (models.TimeEntry, error) {
				return models.TimeEntry{}, errors.NewNotFoundError("Task")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mocks.MockTimeEntryService{CreateFn: tt.createFn}
			handler := NewTimeEntryHandler(svc)

			var bodyBytes []byte
			switch v := tt.body.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				bodyBytes, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPost, "/time-entries", bytes.NewReader(bodyBytes))
			if tt.withCtx {
				req = withUserContext(req, tt.userID)
			}
			w := httptest.NewRecorder()

			err := handler.CreateTimeEntry(w, req)
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

func TestTimeEntryHandler_DeleteTimeEntry(t *testing.T) {
	tests := []struct {
		name       string
		pathID     string
		deleteFn   func(ctx context.Context, id int) error
		wantStatus int
		wantErr    bool
	}{
		{
			name:   "success",
			pathID: "5",
			deleteFn: func(ctx context.Context, id int) error {
				return nil
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:    "invalid id",
			pathID:  "abc",
			wantErr: true,
		},
		{
			name:   "not found",
			pathID: "999",
			deleteFn: func(ctx context.Context, id int) error {
				return errors.NewNotFoundError("Time entry")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mocks.MockTimeEntryService{DeleteFn: tt.deleteFn}
			handler := NewTimeEntryHandler(svc)

			req := httptest.NewRequest(http.MethodDelete, "/time-entries/"+tt.pathID, nil)
			req.SetPathValue("id", tt.pathID)
			w := httptest.NewRecorder()

			err := handler.DeleteTimeEntry(w, req)
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

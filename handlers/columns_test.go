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

func TestColumnHandler_ListColumns(t *testing.T) {
	tests := []struct {
		name       string
		listFn     func(ctx context.Context) ([]models.Column, error)
		wantStatus int
		wantCount  int
		wantErr    bool
	}{
		{
			name: "success",
			listFn: func(ctx context.Context) ([]models.Column, error) {
				return []models.Column{{ID: 1, Title: "To Do"}, {ID: 2, Title: "Done"}}, nil
			},
			wantStatus: http.StatusOK,
			wantCount:  2,
		},
		{
			name: "service error",
			listFn: func(ctx context.Context) ([]models.Column, error) {
				return nil, errors.NewInternalError()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mocks.MockColumnService{ListFn: tt.listFn}
			handler := NewColumnHandler(svc)

			req := httptest.NewRequest(http.MethodGet, "/columns", nil)
			w := httptest.NewRecorder()

			err := handler.ListColumns(w, req)
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

			var cols []models.Column
			json.NewDecoder(w.Body).Decode(&cols)
			if len(cols) != tt.wantCount {
				t.Errorf("expected %d columns, got %d", tt.wantCount, len(cols))
			}
		})
	}
}

func TestColumnHandler_CreateColumn(t *testing.T) {
	tests := []struct {
		name       string
		body       interface{}
		createFn   func(ctx context.Context, req models.CreateColumnRequest) (models.Column, error)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "success",
			body: models.CreateColumnRequest{Title: "In Progress", Color: "#ff0000"},
			createFn: func(ctx context.Context, req models.CreateColumnRequest) (models.Column, error) {
				return models.Column{ID: 1, Title: req.Title, Color: req.Color}, nil
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:    "invalid json",
			body:    "bad",
			wantErr: true,
		},
		{
			name:    "empty title",
			body:    models.CreateColumnRequest{Title: "", Color: "#000"},
			wantErr: true,
		},
		{
			name: "service error",
			body: models.CreateColumnRequest{Title: "Test"},
			createFn: func(ctx context.Context, req models.CreateColumnRequest) (models.Column, error) {
				return models.Column{}, errors.NewInternalError()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mocks.MockColumnService{CreateFn: tt.createFn}
			handler := NewColumnHandler(svc)

			var bodyBytes []byte
			switch v := tt.body.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				bodyBytes, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPost, "/columns", bytes.NewReader(bodyBytes))
			w := httptest.NewRecorder()

			err := handler.CreateColumn(w, req)
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

func TestColumnHandler_UpdateColumn(t *testing.T) {
	tests := []struct {
		name       string
		pathID     string
		body       interface{}
		updateFn   func(ctx context.Context, id int, req models.UpdateColumnRequest) (models.Column, error)
		wantStatus int
		wantErr    bool
	}{
		{
			name:   "success",
			pathID: "1",
			body:   models.UpdateColumnRequest{Title: "Updated"},
			updateFn: func(ctx context.Context, id int, req models.UpdateColumnRequest) (models.Column, error) {
				return models.Column{ID: id, Title: req.Title}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:    "invalid id",
			pathID:  "abc",
			body:    models.UpdateColumnRequest{},
			wantErr: true,
		},
		{
			name:    "invalid json",
			pathID:  "1",
			body:    "bad",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mocks.MockColumnService{UpdateFn: tt.updateFn}
			handler := NewColumnHandler(svc)

			var bodyBytes []byte
			switch v := tt.body.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				bodyBytes, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPut, "/columns/"+tt.pathID, bytes.NewReader(bodyBytes))
			req.SetPathValue("id", tt.pathID)
			w := httptest.NewRecorder()

			err := handler.UpdateColumn(w, req)
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

func TestColumnHandler_DeleteColumn(t *testing.T) {
	tests := []struct {
		name       string
		pathID     string
		deleteFn   func(ctx context.Context, id int) error
		wantStatus int
		wantErr    bool
	}{
		{
			name:   "success",
			pathID: "3",
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
				return errors.NewNotFoundError("Column")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mocks.MockColumnService{DeleteFn: tt.deleteFn}
			handler := NewColumnHandler(svc)

			req := httptest.NewRequest(http.MethodDelete, "/columns/"+tt.pathID, nil)
			req.SetPathValue("id", tt.pathID)
			w := httptest.NewRecorder()

			err := handler.DeleteColumn(w, req)
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

func TestColumnHandler_ReorderColumns(t *testing.T) {
	tests := []struct {
		name       string
		body       interface{}
		reorderFn  func(ctx context.Context, columnIDs []int) ([]models.Column, error)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "success",
			body: models.ReorderColumnsRequest{ColumnIDs: []int{3, 1, 2}},
			reorderFn: func(ctx context.Context, columnIDs []int) ([]models.Column, error) {
				return []models.Column{{ID: 3}, {ID: 1}, {ID: 2}}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:    "invalid json",
			body:    "bad",
			wantErr: true,
		},
		{
			name:    "empty column ids",
			body:    models.ReorderColumnsRequest{ColumnIDs: []int{}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mocks.MockColumnService{ReorderFn: tt.reorderFn}
			handler := NewColumnHandler(svc)

			var bodyBytes []byte
			switch v := tt.body.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				bodyBytes, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPatch, "/columns/reorder", bytes.NewReader(bodyBytes))
			w := httptest.NewRecorder()

			err := handler.ReorderColumns(w, req)
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

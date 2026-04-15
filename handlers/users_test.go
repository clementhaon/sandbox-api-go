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

func TestUserHandler_ListUsers(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		listFn     func(ctx context.Context, params models.UserListParams) (models.UsersListResponse, error)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "success",
			url:  "/users?page=1&pageSize=10",
			listFn: func(ctx context.Context, params models.UserListParams) (models.UsersListResponse, error) {
				return models.UsersListResponse{
					Data:       []models.UserResponse{{ID: 1, Username: "alice"}},
					Pagination: models.Pagination{Page: 1, PageSize: 10, Total: 1, TotalPages: 1},
				}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "service error",
			url:  "/users",
			listFn: func(ctx context.Context, params models.UserListParams) (models.UsersListResponse, error) {
				return models.UsersListResponse{}, errors.NewInternalError()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mocks.MockUserService{ListFn: tt.listFn}
			handler := NewUserHandler(svc)

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			err := handler.ListUsers(w, req)
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

func TestUserHandler_GetUser(t *testing.T) {
	tests := []struct {
		name       string
		pathID     string
		getByIDFn  func(ctx context.Context, id int) (models.UserResponse, error)
		wantStatus int
		wantErr    bool
		wantCode   errors.ErrorCode
	}{
		{
			name:   "success",
			pathID: "1",
			getByIDFn: func(ctx context.Context, id int) (models.UserResponse, error) {
				return models.UserResponse{ID: 1, Username: "alice"}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:    "invalid id",
			pathID:  "abc",
			wantErr: true,
		},
		{
			name:   "not found",
			pathID: "999",
			getByIDFn: func(ctx context.Context, id int) (models.UserResponse, error) {
				return models.UserResponse{}, errors.NewNotFoundError("User")
			},
			wantErr:  true,
			wantCode: errors.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mocks.MockUserService{GetByIDFn: tt.getByIDFn}
			handler := NewUserHandler(svc)

			req := httptest.NewRequest(http.MethodGet, "/users/"+tt.pathID, nil)
			req.SetPathValue("id", tt.pathID)
			w := httptest.NewRecorder()

			err := handler.GetUser(w, req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantCode != "" {
					appErr, ok := errors.IsAppError(err)
					if !ok {
						t.Fatal("expected AppError")
					}
					if appErr.Code != tt.wantCode {
						t.Errorf("expected code %s, got %s", tt.wantCode, appErr.Code)
					}
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

func TestUserHandler_CreateUser(t *testing.T) {
	tests := []struct {
		name       string
		body       interface{}
		createFn   func(ctx context.Context, req models.CreateUserRequest) (models.UserResponse, error)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "success",
			body: models.CreateUserRequest{Email: "a@b.com", Username: "alice", Password: "pass123"},
			createFn: func(ctx context.Context, req models.CreateUserRequest) (models.UserResponse, error) {
				return models.UserResponse{ID: 1, Username: req.Username, Email: req.Email}, nil
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:    "invalid json",
			body:    "not json",
			wantErr: true,
		},
		{
			name: "service error",
			body: models.CreateUserRequest{Email: "a@b.com", Username: "alice", Password: "pass123"},
			createFn: func(ctx context.Context, req models.CreateUserRequest) (models.UserResponse, error) {
				return models.UserResponse{}, errors.NewUserExistsError()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mocks.MockUserService{CreateFn: tt.createFn}
			handler := NewUserHandler(svc)

			var bodyBytes []byte
			switch v := tt.body.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				bodyBytes, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(bodyBytes))
			w := httptest.NewRecorder()

			err := handler.CreateUser(w, req)
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

func TestUserHandler_UpdateUser(t *testing.T) {
	tests := []struct {
		name       string
		pathID     string
		body       interface{}
		updateFn   func(ctx context.Context, id int, req models.UpdateUserRequest) (models.UserResponse, error)
		wantStatus int
		wantErr    bool
	}{
		{
			name:   "success",
			pathID: "1",
			body:   models.UpdateUserRequest{Username: "newname"},
			updateFn: func(ctx context.Context, id int, req models.UpdateUserRequest) (models.UserResponse, error) {
				return models.UserResponse{ID: id, Username: req.Username}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:    "invalid id",
			pathID:  "abc",
			body:    models.UpdateUserRequest{},
			wantErr: true,
		},
		{
			name:    "invalid json",
			pathID:  "1",
			body:    "bad json",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mocks.MockUserService{UpdateFn: tt.updateFn}
			handler := NewUserHandler(svc)

			var bodyBytes []byte
			switch v := tt.body.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				bodyBytes, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPut, "/users/"+tt.pathID, bytes.NewReader(bodyBytes))
			req.SetPathValue("id", tt.pathID)
			w := httptest.NewRecorder()

			err := handler.UpdateUser(w, req)
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

func TestUserHandler_UpdateUserStatus(t *testing.T) {
	tests := []struct {
		name           string
		pathID         string
		body           interface{}
		updateStatusFn func(ctx context.Context, id int, status string) (models.UserResponse, error)
		wantStatus     int
		wantErr        bool
	}{
		{
			name:   "success",
			pathID: "1",
			body:   models.UpdateUserStatusRequest{Status: "active"},
			updateStatusFn: func(ctx context.Context, id int, status string) (models.UserResponse, error) {
				return models.UserResponse{ID: id, Status: status}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:    "invalid id",
			pathID:  "abc",
			body:    models.UpdateUserStatusRequest{Status: "active"},
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
			svc := &mocks.MockUserService{UpdateStatusFn: tt.updateStatusFn}
			handler := NewUserHandler(svc)

			var bodyBytes []byte
			switch v := tt.body.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				bodyBytes, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPatch, "/users/"+tt.pathID+"/status", bytes.NewReader(bodyBytes))
			req.SetPathValue("id", tt.pathID)
			w := httptest.NewRecorder()

			err := handler.UpdateUserStatus(w, req)
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

func TestUserHandler_DeleteUser(t *testing.T) {
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
				return errors.NewNotFoundError("User")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mocks.MockUserService{DeleteFn: tt.deleteFn}
			handler := NewUserHandler(svc)

			req := httptest.NewRequest(http.MethodDelete, "/users/"+tt.pathID, nil)
			req.SetPathValue("id", tt.pathID)
			w := httptest.NewRecorder()

			err := handler.DeleteUser(w, req)
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

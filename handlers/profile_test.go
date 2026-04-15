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

func TestProfileHandler_HandleGetProfile(t *testing.T) {
	tests := []struct {
		name         string
		userID       int
		withContext   bool
		getProfileFn func(ctx context.Context, userID int) (models.User, error)
		wantStatus   int
		wantErr      bool
	}{
		{
			name:       "success",
			userID:     1,
			withContext: true,
			getProfileFn: func(ctx context.Context, userID int) (models.User, error) {
				return models.User{ID: userID, Username: "alice"}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "no user context",
			withContext: false,
			wantErr:    true,
		},
		{
			name:       "not found",
			userID:     999,
			withContext: true,
			getProfileFn: func(ctx context.Context, userID int) (models.User, error) {
				return models.User{}, errors.NewNotFoundError("User")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mocks.MockProfileService{GetProfileFn: tt.getProfileFn}
			handler := NewProfileHandler(svc)

			req := httptest.NewRequest(http.MethodGet, "/profile", nil)
			if tt.withContext {
				req = withUserContext(req, tt.userID)
			}
			w := httptest.NewRecorder()

			err := handler.HandleGetProfile(w, req)
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

func TestProfileHandler_HandleUpdateProfile(t *testing.T) {
	firstName := "Alice"
	tests := []struct {
		name            string
		userID          int
		withContext      bool
		body            interface{}
		updateProfileFn func(ctx context.Context, userID int, req models.UpdateProfileRequest) (models.User, error)
		wantStatus      int
		wantErr         bool
	}{
		{
			name:       "success",
			userID:     1,
			withContext: true,
			body:       models.UpdateProfileRequest{FirstName: &firstName},
			updateProfileFn: func(ctx context.Context, userID int, req models.UpdateProfileRequest) (models.User, error) {
				return models.User{ID: userID, Username: "alice"}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "no user context",
			withContext: false,
			body:       models.UpdateProfileRequest{},
			wantErr:    true,
		},
		{
			name:       "invalid json",
			userID:     1,
			withContext: true,
			body:       "bad json",
			wantErr:    true,
		},
		{
			name:       "service error",
			userID:     1,
			withContext: true,
			body:       models.UpdateProfileRequest{FirstName: &firstName},
			updateProfileFn: func(ctx context.Context, userID int, req models.UpdateProfileRequest) (models.User, error) {
				return models.User{}, errors.NewInternalError()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mocks.MockProfileService{UpdateProfileFn: tt.updateProfileFn}
			handler := NewProfileHandler(svc)

			var bodyBytes []byte
			switch v := tt.body.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				bodyBytes, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPut, "/profile", bytes.NewReader(bodyBytes))
			if tt.withContext {
				req = withUserContext(req, tt.userID)
			}
			w := httptest.NewRecorder()

			err := handler.HandleUpdateProfile(w, req)
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

			var resp map[string]interface{}
			json.NewDecoder(w.Body).Decode(&resp)
			if resp["message"] != "Profile updated successfully" {
				t.Errorf("expected success message, got %v", resp["message"])
			}
		})
	}
}

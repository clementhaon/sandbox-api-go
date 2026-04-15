package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestErrorConstructors(t *testing.T) {
	tests := []struct {
		name       string
		fn         func() *AppError
		wantCode   ErrorCode
		wantStatus int
		wantType   ErrorType
	}{
		{
			name:       "NewAuthRequiredError",
			fn:         NewAuthRequiredError,
			wantCode:   ErrAuthRequired,
			wantStatus: http.StatusUnauthorized,
			wantType:   ErrorTypeClient,
		},
		{
			name:       "NewInvalidTokenError",
			fn:         NewInvalidTokenError,
			wantCode:   ErrInvalidToken,
			wantStatus: http.StatusUnauthorized,
			wantType:   ErrorTypeClient,
		},
		{
			name:       "NewTokenExpiredError",
			fn:         NewTokenExpiredError,
			wantCode:   ErrTokenExpired,
			wantStatus: http.StatusUnauthorized,
			wantType:   ErrorTypeClient,
		},
		{
			name:       "NewInvalidCredentialsError",
			fn:         NewInvalidCredentialsError,
			wantCode:   ErrInvalidCredentials,
			wantStatus: http.StatusUnauthorized,
			wantType:   ErrorTypeClient,
		},
		{
			name:       "NewUserExistsError",
			fn:         NewUserExistsError,
			wantCode:   ErrUserExists,
			wantStatus: http.StatusConflict,
			wantType:   ErrorTypeClient,
		},
		{
			name:       "NewInvalidJSONError",
			fn:         NewInvalidJSONError,
			wantCode:   ErrInvalidJSON,
			wantStatus: http.StatusBadRequest,
			wantType:   ErrorTypeClient,
		},
		{
			name:       "NewNotFoundError",
			fn:         func() *AppError { return NewNotFoundError("Task") },
			wantCode:   ErrNotFound,
			wantStatus: http.StatusNotFound,
			wantType:   ErrorTypeClient,
		},
		{
			name:       "NewForbiddenError",
			fn:         NewForbiddenError,
			wantCode:   ErrForbidden,
			wantStatus: http.StatusForbidden,
			wantType:   ErrorTypeClient,
		},
		{
			name:       "NewInternalError",
			fn:         NewInternalError,
			wantCode:   ErrInternal,
			wantStatus: http.StatusInternalServerError,
			wantType:   ErrorTypeServer,
		},
		{
			name:       "NewDatabaseError",
			fn:         NewDatabaseError,
			wantCode:   ErrDatabase,
			wantStatus: http.StatusInternalServerError,
			wantType:   ErrorTypeServer,
		},
		{
			name:       "NewServiceUnavailableError",
			fn:         NewServiceUnavailableError,
			wantCode:   ErrServiceUnavailable,
			wantStatus: http.StatusServiceUnavailable,
			wantType:   ErrorTypeServer,
		},
		{
			name:       "NewTooManyRequestsError",
			fn:         NewTooManyRequestsError,
			wantCode:   ErrTooManyRequests,
			wantStatus: http.StatusTooManyRequests,
			wantType:   ErrorTypeClient,
		},
		{
			name:       "NewMethodNotAllowedError",
			fn:         NewMethodNotAllowedError,
			wantCode:   ErrMethodNotAllowed,
			wantStatus: http.StatusMethodNotAllowed,
			wantType:   ErrorTypeClient,
		},
		{
			name:       "NewPayloadTooLargeError",
			fn:         NewPayloadTooLargeError,
			wantCode:   ErrPayloadTooLarge,
			wantStatus: http.StatusRequestEntityTooLarge,
			wantType:   ErrorTypeClient,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", err.Code, tt.wantCode)
			}
			if err.StatusCode != tt.wantStatus {
				t.Errorf("StatusCode = %d, want %d", err.StatusCode, tt.wantStatus)
			}
			if err.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", err.Type, tt.wantType)
			}
			if err.Message == "" {
				t.Error("Message should not be empty")
			}
			if err.Timestamp.IsZero() {
				t.Error("Timestamp should be set")
			}
		})
	}
}

func TestAppError_Error(t *testing.T) {
	t.Run("without cause", func(t *testing.T) {
		err := NewNotFoundError("User")
		got := err.Error()
		want := "NOT_FOUND: User not found"
		if got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("with cause", func(t *testing.T) {
		cause := fmt.Errorf("connection refused")
		err := NewDatabaseError().WithCause(cause)
		got := err.Error()
		if !strings.Contains(got, "DATABASE_ERROR") {
			t.Errorf("expected error string to contain code, got %q", got)
		}
		if !strings.Contains(got, "caused by: connection refused") {
			t.Errorf("expected error string to contain cause, got %q", got)
		}
	})
}

func TestAppError_WithCause(t *testing.T) {
	cause := fmt.Errorf("root cause")
	err := NewInternalError().WithCause(cause)
	if err.Cause != cause {
		t.Error("WithCause should set Cause field")
	}
}

func TestAppError_WithRequestID(t *testing.T) {
	err := NewInternalError().WithRequestID("req-123")
	if err.RequestID != "req-123" {
		t.Errorf("RequestID = %q, want %q", err.RequestID, "req-123")
	}
}

func TestIsAppError(t *testing.T) {
	t.Run("returns true for AppError", func(t *testing.T) {
		appErr := NewNotFoundError("Item")
		got, ok := IsAppError(appErr)
		if !ok {
			t.Fatal("expected ok to be true")
		}
		if got != appErr {
			t.Error("expected returned error to match input")
		}
	})

	t.Run("returns false for regular error", func(t *testing.T) {
		regularErr := fmt.Errorf("some error")
		got, ok := IsAppError(regularErr)
		if ok {
			t.Fatal("expected ok to be false")
		}
		if got != nil {
			t.Error("expected nil AppError for regular error")
		}
	})
}

func TestWriteError(t *testing.T) {
	t.Run("writes correct status code and JSON", func(t *testing.T) {
		appErr := NewNotFoundError("Task").WithRequestID("req-456")
		rec := httptest.NewRecorder()

		WriteError(rec, appErr)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status code = %d, want %d", rec.Code, http.StatusNotFound)
		}

		ct := rec.Header().Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("Content-Type = %q, want %q", ct, "application/json")
		}

		var resp ErrorResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Success {
			t.Error("expected success to be false")
		}
		if resp.Error == nil {
			t.Fatal("expected error field to be non-nil")
		}
		if resp.Error.Code != ErrNotFound {
			t.Errorf("error code = %q, want %q", resp.Error.Code, ErrNotFound)
		}
		if resp.Error.RequestID != "req-456" {
			t.Errorf("request_id = %q, want %q", resp.Error.RequestID, "req-456")
		}
	})
}

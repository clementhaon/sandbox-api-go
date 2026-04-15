package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clementhaon/sandbox-api-go/errors"
)

func TestErrorMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		handler    ErrorHandler
		wantStatus int
		wantError  bool
	}{
		{
			name: "handler returns nil gives 200 OK",
			handler: func(w http.ResponseWriter, r *http.Request) error {
				w.WriteHeader(http.StatusOK)
				return nil
			},
			wantStatus: http.StatusOK,
			wantError:  false,
		},
		{
			name: "handler returns AppError gives correct status",
			handler: func(w http.ResponseWriter, r *http.Request) error {
				return errors.NewNotFoundError("Widget")
			},
			wantStatus: http.StatusNotFound,
			wantError:  true,
		},
		{
			name: "handler returns generic error gives 500",
			handler: func(w http.ResponseWriter, r *http.Request) error {
				return fmt.Errorf("something went wrong")
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wrapped := ErrorMiddleware(tc.handler)
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			wrapped.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("got status %d, want %d", rec.Code, tc.wantStatus)
			}

			if tc.wantError {
				var body map[string]interface{}
				if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if _, ok := body["error"]; !ok {
					t.Error("expected error field in response")
				}
			}

			if rid := rec.Header().Get("X-Request-ID"); rid == "" {
				t.Error("expected X-Request-ID header to be set")
			}
		})
	}
}

func TestPanicRecoveryMiddleware(t *testing.T) {
	panicking := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	handler := PanicRecoveryMiddleware(panicking)

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if _, ok := body["error"]; !ok {
		t.Error("expected error field in response after panic recovery")
	}
}

func TestErrorMiddleware_MaxBytesError(t *testing.T) {
	handler := ErrorHandler(func(w http.ResponseWriter, r *http.Request) error {
		return &http.MaxBytesError{Limit: 1024}
	})

	wrapped := ErrorMiddleware(handler)
	req := httptest.NewRequest(http.MethodPost, "/upload", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	errObj, ok := body["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected error object in response")
	}
	if code, _ := errObj["code"].(string); code != string(errors.ErrPayloadTooLarge) {
		t.Errorf("got error code %q, want %q", code, errors.ErrPayloadTooLarge)
	}
}

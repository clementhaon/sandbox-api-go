package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMaxBytesMiddleware(t *testing.T) {
	t.Run("request body under limit passes through", func(t *testing.T) {
		var bodyRead string
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("unexpected error reading body: %v", err)
			}
			bodyRead = string(b)
			w.WriteHeader(http.StatusOK)
		})

		handler := MaxBytesMiddleware(1024)(inner)

		body := strings.NewReader("hello")
		req := httptest.NewRequest(http.MethodPost, "/test", body)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("got status %d, want %d", rec.Code, http.StatusOK)
		}
		if bodyRead != "hello" {
			t.Errorf("got body %q, want %q", bodyRead, "hello")
		}
	})

	t.Run("request body over limit causes error when reading", func(t *testing.T) {
		var readErr error
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, readErr = io.ReadAll(r.Body)
		})

		handler := MaxBytesMiddleware(5)(inner)

		body := strings.NewReader("this body is way too long for the limit")
		req := httptest.NewRequest(http.MethodPost, "/test", body)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if readErr == nil {
			t.Error("expected error when reading oversized body, got nil")
		}
	})
}

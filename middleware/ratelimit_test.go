package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	t.Run("first request is allowed", func(t *testing.T) {
		rl := NewRateLimiter(5, time.Minute)
		defer rl.Stop()

		handler := rl.Limit(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("got status %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("requests within burst limit are allowed", func(t *testing.T) {
		burst := 5
		rl := NewRateLimiter(burst, time.Minute)
		defer rl.Stop()

		handler := rl.Limit(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		for i := 0; i < burst; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "10.0.0.1:12345"
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("request %d: got status %d, want %d", i+1, rec.Code, http.StatusOK)
			}
		}
	})

	t.Run("requests exceeding burst return 429", func(t *testing.T) {
		burst := 3
		rl := NewRateLimiter(burst, time.Minute)
		defer rl.Stop()

		handler := rl.Limit(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// Exhaust the burst
		for i := 0; i < burst; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "10.0.0.2:12345"
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}

		// Next request should be rate limited
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.2:12345"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusTooManyRequests {
			t.Errorf("got status %d, want %d", rec.Code, http.StatusTooManyRequests)
		}
	})

	t.Run("stop does not panic", func(t *testing.T) {
		rl := NewRateLimiter(10, time.Second)
		rl.Stop()
		// If we get here without panicking, the test passes.
	})
}

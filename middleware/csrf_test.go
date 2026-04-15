package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCSRFMiddleware(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name       string
		method     string
		path       string
		cookie     string
		header     string
		wantStatus int
	}{
		{
			name:       "GET requests pass without CSRF token",
			method:     http.MethodGet,
			path:       "/tasks",
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST without CSRF token is rejected",
			method:     http.MethodPost,
			path:       "/tasks",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "POST with valid matching cookie and header passes",
			method:     http.MethodPost,
			path:       "/tasks",
			cookie:     "valid-csrf-token",
			header:     "valid-csrf-token",
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST with mismatched cookie and header is rejected",
			method:     http.MethodPost,
			path:       "/tasks",
			cookie:     "token-a",
			header:     "token-b",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "exempt path /auth/login passes without CSRF",
			method:     http.MethodPost,
			path:       "/auth/login",
			wantStatus: http.StatusOK,
		},
		{
			name:       "exempt path /auth/register passes without CSRF",
			method:     http.MethodPost,
			path:       "/auth/register",
			wantStatus: http.StatusOK,
		},
		{
			name:       "exempt path /auth/logout passes without CSRF",
			method:     http.MethodPost,
			path:       "/auth/logout",
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := CSRFMiddleware(ok)

			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "csrf_token", Value: tc.cookie})
			}
			if tc.header != "" {
				req.Header.Set("X-CSRF-Token", tc.header)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("got status %d, want %d", rec.Code, tc.wantStatus)
			}
		})
	}
}

func TestGenerateCSRFToken(t *testing.T) {
	token := GenerateCSRFToken()
	if token == "" {
		t.Error("expected non-empty CSRF token")
	}
	if len(token) != 64 { // 32 bytes hex-encoded
		t.Errorf("expected token length 64, got %d", len(token))
	}
}

func TestSetCSRFCookie(t *testing.T) {
	rec := httptest.NewRecorder()
	token := SetCSRFCookie(rec, false)

	if token == "" {
		t.Error("expected non-empty token from SetCSRFCookie")
	}

	cookies := rec.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "csrf_token" {
			found = true
			if c.Value != token {
				t.Errorf("cookie value %q does not match returned token %q", c.Value, token)
			}
			if c.Secure {
				t.Error("expected Secure=false for non-production")
			}
			break
		}
	}
	if !found {
		t.Error("csrf_token cookie not found in response")
	}
}

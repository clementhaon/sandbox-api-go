package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/clementhaon/sandbox-api-go/auth"
	"github.com/clementhaon/sandbox-api-go/models"
)

const testSecret = "test-secret-key-minimum-16-chars"

func newTestJWTManager(t *testing.T) *auth.JWTManager {
	t.Helper()
	mgr, err := auth.NewJWTManager(testSecret)
	if err != nil {
		t.Fatalf("failed to create JWTManager: %v", err)
	}
	return mgr
}

func generateTestToken(t *testing.T, mgr *auth.JWTManager) string {
	t.Helper()
	user := models.User{
		ID:       42,
		Username: "testuser",
		Role:     "user",
	}
	token, err := mgr.GenerateToken(user)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	return token
}

func TestNewAuthMiddleware(t *testing.T) {
	jwtMgr := newTestJWTManager(t)
	blacklist := auth.NewTokenBlacklist()
	defer blacklist.Stop()

	okHandler := func(w http.ResponseWriter, r *http.Request) error {
		claims, ok := r.Context().Value(UserContextKey).(*models.Claims)
		if !ok || claims == nil {
			t.Error("expected claims in context")
		}
		w.WriteHeader(http.StatusOK)
		return nil
	}

	tests := []struct {
		name       string
		setup      func(r *http.Request, token string) *http.Request
		wantStatus int
		wantClaims bool
		blacklist  bool
	}{
		{
			name: "token from cookie is accepted",
			setup: func(r *http.Request, token string) *http.Request {
				r.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
				return r
			},
			wantStatus: http.StatusOK,
			wantClaims: true,
		},
		{
			name: "token from Authorization Bearer header is accepted",
			setup: func(r *http.Request, token string) *http.Request {
				r.Header.Set("Authorization", "Bearer "+token)
				return r
			},
			wantStatus: http.StatusOK,
			wantClaims: true,
		},
		{
			name: "missing token returns auth error",
			setup: func(r *http.Request, token string) *http.Request {
				return r
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid Bearer format returns error",
			setup: func(r *http.Request, token string) *http.Request {
				r.Header.Set("Authorization", "Token "+token)
				return r
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "blacklisted token is rejected",
			setup: func(r *http.Request, token string) *http.Request {
				r.Header.Set("Authorization", "Bearer "+token)
				return r
			},
			wantStatus: http.StatusUnauthorized,
			blacklist:  true,
		},
		{
			name: "valid token adds claims to context",
			setup: func(r *http.Request, token string) *http.Request {
				r.Header.Set("Authorization", "Bearer "+token)
				return r
			},
			wantStatus: http.StatusOK,
			wantClaims: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token := generateTestToken(t, jwtMgr)

			bl := auth.NewTokenBlacklist()
			defer bl.Stop()
			if tc.blacklist {
				bl.Add(token, time.Now().Add(time.Hour))
			}

			middleware := NewAuthMiddleware(jwtMgr, bl)
			handler := middleware(okHandler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req = tc.setup(req, token)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("got status %d, want %d", rec.Code, tc.wantStatus)
			}

			if tc.wantStatus != http.StatusOK {
				var body map[string]interface{}
				if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
					t.Fatalf("failed to decode error response: %v", err)
				}
				if _, ok := body["error"]; !ok {
					t.Error("expected error field in response body")
				}
			}
		})
	}
}

func TestAuthMiddleware_ClaimsInContext(t *testing.T) {
	jwtMgr := newTestJWTManager(t)

	var capturedClaims *models.Claims

	handler := func(w http.ResponseWriter, r *http.Request) error {
		claims, ok := r.Context().Value(UserContextKey).(*models.Claims)
		if !ok {
			t.Fatal("claims not found in context")
		}
		capturedClaims = claims
		return nil
	}

	middleware := NewAuthMiddleware(jwtMgr, nil)
	wrapped := middleware(handler)

	token := generateTestToken(t, jwtMgr)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec.Code)
	}
	if capturedClaims == nil {
		t.Fatal("claims were not captured")
	}
	if capturedClaims.UserID != 42 {
		t.Errorf("got UserID %d, want 42", capturedClaims.UserID)
	}
	if capturedClaims.Username != "testuser" {
		t.Errorf("got Username %q, want %q", capturedClaims.Username, "testuser")
	}
}

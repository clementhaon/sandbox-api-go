package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clementhaon/sandbox-api-go/auth"
	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/mocks"
	"github.com/clementhaon/sandbox-api-go/models"
)

func newTestJWTManager(t *testing.T) *auth.JWTManager {
	t.Helper()
	jm, err := auth.NewJWTManager("test-secret-key-minimum-16-chars")
	if err != nil {
		t.Fatalf("failed to create JWTManager: %v", err)
	}
	return jm
}

func newTestAuthHandler(svc *mocks.MockAuthService) *AuthHandler {
	jm, _ := auth.NewJWTManager("test-secret-key-minimum-16-chars")
	bl := auth.NewTokenBlacklist()
	return NewAuthHandler(svc, jm, bl)
}

func TestAuthHandler_Register_Success(t *testing.T) {
	svc := &mocks.MockAuthService{
		RegisterFn: func(ctx context.Context, req models.RegisterRequest) (models.User, string, error) {
			return models.User{
				ID:       1,
				Username: req.Username,
				Email:    req.Email,
				IsActive: true,
				Role:     "user",
			}, "jwt-token-here", nil
		},
	}

	handler := newTestAuthHandler(svc)
	body, _ := json.Marshal(models.RegisterRequest{
		Username: "johndoe",
		Email:    "john@example.com",
		Password: "Password1",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	w := httptest.NewRecorder()

	err := handler.HandleRegister(w, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	// Check cookie is set
	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "auth_token" && c.Value == "jwt-token-here" {
			found = true
		}
	}
	if !found {
		t.Error("expected auth_token cookie to be set")
	}

	var resp models.AuthResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.User.Username != "johndoe" {
		t.Errorf("expected username johndoe, got %s", resp.User.Username)
	}
	if resp.Message != "Registration successful" {
		t.Errorf("expected 'Registration successful', got '%s'", resp.Message)
	}
}

func TestAuthHandler_Register_InvalidJSON(t *testing.T) {
	handler := newTestAuthHandler(&mocks.MockAuthService{})

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()

	err := handler.HandleRegister(w, req)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestAuthHandler_Login_Success(t *testing.T) {
	svc := &mocks.MockAuthService{
		LoginFn: func(ctx context.Context, req models.LoginRequest) (models.User, string, error) {
			return models.User{
				ID:       1,
				Username: "johndoe",
				Email:    req.Email,
			}, "jwt-token", nil
		},
	}

	handler := newTestAuthHandler(svc)
	body, _ := json.Marshal(models.LoginRequest{
		Email:    "john@example.com",
		Password: "Password1",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	err := handler.HandleLogin(w, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp models.AuthResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Message != "Login successful" {
		t.Errorf("expected 'Login successful', got '%s'", resp.Message)
	}
}

func TestAuthHandler_Login_ServiceError(t *testing.T) {
	svc := &mocks.MockAuthService{
		LoginFn: func(ctx context.Context, req models.LoginRequest) (models.User, string, error) {
			return models.User{}, "", errors.NewInvalidCredentialsError()
		},
	}

	handler := newTestAuthHandler(svc)
	body, _ := json.Marshal(models.LoginRequest{
		Email:    "john@example.com",
		Password: "wrong",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	err := handler.HandleLogin(w, req)
	if err == nil {
		t.Fatal("expected error")
	}
	appErr, ok := errors.IsAppError(err)
	if !ok {
		t.Fatal("expected AppError")
	}
	if appErr.Code != errors.ErrInvalidCredentials {
		t.Errorf("expected INVALID_CREDENTIALS, got %s", appErr.Code)
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	handler := newTestAuthHandler(&mocks.MockAuthService{})

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	w := httptest.NewRecorder()

	err := handler.HandleLogout(w, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cookies := w.Result().Cookies()
	for _, c := range cookies {
		if c.Name == "auth_token" {
			if c.MaxAge != -1 {
				t.Errorf("expected MaxAge -1 to clear cookie, got %d", c.MaxAge)
			}
			if c.Value != "" {
				t.Errorf("expected empty cookie value, got '%s'", c.Value)
			}
		}
	}
}

func TestAuthHandler_Logout_BlacklistsToken(t *testing.T) {
	jm := newTestJWTManager(t)
	bl := auth.NewTokenBlacklist()
	defer bl.Stop()
	handler := NewAuthHandler(&mocks.MockAuthService{}, jm, bl)

	// Generate a valid token
	user := models.User{ID: 1, Username: "test", Role: "user"}
	token, err := jm.GenerateToken(user)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Logout with the token in cookie
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
	w := httptest.NewRecorder()

	err = handler.HandleLogout(w, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Token should now be blacklisted
	if !bl.IsBlacklisted(token) {
		t.Error("expected token to be blacklisted after logout")
	}
}

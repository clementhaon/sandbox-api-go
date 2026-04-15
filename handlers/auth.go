package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/clementhaon/sandbox-api-go/auth"
	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/logger"
	"github.com/clementhaon/sandbox-api-go/middleware"
	"github.com/clementhaon/sandbox-api-go/models"
	"github.com/clementhaon/sandbox-api-go/services"
)

type AuthHandler struct {
	authService services.AuthService
	jwtManager  *auth.JWTManager
	blacklist   *auth.TokenBlacklist
}

func NewAuthHandler(s services.AuthService, jwtManager *auth.JWTManager, blacklist *auth.TokenBlacklist) *AuthHandler {
	return &AuthHandler{authService: s, jwtManager: jwtManager, blacklist: blacklist}
}

func (h *AuthHandler) HandleRegister(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.WarnContext(r.Context(), "Invalid JSON in register request", map[string]interface{}{
			"error": err.Error(),
		})
		return errors.NewInvalidJSONError()
	}

	user, token, err := h.authService.Register(r.Context(), req)
	if err != nil {
		return err
	}

	isProduction := os.Getenv("APP_ENV") == "production"
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		MaxAge:   24 * 60 * 60,
		HttpOnly: true,
		Secure:   isProduction,
		SameSite: http.SameSiteStrictMode,
	})

	csrfToken := middleware.SetCSRFCookie(w, isProduction)

	response := models.AuthResponse{
		User:    user,
		Message: "Registration successful",
	}

	w.Header().Set("X-CSRF-Token", csrfToken)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
	return nil
}

func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.WarnContext(r.Context(), "Invalid JSON in login request", map[string]interface{}{
			"error": err.Error(),
		})
		return errors.NewInvalidJSONError()
	}

	user, token, err := h.authService.Login(r.Context(), req)
	if err != nil {
		return err
	}

	isProduction := os.Getenv("APP_ENV") == "production"
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		MaxAge:   24 * 60 * 60,
		HttpOnly: true,
		Secure:   isProduction,
		SameSite: http.SameSiteStrictMode,
	})

	csrfToken := middleware.SetCSRFCookie(w, isProduction)
	w.Header().Set("X-CSRF-Token", csrfToken)

	response := models.AuthResponse{
		User:    user,
		Message: "Login successful",
	}

	json.NewEncoder(w).Encode(response)
	return nil
}

func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	logger.InfoContext(r.Context(), "User logout requested")

	// Extract and blacklist the current token
	if token := h.extractToken(r); token != "" {
		if claims, err := h.jwtManager.ValidateToken(token); err == nil {
			h.blacklist.Add(token, claims.ExpiresAt)
		}
	}

	isProduction := os.Getenv("APP_ENV") == "production"
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isProduction,
		SameSite: http.SameSiteStrictMode,
	})
	middleware.ClearCSRFCookie(w, isProduction)

	json.NewEncoder(w).Encode(map[string]string{
		"message": "Logout successful",
	})
	return nil
}

func (h *AuthHandler) HandleGetUser(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		logger.ErrorContext(r.Context(), "Missing user context in authenticated request", nil)
		return errors.NewInternalError().WithDetails(map[string]interface{}{
			"issue": "user_context_missing",
		})
	}

	json.NewEncoder(w).Encode(claims)
	return nil
}

// extractToken extracts the JWT token from cookie or Authorization header.
func (h *AuthHandler) extractToken(r *http.Request) string {
	if cookie, err := r.Cookie("auth_token"); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	authHeader := r.Header.Get("Authorization")
	if parts := strings.Split(authHeader, " "); len(parts) == 2 && parts[0] == "Bearer" {
		return parts[1]
	}
	return ""
}

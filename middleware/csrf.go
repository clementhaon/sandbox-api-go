package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/clementhaon/sandbox-api-go/errors"
)

// GenerateCSRFToken generates a cryptographically secure CSRF token.
func GenerateCSRFToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

// CSRFMiddleware validates CSRF tokens for state-changing requests.
// It uses the double-submit cookie pattern: the csrf_token cookie must match
// the X-CSRF-Token header on POST/PUT/PATCH/DELETE requests.
func CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip safe methods
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		// Skip public auth routes (login, register, logout)
		if isCSRFExemptPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Validate CSRF token
		cookie, err := r.Cookie("csrf_token")
		if err != nil || cookie.Value == "" {
			appErr := errors.NewForbiddenError()
			appErr.Message = "Missing CSRF token"
			errors.WriteError(w, appErr)
			return
		}

		headerToken := r.Header.Get("X-CSRF-Token")
		if headerToken == "" || headerToken != cookie.Value {
			appErr := errors.NewForbiddenError()
			appErr.Message = "Invalid CSRF token"
			errors.WriteError(w, appErr)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isCSRFExemptPath(path string) bool {
	return path == "/auth/login" || path == "/auth/register" || path == "/auth/logout"
}

// SetCSRFCookie sets the csrf_token cookie (readable by JavaScript).
func SetCSRFCookie(w http.ResponseWriter, isProduction bool) string {
	token := GenerateCSRFToken()
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    token,
		Path:     "/",
		MaxAge:   24 * 60 * 60,
		HttpOnly: false, // Must be readable by JS
		Secure:   isProduction,
		SameSite: http.SameSiteStrictMode,
	})
	return token
}

// ClearCSRFCookie clears the csrf_token cookie.
func ClearCSRFCookie(w http.ResponseWriter, isProduction bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: false,
		Secure:   isProduction,
		SameSite: http.SameSiteStrictMode,
	})
}

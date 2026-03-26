package middleware

import (
	"context"
	"github.com/clementhaon/sandbox-api-go/auth"
	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/logger"
	"net/http"
	"strings"
)

type contextKey string

const UserContextKey contextKey = "user"

// AuthMiddleware validates the JWT token in requests
func AuthMiddleware(handler ErrorHandler) http.HandlerFunc {
	return ErrorMiddleware(func(w http.ResponseWriter, r *http.Request) error {
		var token string

		// Try to get token from cookie first
		if cookie, err := r.Cookie("auth_token"); err == nil && cookie.Value != "" {
			token = cookie.Value
		} else {
			// Fallback: get token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				logger.WarnContext(r.Context(), "Authentication attempt without token")
				return errors.NewAuthRequiredError().WithDetails(map[string]interface{}{
					"message": "Token required in cookie or Authorization header",
				})
			}

			// Check "Bearer <token>" format
			tokenParts := strings.Split(authHeader, " ")
			if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
				logger.WarnContext(r.Context(), "Invalid token format in Authorization header")
				return errors.NewInvalidTokenError().WithDetails(map[string]interface{}{
					"expected_format": "Bearer <token>",
				})
			}
			token = tokenParts[1]
		}

		// Validate the token
		claims, err := auth.ValidateToken(token)
		if err != nil {
			logger.WarnContext(r.Context(), "Invalid or expired token", map[string]interface{}{
				"error": err.Error(),
			})
			return errors.NewInvalidTokenError().WithCause(err)
		}

		// Add user information to context
		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		ctx = context.WithValue(ctx, logger.UserIDKey, claims.UserID)

		return handler(w, r.WithContext(ctx))
	})
}

package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/clementhaon/sandbox-api-go/pkg/auth"
	"github.com/clementhaon/sandbox-api-go/pkg/errors"
	"github.com/clementhaon/sandbox-api-go/pkg/logger"
)

type contextKey string

const UserContextKey contextKey = "user"

func NewAuthMiddleware(jwtManager *auth.JWTManager) func(ErrorHandler) http.HandlerFunc {
	return func(handler ErrorHandler) http.HandlerFunc {
		return ErrorMiddleware(func(w http.ResponseWriter, r *http.Request) error {
			var token string

			if cookie, err := r.Cookie("auth_token"); err == nil && cookie.Value != "" {
				token = cookie.Value
			} else {
				authHeader := r.Header.Get("Authorization")
				if authHeader == "" {
					logger.WarnContext(r.Context(), "Authentication attempt without token")
					return errors.NewAuthRequiredError().WithDetails(map[string]interface{}{
						"message": "Token required in cookie or Authorization header",
					})
				}

				tokenParts := strings.Split(authHeader, " ")
				if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
					logger.WarnContext(r.Context(), "Invalid token format in Authorization header")
					return errors.NewInvalidTokenError().WithDetails(map[string]interface{}{
						"expected_format": "Bearer <token>",
					})
				}
				token = tokenParts[1]
			}

			claims, err := jwtManager.ValidateToken(token)
			if err != nil {
				logger.WarnContext(r.Context(), "Invalid or expired token", map[string]interface{}{
					"error": err.Error(),
				})
				return errors.NewInvalidTokenError().WithCause(err)
			}

			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			ctx = context.WithValue(ctx, logger.UserIDKey, claims.UserID)

			return handler(w, r.WithContext(ctx))
		})
	}
}

package middleware

import (
	"context"
	"net/http"
	"sandbox-api-go/auth"
	"sandbox-api-go/errors"
	"sandbox-api-go/logger"
	"strings"
)

type contextKey string

const UserContextKey contextKey = "user"

// AuthMiddleware vérifie le token JWT dans les requêtes
func AuthMiddleware(handler ErrorHandler) http.HandlerFunc {
	return ErrorMiddleware(func(w http.ResponseWriter, r *http.Request) error {
		var token string

		// Essayer de récupérer le token depuis le cookie d'abord
		if cookie, err := r.Cookie("auth_token"); err == nil && cookie.Value != "" {
			token = cookie.Value
		} else {
			// Fallback : récupérer le token depuis l'en-tête Authorization
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				logger.WarnContext(r.Context(), "Authentication attempt without token")
				return errors.NewAuthRequiredError().WithDetails(map[string]interface{}{
					"message": "Token required in cookie or Authorization header",
				})
			}

			// Vérifier le format "Bearer <token>"
			tokenParts := strings.Split(authHeader, " ")
			if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
				logger.WarnContext(r.Context(), "Invalid token format in Authorization header")
				return errors.NewInvalidTokenError().WithDetails(map[string]interface{}{
					"expected_format": "Bearer <token>",
				})
			}
			token = tokenParts[1]
		}

		// Valider le token
		claims, err := auth.ValidateToken(token)
		if err != nil {
			logger.WarnContext(r.Context(), "Invalid or expired token", map[string]interface{}{
				"error": err.Error(),
			})
			return errors.NewInvalidTokenError().WithCause(err)
		}

		// Ajouter les informations utilisateur au contexte
		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		ctx = context.WithValue(ctx, logger.UserIDKey, claims.UserID)
		
		return handler(w, r.WithContext(ctx))
	})
} 
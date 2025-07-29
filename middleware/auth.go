package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"sandbox-api-go/auth"
	"strings"
)

type contextKey string

const UserContextKey contextKey = "user"

// AuthMiddleware vérifie le token JWT dans les requêtes
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Récupérer le token depuis l'en-tête Authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Token d'authentification requis",
			})
			return
		}

		// Vérifier le format "Bearer <token>"
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Format de token invalide. Utilisez 'Bearer <token>'",
			})
			return
		}

		// Valider le token
		claims, err := auth.ValidateToken(tokenParts[1])
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Token invalide ou expiré",
			})
			return
		}

		// Ajouter les informations utilisateur au contexte
		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
} 
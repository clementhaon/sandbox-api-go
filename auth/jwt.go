package auth

import (
	"fmt"
	"github.com/clementhaon/sandbox-api-go/models"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWT secret key loaded from environment variables
var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

// GenerateToken génère un token JWT pour un utilisateur
func GenerateToken(user models.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
		"exp":      time.Now().Add(time.Hour * 24).Unix(), // Expire dans 24h
	}

	// Add optional fields if present
	if user.FirstName.Valid {
		claims["first_name"] = user.FirstName.String
	}
	if user.LastName.Valid {
		claims["last_name"] = user.LastName.String
	}
	if user.AvatarURL.Valid {
		claims["avatar_url"] = user.AvatarURL.String
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateToken valide un token JWT et retourne les claims
func ValidateToken(tokenString string) (*models.Claims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID := int(claims["user_id"].(float64))
		username := claims["username"].(string)
		exp := int64(claims["exp"].(float64))

		result := &models.Claims{
			UserID:    userID,
			Username:  username,
			ExpiresAt: time.Unix(exp, 0),
		}

		// Extract optional fields if they exist
		if role, ok := claims["role"].(string); ok {
			result.Role = role
		}
		if firstName, ok := claims["first_name"].(string); ok {
			result.FirstName = firstName
		}
		if lastName, ok := claims["last_name"].(string); ok {
			result.LastName = lastName
		}
		if avatarURL, ok := claims["avatar_url"].(string); ok {
			result.AvatarURL = avatarURL
		}

		return result, nil
	}

	return nil, fmt.Errorf("invalid token")
}

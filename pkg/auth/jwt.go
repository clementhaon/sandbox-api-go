package auth

import (
	"fmt"
	"time"

	"github.com/clementhaon/sandbox-api-go/pkg/models"
	"github.com/golang-jwt/jwt/v5"
)

type JWTManager struct {
	secret []byte
}

func NewJWTManager(secret string) (*JWTManager, error) {
	if len(secret) < 16 {
		return nil, fmt.Errorf("JWT secret must be at least 16 characters long")
	}
	return &JWTManager{secret: []byte(secret)}, nil
}

func (m *JWTManager) GenerateToken(user models.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	}

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
	return token.SignedString(m.secret)
}

func (m *JWTManager) ValidateToken(tokenString string) (*models.Claims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
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

package auth

import (
	"fmt"
	"sandbox-api-go/models"
	"time"
	"sandbox-api-go/config"
	"github.com/golang-jwt/jwt/v5"
)

// Récupération de la clé secrète JWT à partir des variables d'environnement
var jwtSecret = []byte(config.GetEnv("JWT_SECRET", "votre-secret-super-securise-ici"))

// GenerateToken génère un token JWT pour un utilisateur
func GenerateToken(user models.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(), // Expire dans 24h
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateToken valide un token JWT et retourne les claims
func ValidateToken(tokenString string) (*models.Claims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("méthode de signature inattendue: %v", token.Header["alg"])
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

		return &models.Claims{
			UserID:    userID,
			Username:  username,
			ExpiresAt: time.Unix(exp, 0),
		}, nil
	}

	return nil, fmt.Errorf("token invalide")
} 
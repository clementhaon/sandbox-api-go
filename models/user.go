package models

import "time"

// User represents a user in the system
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"-"` // Le "-" empêche l'export en JSON pour la sécurité
}

// LoginRequest represents login credentials
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterRequest represents registration data
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse represents the response after authentication
type AuthResponse struct {
	Token   string `json:"token"`
	User    User   `json:"user"`
	Message string `json:"message"`
}

// Claims represents JWT claims
type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	ExpiresAt time.Time `json:"exp"`
} 
package models

import (
	"database/sql"
	"time"
)

// User represents a user in the system
type User struct {
	ID          int            `json:"id"`
	Username    string         `json:"username"`
	Email       string         `json:"email"`
	Password    string         `json:"-"` // Le "-" empêche l'export en JSON pour la sécurité
	FirstName   sql.NullString `json:"first_name,omitempty"`
	LastName    sql.NullString `json:"last_name,omitempty"`
	AvatarURL   sql.NullString `json:"avatar_url,omitempty"`
	IsActive    bool           `json:"is_active"`
	LastLoginAt sql.NullTime   `json:"last_login_at,omitempty"`
	Role        string         `json:"role"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// LoginRequest represents login credentials
type LoginRequest struct {
	Email string `json:"email"`
	Password string `json:"password"`
}

// RegisterRequest represents registration data
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UpdateProfileRequest represents profile update data
// Note: email and password cannot be updated through this endpoint
type UpdateProfileRequest struct {
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
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
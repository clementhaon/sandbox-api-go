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
	Email    string `json:"email"`
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
	User    User   `json:"user"`
	Message string `json:"message"`
}

// Claims represents JWT claims
type Claims struct {
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	Role      string    `json:"role,omitempty"`
	FirstName string    `json:"first_name,omitempty"`
	LastName  string    `json:"last_name,omitempty"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	ExpiresAt time.Time `json:"exp"`
}

// UserResponse represents a user in API responses (with proper JSON formatting)
type UserResponse struct {
	ID        int        `json:"id"`
	Email     string     `json:"email"`
	Username  string     `json:"username"`
	FirstName string     `json:"firstName,omitempty"`
	LastName  string     `json:"lastName,omitempty"`
	AvatarURL string     `json:"avatarUrl,omitempty"`
	Role      string     `json:"role"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	LastLogin *time.Time `json:"lastLogin,omitempty"`
}

// UserFromDB converts a User to UserResponse
func UserFromDB(u User) UserResponse {
	resp := UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Username:  u.Username,
		Role:      u.Role,
		Status:    StatusActive,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
	if !u.IsActive {
		resp.Status = StatusInactive
	}
	if u.FirstName.Valid {
		resp.FirstName = u.FirstName.String
	}
	if u.LastName.Valid {
		resp.LastName = u.LastName.String
	}
	if u.AvatarURL.Valid {
		resp.AvatarURL = u.AvatarURL.String
	}
	if u.LastLoginAt.Valid {
		resp.LastLogin = &u.LastLoginAt.Time
	}
	return resp
}

// Pagination represents pagination info in responses
type Pagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

// UsersListResponse represents the paginated users list response
type UsersListResponse struct {
	Data       []UserResponse `json:"data"`
	Pagination Pagination     `json:"pagination"`
}

// CreateUserRequest represents the request to create a user
type CreateUserRequest struct {
	Email     string `json:"email"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	Role      string `json:"role,omitempty"`
}

// UpdateUserRequest represents the request to update a user
type UpdateUserRequest struct {
	Email     string `json:"email,omitempty"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	AvatarURL string `json:"avatarUrl,omitempty"`
	Role      string `json:"role,omitempty"`
}

// UpdateUserStatusRequest represents the request to update user status
type UpdateUserStatusRequest struct {
	Status string `json:"status"`
}

// UserListParams represents query parameters for listing users
type UserListParams struct {
	Page      int
	PageSize  int
	SortBy    string
	SortOrder string
	Search    string
	Role      string
	Status    string
}

package models

import (
	"database/sql"
	"time"
)

type User struct {
	ID          int            `json:"id"`
	Username    string         `json:"username"`
	Email       string         `json:"email"`
	Password    string         `json:"-"`
	FirstName   sql.NullString `json:"first_name,omitempty"`
	LastName    sql.NullString `json:"last_name,omitempty"`
	AvatarURL   sql.NullString `json:"avatar_url,omitempty"`
	IsActive    bool           `json:"is_active"`
	LastLoginAt sql.NullTime   `json:"last_login_at,omitempty"`
	Role        string         `json:"role"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UpdateProfileRequest struct {
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

type AuthResponse struct {
	Token   string `json:"token"`
	User    User   `json:"user"`
	Message string `json:"message"`
}

type Claims struct {
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	Role      string    `json:"role,omitempty"`
	FirstName string    `json:"first_name,omitempty"`
	LastName  string    `json:"last_name,omitempty"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	ExpiresAt time.Time `json:"exp"`
}

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

func UserFromDB(u User) UserResponse {
	resp := UserResponse{
		ID: u.ID, Email: u.Email, Username: u.Username, Role: u.Role,
		Status: StatusActive, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
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

// UserBrief is used for inter-service communication and task assignees
type UserBrief struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatarUrl,omitempty"`
}

type Pagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

type UsersListResponse struct {
	Data       []UserResponse `json:"data"`
	Pagination Pagination     `json:"pagination"`
}

type CreateUserRequest struct {
	Email     string `json:"email"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	Role      string `json:"role,omitempty"`
}

type UpdateUserRequest struct {
	Email     string `json:"email,omitempty"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	AvatarURL string `json:"avatarUrl,omitempty"`
	Role      string `json:"role,omitempty"`
}

type UpdateUserStatusRequest struct {
	Status string `json:"status"`
}

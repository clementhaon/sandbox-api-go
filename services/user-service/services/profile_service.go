package services

import (
	"context"
	"database/sql"
	"time"

	"github.com/clementhaon/sandbox-api-go/pkg/errors"
	"github.com/clementhaon/sandbox-api-go/pkg/logger"
	"github.com/clementhaon/sandbox-api-go/pkg/models"
)

type ProfileService interface {
	GetProfile(ctx context.Context, userID int) (models.User, error)
	UpdateProfile(ctx context.Context, userID int, req models.UpdateProfileRequest) (models.User, error)
}

type profileService struct {
	db *sql.DB
}

func NewProfileService(db *sql.DB) ProfileService {
	return &profileService{db: db}
}

func (s *profileService) GetProfile(ctx context.Context, userID int) (models.User, error) {
	var user models.User
	startTime := time.Now()
	err := s.db.QueryRow(
		`SELECT id, username, email, first_name, last_name, avatar_url, is_active, last_login_at, role, created_at, updated_at
		FROM users WHERE id = $1`,
		userID,
	).Scan(&user.ID, &user.Username, &user.Email, &user.FirstName, &user.LastName,
		&user.AvatarURL, &user.IsActive, &user.LastLoginAt, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	logger.LogDatabaseOperation(ctx, "SELECT", "users", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return models.User{}, errors.NewNotFoundError("User")
	} else if err != nil {
		logger.ErrorContext(ctx, "Database error fetching user profile", err)
		return models.User{}, errors.NewDatabaseError().WithCause(err)
	}

	logger.InfoContext(ctx, "User profile retrieved", map[string]interface{}{
		"user_id": user.ID,
	})

	return user, nil
}

func (s *profileService) UpdateProfile(ctx context.Context, userID int, req models.UpdateProfileRequest) (models.User, error) {
	var firstName, lastName, avatarURL sql.NullString

	if req.FirstName != nil {
		firstName = sql.NullString{String: *req.FirstName, Valid: true}
	}
	if req.LastName != nil {
		lastName = sql.NullString{String: *req.LastName, Valid: true}
	}
	if req.AvatarURL != nil {
		avatarURL = sql.NullString{String: *req.AvatarURL, Valid: true}
	}

	startTime := time.Now()
	_, err := s.db.Exec(
		`UPDATE users
		SET first_name = COALESCE($1, first_name),
		    last_name = COALESCE($2, last_name),
		    avatar_url = COALESCE($3, avatar_url),
		    updated_at = NOW()
		WHERE id = $4`,
		firstName, lastName, avatarURL, userID,
	)
	logger.LogDatabaseOperation(ctx, "UPDATE", "users", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Database error updating user profile", err)
		return models.User{}, errors.NewDatabaseError().WithCause(err)
	}

	var updatedUser models.User
	startTime = time.Now()
	err = s.db.QueryRow(
		`SELECT id, username, email, first_name, last_name, avatar_url, is_active, last_login_at, role, created_at, updated_at
		FROM users WHERE id = $1`,
		userID,
	).Scan(&updatedUser.ID, &updatedUser.Username, &updatedUser.Email, &updatedUser.FirstName,
		&updatedUser.LastName, &updatedUser.AvatarURL, &updatedUser.IsActive, &updatedUser.LastLoginAt,
		&updatedUser.Role, &updatedUser.CreatedAt, &updatedUser.UpdatedAt)
	logger.LogDatabaseOperation(ctx, "SELECT", "users", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Database error fetching updated profile", err)
		return models.User{}, errors.NewDatabaseError().WithCause(err)
	}

	logger.InfoContext(ctx, "User profile updated successfully", map[string]interface{}{
		"user_id": updatedUser.ID,
	})

	return updatedUser, nil
}

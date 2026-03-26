package handlers

import (
	"database/sql"
	"encoding/json"
	"github.com/clementhaon/sandbox-api-go/database"
	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/logger"
	"github.com/clementhaon/sandbox-api-go/middleware"
	"github.com/clementhaon/sandbox-api-go/models"
	"net/http"
	"time"
)

// HandleGetProfile récupère le profil complet de l'utilisateur connecté
func HandleGetProfile(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		return errors.NewMethodNotAllowedError()
	}

	// Get user from context
	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)

	logger.Info("HandleGetProfile", map[string]interface{}{
		"claims": claims,
	})
	if !ok {
		logger.ErrorContext(r.Context(), "Missing user context in authenticated request", nil)
		return errors.NewInternalError().WithDetails(map[string]interface{}{
			"issue": "user_context_missing",
		})
	}

	// Fetch full profile from database
	var user models.User
	startTime := time.Now()
	err := database.DB.QueryRow(
		`SELECT id, username, email, first_name, last_name, avatar_url, is_active, last_login_at, role, created_at, updated_at
		FROM users WHERE id = $1`,
		claims.UserID,
	).Scan(&user.ID, &user.Username, &user.Email, &user.FirstName, &user.LastName,
		&user.AvatarURL, &user.IsActive, &user.LastLoginAt, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	logger.LogDatabaseOperation(r.Context(), "SELECT", "users", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return errors.NewNotFoundError("User")
	} else if err != nil {
		logger.ErrorContext(r.Context(), "Database error fetching user profile", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	logger.InfoContext(r.Context(), "User profile retrieved", map[string]interface{}{
		"user_id": user.ID,
	})

	json.NewEncoder(w).Encode(user)
	return nil
}

// HandleUpdateProfile met à jour le profil de l'utilisateur (sans email ni password)
func HandleUpdateProfile(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPut {
		return errors.NewMethodNotAllowedError()
	}

	// Get user from context
	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		logger.ErrorContext(r.Context(), "Missing user context in authenticated request", nil)
		return errors.NewInternalError().WithDetails(map[string]interface{}{
			"issue": "user_context_missing",
		})
	}

	var req models.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.WarnContext(r.Context(), "Invalid JSON in update profile request", map[string]interface{}{
			"error": err.Error(),
		})
		return errors.NewInvalidJSONError()
	}

	// Prepare values for update
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

	// Update profile in the database
	startTime := time.Now()
	_, err := database.DB.Exec(
		`UPDATE users
		SET first_name = COALESCE($1, first_name),
		    last_name = COALESCE($2, last_name),
		    avatar_url = COALESCE($3, avatar_url),
		    updated_at = NOW()
		WHERE id = $4`,
		firstName, lastName, avatarURL, claims.UserID,
	)
	logger.LogDatabaseOperation(r.Context(), "UPDATE", "users", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(r.Context(), "Database error updating user profile", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	// Fetch updated profile
	var updatedUser models.User
	startTime = time.Now()
	err = database.DB.QueryRow(
		`SELECT id, username, email, first_name, last_name, avatar_url, is_active, last_login_at, role, created_at, updated_at
		FROM users WHERE id = $1`,
		claims.UserID,
	).Scan(&updatedUser.ID, &updatedUser.Username, &updatedUser.Email, &updatedUser.FirstName,
		&updatedUser.LastName, &updatedUser.AvatarURL, &updatedUser.IsActive, &updatedUser.LastLoginAt,
		&updatedUser.Role, &updatedUser.CreatedAt, &updatedUser.UpdatedAt)
	logger.LogDatabaseOperation(r.Context(), "SELECT", "users", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(r.Context(), "Database error fetching updated profile", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	logger.InfoContext(r.Context(), "User profile updated successfully", map[string]interface{}{
		"user_id": updatedUser.ID,
	})

	response := map[string]interface{}{
		"message": "Profile updated successfully",
		"user":    updatedUser,
	}

	json.NewEncoder(w).Encode(response)
	return nil
}

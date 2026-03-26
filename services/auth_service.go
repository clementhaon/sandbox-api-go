package services

import (
	"context"
	"database/sql"
	"time"

	"github.com/clementhaon/sandbox-api-go/auth"
	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/logger"
	"github.com/clementhaon/sandbox-api-go/metrics"
	"github.com/clementhaon/sandbox-api-go/models"
	"github.com/clementhaon/sandbox-api-go/validation"

	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Register(ctx context.Context, req models.RegisterRequest) (models.User, string, error)
	Login(ctx context.Context, req models.LoginRequest) (models.User, string, error)
}

type authService struct {
	db *sql.DB
}

func NewAuthService(db *sql.DB) AuthService {
	return &authService{db: db}
}

func (s *authService) Register(ctx context.Context, req models.RegisterRequest) (models.User, string, error) {
	if validationErr := validation.ValidateRegisterRequest(req.Username, req.Email, req.Password); validationErr != nil {
		return models.User{}, "", validationErr
	}

	var existingUser models.User
	startTime := time.Now()
	err := s.db.QueryRow("SELECT id FROM users WHERE username = $1 OR email = $2", req.Username, req.Email).Scan(&existingUser.ID)
	logger.LogDatabaseOperation(ctx, "SELECT", "users", time.Since(startTime), err)

	if err == nil {
		return models.User{}, "", errors.NewUserExistsError()
	} else if err != sql.ErrNoRows {
		logger.ErrorContext(ctx, "Database error checking existing user", err)
		return models.User{}, "", errors.NewDatabaseError().WithCause(err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.ErrorContext(ctx, "Error hashing password", err)
		return models.User{}, "", errors.NewInternalError().WithCause(err)
	}

	var newUser models.User
	startTime = time.Now()
	err = s.db.QueryRow(
		`INSERT INTO users (username, email, password, is_active, role)
		VALUES ($1, $2, $3, true, 'user')
		RETURNING id, username, email, first_name, last_name, avatar_url, is_active, last_login_at, role, created_at, updated_at`,
		req.Username, req.Email, string(hashedPassword),
	).Scan(&newUser.ID, &newUser.Username, &newUser.Email, &newUser.FirstName, &newUser.LastName,
		&newUser.AvatarURL, &newUser.IsActive, &newUser.LastLoginAt, &newUser.Role, &newUser.CreatedAt, &newUser.UpdatedAt)
	logger.LogDatabaseOperation(ctx, "INSERT", "users", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error creating user", err)
		return models.User{}, "", errors.NewDatabaseError().WithCause(err)
	}

	token, err := auth.GenerateToken(newUser)
	if err != nil {
		logger.ErrorContext(ctx, "Error generating JWT token", err)
		return models.User{}, "", errors.NewInternalError().WithCause(err)
	}

	logger.InfoContext(ctx, "User registered successfully", map[string]interface{}{
		"user_id":  newUser.ID,
		"username": newUser.Username,
	})
	metrics.RecordAuthAttempt("register", "success")

	return newUser, token, nil
}

func (s *authService) Login(ctx context.Context, req models.LoginRequest) (models.User, string, error) {
	if validationErr := validation.ValidateLoginRequest(req.Email, req.Password); validationErr != nil {
		return models.User{}, "", validationErr
	}

	var foundUser models.User
	var hashedPassword string
	startTime := time.Now()
	err := s.db.QueryRow(
		`SELECT id, username, email, password, first_name, last_name, avatar_url, is_active, last_login_at, role, created_at, updated_at
		FROM users WHERE Email = $1`,
		req.Email,
	).Scan(&foundUser.ID, &foundUser.Username, &foundUser.Email, &hashedPassword, &foundUser.FirstName,
		&foundUser.LastName, &foundUser.AvatarURL, &foundUser.IsActive, &foundUser.LastLoginAt,
		&foundUser.Role, &foundUser.CreatedAt, &foundUser.UpdatedAt)
	logger.LogDatabaseOperation(ctx, "SELECT", "users", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		logger.WarnContext(ctx, "Login attempt with non-existent email", map[string]interface{}{
			"email": req.Email,
		})
		return models.User{}, "", errors.NewInvalidCredentialsError()
	} else if err != nil {
		logger.ErrorContext(ctx, "Database error during login", err)
		return models.User{}, "", errors.NewDatabaseError().WithCause(err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)); err != nil {
		logger.WarnContext(ctx, "Login attempt with invalid password", map[string]interface{}{
			"user_id": foundUser.ID,
			"email":   req.Email,
		})
		return models.User{}, "", errors.NewInvalidCredentialsError()
	}

	startTime = time.Now()
	_, err = s.db.Exec("UPDATE users SET last_login_at = NOW() WHERE id = $1", foundUser.ID)
	logger.LogDatabaseOperation(ctx, "UPDATE", "users", time.Since(startTime), err)
	if err != nil {
		logger.WarnContext(ctx, "Failed to update last_login_at", map[string]interface{}{
			"user_id": foundUser.ID,
			"error":   err.Error(),
		})
	}

	token, err := auth.GenerateToken(foundUser)
	if err != nil {
		logger.ErrorContext(ctx, "Error generating JWT token for login", err)
		return models.User{}, "", errors.NewInternalError().WithCause(err)
	}

	logger.InfoContext(ctx, "User logged in successfully", map[string]interface{}{
		"user_id":  foundUser.ID,
		"username": foundUser.Username,
		"email":    foundUser.Email,
	})
	metrics.RecordAuthAttempt("login", "success")

	return foundUser, token, nil
}

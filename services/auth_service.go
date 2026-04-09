package services

import (
	"context"

	"github.com/clementhaon/sandbox-api-go/auth"
	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/logger"
	"github.com/clementhaon/sandbox-api-go/metrics"
	"github.com/clementhaon/sandbox-api-go/models"
	"github.com/clementhaon/sandbox-api-go/repository"
	"github.com/clementhaon/sandbox-api-go/validation"

	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Register(ctx context.Context, req models.RegisterRequest) (models.User, string, error)
	Login(ctx context.Context, req models.LoginRequest) (models.User, string, error)
}

type authService struct {
	userRepo   repository.UserRepository
	jwtManager *auth.JWTManager
}

func NewAuthService(userRepo repository.UserRepository, jwtManager *auth.JWTManager) AuthService {
	return &authService{userRepo: userRepo, jwtManager: jwtManager}
}

func (s *authService) Register(ctx context.Context, req models.RegisterRequest) (models.User, string, error) {
	if validationErr := validation.ValidateRegisterRequest(req.Username, req.Email, req.Password); validationErr != nil {
		return models.User{}, "", validationErr
	}

	exists, err := s.userRepo.ExistsByUsernameOrEmail(ctx, req.Username, req.Email)
	if err != nil {
		return models.User{}, "", err
	}
	if exists {
		return models.User{}, "", errors.NewUserExistsError()
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.ErrorContext(ctx, "Error hashing password", err)
		return models.User{}, "", errors.NewInternalError().WithCause(err)
	}

	newUser, err := s.userRepo.CreateAuth(ctx, req.Username, req.Email, string(hashedPassword))
	if err != nil {
		return models.User{}, "", err
	}

	token, err := s.jwtManager.GenerateToken(newUser)
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

	foundUser, hashedPassword, err := s.userRepo.FindByEmailWithPassword(ctx, req.Email)
	if err != nil {
		if _, ok := errors.IsAppError(err); ok {
			logger.WarnContext(ctx, "Login attempt with non-existent email", map[string]interface{}{
				"email": req.Email,
			})
		}
		return models.User{}, "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)); err != nil {
		logger.WarnContext(ctx, "Login attempt with invalid password", map[string]interface{}{
			"user_id": foundUser.ID,
			"email":   req.Email,
		})
		return models.User{}, "", errors.NewInvalidCredentialsError()
	}

	if err := s.userRepo.UpdateLastLogin(ctx, foundUser.ID); err != nil {
		logger.WarnContext(ctx, "Failed to update last_login_at", map[string]interface{}{
			"user_id": foundUser.ID,
			"error":   err.Error(),
		})
	}

	token, err := s.jwtManager.GenerateToken(foundUser)
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

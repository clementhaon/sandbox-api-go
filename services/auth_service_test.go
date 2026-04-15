package services

import (
	"context"
	"testing"

	"github.com/clementhaon/sandbox-api-go/auth"
	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/mocks"
	"github.com/clementhaon/sandbox-api-go/models"

	"golang.org/x/crypto/bcrypt"
)

func newJWTManager(t *testing.T) *auth.JWTManager {
	t.Helper()
	jm, err := auth.NewJWTManager("test-secret-key-for-testing-only")
	if err != nil {
		t.Fatalf("failed to create JWT manager: %v", err)
	}
	return jm
}

func TestAuthService_Register_Success(t *testing.T) {
	userRepo := &mocks.MockUserRepository{
		ExistsByUsernameOrEmailFn: func(ctx context.Context, username, email string) (bool, error) {
			return false, nil
		},
		CreateAuthFn: func(ctx context.Context, username, email, hashedPassword string) (models.User, error) {
			return models.User{
				ID:       1,
				Username: username,
				Email:    email,
				IsActive: true,
				Role:     "user",
			}, nil
		},
	}

	svc := NewAuthService(userRepo, newJWTManager(t))
	user, token, err := svc.Register(context.Background(), models.RegisterRequest{
		Username: "johndoe",
		Email:    "john@example.com",
		Password: "Password1",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Username != "johndoe" {
		t.Errorf("expected username johndoe, got %s", user.Username)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
}

func TestAuthService_Register_UserExists(t *testing.T) {
	userRepo := &mocks.MockUserRepository{
		ExistsByUsernameOrEmailFn: func(ctx context.Context, username, email string) (bool, error) {
			return true, nil
		},
	}

	svc := NewAuthService(userRepo, newJWTManager(t))
	_, _, err := svc.Register(context.Background(), models.RegisterRequest{
		Username: "johndoe",
		Email:    "john@example.com",
		Password: "Password1",
	})

	if err == nil {
		t.Fatal("expected error for existing user")
	}
	appErr, ok := errors.IsAppError(err)
	if !ok {
		t.Fatal("expected AppError")
	}
	if appErr.Code != errors.ErrUserExists {
		t.Errorf("expected USER_EXISTS, got %s", appErr.Code)
	}
}

func TestAuthService_Register_ValidationError(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	svc := NewAuthService(userRepo, newJWTManager(t))

	tests := []struct {
		name string
		req  models.RegisterRequest
	}{
		{"empty username", models.RegisterRequest{Username: "", Email: "a@b.com", Password: "Password1"}},
		{"empty email", models.RegisterRequest{Username: "johndoe", Email: "", Password: "Password1"}},
		{"empty password", models.RegisterRequest{Username: "johndoe", Email: "a@b.com", Password: ""}},
		{"weak password", models.RegisterRequest{Username: "johndoe", Email: "a@b.com", Password: "weak"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := svc.Register(context.Background(), tt.req)
			if err == nil {
				t.Error("expected validation error")
			}
		})
	}
}

func TestAuthService_Login_Success(t *testing.T) {
	hashedPwd, _ := bcrypt.GenerateFromPassword([]byte("Password1"), bcrypt.MinCost)

	userRepo := &mocks.MockUserRepository{
		FindByEmailWithPasswordFn: func(ctx context.Context, email string) (models.User, string, error) {
			return models.User{
				ID:       1,
				Username: "johndoe",
				Email:    email,
				IsActive: true,
				Role:     "user",
			}, string(hashedPwd), nil
		},
		UpdateLastLoginFn: func(ctx context.Context, userID int) error {
			return nil
		},
	}

	svc := NewAuthService(userRepo, newJWTManager(t))
	user, token, err := svc.Login(context.Background(), models.LoginRequest{
		Email:    "john@example.com",
		Password: "Password1",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != 1 {
		t.Errorf("expected user ID 1, got %d", user.ID)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	hashedPwd, _ := bcrypt.GenerateFromPassword([]byte("Password1"), bcrypt.MinCost)

	userRepo := &mocks.MockUserRepository{
		FindByEmailWithPasswordFn: func(ctx context.Context, email string) (models.User, string, error) {
			return models.User{ID: 1, Email: email}, string(hashedPwd), nil
		},
	}

	svc := NewAuthService(userRepo, newJWTManager(t))
	_, _, err := svc.Login(context.Background(), models.LoginRequest{
		Email:    "john@example.com",
		Password: "WrongPassword1",
	})

	if err == nil {
		t.Fatal("expected error for wrong password")
	}
	appErr, ok := errors.IsAppError(err)
	if !ok {
		t.Fatal("expected AppError")
	}
	if appErr.Code != errors.ErrInvalidCredentials {
		t.Errorf("expected INVALID_CREDENTIALS, got %s", appErr.Code)
	}
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	userRepo := &mocks.MockUserRepository{
		FindByEmailWithPasswordFn: func(ctx context.Context, email string) (models.User, string, error) {
			return models.User{}, "", errors.NewInvalidCredentialsError()
		},
	}

	svc := NewAuthService(userRepo, newJWTManager(t))
	_, _, err := svc.Login(context.Background(), models.LoginRequest{
		Email:    "unknown@example.com",
		Password: "Password1",
	})

	if err == nil {
		t.Fatal("expected error for non-existent user")
	}
}

func TestAuthService_Login_ValidationError(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	svc := NewAuthService(userRepo, newJWTManager(t))

	_, _, err := svc.Login(context.Background(), models.LoginRequest{
		Email:    "",
		Password: "Password1",
	})
	if err == nil {
		t.Error("expected validation error for empty email")
	}
}

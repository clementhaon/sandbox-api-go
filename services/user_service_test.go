package services

import (
	"context"
	"testing"

	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/mocks"
	"github.com/clementhaon/sandbox-api-go/models"
)

func TestUserService_List(t *testing.T) {
	repo := &mocks.MockUserRepository{
		ListFn: func(ctx context.Context, params models.UserListParams) ([]models.User, int, error) {
			return []models.User{
				{ID: 1, Username: "alice", Email: "alice@test.com", IsActive: true, Role: "user"},
				{ID: 2, Username: "bob", Email: "bob@test.com", IsActive: true, Role: "admin"},
			}, 2, nil
		},
	}

	svc := NewUserService(repo)
	resp, err := svc.List(context.Background(), models.UserListParams{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Errorf("expected 2 users, got %d", len(resp.Data))
	}
	if resp.Pagination.Total != 2 {
		t.Errorf("expected total 2, got %d", resp.Pagination.Total)
	}
}

func TestUserService_List_DefaultPagination(t *testing.T) {
	repo := &mocks.MockUserRepository{
		ListFn: func(ctx context.Context, params models.UserListParams) ([]models.User, int, error) {
			return []models.User{}, 0, nil
		},
	}

	svc := NewUserService(repo)
	resp, err := svc.List(context.Background(), models.UserListParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Pagination.Page != 1 {
		t.Errorf("expected page 1, got %d", resp.Pagination.Page)
	}
	if resp.Pagination.PageSize != 20 {
		t.Errorf("expected pageSize 20, got %d", resp.Pagination.PageSize)
	}
}

func TestUserService_GetByID_Success(t *testing.T) {
	repo := &mocks.MockUserRepository{
		GetByIDFn: func(ctx context.Context, id int) (models.User, error) {
			return models.User{ID: 1, Username: "alice", Email: "alice@test.com", IsActive: true, Role: "user"}, nil
		},
	}

	svc := NewUserService(repo)
	user, err := svc.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Username != "alice" {
		t.Errorf("expected 'alice', got '%s'", user.Username)
	}
}

func TestUserService_GetByID_NotFound(t *testing.T) {
	repo := &mocks.MockUserRepository{
		GetByIDFn: func(ctx context.Context, id int) (models.User, error) {
			return models.User{}, errors.NewNotFoundError("User")
		},
	}

	svc := NewUserService(repo)
	_, err := svc.GetByID(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUserService_Create_Success(t *testing.T) {
	repo := &mocks.MockUserRepository{
		ExistsByUsernameOrEmailFn: func(ctx context.Context, username, email string) (bool, error) {
			return false, nil
		},
		CreateFn: func(ctx context.Context, username, email, hashedPassword, firstName, lastName, role string) (models.User, error) {
			return models.User{ID: 1, Username: username, Email: email, Role: role, IsActive: true}, nil
		},
	}

	svc := NewUserService(repo)
	user, err := svc.Create(context.Background(), models.CreateUserRequest{
		Username: "newuser",
		Email:    "new@test.com",
		Password: "password",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Username != "newuser" {
		t.Errorf("expected 'newuser', got '%s'", user.Username)
	}
	if user.Role != "user" {
		t.Errorf("expected default role 'user', got '%s'", user.Role)
	}
}

func TestUserService_Create_MissingFields(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	svc := NewUserService(repo)

	_, err := svc.Create(context.Background(), models.CreateUserRequest{Username: "a", Email: ""})
	if err == nil {
		t.Fatal("expected error for missing fields")
	}
}

func TestUserService_Create_InvalidRole(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	svc := NewUserService(repo)

	_, err := svc.Create(context.Background(), models.CreateUserRequest{
		Username: "test",
		Email:    "t@t.com",
		Password: "pass",
		Role:     "superadmin",
	})
	if err == nil {
		t.Fatal("expected error for invalid role")
	}
}

func TestUserService_Create_UserExists(t *testing.T) {
	repo := &mocks.MockUserRepository{
		ExistsByUsernameOrEmailFn: func(ctx context.Context, username, email string) (bool, error) {
			return true, nil
		},
	}

	svc := NewUserService(repo)
	_, err := svc.Create(context.Background(), models.CreateUserRequest{
		Username: "existing",
		Email:    "existing@test.com",
		Password: "pass",
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

func TestUserService_Update_NotFound(t *testing.T) {
	repo := &mocks.MockUserRepository{
		ExistsFn: func(ctx context.Context, id int) (bool, error) {
			return false, nil
		},
	}

	svc := NewUserService(repo)
	_, err := svc.Update(context.Background(), 999, models.UpdateUserRequest{Email: "new@test.com"})
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestUserService_Update_InvalidRole(t *testing.T) {
	repo := &mocks.MockUserRepository{
		ExistsFn: func(ctx context.Context, id int) (bool, error) {
			return true, nil
		},
	}

	svc := NewUserService(repo)
	_, err := svc.Update(context.Background(), 1, models.UpdateUserRequest{Role: "superadmin"})
	if err == nil {
		t.Fatal("expected error for invalid role")
	}
}

func TestUserService_UpdateStatus_InvalidStatus(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	svc := NewUserService(repo)

	_, err := svc.UpdateStatus(context.Background(), 1, "unknown")
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestUserService_UpdateStatus_Success(t *testing.T) {
	repo := &mocks.MockUserRepository{
		UpdateStatusFn: func(ctx context.Context, id int, isActive bool) (models.User, error) {
			return models.User{ID: id, IsActive: isActive, Role: "user"}, nil
		},
	}

	svc := NewUserService(repo)
	user, err := svc.UpdateStatus(context.Background(), 1, "inactive")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Status != "inactive" {
		t.Errorf("expected status 'inactive', got '%s'", user.Status)
	}
}

func TestUserService_Delete(t *testing.T) {
	deletedID := 0
	repo := &mocks.MockUserRepository{
		DeleteFn: func(ctx context.Context, id int) error {
			deletedID = id
			return nil
		},
	}

	svc := NewUserService(repo)
	err := svc.Delete(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedID != 5 {
		t.Errorf("expected delete ID 5, got %d", deletedID)
	}
}

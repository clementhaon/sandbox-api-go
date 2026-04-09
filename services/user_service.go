package services

import (
	"context"

	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/logger"
	"github.com/clementhaon/sandbox-api-go/models"
	"github.com/clementhaon/sandbox-api-go/repository"

	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	List(ctx context.Context, params models.UserListParams) (models.UsersListResponse, error)
	GetByID(ctx context.Context, id int) (models.UserResponse, error)
	Create(ctx context.Context, req models.CreateUserRequest) (models.UserResponse, error)
	Update(ctx context.Context, id int, req models.UpdateUserRequest) (models.UserResponse, error)
	UpdateStatus(ctx context.Context, id int, status string) (models.UserResponse, error)
	Delete(ctx context.Context, id int) error
}

type userService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) UserService {
	return &userService{userRepo: userRepo}
}

func (s *userService) List(ctx context.Context, params models.UserListParams) (models.UsersListResponse, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	users, total, err := s.userRepo.List(ctx, params)
	if err != nil {
		return models.UsersListResponse{}, err
	}

	userResponses := make([]models.UserResponse, len(users))
	for i, u := range users {
		userResponses[i] = models.UserFromDB(u)
	}

	totalPages := (total + params.PageSize - 1) / params.PageSize
	return models.UsersListResponse{
		Data: userResponses,
		Pagination: models.Pagination{
			Page:       params.Page,
			PageSize:   params.PageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *userService) GetByID(ctx context.Context, id int) (models.UserResponse, error) {
	u, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return models.UserResponse{}, err
	}
	return models.UserFromDB(u), nil
}

func (s *userService) Create(ctx context.Context, req models.CreateUserRequest) (models.UserResponse, error) {
	if req.Email == "" || req.Username == "" || req.Password == "" {
		return models.UserResponse{}, errors.NewBadRequestError("Email, username and password are required")
	}

	if req.Role == "" {
		req.Role = models.RoleUser
	}

	validRole := false
	for _, r := range models.ValidRoles() {
		if req.Role == r {
			validRole = true
			break
		}
	}
	if !validRole {
		return models.UserResponse{}, errors.NewBadRequestError("Invalid role")
	}

	exists, err := s.userRepo.ExistsByUsernameOrEmail(ctx, req.Username, req.Email)
	if err != nil {
		return models.UserResponse{}, err
	}
	if exists {
		return models.UserResponse{}, errors.NewUserExistsError()
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.ErrorContext(ctx, "Error hashing password", err)
		return models.UserResponse{}, errors.NewInternalError().WithCause(err)
	}

	u, err := s.userRepo.Create(ctx, req.Username, req.Email, string(hashedPassword), req.FirstName, req.LastName, req.Role)
	if err != nil {
		return models.UserResponse{}, err
	}
	return models.UserFromDB(u), nil
}

func (s *userService) Update(ctx context.Context, id int, req models.UpdateUserRequest) (models.UserResponse, error) {
	exists, err := s.userRepo.Exists(ctx, id)
	if err != nil {
		return models.UserResponse{}, err
	}
	if !exists {
		return models.UserResponse{}, errors.NewNotFoundError("User not found")
	}

	if req.Role != "" {
		validRole := false
		for _, r := range models.ValidRoles() {
			if req.Role == r {
				validRole = true
				break
			}
		}
		if !validRole {
			return models.UserResponse{}, errors.NewBadRequestError("Invalid role")
		}
	}

	u, err := s.userRepo.Update(ctx, id, req)
	if err != nil {
		return models.UserResponse{}, err
	}
	return models.UserFromDB(u), nil
}

func (s *userService) UpdateStatus(ctx context.Context, id int, status string) (models.UserResponse, error) {
	if status != "active" && status != "inactive" {
		return models.UserResponse{}, errors.NewBadRequestError("Status must be 'active' or 'inactive'")
	}

	isActive := status == "active"
	u, err := s.userRepo.UpdateStatus(ctx, id, isActive)
	if err != nil {
		return models.UserResponse{}, err
	}
	return models.UserFromDB(u), nil
}

func (s *userService) Delete(ctx context.Context, id int) error {
	return s.userRepo.Delete(ctx, id)
}

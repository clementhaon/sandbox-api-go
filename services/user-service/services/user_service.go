package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/clementhaon/sandbox-api-go/pkg/errors"
	"github.com/clementhaon/sandbox-api-go/pkg/logger"
	"github.com/clementhaon/sandbox-api-go/pkg/models"

	"golang.org/x/crypto/bcrypt"
)

type UserListParams struct {
	Page      int
	PageSize  int
	SortBy    string
	SortOrder string
	Search    string
	Role      string
	Status    string
}

type UserService interface {
	List(ctx context.Context, params UserListParams) (models.UsersListResponse, error)
	GetByID(ctx context.Context, id int) (models.UserResponse, error)
	Create(ctx context.Context, req models.CreateUserRequest) (models.UserResponse, error)
	Update(ctx context.Context, id int, req models.UpdateUserRequest) (models.UserResponse, error)
	UpdateStatus(ctx context.Context, id int, status string) (models.UserResponse, error)
	Delete(ctx context.Context, id int) error
}

type userService struct {
	db *sql.DB
}

func NewUserService(db *sql.DB) UserService {
	return &userService{db: db}
}

func (s *userService) List(ctx context.Context, params UserListParams) (models.UsersListResponse, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	validSortFields := map[string]string{
		"id": "id", "email": "email", "username": "username",
		"role": "role", "status": "is_active", "createdAt": "created_at",
	}
	sortField, ok := validSortFields[params.SortBy]
	if !ok {
		sortField = "id"
	}

	sortOrder := strings.ToUpper(params.SortOrder)
	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "ASC"
	}

	baseQuery := `FROM users WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if params.Search != "" {
		baseQuery += fmt.Sprintf(` AND (email ILIKE $%d OR username ILIKE $%d OR first_name ILIKE $%d OR last_name ILIKE $%d)`, argIndex, argIndex, argIndex, argIndex)
		args = append(args, "%"+params.Search+"%")
		argIndex++
	}
	if params.Role != "" {
		baseQuery += fmt.Sprintf(` AND role = $%d`, argIndex)
		args = append(args, params.Role)
		argIndex++
	}
	if params.Status != "" {
		isActive := params.Status == "active"
		baseQuery += fmt.Sprintf(` AND is_active = $%d`, argIndex)
		args = append(args, isActive)
		argIndex++
	}

	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	startTime := time.Now()
	err := s.db.QueryRow(countQuery, args...).Scan(&total)
	logger.LogDatabaseOperation(ctx, "SELECT COUNT", "users", time.Since(startTime), err)
	if err != nil {
		logger.ErrorContext(ctx, "Error counting users", err)
		return models.UsersListResponse{}, errors.NewDatabaseError().WithCause(err)
	}

	offset := (params.Page - 1) * params.PageSize
	selectQuery := fmt.Sprintf(`SELECT id, username, email, first_name, last_name, avatar_url, is_active, last_login_at, role, created_at, updated_at %s ORDER BY %s %s LIMIT $%d OFFSET $%d`,
		baseQuery, sortField, sortOrder, argIndex, argIndex+1)
	args = append(args, params.PageSize, offset)

	startTime = time.Now()
	rows, err := s.db.Query(selectQuery, args...)
	logger.LogDatabaseOperation(ctx, "SELECT", "users", time.Since(startTime), err)
	if err != nil {
		logger.ErrorContext(ctx, "Error querying users", err)
		return models.UsersListResponse{}, errors.NewDatabaseError().WithCause(err)
	}
	defer rows.Close()

	users := []models.UserResponse{}
	for rows.Next() {
		var u models.User
		err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.FirstName, &u.LastName,
			&u.AvatarURL, &u.IsActive, &u.LastLoginAt, &u.Role, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			logger.ErrorContext(ctx, "Error scanning user row", err)
			return models.UsersListResponse{}, errors.NewDatabaseError().WithCause(err)
		}
		users = append(users, models.UserFromDB(u))
	}

	totalPages := (total + params.PageSize - 1) / params.PageSize
	return models.UsersListResponse{
		Data: users,
		Pagination: models.Pagination{
			Page:       params.Page,
			PageSize:   params.PageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *userService) GetByID(ctx context.Context, id int) (models.UserResponse, error) {
	var u models.User
	startTime := time.Now()
	err := s.db.QueryRow(
		`SELECT id, username, email, first_name, last_name, avatar_url, is_active, last_login_at, role, created_at, updated_at
		FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.FirstName, &u.LastName,
		&u.AvatarURL, &u.IsActive, &u.LastLoginAt, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	logger.LogDatabaseOperation(ctx, "SELECT", "users", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return models.UserResponse{}, errors.NewNotFoundError("User not found")
	} else if err != nil {
		logger.ErrorContext(ctx, "Error fetching user", err)
		return models.UserResponse{}, errors.NewDatabaseError().WithCause(err)
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

	var existingID int
	startTime := time.Now()
	err := s.db.QueryRow("SELECT id FROM users WHERE username = $1 OR email = $2", req.Username, req.Email).Scan(&existingID)
	logger.LogDatabaseOperation(ctx, "SELECT", "users", time.Since(startTime), err)

	if err == nil {
		return models.UserResponse{}, errors.NewUserExistsError()
	} else if err != sql.ErrNoRows {
		logger.ErrorContext(ctx, "Database error checking existing user", err)
		return models.UserResponse{}, errors.NewDatabaseError().WithCause(err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.ErrorContext(ctx, "Error hashing password", err)
		return models.UserResponse{}, errors.NewInternalError().WithCause(err)
	}

	var u models.User
	startTime = time.Now()
	err = s.db.QueryRow(
		`INSERT INTO users (username, email, password, first_name, last_name, is_active, role)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), true, $6)
		RETURNING id, username, email, first_name, last_name, avatar_url, is_active, last_login_at, role, created_at, updated_at`,
		req.Username, req.Email, string(hashedPassword), req.FirstName, req.LastName, req.Role,
	).Scan(&u.ID, &u.Username, &u.Email, &u.FirstName, &u.LastName,
		&u.AvatarURL, &u.IsActive, &u.LastLoginAt, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	logger.LogDatabaseOperation(ctx, "INSERT", "users", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error creating user", err)
		return models.UserResponse{}, errors.NewDatabaseError().WithCause(err)
	}

	return models.UserFromDB(u), nil
}

func (s *userService) Update(ctx context.Context, id int, req models.UpdateUserRequest) (models.UserResponse, error) {
	var existingID int
	startTime := time.Now()
	err := s.db.QueryRow("SELECT id FROM users WHERE id = $1", id).Scan(&existingID)
	logger.LogDatabaseOperation(ctx, "SELECT", "users", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return models.UserResponse{}, errors.NewNotFoundError("User not found")
	} else if err != nil {
		logger.ErrorContext(ctx, "Database error checking user", err)
		return models.UserResponse{}, errors.NewDatabaseError().WithCause(err)
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

	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Email != "" {
		setParts = append(setParts, fmt.Sprintf("email = $%d", argIndex))
		args = append(args, req.Email)
		argIndex++
	}
	if req.Username != "" {
		setParts = append(setParts, fmt.Sprintf("username = $%d", argIndex))
		args = append(args, req.Username)
		argIndex++
	}
	if req.FirstName != "" {
		setParts = append(setParts, fmt.Sprintf("first_name = $%d", argIndex))
		args = append(args, req.FirstName)
		argIndex++
	}
	if req.LastName != "" {
		setParts = append(setParts, fmt.Sprintf("last_name = $%d", argIndex))
		args = append(args, req.LastName)
		argIndex++
	}
	if req.AvatarURL != "" {
		setParts = append(setParts, fmt.Sprintf("avatar_url = $%d", argIndex))
		args = append(args, req.AvatarURL)
		argIndex++
	}
	if req.Role != "" {
		setParts = append(setParts, fmt.Sprintf("role = $%d", argIndex))
		args = append(args, req.Role)
		argIndex++
	}

	if len(setParts) == 0 {
		return models.UserResponse{}, errors.NewBadRequestError("No fields to update")
	}

	setParts = append(setParts, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf(`UPDATE users SET %s WHERE id = $%d
		RETURNING id, username, email, first_name, last_name, avatar_url, is_active, last_login_at, role, created_at, updated_at`,
		strings.Join(setParts, ", "), argIndex)

	var u models.User
	startTime = time.Now()
	err = s.db.QueryRow(query, args...).Scan(&u.ID, &u.Username, &u.Email, &u.FirstName, &u.LastName,
		&u.AvatarURL, &u.IsActive, &u.LastLoginAt, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	logger.LogDatabaseOperation(ctx, "UPDATE", "users", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error updating user", err)
		return models.UserResponse{}, errors.NewDatabaseError().WithCause(err)
	}

	return models.UserFromDB(u), nil
}

func (s *userService) UpdateStatus(ctx context.Context, id int, status string) (models.UserResponse, error) {
	isActive := status == "active"
	if status != "active" && status != "inactive" {
		return models.UserResponse{}, errors.NewBadRequestError("Status must be 'active' or 'inactive'")
	}

	var u models.User
	startTime := time.Now()
	err := s.db.QueryRow(
		`UPDATE users SET is_active = $1, updated_at = NOW() WHERE id = $2
		RETURNING id, username, email, first_name, last_name, avatar_url, is_active, last_login_at, role, created_at, updated_at`,
		isActive, id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.FirstName, &u.LastName,
		&u.AvatarURL, &u.IsActive, &u.LastLoginAt, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	logger.LogDatabaseOperation(ctx, "UPDATE", "users", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return models.UserResponse{}, errors.NewNotFoundError("User not found")
	} else if err != nil {
		logger.ErrorContext(ctx, "Error updating user status", err)
		return models.UserResponse{}, errors.NewDatabaseError().WithCause(err)
	}

	return models.UserFromDB(u), nil
}

func (s *userService) Delete(ctx context.Context, id int) error {
	startTime := time.Now()
	result, err := s.db.Exec("DELETE FROM users WHERE id = $1", id)
	logger.LogDatabaseOperation(ctx, "DELETE", "users", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error deleting user", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.NewNotFoundError("User not found")
	}

	return nil
}

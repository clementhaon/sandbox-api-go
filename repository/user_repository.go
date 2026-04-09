package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/logger"
	"github.com/clementhaon/sandbox-api-go/models"
)

type UserRepository interface {
	// Auth operations
	ExistsByUsernameOrEmail(ctx context.Context, username, email string) (bool, error)
	CreateAuth(ctx context.Context, username, email, hashedPassword string) (models.User, error)
	FindByEmailWithPassword(ctx context.Context, email string) (models.User, string, error)
	UpdateLastLogin(ctx context.Context, userID int) error

	// User CRUD
	List(ctx context.Context, params models.UserListParams) ([]models.User, int, error)
	GetByID(ctx context.Context, id int) (models.User, error)
	Exists(ctx context.Context, id int) (bool, error)
	Create(ctx context.Context, username, email, hashedPassword, firstName, lastName, role string) (models.User, error)
	Update(ctx context.Context, id int, req models.UpdateUserRequest) (models.User, error)
	UpdateStatus(ctx context.Context, id int, isActive bool) (models.User, error)
	Delete(ctx context.Context, id int) error

	// Profile operations
	UpdateProfile(ctx context.Context, userID int, firstName, lastName, avatarURL sql.NullString) error
}

type postgresUserRepo struct {
	db *sql.DB
}

func NewPostgresUserRepository(db *sql.DB) UserRepository {
	return &postgresUserRepo{db: db}
}

const userColumns = `id, username, email, first_name, last_name, avatar_url, is_active, last_login_at, role, created_at, updated_at`

func scanUser(row interface{ Scan(...any) error }) (models.User, error) {
	var u models.User
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.FirstName, &u.LastName,
		&u.AvatarURL, &u.IsActive, &u.LastLoginAt, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

// --- Auth operations ---

func (r *postgresUserRepo) ExistsByUsernameOrEmail(ctx context.Context, username, email string) (bool, error) {
	var id int
	startTime := time.Now()
	err := r.db.QueryRowContext(ctx, "SELECT id FROM users WHERE username = $1 OR email = $2", username, email).Scan(&id)
	logger.LogDatabaseOperation(ctx, "SELECT", "users", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		logger.ErrorContext(ctx, "Database error checking existing user", err)
		return false, errors.NewDatabaseError().WithCause(err)
	}
	return true, nil
}

func (r *postgresUserRepo) CreateAuth(ctx context.Context, username, email, hashedPassword string) (models.User, error) {
	startTime := time.Now()
	u, err := scanUser(r.db.QueryRowContext(ctx,
		`INSERT INTO users (username, email, password, is_active, role)
		VALUES ($1, $2, $3, true, 'user')
		RETURNING `+userColumns,
		username, email, hashedPassword,
	))
	logger.LogDatabaseOperation(ctx, "INSERT", "users", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error creating user", err)
		return models.User{}, errors.NewDatabaseError().WithCause(err)
	}
	return u, nil
}

func (r *postgresUserRepo) FindByEmailWithPassword(ctx context.Context, email string) (models.User, string, error) {
	var u models.User
	var hashedPassword string
	startTime := time.Now()
	err := r.db.QueryRowContext(ctx,
		`SELECT id, username, email, password, first_name, last_name, avatar_url, is_active, last_login_at, role, created_at, updated_at
		FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Username, &u.Email, &hashedPassword, &u.FirstName,
		&u.LastName, &u.AvatarURL, &u.IsActive, &u.LastLoginAt,
		&u.Role, &u.CreatedAt, &u.UpdatedAt)
	logger.LogDatabaseOperation(ctx, "SELECT", "users", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return models.User{}, "", errors.NewInvalidCredentialsError()
	}
	if err != nil {
		logger.ErrorContext(ctx, "Database error during login", err)
		return models.User{}, "", errors.NewDatabaseError().WithCause(err)
	}
	return u, hashedPassword, nil
}

func (r *postgresUserRepo) UpdateLastLogin(ctx context.Context, userID int) error {
	startTime := time.Now()
	_, err := r.db.ExecContext(ctx, "UPDATE users SET last_login_at = NOW() WHERE id = $1", userID)
	logger.LogDatabaseOperation(ctx, "UPDATE", "users", time.Since(startTime), err)
	return err
}

// --- User CRUD ---

func (r *postgresUserRepo) List(ctx context.Context, params models.UserListParams) ([]models.User, int, error) {
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
	startTime := time.Now()
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) "+baseQuery, args...).Scan(&total)
	logger.LogDatabaseOperation(ctx, "SELECT COUNT", "users", time.Since(startTime), err)
	if err != nil {
		logger.ErrorContext(ctx, "Error counting users", err)
		return nil, 0, errors.NewDatabaseError().WithCause(err)
	}

	offset := (params.Page - 1) * params.PageSize
	selectQuery := fmt.Sprintf(`SELECT %s %s ORDER BY %s %s LIMIT $%d OFFSET $%d`,
		userColumns, baseQuery, sortField, sortOrder, argIndex, argIndex+1)
	args = append(args, params.PageSize, offset)

	startTime = time.Now()
	rows, err := r.db.QueryContext(ctx, selectQuery, args...)
	logger.LogDatabaseOperation(ctx, "SELECT", "users", time.Since(startTime), err)
	if err != nil {
		logger.ErrorContext(ctx, "Error querying users", err)
		return nil, 0, errors.NewDatabaseError().WithCause(err)
	}
	defer rows.Close()

	users := []models.User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			logger.ErrorContext(ctx, "Error scanning user row", err)
			return nil, 0, errors.NewDatabaseError().WithCause(err)
		}
		users = append(users, u)
	}

	return users, total, nil
}

func (r *postgresUserRepo) GetByID(ctx context.Context, id int) (models.User, error) {
	startTime := time.Now()
	u, err := scanUser(r.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE id = $1`, id))
	logger.LogDatabaseOperation(ctx, "SELECT", "users", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return models.User{}, errors.NewNotFoundError("User")
	}
	if err != nil {
		logger.ErrorContext(ctx, "Error fetching user", err)
		return models.User{}, errors.NewDatabaseError().WithCause(err)
	}
	return u, nil
}

func (r *postgresUserRepo) Exists(ctx context.Context, id int) (bool, error) {
	var existingID int
	startTime := time.Now()
	err := r.db.QueryRowContext(ctx, "SELECT id FROM users WHERE id = $1", id).Scan(&existingID)
	logger.LogDatabaseOperation(ctx, "SELECT", "users", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		logger.ErrorContext(ctx, "Database error checking user", err)
		return false, errors.NewDatabaseError().WithCause(err)
	}
	return true, nil
}

func (r *postgresUserRepo) Create(ctx context.Context, username, email, hashedPassword, firstName, lastName, role string) (models.User, error) {
	startTime := time.Now()
	u, err := scanUser(r.db.QueryRowContext(ctx,
		`INSERT INTO users (username, email, password, first_name, last_name, is_active, role)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), true, $6)
		RETURNING `+userColumns,
		username, email, hashedPassword, firstName, lastName, role,
	))
	logger.LogDatabaseOperation(ctx, "INSERT", "users", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error creating user", err)
		return models.User{}, errors.NewDatabaseError().WithCause(err)
	}
	return u, nil
}

func (r *postgresUserRepo) Update(ctx context.Context, id int, req models.UpdateUserRequest) (models.User, error) {
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
		return models.User{}, errors.NewBadRequestError("No fields to update")
	}

	setParts = append(setParts, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf(`UPDATE users SET %s WHERE id = $%d RETURNING %s`,
		strings.Join(setParts, ", "), argIndex, userColumns)

	startTime := time.Now()
	u, err := scanUser(r.db.QueryRowContext(ctx, query, args...))
	logger.LogDatabaseOperation(ctx, "UPDATE", "users", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error updating user", err)
		return models.User{}, errors.NewDatabaseError().WithCause(err)
	}
	return u, nil
}

func (r *postgresUserRepo) UpdateStatus(ctx context.Context, id int, isActive bool) (models.User, error) {
	startTime := time.Now()
	u, err := scanUser(r.db.QueryRowContext(ctx,
		`UPDATE users SET is_active = $1, updated_at = NOW() WHERE id = $2 RETURNING `+userColumns,
		isActive, id,
	))
	logger.LogDatabaseOperation(ctx, "UPDATE", "users", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return models.User{}, errors.NewNotFoundError("User not found")
	}
	if err != nil {
		logger.ErrorContext(ctx, "Error updating user status", err)
		return models.User{}, errors.NewDatabaseError().WithCause(err)
	}
	return u, nil
}

func (r *postgresUserRepo) Delete(ctx context.Context, id int) error {
	startTime := time.Now()
	result, err := r.db.ExecContext(ctx, "DELETE FROM users WHERE id = $1", id)
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

// --- Profile operations ---

func (r *postgresUserRepo) UpdateProfile(ctx context.Context, userID int, firstName, lastName, avatarURL sql.NullString) error {
	startTime := time.Now()
	_, err := r.db.ExecContext(ctx,
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
		return errors.NewDatabaseError().WithCause(err)
	}
	return nil
}

package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"sandbox-api-go/database"
	"sandbox-api-go/errors"
	"sandbox-api-go/logger"
	"sandbox-api-go/models"

	"golang.org/x/crypto/bcrypt"
)

// ListUsers handles GET /users - paginated list with filters
func ListUsers(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	// Parse query params
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	sortBy := r.URL.Query().Get("sortBy")
	validSortFields := map[string]string{
		"id":        "id",
		"email":     "email",
		"username":  "username",
		"role":      "role",
		"status":    "is_active",
		"createdAt": "created_at",
	}
	sortField, ok := validSortFields[sortBy]
	if !ok {
		sortField = "id"
	}

	sortOrder := strings.ToUpper(r.URL.Query().Get("sortOrder"))
	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "ASC"
	}

	search := r.URL.Query().Get("search")
	role := r.URL.Query().Get("role")
	status := r.URL.Query().Get("status")

	// Build query
	baseQuery := `FROM users WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if search != "" {
		baseQuery += fmt.Sprintf(` AND (email ILIKE $%d OR username ILIKE $%d OR first_name ILIKE $%d OR last_name ILIKE $%d)`, argIndex, argIndex, argIndex, argIndex)
		args = append(args, "%"+search+"%")
		argIndex++
	}

	if role != "" {
		baseQuery += fmt.Sprintf(` AND role = $%d`, argIndex)
		args = append(args, role)
		argIndex++
	}

	if status != "" {
		isActive := status == "active"
		baseQuery += fmt.Sprintf(` AND is_active = $%d`, argIndex)
		args = append(args, isActive)
		argIndex++
	}

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	startTime := time.Now()
	err := database.DB.QueryRow(countQuery, args...).Scan(&total)
	logger.LogDatabaseOperation(r.Context(), "SELECT COUNT", "users", time.Since(startTime), err)
	if err != nil {
		logger.ErrorContext(r.Context(), "Error counting users", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	// Get users
	offset := (page - 1) * pageSize
	selectQuery := fmt.Sprintf(`SELECT id, username, email, first_name, last_name, avatar_url, is_active, last_login_at, role, created_at, updated_at %s ORDER BY %s %s LIMIT $%d OFFSET $%d`,
		baseQuery, sortField, sortOrder, argIndex, argIndex+1)
	args = append(args, pageSize, offset)

	startTime = time.Now()
	rows, err := database.DB.Query(selectQuery, args...)
	logger.LogDatabaseOperation(r.Context(), "SELECT", "users", time.Since(startTime), err)
	if err != nil {
		logger.ErrorContext(r.Context(), "Error querying users", err)
		return errors.NewDatabaseError().WithCause(err)
	}
	defer rows.Close()

	users := []models.UserResponse{}
	for rows.Next() {
		var u models.User
		err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.FirstName, &u.LastName,
			&u.AvatarURL, &u.IsActive, &u.LastLoginAt, &u.Role, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			logger.ErrorContext(r.Context(), "Error scanning user row", err)
			return errors.NewDatabaseError().WithCause(err)
		}
		users = append(users, models.UserFromDB(u))
	}

	totalPages := (total + pageSize - 1) / pageSize
	response := models.UsersListResponse{
		Data: users,
		Pagination: models.Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	json.NewEncoder(w).Encode(response)
	return nil
}

// GetUser handles GET /users/{id}
func GetUser(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return errors.NewBadRequestError("Invalid user ID")
	}

	var u models.User
	startTime := time.Now()
	err = database.DB.QueryRow(
		`SELECT id, username, email, first_name, last_name, avatar_url, is_active, last_login_at, role, created_at, updated_at
		FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.FirstName, &u.LastName,
		&u.AvatarURL, &u.IsActive, &u.LastLoginAt, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	logger.LogDatabaseOperation(r.Context(), "SELECT", "users", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return errors.NewNotFoundError("User not found")
	} else if err != nil {
		logger.ErrorContext(r.Context(), "Error fetching user", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	json.NewEncoder(w).Encode(models.UserFromDB(u))
	return nil
}

// CreateUser handles POST /users
func CreateUser(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewInvalidJSONError()
	}

	// Validate required fields
	if req.Email == "" || req.Username == "" || req.Password == "" {
		return errors.NewBadRequestError("Email, username and password are required")
	}

	// Set default role if not provided
	if req.Role == "" {
		req.Role = models.RoleUser
	}

	// Validate role
	validRole := false
	for _, r := range models.ValidRoles() {
		if req.Role == r {
			validRole = true
			break
		}
	}
	if !validRole {
		return errors.NewBadRequestError("Invalid role")
	}

	// Check if user exists
	var existingID int
	startTime := time.Now()
	err := database.DB.QueryRow("SELECT id FROM users WHERE username = $1 OR email = $2", req.Username, req.Email).Scan(&existingID)
	logger.LogDatabaseOperation(r.Context(), "SELECT", "users", time.Since(startTime), err)

	if err == nil {
		return errors.NewUserExistsError()
	} else if err != sql.ErrNoRows {
		logger.ErrorContext(r.Context(), "Database error checking existing user", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.ErrorContext(r.Context(), "Error hashing password", err)
		return errors.NewInternalError().WithCause(err)
	}

	// Insert user
	var u models.User
	startTime = time.Now()
	err = database.DB.QueryRow(
		`INSERT INTO users (username, email, password, first_name, last_name, is_active, role)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), true, $6)
		RETURNING id, username, email, first_name, last_name, avatar_url, is_active, last_login_at, role, created_at, updated_at`,
		req.Username, req.Email, string(hashedPassword), req.FirstName, req.LastName, req.Role,
	).Scan(&u.ID, &u.Username, &u.Email, &u.FirstName, &u.LastName,
		&u.AvatarURL, &u.IsActive, &u.LastLoginAt, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	logger.LogDatabaseOperation(r.Context(), "INSERT", "users", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(r.Context(), "Error creating user", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.UserFromDB(u))
	return nil
}

// UpdateUser handles PUT /users/{id}
func UpdateUser(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return errors.NewBadRequestError("Invalid user ID")
	}

	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewInvalidJSONError()
	}

	// Check if user exists
	var existingID int
	startTime := time.Now()
	err = database.DB.QueryRow("SELECT id FROM users WHERE id = $1", id).Scan(&existingID)
	logger.LogDatabaseOperation(r.Context(), "SELECT", "users", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return errors.NewNotFoundError("User not found")
	} else if err != nil {
		logger.ErrorContext(r.Context(), "Database error checking user", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	// Validate role if provided
	if req.Role != "" {
		validRole := false
		for _, r := range models.ValidRoles() {
			if req.Role == r {
				validRole = true
				break
			}
		}
		if !validRole {
			return errors.NewBadRequestError("Invalid role")
		}
	}

	// Build update query dynamically
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
		return errors.NewBadRequestError("No fields to update")
	}

	setParts = append(setParts, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf(`UPDATE users SET %s WHERE id = $%d
		RETURNING id, username, email, first_name, last_name, avatar_url, is_active, last_login_at, role, created_at, updated_at`,
		strings.Join(setParts, ", "), argIndex)

	var u models.User
	startTime = time.Now()
	err = database.DB.QueryRow(query, args...).Scan(&u.ID, &u.Username, &u.Email, &u.FirstName, &u.LastName,
		&u.AvatarURL, &u.IsActive, &u.LastLoginAt, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	logger.LogDatabaseOperation(r.Context(), "UPDATE", "users", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(r.Context(), "Error updating user", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	json.NewEncoder(w).Encode(models.UserFromDB(u))
	return nil
}

// UpdateUserStatus handles PATCH /users/{id}/status
func UpdateUserStatus(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return errors.NewBadRequestError("Invalid user ID")
	}

	var req models.UpdateUserStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewInvalidJSONError()
	}

	// Validate status
	isActive := req.Status == "active"
	if req.Status != "active" && req.Status != "inactive" {
		return errors.NewBadRequestError("Status must be 'active' or 'inactive'")
	}

	var u models.User
	startTime := time.Now()
	err = database.DB.QueryRow(
		`UPDATE users SET is_active = $1, updated_at = NOW() WHERE id = $2
		RETURNING id, username, email, first_name, last_name, avatar_url, is_active, last_login_at, role, created_at, updated_at`,
		isActive, id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.FirstName, &u.LastName,
		&u.AvatarURL, &u.IsActive, &u.LastLoginAt, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	logger.LogDatabaseOperation(r.Context(), "UPDATE", "users", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return errors.NewNotFoundError("User not found")
	} else if err != nil {
		logger.ErrorContext(r.Context(), "Error updating user status", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	json.NewEncoder(w).Encode(models.UserFromDB(u))
	return nil
}

// DeleteUser handles DELETE /users/{id}
func DeleteUser(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return errors.NewBadRequestError("Invalid user ID")
	}

	startTime := time.Now()
	result, err := database.DB.Exec("DELETE FROM users WHERE id = $1", id)
	logger.LogDatabaseOperation(r.Context(), "DELETE", "users", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(r.Context(), "Error deleting user", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.NewNotFoundError("User not found")
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sandbox-api-go/auth"
	"sandbox-api-go/database"
	"sandbox-api-go/models"
	"sandbox-api-go/errors"
	"sandbox-api-go/logger"
	"sandbox-api-go/metrics"
	"sandbox-api-go/validation"
	"golang.org/x/crypto/bcrypt"
	"sandbox-api-go/middleware"
	"time"
)

// HandleRegister gère l'inscription d'un nouvel utilisateur
func HandleRegister(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		return errors.NewMethodNotAllowedError()
	}

	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.WarnContext(r.Context(), "Invalid JSON in register request", map[string]interface{}{
			"error": err.Error(),
		})
		return errors.NewInvalidJSONError()
	}

	// Validation
	if validationErr := validation.ValidateRegisterRequest(req.Username, req.Email, req.Password); validationErr != nil {
		return validationErr
	}

	// Vérifier si l'utilisateur existe déjà
	var existingUser models.User
	startTime := time.Now()
	err := database.DB.QueryRow("SELECT id FROM users WHERE username = $1 OR email = $2", req.Username, req.Email).Scan(&existingUser.ID)
	logger.LogDatabaseOperation(r.Context(), "SELECT", "users", time.Since(startTime), err)
	
	if err == nil {
		return errors.NewUserExistsError()
	} else if err != sql.ErrNoRows {
		logger.ErrorContext(r.Context(), "Database error checking existing user", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	// Hasher le mot de passe
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.ErrorContext(r.Context(), "Error hashing password", err)
		return errors.NewInternalError().WithCause(err)
	}

	// Créer le nouvel utilisateur dans la base de données
	var newUser models.User
	startTime = time.Now()
	err = database.DB.QueryRow(
		`INSERT INTO users (username, email, password, is_active, role)
		VALUES ($1, $2, $3, true, 'user')
		RETURNING id, username, email, first_name, last_name, avatar_url, is_active, last_login_at, role, created_at, updated_at`,
		req.Username, req.Email, string(hashedPassword),
	).Scan(&newUser.ID, &newUser.Username, &newUser.Email, &newUser.FirstName, &newUser.LastName,
		&newUser.AvatarURL, &newUser.IsActive, &newUser.LastLoginAt, &newUser.Role, &newUser.CreatedAt, &newUser.UpdatedAt)
	logger.LogDatabaseOperation(r.Context(), "INSERT", "users", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(r.Context(), "Error creating user", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	// Générer le token
	token, err := auth.GenerateToken(newUser)
	if err != nil {
		logger.ErrorContext(r.Context(), "Error generating JWT token", err)
		return errors.NewInternalError().WithCause(err)
	}

	// Créer le cookie HTTPOnly sécurisé
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		MaxAge:   24 * 60 * 60, // 24 heures en secondes
		HttpOnly: true,         // Empêche l'accès via JavaScript
		Secure:   false,        // À mettre à true en production avec HTTPS
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, cookie)

	// Add user ID to context for logging
	ctx := r.Context()
	if requestID, ok := ctx.Value(logger.RequestIDKey).(string); ok {
		logger.InfoContext(ctx, "User registered successfully", map[string]interface{}{
			"user_id":    newUser.ID,
			"username":   newUser.Username,
			"request_id": requestID,
		})
	}
	metrics.RecordAuthAttempt("register", "success")

	// Réponse sans le token (maintenant dans le cookie)
	response := models.AuthResponse{
		Token:   "", // Token retiré de la réponse JSON
		User:    newUser,
		Message: "Inscription réussie",
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
	return nil
}

// HandleLogin gère la connexion d'un utilisateur
func HandleLogin(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		return errors.NewMethodNotAllowedError()
	}

	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.WarnContext(r.Context(), "Invalid JSON in login request", map[string]interface{}{
			"error": err.Error(),
		})
		return errors.NewInvalidJSONError()
	}

	// Validation
	if validationErr := validation.ValidateLoginRequest(req.Email, req.Password); validationErr != nil {
		return validationErr
	}

	// Chercher l'utilisateur dans la base de données
	var foundUser models.User
	var hashedPassword string
	startTime := time.Now()
	err := database.DB.QueryRow(
		`SELECT id, username, email, password, first_name, last_name, avatar_url, is_active, last_login_at, role, created_at, updated_at
		FROM users WHERE Email = $1`,
		req.Email,
	).Scan(&foundUser.ID, &foundUser.Username, &foundUser.Email, &hashedPassword, &foundUser.FirstName,
		&foundUser.LastName, &foundUser.AvatarURL, &foundUser.IsActive, &foundUser.LastLoginAt,
		&foundUser.Role, &foundUser.CreatedAt, &foundUser.UpdatedAt)
	logger.LogDatabaseOperation(r.Context(), "SELECT", "users", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		logger.WarnContext(r.Context(), "Login attempt with non-existent email", map[string]interface{}{
			"email": req.Email,
		})
		return errors.NewInvalidCredentialsError()
	} else if err != nil {
		logger.ErrorContext(r.Context(), "Database error during login", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	// Vérifier le mot de passe
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)); err != nil {
		logger.WarnContext(r.Context(), "Login attempt with invalid password", map[string]interface{}{
			"user_id": foundUser.ID,
			"email":   req.Email,
		})
		return errors.NewInvalidCredentialsError()
	}

	// Mettre à jour last_login_at
	startTime = time.Now()
	_, err = database.DB.Exec("UPDATE users SET last_login_at = NOW() WHERE id = $1", foundUser.ID)
	logger.LogDatabaseOperation(r.Context(), "UPDATE", "users", time.Since(startTime), err)
	if err != nil {
		logger.WarnContext(r.Context(), "Failed to update last_login_at", map[string]interface{}{
			"user_id": foundUser.ID,
			"error":   err.Error(),
		})
		// Non-blocking error, continue with login
	}

	// Générer le token
	token, err := auth.GenerateToken(foundUser)
	if err != nil {
		logger.ErrorContext(r.Context(), "Error generating JWT token for login", err)
		return errors.NewInternalError().WithCause(err)
	}

	// Créer le cookie HTTPOnly sécurisé
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		MaxAge:   24 * 60 * 60, // 24 heures en secondes
		HttpOnly: true,         // Empêche l'accès via JavaScript
		Secure:   false,        // À mettre à true en production avec HTTPS
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, cookie)

	// Log successful login
	logger.InfoContext(r.Context(), "User logged in successfully", map[string]interface{}{
		"user_id":  foundUser.ID,
		"username": foundUser.Username,
		"email":    foundUser.Email,
	})
	metrics.RecordAuthAttempt("login", "success")

	// Réponse sans le token (maintenant dans le cookie)
	response := models.AuthResponse{
		Token:   "", // Token retiré de la réponse JSON
		User:    foundUser,
		Message: "Connexion réussie",
	}

	json.NewEncoder(w).Encode(response)
	return nil
}

// HandleLogout gère la déconnexion d'un utilisateur
func HandleLogout(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		return errors.NewMethodNotAllowedError()
	}

	// Log logout attempt
	logger.InfoContext(r.Context(), "User logout requested")

	// Supprimer le cookie en définissant MaxAge à -1
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // Supprime le cookie
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, cookie)

	// Réponse de confirmation
	response := map[string]string{
		"message": "Déconnexion réussie",
	}

	json.NewEncoder(w).Encode(response)
	return nil
} 


func HandleGetUser(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		return errors.NewMethodNotAllowedError()
	}

	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		logger.ErrorContext(r.Context(), "Missing user context in authenticated request", nil)
		return errors.NewInternalError().WithDetails(map[string]interface{}{
			"issue": "user_context_missing",
		})
	}

	json.NewEncoder(w).Encode(claims)
	return nil
}





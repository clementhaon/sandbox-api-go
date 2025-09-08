package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sandbox-api-go/auth"
	"sandbox-api-go/database"
	"sandbox-api-go/models"
	"golang.org/x/crypto/bcrypt"
	"sandbox-api-go/middleware"
)

// HandleRegister gère l'inscription d'un nouvel utilisateur
func HandleRegister(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Méthode non autorisée"})
		return
	}

	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "JSON invalide"})
		return
	}

	// Validation
	if req.Username == "" || req.Email == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Tous les champs sont requis"})
		return
	}

	// Vérifier si l'utilisateur existe déjà
	var existingUser models.User
	err := database.DB.QueryRow("SELECT id FROM users WHERE username = $1 OR email = $2", req.Username, req.Email).Scan(&existingUser.ID)
	if err == nil {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "Nom d'utilisateur ou email déjà utilisé"})
		return
	} else if err != sql.ErrNoRows {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Erreur lors de la vérification de l'utilisateur"})
		return
	}

	// Hasher le mot de passe
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Erreur lors du hachage du mot de passe"})
		return
	}

	// Créer le nouvel utilisateur dans la base de données
	var newUser models.User
	err = database.DB.QueryRow(
		"INSERT INTO users (username, email, password) VALUES ($1, $2, $3) RETURNING id, username, email",
		req.Username, req.Email, string(hashedPassword),
	).Scan(&newUser.ID, &newUser.Username, &newUser.Email)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Erreur lors de la création de l'utilisateur"})
		return
	}

	// Générer le token
	token, err := auth.GenerateToken(newUser)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Erreur lors de la génération du token"})
		return
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

	// Réponse sans le token (maintenant dans le cookie)
	response := models.AuthResponse{
		Token:   "", // Token retiré de la réponse JSON
		User:    newUser,
		Message: "Inscription réussie",
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// HandleLogin gère la connexion d'un utilisateur
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Méthode non autorisée"})
		return
	}

	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "JSON invalide"})
		return
	}

	// Validation
	if req.Email == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Nom d'utilisateur et mot de passe requis"})
		return
	}

	// Chercher l'utilisateur dans la base de données
	var foundUser models.User
	var hashedPassword string
	err := database.DB.QueryRow(
		"SELECT id, username, email, password FROM users WHERE Email = $1",
		req.Email,
	).Scan(&foundUser.ID, &foundUser.Username, &foundUser.Email, &hashedPassword)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Identifiants invalides 1"})
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Erreur lors de la recherche de l'utilisateur"})
		return
	}

	// Vérifier le mot de passe
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Identifiants invalides 2"})
		return
	}

	// Générer le token
	token, err := auth.GenerateToken(foundUser)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Erreur lors de la génération du token"})
		return
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

	// Réponse sans le token (maintenant dans le cookie)
	response := models.AuthResponse{
		Token:   "", // Token retiré de la réponse JSON
		User:    foundUser,
		Message: "Connexion réussie",
	}

	json.NewEncoder(w).Encode(response)
}

// HandleLogout gère la déconnexion d'un utilisateur
func HandleLogout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Méthode non autorisée"})
		return
	}

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
} 


func HandleGetUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Méthode non autorisée"})
		return
	}

	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Erreur d'authentification"})
		return
	}

	json.NewEncoder(w).Encode(claims)
}





package handlers

import (
	"encoding/json"
	"net/http"
	"sandbox-api-go/auth"
	"sandbox-api-go/models"
)

// Stockage en mémoire pour les utilisateurs (en prod, utilisez une vraie DB)
var users []models.User
var userNextID = 1

func init() {
	// Utilisateur de test
	users = []models.User{
		{ID: 1, Username: "admin", Email: "admin@example.com", Password: "password123"},
	}
	userNextID = 2
}

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
	for _, user := range users {
		if user.Username == req.Username {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": "Nom d'utilisateur déjà pris"})
			return
		}
		if user.Email == req.Email {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": "Email déjà utilisé"})
			return
		}
	}

	// Créer le nouvel utilisateur
	newUser := models.User{
		ID:       userNextID,
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password, // En prod, hasher le mot de passe !
	}
	userNextID++
	users = append(users, newUser)

	// Générer le token
	token, err := auth.GenerateToken(newUser)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Erreur lors de la génération du token"})
		return
	}

	// Réponse
	response := models.AuthResponse{
		Token:   token,
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
	if req.Username == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Nom d'utilisateur et mot de passe requis"})
		return
	}

	// Chercher l'utilisateur
	var foundUser *models.User
	for _, user := range users {
		if user.Username == req.Username && user.Password == req.Password {
			foundUser = &user
			break
		}
	}

	if foundUser == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Identifiants invalides"})
		return
	}

	// Générer le token
	token, err := auth.GenerateToken(*foundUser)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Erreur lors de la génération du token"})
		return
	}

	// Réponse
	response := models.AuthResponse{
		Token:   token,
		User:    *foundUser,
		Message: "Connexion réussie",
	}

	json.NewEncoder(w).Encode(response)
} 
package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/clementhaon/sandbox-api-go/pkg/models"
)

type InternalHandler struct {
	db *sql.DB
}

func NewInternalHandler(db *sql.DB) *InternalHandler {
	return &InternalHandler{db: db}
}

func (h *InternalHandler) GetUserBrief(w http.ResponseWriter, r *http.Request) error {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	var user models.UserBrief
	var avatarURL sql.NullString
	err = h.db.QueryRow("SELECT id, username, avatar_url FROM users WHERE id = $1", id).
		Scan(&user.ID, &user.Username, &avatarURL)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		return nil
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return nil
	}
	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
	return nil
}

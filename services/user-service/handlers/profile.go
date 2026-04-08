package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/clementhaon/sandbox-api-go/pkg/errors"
	"github.com/clementhaon/sandbox-api-go/pkg/logger"
	"github.com/clementhaon/sandbox-api-go/pkg/middleware"
	"github.com/clementhaon/sandbox-api-go/pkg/models"
	"github.com/clementhaon/sandbox-api-go/services/user-service/services"
)

type ProfileHandler struct {
	profileService services.ProfileService
}

func NewProfileHandler(s services.ProfileService) *ProfileHandler {
	return &ProfileHandler{profileService: s}
}

func (h *ProfileHandler) HandleGetProfile(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		logger.ErrorContext(r.Context(), "Missing user context in authenticated request", nil)
		return errors.NewInternalError().WithDetails(map[string]interface{}{
			"issue": "user_context_missing",
		})
	}

	user, err := h.profileService.GetProfile(r.Context(), claims.UserID)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(user)
	return nil
}

func (h *ProfileHandler) HandleUpdateProfile(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		logger.ErrorContext(r.Context(), "Missing user context in authenticated request", nil)
		return errors.NewInternalError().WithDetails(map[string]interface{}{
			"issue": "user_context_missing",
		})
	}

	var req models.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.WarnContext(r.Context(), "Invalid JSON in update profile request", map[string]interface{}{
			"error": err.Error(),
		})
		return errors.NewInvalidJSONError()
	}

	updatedUser, err := h.profileService.UpdateProfile(r.Context(), claims.UserID, req)
	if err != nil {
		return err
	}

	response := map[string]interface{}{
		"message": "Profile updated successfully",
		"user":    updatedUser,
	}

	json.NewEncoder(w).Encode(response)
	return nil
}

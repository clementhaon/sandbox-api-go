package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/middleware"
	"github.com/clementhaon/sandbox-api-go/models"
	"github.com/clementhaon/sandbox-api-go/services"
)

type MediaHandler struct {
	mediaService services.MediaService
}

func NewMediaHandler(s services.MediaService) *MediaHandler {
	return &MediaHandler{mediaService: s}
}

func (h *MediaHandler) HandleGetPresignedUploadURL(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return errors.NewMethodNotAllowedError()
	}

	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		return errors.NewAuthRequiredError()
	}

	var req models.PresignedUploadURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewInvalidJSONError()
	}

	response, err := h.mediaService.GetPresignedUploadURL(r.Context(), claims.UserID, req.Filename, req.MimeType)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	return nil
}

func (h *MediaHandler) HandleConfirmUpload(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return errors.NewMethodNotAllowedError()
	}

	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		return errors.NewInvalidTokenError()
	}

	var req struct {
		ObjectKey        string `json:"object_key"`
		OriginalFilename string `json:"original_filename"`
		MimeType         string `json:"mime_type"`
		BucketName       string `json:"bucket_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewBadRequestError("Invalid request body")
	}

	media, err := h.mediaService.ConfirmUpload(r.Context(), claims.UserID, req.ObjectKey, req.OriginalFilename, req.MimeType, req.BucketName)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(media)
	return nil
}

func (h *MediaHandler) HandleGetUserMedia(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		return errors.NewMethodNotAllowedError()
	}

	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		return errors.NewInternalError().WithDetails(map[string]interface{}{
			"issue": "user_context_missing",
		})
	}

	page := 1
	if pageParam := r.URL.Query().Get("page"); pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		}
	}

	response, err := h.mediaService.ListUserMedia(r.Context(), claims.UserID, page)
	if err != nil {
		return err
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	return nil
}

func (h *MediaHandler) HandleGetMediaByID(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return errors.NewMethodNotAllowedError()
	}

	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		return errors.NewUnauthorizedError("User ID not found in context")
	}

	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 {
		return errors.NewBadRequestError("Invalid media ID")
	}

	mediaID, err := strconv.Atoi(pathParts[len(pathParts)-1])
	if err != nil {
		return errors.NewBadRequestError("Invalid media ID")
	}

	media, err := h.mediaService.GetByID(r.Context(), claims.UserID, mediaID)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(media)
	return nil
}

func (h *MediaHandler) HandleGetPresignedDownloadURL(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return errors.NewMethodNotAllowedError()
	}

	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		return errors.NewUnauthorizedError("User ID not found in context")
	}

	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		return errors.NewBadRequestError("Invalid media ID")
	}

	mediaID, err := strconv.Atoi(pathParts[len(pathParts)-2])
	if err != nil {
		return errors.NewBadRequestError("Invalid media ID")
	}

	response, err := h.mediaService.GetPresignedDownloadURL(r.Context(), claims.UserID, mediaID)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	return nil
}

func (h *MediaHandler) HandleDeleteMedia(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodDelete {
		return errors.NewMethodNotAllowedError()
	}

	claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
	if !ok {
		return errors.NewUnauthorizedError("User ID not found in context")
	}

	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 {
		return errors.NewBadRequestError("Invalid media ID")
	}

	mediaID, err := strconv.Atoi(pathParts[len(pathParts)-1])
	if err != nil {
		return errors.NewBadRequestError("Invalid media ID")
	}

	if err := h.mediaService.Delete(r.Context(), claims.UserID, mediaID); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Media %d deleted successfully", mediaID),
	})
	return nil
}

package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/clementhaon/sandbox-api-go/database"
	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/logger"
	"github.com/clementhaon/sandbox-api-go/middleware"
	"github.com/clementhaon/sandbox-api-go/models"
	"github.com/clementhaon/sandbox-api-go/storage"
	"net/http"
	"strconv"
	"strings"
)

func HandleGetPresignedUploadURL(w http.ResponseWriter, r *http.Request) error {
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

	if req.Filename == "" {
		return errors.NewMissingFieldError("Filename is required")
	}

	if req.MimeType == "" {
		return errors.NewMissingFieldError("MIME type is required")
	}

	uploadURL, objectKey, err := storage.GeneratePresignedUploadURL(req.Filename, req.MimeType, claims.UserID)
	if err != nil {
		logger.Error("Failed to generate presigned upload URL", err)
		return errors.NewInternalError()
	}

	response := models.PresignedUploadURLResponse{
		UploadURL: uploadURL,
		ObjectKey: objectKey,
		ExpiresIn: 604800, // 7 jours en secondes (7 * 24 * 60 * 60)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	return nil
}

func HandleConfirmUpload(w http.ResponseWriter, r *http.Request) error {
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

	if req.ObjectKey == "" || req.OriginalFilename == "" || req.MimeType == "" || req.BucketName == "" {
		return errors.NewBadRequestError("Missing required fields")
	}

	objInfo, err := storage.GetObjectInfo(req.ObjectKey)
	if err != nil {
		logger.Error("Failed to get object info", err)
		return errors.NewBadRequestError("Object not found or upload incomplete")
	}

	query := `
		INSERT INTO media (user_id, object_key, bucket_name, original_filename, file_size, mime_type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	var media models.Media
	media.UserID = claims.UserID
	media.ObjectKey = req.ObjectKey
	media.BucketName = req.BucketName
	media.OriginalFilename = req.OriginalFilename
	media.FileSize = objInfo.Size
	media.MimeType = req.MimeType

	err = database.DB.QueryRow(
		query,
		media.UserID,
		media.ObjectKey,
		media.BucketName,
		media.OriginalFilename,
		media.FileSize,
		media.MimeType,
	).Scan(&media.ID, &media.CreatedAt, &media.UpdatedAt)

	if err != nil {
		logger.Error("Failed to save media record", err)
		return errors.NewInternalServerError("Failed to save media record")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(media)
	return nil
}

func HandleGetUserMedia(w http.ResponseWriter, r *http.Request) error {
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

	// Récupérer la page depuis les query parameters (par défaut: 1)
	page := 1
	if pageParam := r.URL.Query().Get("page"); pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		}
	}

	// Limite fixe de 50 items par page
	limit := 50
	offset := (page - 1) * limit

	// Compter le nombre total de médias pour cet utilisateur
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM media WHERE user_id = $1`
	err := database.DB.QueryRow(countQuery, claims.UserID).Scan(&totalCount)
	if err != nil {
		logger.Error("Failed to count media", err)
		return errors.NewInternalServerError("Failed to retrieve media count")
	}

	// Calculer le nombre total de pages
	totalPages := (totalCount + limit - 1) / limit
	if totalPages == 0 {
		totalPages = 1
	}

	// Récupérer les médias avec pagination
	query := `
		SELECT id, user_id, object_key, bucket_name, original_filename, file_size, mime_type, created_at, updated_at
		FROM media
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := database.DB.Query(query, claims.UserID, limit, offset)
	if err != nil {
		logger.Error("Failed to query media", err)
		return errors.NewInternalServerError("Failed to retrieve media")
	}
	defer rows.Close()

	var mediaList []models.Media
	for rows.Next() {
		var media models.Media
		err := rows.Scan(
			&media.ID,
			&media.UserID,
			&media.ObjectKey,
			&media.BucketName,
			&media.OriginalFilename,
			&media.FileSize,
			&media.MimeType,
			&media.CreatedAt,
			&media.UpdatedAt,
		)
		if err != nil {
			logger.Error("Failed to scan media row", err)
			continue
		}
		// Générer une URL présignée pour accéder au fichier
		presignedURL, err := storage.GeneratePresignedDownloadURL(media.ObjectKey)
		if err != nil {
			logger.Error("Failed to generate presigned URL for media", err)
			// En cas d'erreur, on peut mettre une URL vide ou continuer sans cette entrée
			media.URL = ""
		} else {
			media.URL = presignedURL
		}
		mediaList = append(mediaList, media)
	}

	if mediaList == nil {
		mediaList = []models.Media{}
	}

	response := models.MediaListResponse{
		Media:      mediaList,
		Page:       page,
		Limit:      limit,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	return nil
}

func HandleGetMediaByID(w http.ResponseWriter, r *http.Request) error {
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

	var media models.Media
	query := `
		SELECT id, user_id, object_key, bucket_name, original_filename, file_size, mime_type, created_at, updated_at
		FROM media
		WHERE id = $1 AND user_id = $2
	`

	err = database.DB.QueryRow(query, mediaID, claims.UserID).Scan(
		&media.ID,
		&media.UserID,
		&media.ObjectKey,
		&media.BucketName,
		&media.OriginalFilename,
		&media.FileSize,
		&media.MimeType,
		&media.CreatedAt,
		&media.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return errors.NewNotFoundError("Media")
	}
	if err != nil {
		logger.Error("Failed to query media", err)
		return errors.NewInternalServerError("Failed to retrieve media")
	}

	// Générer une URL présignée pour accéder au fichier
	presignedURL, err := storage.GeneratePresignedDownloadURL(media.ObjectKey)
	if err != nil {
		logger.Error("Failed to generate presigned URL for media", err)
		media.URL = ""
	} else {
		media.URL = presignedURL
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(media)
	return nil
}

func HandleGetPresignedDownloadURL(w http.ResponseWriter, r *http.Request) error {
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

	var objectKey string
	query := `SELECT object_key FROM media WHERE id = $1 AND user_id = $2`
	err = database.DB.QueryRow(query, mediaID, claims.UserID).Scan(&objectKey)

	if err == sql.ErrNoRows {
		return errors.NewNotFoundError("Media")
	}
	if err != nil {
		logger.Error("Failed to query media", err)
		return errors.NewInternalServerError("Failed to retrieve media")
	}

	downloadURL, err := storage.GeneratePresignedDownloadURL(objectKey)
	if err != nil {
		logger.Error("Failed to generate presigned download URL", err)
		return errors.NewInternalServerError("Failed to generate download URL")
	}

	response := models.PresignedDownloadURLResponse{
		DownloadURL: downloadURL,
		ExpiresIn:   604800, // 7 jours en secondes (7 * 24 * 60 * 60)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	return nil
}

func HandleDeleteMedia(w http.ResponseWriter, r *http.Request) error {
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

	var objectKey string
	query := `SELECT object_key FROM media WHERE id = $1 AND user_id = $2`
	err = database.DB.QueryRow(query, mediaID, claims.UserID).Scan(&objectKey)

	if err == sql.ErrNoRows {
		return errors.NewNotFoundError("Media")
	}
	if err != nil {
		logger.Error("Failed to query media", err)
		return errors.NewInternalServerError("Failed to retrieve media")
	}

	if err := storage.DeleteObject(objectKey); err != nil {
		logger.Error("Failed to delete object from MinIO", err)
	}

	deleteQuery := `DELETE FROM media WHERE id = $1 AND user_id = $2`
	_, err = database.DB.Exec(deleteQuery, mediaID, claims.UserID)
	if err != nil {
		logger.Error("Failed to delete media record", err)
		return errors.NewInternalServerError("Failed to delete media")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Media %d deleted successfully", mediaID),
	})
	return nil
}

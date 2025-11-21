package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sandbox-api-go/database"
	"sandbox-api-go/errors"
	"sandbox-api-go/logger"
	"sandbox-api-go/models"
	"sandbox-api-go/storage"
	"strconv"
	"strings"
	"sandbox-api-go/middleware"
)

func HandleGetPresignedUploadURL(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return errors.NewMethodNotAllowedError()
	}

	userID, ok := r.Context().Value("user_id").(int)
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

	uploadURL, objectKey, err := storage.GeneratePresignedUploadURL(req.Filename, req.MimeType, userID)
	if err != nil {
		logger.Error("Failed to generate presigned upload URL", err)
		return errors.NewInternalError()
	}

	response := models.PresignedUploadURLResponse{
		UploadURL: uploadURL,
		ObjectKey: objectKey,
		ExpiresIn: 3600,
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

	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		return errors.NewInvalidTokenError()
	}

	var req struct {
		ObjectKey        string `json:"object_key"`
		OriginalFilename string `json:"original_filename"`
		MimeType         string `json:"mime_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewBadRequestError("Invalid request body")
	}

	if req.ObjectKey == "" || req.OriginalFilename == "" || req.MimeType == "" {
		return errors.NewBadRequestError("Missing required fields")
	}

	objInfo, err := storage.GetObjectInfo(req.ObjectKey)
	if err != nil {
		logger.Error("Failed to get object info", err)
		return errors.NewBadRequestError("Object not found or upload incomplete")
	}

	bucketName := objInfo.Key
	if idx := strings.Index(objInfo.Key, "/"); idx > 0 {
		bucketName = "user-uploads"
	}

	query := `
		INSERT INTO media (user_id, object_key, bucket_name, original_filename, file_size, mime_type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	var media models.Media
	media.UserID = userID
	media.ObjectKey = req.ObjectKey
	media.BucketName = bucketName
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
	query := `
		SELECT id, user_id, object_key, bucket_name, original_filename, file_size, mime_type, created_at, updated_at
		FROM media
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := database.DB.Query(query, claims.UserID)
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
		mediaList = append(mediaList, media)
	}

	if mediaList == nil {
		mediaList = []models.Media{}
	}

	response := models.MediaListResponse{
		Media: mediaList,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	return nil
}

func HandleGetMediaByID(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return errors.NewMethodNotAllowedError()
	}

	userID, ok := r.Context().Value("user_id").(int)
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

	err = database.DB.QueryRow(query, mediaID, userID).Scan(
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(media)
	return nil
}

func HandleGetPresignedDownloadURL(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return errors.NewMethodNotAllowedError()
	}

	userID, ok := r.Context().Value("user_id").(int)
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
	err = database.DB.QueryRow(query, mediaID, userID).Scan(&objectKey)

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
		ExpiresIn:   3600,
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

	userID, ok := r.Context().Value("user_id").(int)
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
	err = database.DB.QueryRow(query, mediaID, userID).Scan(&objectKey)

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
	_, err = database.DB.Exec(deleteQuery, mediaID, userID)
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

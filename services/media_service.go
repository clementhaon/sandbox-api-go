package services

import (
	"context"
	"database/sql"
	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/logger"
	"github.com/clementhaon/sandbox-api-go/models"
	"github.com/clementhaon/sandbox-api-go/storage"
)

type MediaService interface {
	GetPresignedUploadURL(ctx context.Context, userID int, filename, mimeType string) (models.PresignedUploadURLResponse, error)
	ConfirmUpload(ctx context.Context, userID int, objectKey, originalFilename, mimeType, bucketName string) (models.Media, error)
	ListUserMedia(ctx context.Context, userID int, page int) (models.MediaListResponse, error)
	GetByID(ctx context.Context, userID int, mediaID int) (models.Media, error)
	GetPresignedDownloadURL(ctx context.Context, userID int, mediaID int) (models.PresignedDownloadURLResponse, error)
	Delete(ctx context.Context, userID int, mediaID int) error
}

type mediaService struct {
	db *sql.DB
}

func NewMediaService(db *sql.DB) MediaService {
	return &mediaService{db: db}
}

func (s *mediaService) GetPresignedUploadURL(ctx context.Context, userID int, filename, mimeType string) (models.PresignedUploadURLResponse, error) {
	if filename == "" {
		return models.PresignedUploadURLResponse{}, errors.NewMissingFieldError("Filename is required")
	}
	if mimeType == "" {
		return models.PresignedUploadURLResponse{}, errors.NewMissingFieldError("MIME type is required")
	}

	uploadURL, objectKey, err := storage.GeneratePresignedUploadURL(filename, mimeType, userID)
	if err != nil {
		logger.Error("Failed to generate presigned upload URL", err)
		return models.PresignedUploadURLResponse{}, errors.NewInternalError()
	}

	return models.PresignedUploadURLResponse{
		UploadURL: uploadURL,
		ObjectKey: objectKey,
		ExpiresIn: 604800, // 7 days in seconds
	}, nil
}

func (s *mediaService) ConfirmUpload(ctx context.Context, userID int, objectKey, originalFilename, mimeType, bucketName string) (models.Media, error) {
	if objectKey == "" || originalFilename == "" || mimeType == "" || bucketName == "" {
		return models.Media{}, errors.NewBadRequestError("Missing required fields")
	}

	objInfo, err := storage.GetObjectInfo(objectKey)
	if err != nil {
		logger.Error("Failed to get object info", err)
		return models.Media{}, errors.NewBadRequestError("Object not found or upload incomplete")
	}

	query := `
		INSERT INTO media (user_id, object_key, bucket_name, original_filename, file_size, mime_type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	var media models.Media
	media.UserID = userID
	media.ObjectKey = objectKey
	media.BucketName = bucketName
	media.OriginalFilename = originalFilename
	media.FileSize = objInfo.Size
	media.MimeType = mimeType

	err = s.db.QueryRow(
		query,
		media.UserID, media.ObjectKey, media.BucketName,
		media.OriginalFilename, media.FileSize, media.MimeType,
	).Scan(&media.ID, &media.CreatedAt, &media.UpdatedAt)

	if err != nil {
		logger.Error("Failed to save media record", err)
		return models.Media{}, errors.NewInternalServerError("Failed to save media record")
	}

	return media, nil
}

func (s *mediaService) ListUserMedia(ctx context.Context, userID int, page int) (models.MediaListResponse, error) {
	if page < 1 {
		page = 1
	}
	limit := 50
	offset := (page - 1) * limit

	var totalCount int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM media WHERE user_id = $1`, userID).Scan(&totalCount)
	if err != nil {
		logger.Error("Failed to count media", err)
		return models.MediaListResponse{}, errors.NewInternalServerError("Failed to retrieve media count")
	}

	totalPages := (totalCount + limit - 1) / limit
	if totalPages == 0 {
		totalPages = 1
	}

	rows, err := s.db.Query(`
		SELECT id, user_id, object_key, bucket_name, original_filename, file_size, mime_type, created_at, updated_at
		FROM media
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		logger.Error("Failed to query media", err)
		return models.MediaListResponse{}, errors.NewInternalServerError("Failed to retrieve media")
	}
	defer rows.Close()

	var mediaList []models.Media
	for rows.Next() {
		var media models.Media
		err := rows.Scan(
			&media.ID, &media.UserID, &media.ObjectKey, &media.BucketName,
			&media.OriginalFilename, &media.FileSize, &media.MimeType,
			&media.CreatedAt, &media.UpdatedAt,
		)
		if err != nil {
			logger.Error("Failed to scan media row", err)
			continue
		}
		presignedURL, err := storage.GeneratePresignedDownloadURL(media.ObjectKey)
		if err != nil {
			logger.Error("Failed to generate presigned URL for media", err)
			media.URL = ""
		} else {
			media.URL = presignedURL
		}
		mediaList = append(mediaList, media)
	}

	if mediaList == nil {
		mediaList = []models.Media{}
	}

	return models.MediaListResponse{
		Media:      mediaList,
		Page:       page,
		Limit:      limit,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}

func (s *mediaService) GetByID(ctx context.Context, userID int, mediaID int) (models.Media, error) {
	var media models.Media
	err := s.db.QueryRow(`
		SELECT id, user_id, object_key, bucket_name, original_filename, file_size, mime_type, created_at, updated_at
		FROM media
		WHERE id = $1 AND user_id = $2
	`, mediaID, userID).Scan(
		&media.ID, &media.UserID, &media.ObjectKey, &media.BucketName,
		&media.OriginalFilename, &media.FileSize, &media.MimeType,
		&media.CreatedAt, &media.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return models.Media{}, errors.NewNotFoundError("Media")
	}
	if err != nil {
		logger.Error("Failed to query media", err)
		return models.Media{}, errors.NewInternalServerError("Failed to retrieve media")
	}

	presignedURL, err := storage.GeneratePresignedDownloadURL(media.ObjectKey)
	if err != nil {
		logger.Error("Failed to generate presigned URL for media", err)
		media.URL = ""
	} else {
		media.URL = presignedURL
	}

	return media, nil
}

func (s *mediaService) GetPresignedDownloadURL(ctx context.Context, userID int, mediaID int) (models.PresignedDownloadURLResponse, error) {
	var objectKey string
	err := s.db.QueryRow(`SELECT object_key FROM media WHERE id = $1 AND user_id = $2`, mediaID, userID).Scan(&objectKey)

	if err == sql.ErrNoRows {
		return models.PresignedDownloadURLResponse{}, errors.NewNotFoundError("Media")
	}
	if err != nil {
		logger.Error("Failed to query media", err)
		return models.PresignedDownloadURLResponse{}, errors.NewInternalServerError("Failed to retrieve media")
	}

	downloadURL, err := storage.GeneratePresignedDownloadURL(objectKey)
	if err != nil {
		logger.Error("Failed to generate presigned download URL", err)
		return models.PresignedDownloadURLResponse{}, errors.NewInternalServerError("Failed to generate download URL")
	}

	return models.PresignedDownloadURLResponse{
		DownloadURL: downloadURL,
		ExpiresIn:   604800, // 7 days in seconds
	}, nil
}

func (s *mediaService) Delete(ctx context.Context, userID int, mediaID int) error {
	var objectKey string
	err := s.db.QueryRow(`SELECT object_key FROM media WHERE id = $1 AND user_id = $2`, mediaID, userID).Scan(&objectKey)

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

	_, err = s.db.Exec(`DELETE FROM media WHERE id = $1 AND user_id = $2`, mediaID, userID)
	if err != nil {
		logger.Error("Failed to delete media record", err)
		return errors.NewInternalServerError("Failed to delete media")
	}

	return nil
}

package repository

import (
	"context"
	"database/sql"

	"github.com/clementhaon/sandbox-api-go/database"
	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/logger"
	"github.com/clementhaon/sandbox-api-go/models"
)

type MediaRepository interface {
	Create(ctx context.Context, userID int, objectKey, bucketName, originalFilename, mimeType string, fileSize int64) (models.Media, error)
	Count(ctx context.Context, userID int) (int, error)
	List(ctx context.Context, userID int, limit, offset int) ([]models.Media, error)
	GetByID(ctx context.Context, userID int, mediaID int) (models.Media, error)
	GetObjectKey(ctx context.Context, userID int, mediaID int) (string, error)
	Delete(ctx context.Context, userID int, mediaID int) error
	WithQuerier(q database.Querier) MediaRepository
}

type postgresMediaRepo struct {
	db database.Querier
}

func NewPostgresMediaRepository(db *sql.DB) MediaRepository {
	return &postgresMediaRepo{db: db}
}

func (r *postgresMediaRepo) WithQuerier(q database.Querier) MediaRepository {
	return &postgresMediaRepo{db: q}
}

func (r *postgresMediaRepo) Create(ctx context.Context, userID int, objectKey, bucketName, originalFilename, mimeType string, fileSize int64) (models.Media, error) {
	var media models.Media
	media.UserID = userID
	media.ObjectKey = objectKey
	media.BucketName = bucketName
	media.OriginalFilename = originalFilename
	media.FileSize = fileSize
	media.MimeType = mimeType

	err := r.db.QueryRowContext(ctx, `
		INSERT INTO media (user_id, object_key, bucket_name, original_filename, file_size, mime_type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`, userID, objectKey, bucketName, originalFilename, fileSize, mimeType).
		Scan(&media.ID, &media.CreatedAt, &media.UpdatedAt)

	if err != nil {
		logger.Error("Failed to save media record", err)
		return models.Media{}, errors.NewInternalServerError("Failed to save media record")
	}
	return media, nil
}

func (r *postgresMediaRepo) Count(ctx context.Context, userID int) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM media WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		logger.Error("Failed to count media", err)
		return 0, errors.NewInternalServerError("Failed to retrieve media count")
	}
	return count, nil
}

func (r *postgresMediaRepo) List(ctx context.Context, userID int, limit, offset int) ([]models.Media, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, object_key, bucket_name, original_filename, file_size, mime_type, created_at, updated_at
		FROM media
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		logger.Error("Failed to query media", err)
		return nil, errors.NewInternalServerError("Failed to retrieve media")
	}
	defer rows.Close()

	var mediaList []models.Media
	for rows.Next() {
		var m models.Media
		if err := rows.Scan(&m.ID, &m.UserID, &m.ObjectKey, &m.BucketName, &m.OriginalFilename, &m.FileSize, &m.MimeType, &m.CreatedAt, &m.UpdatedAt); err != nil {
			logger.Error("Failed to scan media row", err)
			continue
		}
		mediaList = append(mediaList, m)
	}
	return mediaList, nil
}

func (r *postgresMediaRepo) GetByID(ctx context.Context, userID int, mediaID int) (models.Media, error) {
	var m models.Media
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, object_key, bucket_name, original_filename, file_size, mime_type, created_at, updated_at
		FROM media
		WHERE id = $1 AND user_id = $2
	`, mediaID, userID).Scan(&m.ID, &m.UserID, &m.ObjectKey, &m.BucketName, &m.OriginalFilename, &m.FileSize, &m.MimeType, &m.CreatedAt, &m.UpdatedAt)

	if err == sql.ErrNoRows {
		return models.Media{}, errors.NewNotFoundError("Media")
	}
	if err != nil {
		logger.Error("Failed to query media", err)
		return models.Media{}, errors.NewInternalServerError("Failed to retrieve media")
	}
	return m, nil
}

func (r *postgresMediaRepo) GetObjectKey(ctx context.Context, userID int, mediaID int) (string, error) {
	var objectKey string
	err := r.db.QueryRowContext(ctx, `SELECT object_key FROM media WHERE id = $1 AND user_id = $2`, mediaID, userID).Scan(&objectKey)

	if err == sql.ErrNoRows {
		return "", errors.NewNotFoundError("Media")
	}
	if err != nil {
		logger.Error("Failed to query media", err)
		return "", errors.NewInternalServerError("Failed to retrieve media")
	}
	return objectKey, nil
}

func (r *postgresMediaRepo) Delete(ctx context.Context, userID int, mediaID int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM media WHERE id = $1 AND user_id = $2`, mediaID, userID)
	if err != nil {
		logger.Error("Failed to delete media record", err)
		return errors.NewInternalServerError("Failed to delete media")
	}
	return nil
}

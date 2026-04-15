package services

import (
	"context"

	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/logger"
	"github.com/clementhaon/sandbox-api-go/models"
	"github.com/clementhaon/sandbox-api-go/repository"
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
	mediaRepo repository.MediaRepository
	storage   storage.StorageClient
}

func NewMediaService(mediaRepo repository.MediaRepository, storage storage.StorageClient) MediaService {
	return &mediaService{mediaRepo: mediaRepo, storage: storage}
}

func (s *mediaService) GetPresignedUploadURL(ctx context.Context, userID int, filename, mimeType string) (models.PresignedUploadURLResponse, error) {
	if filename == "" {
		return models.PresignedUploadURLResponse{}, errors.NewMissingFieldError("Filename is required")
	}
	if mimeType == "" {
		return models.PresignedUploadURLResponse{}, errors.NewMissingFieldError("MIME type is required")
	}

	uploadURL, objectKey, err := s.storage.GeneratePresignedUploadURL(filename, mimeType, userID)
	if err != nil {
		logger.Error("Failed to generate presigned upload URL", err)
		return models.PresignedUploadURLResponse{}, errors.NewInternalError()
	}

	return models.PresignedUploadURLResponse{
		UploadURL: uploadURL,
		ObjectKey: objectKey,
		ExpiresIn: 604800,
	}, nil
}

func (s *mediaService) ConfirmUpload(ctx context.Context, userID int, objectKey, originalFilename, mimeType, bucketName string) (models.Media, error) {
	if objectKey == "" || originalFilename == "" || mimeType == "" || bucketName == "" {
		return models.Media{}, errors.NewBadRequestError("Missing required fields")
	}

	objInfo, err := s.storage.GetObjectInfo(objectKey)
	if err != nil {
		logger.Error("Failed to get object info", err)
		return models.Media{}, errors.NewBadRequestError("Object not found or upload incomplete")
	}

	return s.mediaRepo.Create(ctx, userID, objectKey, bucketName, originalFilename, mimeType, objInfo.Size)
}

func (s *mediaService) ListUserMedia(ctx context.Context, userID int, page int) (models.MediaListResponse, error) {
	if page < 1 {
		page = 1
	}
	limit := 50
	offset := (page - 1) * limit

	totalCount, err := s.mediaRepo.Count(ctx, userID)
	if err != nil {
		return models.MediaListResponse{}, err
	}

	totalPages := (totalCount + limit - 1) / limit
	if totalPages == 0 {
		totalPages = 1
	}

	mediaList, err := s.mediaRepo.List(ctx, userID, limit, offset)
	if err != nil {
		return models.MediaListResponse{}, err
	}

	for i := range mediaList {
		presignedURL, err := s.storage.GeneratePresignedDownloadURL(mediaList[i].ObjectKey)
		if err != nil {
			logger.Error("Failed to generate presigned URL for media", err)
			mediaList[i].URL = ""
		} else {
			mediaList[i].URL = presignedURL
		}
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
	media, err := s.mediaRepo.GetByID(ctx, userID, mediaID)
	if err != nil {
		return models.Media{}, err
	}

	presignedURL, err := s.storage.GeneratePresignedDownloadURL(media.ObjectKey)
	if err != nil {
		logger.Error("Failed to generate presigned URL for media", err)
		media.URL = ""
	} else {
		media.URL = presignedURL
	}

	return media, nil
}

func (s *mediaService) GetPresignedDownloadURL(ctx context.Context, userID int, mediaID int) (models.PresignedDownloadURLResponse, error) {
	objectKey, err := s.mediaRepo.GetObjectKey(ctx, userID, mediaID)
	if err != nil {
		return models.PresignedDownloadURLResponse{}, err
	}

	downloadURL, err := s.storage.GeneratePresignedDownloadURL(objectKey)
	if err != nil {
		logger.Error("Failed to generate presigned download URL", err)
		return models.PresignedDownloadURLResponse{}, errors.NewInternalServerError("Failed to generate download URL")
	}

	return models.PresignedDownloadURLResponse{
		DownloadURL: downloadURL,
		ExpiresIn:   604800,
	}, nil
}

func (s *mediaService) Delete(ctx context.Context, userID int, mediaID int) error {
	objectKey, err := s.mediaRepo.GetObjectKey(ctx, userID, mediaID)
	if err != nil {
		return err
	}

	if err := s.storage.DeleteObject(objectKey); err != nil {
		logger.Error("Failed to delete object from MinIO", err)
	}

	return s.mediaRepo.Delete(ctx, userID, mediaID)
}

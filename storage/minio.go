package storage

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"github.com/clementhaon/sandbox-api-go/config"
	"github.com/clementhaon/sandbox-api-go/logger"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var MinioClient *minio.Client

func InitMinIO() error {
	endpoint := config.GetEnv("MINIO_ENDPOINT", "minio:9000")
	accessKeyID := config.GetEnv("MINIO_ROOT_USER", "minioadmin")
	secretAccessKey := config.GetEnv("MINIO_ROOT_PASSWORD", "minioadmin123")
	useSSL := config.GetEnv("MINIO_USE_SSL", "false") == "true"
	logger.Info("Initializing MinIO client", map[string]interface{}{
		"endpoint":        endpoint,
		"accessKeyID":     accessKeyID,
		"secretAccessKey": secretAccessKey,
		"useSSL":          useSSL,
	})
	var err error
	MinioClient, err = minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize MinIO client: %v", err)
	}

	logger.Info("MinIO client initialized successfully")

	if err := ensureBucketExists(); err != nil {
		return fmt.Errorf("failed to ensure bucket exists: %v", err)
	}

	return nil
}

func ensureBucketExists() error {
	ctx := context.Background()
	bucketName := config.GetEnv("MINIO_BUCKET", "user-uploads")

	exists, err := MinioClient.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %v", err)
	}

	if !exists {
		err = MinioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %v", err)
		}
		logger.Info(fmt.Sprintf("Bucket '%s' created successfully", bucketName))
	} else {
		logger.Info(fmt.Sprintf("Bucket '%s' already exists", bucketName))
	}

	return nil
}

func GeneratePresignedUploadURL(filename, mimeType string, userID int) (string, string, error) {
	ctx := context.Background()
	bucketName := config.GetEnv("MINIO_BUCKET", "user-uploads")

	ext := filepath.Ext(filename)
	baseFilename := strings.TrimSuffix(filename, ext)

	sanitizedBase := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, baseFilename)

	objectKey := fmt.Sprintf("users/%d/%s-%s%s", userID, sanitizedBase, uuid.New().String()[:8], ext)

	reqParams := make(url.Values)
	reqParams.Set("response-content-type", mimeType)

	presignedURL, err := MinioClient.PresignedPutObject(ctx, bucketName, objectKey, 7*24*time.Hour)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate presigned upload URL: %v", err)
	}

	return presignedURL.String(), objectKey, nil
}

func GeneratePresignedDownloadURL(objectKey string) (string, error) {
	ctx := context.Background()
	bucketName := config.GetEnv("MINIO_BUCKET", "user-uploads")

	reqParams := make(url.Values)
	reqParams.Set("response-content-disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(objectKey)))

	presignedURL, err := MinioClient.PresignedGetObject(ctx, bucketName, objectKey, 7*24*time.Hour, reqParams)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned download URL: %v", err)
	}

	return presignedURL.String(), nil
}

func DeleteObject(objectKey string) error {
	ctx := context.Background()
	bucketName := config.GetEnv("MINIO_BUCKET", "user-uploads")

	err := MinioClient.RemoveObject(ctx, bucketName, objectKey, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %v", err)
	}

	logger.Info(fmt.Sprintf("Object '%s' deleted successfully", objectKey))
	return nil
}

func GetObjectInfo(objectKey string) (*minio.ObjectInfo, error) {
	ctx := context.Background()
	bucketName := config.GetEnv("MINIO_BUCKET", "user-uploads")

	objInfo, err := MinioClient.StatObject(ctx, bucketName, objectKey, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object info: %v", err)
	}

	return &objInfo, nil
}


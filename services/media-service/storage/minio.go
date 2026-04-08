package storage

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/clementhaon/sandbox-api-go/pkg/logger"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Storage wraps a MinIO client with a default bucket.
type Storage struct {
	client     *minio.Client
	bucketName string
}

// NewStorage creates and returns a configured Storage, ensuring the bucket exists.
func NewStorage(endpoint, accessKey, secretKey, bucketName string, useSSL bool) (*Storage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MinIO client: %w", err)
	}

	s := &Storage{client: client, bucketName: bucketName}

	if err := s.ensureBucketExists(); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	logger.Info("MinIO client initialized successfully", map[string]interface{}{
		"endpoint": endpoint,
		"bucket":   bucketName,
	})

	return s, nil
}

func (s *Storage) ensureBucketExists() error {
	ctx := context.Background()

	exists, err := s.client.BucketExists(ctx, s.bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		if err := s.client.MakeBucket(ctx, s.bucketName, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		logger.Info(fmt.Sprintf("Bucket '%s' created successfully", s.bucketName))
	}

	return nil
}

func (s *Storage) GeneratePresignedUploadURL(filename, mimeType string, userID int) (string, string, error) {
	ctx := context.Background()

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

	presignedURL, err := s.client.PresignedPutObject(ctx, s.bucketName, objectKey, 7*24*time.Hour)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate presigned upload URL: %w", err)
	}

	return presignedURL.String(), objectKey, nil
}

func (s *Storage) GeneratePresignedDownloadURL(objectKey string) (string, error) {
	ctx := context.Background()

	reqParams := make(url.Values)
	reqParams.Set("response-content-disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(objectKey)))

	presignedURL, err := s.client.PresignedGetObject(ctx, s.bucketName, objectKey, 7*24*time.Hour, reqParams)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned download URL: %w", err)
	}

	return presignedURL.String(), nil
}

func (s *Storage) DeleteObject(objectKey string) error {
	ctx := context.Background()

	if err := s.client.RemoveObject(ctx, s.bucketName, objectKey, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	logger.Info(fmt.Sprintf("Object '%s' deleted successfully", objectKey))
	return nil
}

func (s *Storage) GetObjectInfo(objectKey string) (*minio.ObjectInfo, error) {
	ctx := context.Background()

	objInfo, err := s.client.StatObject(ctx, s.bucketName, objectKey, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object info: %w", err)
	}

	return &objInfo, nil
}

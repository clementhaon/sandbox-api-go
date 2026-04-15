package mocks

import (
	"github.com/minio/minio-go/v7"
)

// MockStorage implements storage.StorageClient for testing.
type MockStorage struct {
	GeneratePresignedUploadURLFn   func(filename, mimeType string, userID int) (string, string, error)
	GeneratePresignedDownloadURLFn func(objectKey string) (string, error)
	DeleteObjectFn                 func(objectKey string) error
	GetObjectInfoFn                func(objectKey string) (*minio.ObjectInfo, error)
}

func (m *MockStorage) GeneratePresignedUploadURL(filename, mimeType string, userID int) (string, string, error) {
	return m.GeneratePresignedUploadURLFn(filename, mimeType, userID)
}

func (m *MockStorage) GeneratePresignedDownloadURL(objectKey string) (string, error) {
	return m.GeneratePresignedDownloadURLFn(objectKey)
}

func (m *MockStorage) DeleteObject(objectKey string) error {
	return m.DeleteObjectFn(objectKey)
}

func (m *MockStorage) GetObjectInfo(objectKey string) (*minio.ObjectInfo, error) {
	return m.GetObjectInfoFn(objectKey)
}

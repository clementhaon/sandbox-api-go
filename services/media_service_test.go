package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/clementhaon/sandbox-api-go/mocks"
	"github.com/clementhaon/sandbox-api-go/models"
	"github.com/minio/minio-go/v7"
)

func TestMediaService_GetPresignedUploadURL(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		mimeType string
		uploadFn func(filename, mimeType string, userID int) (string, string, error)
		wantErr  bool
	}{
		{
			name:     "success",
			filename: "test.png",
			mimeType: "image/png",
			uploadFn: func(filename, mimeType string, userID int) (string, string, error) {
				return "https://example.com/upload", "key123", nil
			},
		},
		{
			name:     "empty filename",
			filename: "",
			mimeType: "image/png",
			wantErr:  true,
		},
		{
			name:     "empty mime type",
			filename: "test.png",
			mimeType: "",
			wantErr:  true,
		},
		{
			name:     "storage error",
			filename: "test.png",
			mimeType: "image/png",
			uploadFn: func(filename, mimeType string, userID int) (string, string, error) {
				return "", "", fmt.Errorf("storage error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &mocks.MockStorage{GeneratePresignedUploadURLFn: tt.uploadFn}
			repo := &mocks.MockMediaRepository{}
			svc := NewMediaService(repo, storage)

			resp, err := svc.GetPresignedUploadURL(context.Background(), 1, tt.filename, tt.mimeType)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.UploadURL == "" {
				t.Error("expected non-empty upload URL")
			}
			if resp.ExpiresIn != 604800 {
				t.Errorf("expected expiresIn 604800, got %d", resp.ExpiresIn)
			}
		})
	}
}

func TestMediaService_ConfirmUpload(t *testing.T) {
	tests := []struct {
		name        string
		objectKey   string
		origFile    string
		mimeType    string
		bucket      string
		getInfoFn   func(objectKey string) (*minio.ObjectInfo, error)
		createFn    func(ctx context.Context, userID int, objectKey, bucketName, originalFilename, mimeType string, fileSize int64) (models.Media, error)
		wantErr     bool
	}{
		{
			name:      "success",
			objectKey: "key1",
			origFile:  "test.png",
			mimeType:  "image/png",
			bucket:    "mybucket",
			getInfoFn: func(objectKey string) (*minio.ObjectInfo, error) {
				return &minio.ObjectInfo{Size: 1024}, nil
			},
			createFn: func(ctx context.Context, userID int, objectKey, bucketName, originalFilename, mimeType string, fileSize int64) (models.Media, error) {
				return models.Media{ID: 1, UserID: userID, ObjectKey: objectKey, FileSize: fileSize}, nil
			},
		},
		{
			name:     "missing fields",
			objectKey: "",
			origFile: "test.png",
			mimeType: "image/png",
			bucket:   "mybucket",
			wantErr:  true,
		},
		{
			name:      "object not found",
			objectKey: "key1",
			origFile:  "test.png",
			mimeType:  "image/png",
			bucket:    "mybucket",
			getInfoFn: func(objectKey string) (*minio.ObjectInfo, error) {
				return nil, fmt.Errorf("not found")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &mocks.MockStorage{GetObjectInfoFn: tt.getInfoFn}
			repo := &mocks.MockMediaRepository{CreateFn: tt.createFn}
			svc := NewMediaService(repo, storage)

			media, err := svc.ConfirmUpload(context.Background(), 1, tt.objectKey, tt.origFile, tt.mimeType, tt.bucket)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if media.ID != 1 {
				t.Errorf("expected media ID 1, got %d", media.ID)
			}
		})
	}
}

func TestMediaService_ListUserMedia(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		countFn    func(ctx context.Context, userID int) (int, error)
		listFn     func(ctx context.Context, userID int, limit, offset int) ([]models.Media, error)
		downloadFn func(objectKey string) (string, error)
		wantErr    bool
		wantPage   int
	}{
		{
			name: "success",
			page: 1,
			countFn: func(ctx context.Context, userID int) (int, error) {
				return 2, nil
			},
			listFn: func(ctx context.Context, userID int, limit, offset int) ([]models.Media, error) {
				return []models.Media{{ID: 1, ObjectKey: "key1"}, {ID: 2, ObjectKey: "key2"}}, nil
			},
			downloadFn: func(objectKey string) (string, error) {
				return "https://example.com/" + objectKey, nil
			},
			wantPage: 1,
		},
		{
			name: "page defaults to 1 when 0",
			page: 0,
			countFn: func(ctx context.Context, userID int) (int, error) {
				return 0, nil
			},
			listFn: func(ctx context.Context, userID int, limit, offset int) ([]models.Media, error) {
				return nil, nil
			},
			downloadFn: func(objectKey string) (string, error) {
				return "", nil
			},
			wantPage: 1,
		},
		{
			name: "count error",
			page: 1,
			countFn: func(ctx context.Context, userID int) (int, error) {
				return 0, fmt.Errorf("db error")
			},
			wantErr: true,
		},
		{
			name: "list error",
			page: 1,
			countFn: func(ctx context.Context, userID int) (int, error) {
				return 1, nil
			},
			listFn: func(ctx context.Context, userID int, limit, offset int) ([]models.Media, error) {
				return nil, fmt.Errorf("db error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &mocks.MockStorage{GeneratePresignedDownloadURLFn: tt.downloadFn}
			repo := &mocks.MockMediaRepository{CountFn: tt.countFn, ListFn: tt.listFn}
			svc := NewMediaService(repo, storage)

			resp, err := svc.ListUserMedia(context.Background(), 1, tt.page)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Page != tt.wantPage {
				t.Errorf("expected page %d, got %d", tt.wantPage, resp.Page)
			}
		})
	}
}

func TestMediaService_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		getByIDFn  func(ctx context.Context, userID int, mediaID int) (models.Media, error)
		downloadFn func(objectKey string) (string, error)
		wantErr    bool
		wantURL    string
	}{
		{
			name: "success",
			getByIDFn: func(ctx context.Context, userID int, mediaID int) (models.Media, error) {
				return models.Media{ID: mediaID, ObjectKey: "key1"}, nil
			},
			downloadFn: func(objectKey string) (string, error) {
				return "https://example.com/dl", nil
			},
			wantURL: "https://example.com/dl",
		},
		{
			name: "not found",
			getByIDFn: func(ctx context.Context, userID int, mediaID int) (models.Media, error) {
				return models.Media{}, fmt.Errorf("not found")
			},
			wantErr: true,
		},
		{
			name: "presigned url error returns media with empty url",
			getByIDFn: func(ctx context.Context, userID int, mediaID int) (models.Media, error) {
				return models.Media{ID: 1, ObjectKey: "key1"}, nil
			},
			downloadFn: func(objectKey string) (string, error) {
				return "", fmt.Errorf("storage error")
			},
			wantURL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &mocks.MockStorage{GeneratePresignedDownloadURLFn: tt.downloadFn}
			repo := &mocks.MockMediaRepository{GetByIDFn: tt.getByIDFn}
			svc := NewMediaService(repo, storage)

			media, err := svc.GetByID(context.Background(), 1, 5)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if media.URL != tt.wantURL {
				t.Errorf("expected URL %q, got %q", tt.wantURL, media.URL)
			}
		})
	}
}

func TestMediaService_GetPresignedDownloadURL(t *testing.T) {
	tests := []struct {
		name         string
		getObjectKeyFn func(ctx context.Context, userID int, mediaID int) (string, error)
		downloadFn   func(objectKey string) (string, error)
		wantErr      bool
	}{
		{
			name: "success",
			getObjectKeyFn: func(ctx context.Context, userID int, mediaID int) (string, error) {
				return "key1", nil
			},
			downloadFn: func(objectKey string) (string, error) {
				return "https://example.com/dl", nil
			},
		},
		{
			name: "object key not found",
			getObjectKeyFn: func(ctx context.Context, userID int, mediaID int) (string, error) {
				return "", fmt.Errorf("not found")
			},
			wantErr: true,
		},
		{
			name: "storage error",
			getObjectKeyFn: func(ctx context.Context, userID int, mediaID int) (string, error) {
				return "key1", nil
			},
			downloadFn: func(objectKey string) (string, error) {
				return "", fmt.Errorf("storage error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &mocks.MockStorage{GeneratePresignedDownloadURLFn: tt.downloadFn}
			repo := &mocks.MockMediaRepository{GetObjectKeyFn: tt.getObjectKeyFn}
			svc := NewMediaService(repo, storage)

			resp, err := svc.GetPresignedDownloadURL(context.Background(), 1, 5)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.DownloadURL == "" {
				t.Error("expected non-empty download URL")
			}
			if resp.ExpiresIn != 604800 {
				t.Errorf("expected expiresIn 604800, got %d", resp.ExpiresIn)
			}
		})
	}
}

func TestMediaService_Delete(t *testing.T) {
	tests := []struct {
		name           string
		getObjectKeyFn func(ctx context.Context, userID int, mediaID int) (string, error)
		deleteObjFn    func(objectKey string) error
		deleteRepoFn   func(ctx context.Context, userID int, mediaID int) error
		wantErr        bool
	}{
		{
			name: "success",
			getObjectKeyFn: func(ctx context.Context, userID int, mediaID int) (string, error) {
				return "key1", nil
			},
			deleteObjFn: func(objectKey string) error {
				return nil
			},
			deleteRepoFn: func(ctx context.Context, userID int, mediaID int) error {
				return nil
			},
		},
		{
			name: "object key not found",
			getObjectKeyFn: func(ctx context.Context, userID int, mediaID int) (string, error) {
				return "", fmt.Errorf("not found")
			},
			wantErr: true,
		},
		{
			name: "storage delete error still deletes from repo",
			getObjectKeyFn: func(ctx context.Context, userID int, mediaID int) (string, error) {
				return "key1", nil
			},
			deleteObjFn: func(objectKey string) error {
				return fmt.Errorf("storage error")
			},
			deleteRepoFn: func(ctx context.Context, userID int, mediaID int) error {
				return nil
			},
		},
		{
			name: "repo delete error",
			getObjectKeyFn: func(ctx context.Context, userID int, mediaID int) (string, error) {
				return "key1", nil
			},
			deleteObjFn: func(objectKey string) error {
				return nil
			},
			deleteRepoFn: func(ctx context.Context, userID int, mediaID int) error {
				return fmt.Errorf("db error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &mocks.MockStorage{DeleteObjectFn: tt.deleteObjFn}
			repo := &mocks.MockMediaRepository{
				GetObjectKeyFn: tt.getObjectKeyFn,
				DeleteFn:       tt.deleteRepoFn,
			}
			svc := NewMediaService(repo, storage)

			err := svc.Delete(context.Background(), 1, 5)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

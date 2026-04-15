package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/mocks"
	"github.com/clementhaon/sandbox-api-go/models"
)

func TestMediaHandler_HandleGetPresignedUploadURL(t *testing.T) {
	tests := []struct {
		name       string
		userID     int
		withCtx    bool
		body       interface{}
		uploadFn   func(ctx context.Context, userID int, filename, mimeType string) (models.PresignedUploadURLResponse, error)
		wantStatus int
		wantErr    bool
	}{
		{
			name:    "success",
			userID:  1,
			withCtx: true,
			body:    models.PresignedUploadURLRequest{Filename: "test.png", MimeType: "image/png"},
			uploadFn: func(ctx context.Context, userID int, filename, mimeType string) (models.PresignedUploadURLResponse, error) {
				return models.PresignedUploadURLResponse{UploadURL: "https://example.com/upload", ObjectKey: "key123", ExpiresIn: 604800}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:    "no user context",
			withCtx: false,
			body:    models.PresignedUploadURLRequest{Filename: "test.png", MimeType: "image/png"},
			wantErr: true,
		},
		{
			name:    "invalid json",
			userID:  1,
			withCtx: true,
			body:    "bad",
			wantErr: true,
		},
		{
			name:    "service error",
			userID:  1,
			withCtx: true,
			body:    models.PresignedUploadURLRequest{Filename: "", MimeType: "image/png"},
			uploadFn: func(ctx context.Context, userID int, filename, mimeType string) (models.PresignedUploadURLResponse, error) {
				return models.PresignedUploadURLResponse{}, errors.NewMissingFieldError("Filename is required")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mocks.MockMediaService{GetPresignedUploadURLFn: tt.uploadFn}
			handler := NewMediaHandler(svc)

			var bodyBytes []byte
			switch v := tt.body.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				bodyBytes, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPost, "/media/upload-url", bytes.NewReader(bodyBytes))
			if tt.withCtx {
				req = withUserContext(req, tt.userID)
			}
			w := httptest.NewRecorder()

			err := handler.HandleGetPresignedUploadURL(w, req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestMediaHandler_HandleConfirmUpload(t *testing.T) {
	tests := []struct {
		name       string
		userID     int
		withCtx    bool
		body       interface{}
		confirmFn  func(ctx context.Context, userID int, objectKey, originalFilename, mimeType, bucketName string) (models.Media, error)
		wantStatus int
		wantErr    bool
	}{
		{
			name:    "success",
			userID:  1,
			withCtx: true,
			body:    models.ConfirmUploadRequest{ObjectKey: "key1", OriginalFilename: "test.png", MimeType: "image/png", BucketName: "bucket"},
			confirmFn: func(ctx context.Context, userID int, objectKey, originalFilename, mimeType, bucketName string) (models.Media, error) {
				return models.Media{ID: 1, UserID: userID, ObjectKey: objectKey}, nil
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:    "no user context",
			withCtx: false,
			body:    models.ConfirmUploadRequest{ObjectKey: "key1"},
			wantErr: true,
		},
		{
			name:    "invalid json",
			userID:  1,
			withCtx: true,
			body:    "bad",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mocks.MockMediaService{ConfirmUploadFn: tt.confirmFn}
			handler := NewMediaHandler(svc)

			var bodyBytes []byte
			switch v := tt.body.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				bodyBytes, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPost, "/media/confirm", bytes.NewReader(bodyBytes))
			if tt.withCtx {
				req = withUserContext(req, tt.userID)
			}
			w := httptest.NewRecorder()

			err := handler.HandleConfirmUpload(w, req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestMediaHandler_HandleGetUserMedia(t *testing.T) {
	tests := []struct {
		name       string
		userID     int
		withCtx    bool
		url        string
		listFn     func(ctx context.Context, userID int, page int) (models.MediaListResponse, error)
		wantStatus int
		wantErr    bool
	}{
		{
			name:    "success default page",
			userID:  1,
			withCtx: true,
			url:     "/media",
			listFn: func(ctx context.Context, userID int, page int) (models.MediaListResponse, error) {
				return models.MediaListResponse{Media: []models.Media{}, Page: 1, TotalCount: 0, TotalPages: 1}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:    "success with page param",
			userID:  1,
			withCtx: true,
			url:     "/media?page=2",
			listFn: func(ctx context.Context, userID int, page int) (models.MediaListResponse, error) {
				return models.MediaListResponse{Media: []models.Media{}, Page: 2}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:    "no user context",
			withCtx: false,
			url:     "/media",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mocks.MockMediaService{ListUserMediaFn: tt.listFn}
			handler := NewMediaHandler(svc)

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			if tt.withCtx {
				req = withUserContext(req, tt.userID)
			}
			w := httptest.NewRecorder()

			err := handler.HandleGetUserMedia(w, req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestMediaHandler_HandleGetMediaByID(t *testing.T) {
	tests := []struct {
		name       string
		userID     int
		withCtx    bool
		pathID     string
		getByIDFn  func(ctx context.Context, userID int, mediaID int) (models.Media, error)
		wantStatus int
		wantErr    bool
	}{
		{
			name:    "success",
			userID:  1,
			withCtx: true,
			pathID:  "5",
			getByIDFn: func(ctx context.Context, userID int, mediaID int) (models.Media, error) {
				return models.Media{ID: mediaID, UserID: userID}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:    "no user context",
			withCtx: false,
			pathID:  "5",
			wantErr: true,
		},
		{
			name:    "invalid id",
			userID:  1,
			withCtx: true,
			pathID:  "abc",
			wantErr: true,
		},
		{
			name:    "not found",
			userID:  1,
			withCtx: true,
			pathID:  "999",
			getByIDFn: func(ctx context.Context, userID int, mediaID int) (models.Media, error) {
				return models.Media{}, errors.NewNotFoundError("Media")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mocks.MockMediaService{GetByIDFn: tt.getByIDFn}
			handler := NewMediaHandler(svc)

			req := httptest.NewRequest(http.MethodGet, "/media/"+tt.pathID, nil)
			req.SetPathValue("id", tt.pathID)
			if tt.withCtx {
				req = withUserContext(req, tt.userID)
			}
			w := httptest.NewRecorder()

			err := handler.HandleGetMediaByID(w, req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestMediaHandler_HandleGetPresignedDownloadURL(t *testing.T) {
	tests := []struct {
		name       string
		userID     int
		withCtx    bool
		pathID     string
		downloadFn func(ctx context.Context, userID int, mediaID int) (models.PresignedDownloadURLResponse, error)
		wantStatus int
		wantErr    bool
	}{
		{
			name:    "success",
			userID:  1,
			withCtx: true,
			pathID:  "5",
			downloadFn: func(ctx context.Context, userID int, mediaID int) (models.PresignedDownloadURLResponse, error) {
				return models.PresignedDownloadURLResponse{DownloadURL: "https://example.com/dl", ExpiresIn: 604800}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:    "no user context",
			withCtx: false,
			pathID:  "5",
			wantErr: true,
		},
		{
			name:    "invalid id",
			userID:  1,
			withCtx: true,
			pathID:  "abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mocks.MockMediaService{GetPresignedDownloadURLFn: tt.downloadFn}
			handler := NewMediaHandler(svc)

			req := httptest.NewRequest(http.MethodGet, "/media/"+tt.pathID+"/download", nil)
			req.SetPathValue("id", tt.pathID)
			if tt.withCtx {
				req = withUserContext(req, tt.userID)
			}
			w := httptest.NewRecorder()

			err := handler.HandleGetPresignedDownloadURL(w, req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestMediaHandler_HandleDeleteMedia(t *testing.T) {
	tests := []struct {
		name       string
		userID     int
		withCtx    bool
		pathID     string
		deleteFn   func(ctx context.Context, userID int, mediaID int) error
		wantStatus int
		wantErr    bool
	}{
		{
			name:    "success",
			userID:  1,
			withCtx: true,
			pathID:  "5",
			deleteFn: func(ctx context.Context, userID int, mediaID int) error {
				return nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:    "no user context",
			withCtx: false,
			pathID:  "5",
			wantErr: true,
		},
		{
			name:    "invalid id",
			userID:  1,
			withCtx: true,
			pathID:  "abc",
			wantErr: true,
		},
		{
			name:    "not found",
			userID:  1,
			withCtx: true,
			pathID:  "999",
			deleteFn: func(ctx context.Context, userID int, mediaID int) error {
				return errors.NewNotFoundError("Media")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mocks.MockMediaService{DeleteFn: tt.deleteFn}
			handler := NewMediaHandler(svc)

			req := httptest.NewRequest(http.MethodDelete, "/media/"+tt.pathID, nil)
			req.SetPathValue("id", tt.pathID)
			if tt.withCtx {
				req = withUserContext(req, tt.userID)
			}
			w := httptest.NewRecorder()

			err := handler.HandleDeleteMedia(w, req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

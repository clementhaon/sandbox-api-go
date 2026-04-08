package models

import "time"

type Media struct {
	ID               int       `json:"id"`
	UserID           int       `json:"user_id"`
	ObjectKey        string    `json:"object_key"`
	BucketName       string    `json:"bucket_name"`
	OriginalFilename string    `json:"original_filename"`
	FileSize         int64     `json:"file_size"`
	MimeType         string    `json:"mime_type"`
	URL              string    `json:"url,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type PresignedUploadURLRequest struct {
	Filename string `json:"filename"`
	MimeType string `json:"mime_type"`
}

type PresignedUploadURLResponse struct {
	UploadURL string `json:"upload_url"`
	ObjectKey string `json:"object_key"`
	ExpiresIn int    `json:"expires_in"`
}

type PresignedDownloadURLResponse struct {
	DownloadURL string `json:"download_url"`
	ExpiresIn   int    `json:"expires_in"`
}

type MediaListResponse struct {
	Media      []Media `json:"media"`
	Page       int     `json:"page"`
	Limit      int     `json:"limit"`
	TotalCount int     `json:"total_count"`
	TotalPages int     `json:"total_pages"`
}

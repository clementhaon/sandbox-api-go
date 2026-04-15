package models

import (
	"time"
)

type Media struct {
	ID               int       `json:"id"`
	UserID           int       `json:"userId"`
	ObjectKey        string    `json:"objectKey"`
	BucketName       string    `json:"bucketName"`
	OriginalFilename string    `json:"originalFilename"`
	FileSize         int64     `json:"fileSize"`
	MimeType         string    `json:"mimeType"`
	URL              string    `json:"url,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type PresignedUploadURLRequest struct {
	Filename string `json:"filename"`
	MimeType string `json:"mimeType"`
}

type PresignedUploadURLResponse struct {
	UploadURL string `json:"uploadUrl"`
	ObjectKey string `json:"objectKey"`
	ExpiresIn int    `json:"expiresIn"`
}

type PresignedDownloadURLResponse struct {
	DownloadURL string `json:"downloadUrl"`
	ExpiresIn   int    `json:"expiresIn"`
}

type ConfirmUploadRequest struct {
	ObjectKey        string `json:"objectKey"`
	OriginalFilename string `json:"originalFilename"`
	MimeType         string `json:"mimeType"`
	BucketName       string `json:"bucketName"`
}

type MediaListResponse struct {
	Media      []Media `json:"media"`
	Page       int     `json:"page"`
	Limit      int     `json:"limit"`
	TotalCount int     `json:"totalCount"`
	TotalPages int     `json:"totalPages"`
}

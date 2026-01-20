package models

import "time"

// Column represents a board column
type Column struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Order     int       `json:"order"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CreateColumnRequest represents the request to create a column
type CreateColumnRequest struct {
	Title string `json:"title"`
	Color string `json:"color,omitempty"`
}

// UpdateColumnRequest represents the request to update a column
type UpdateColumnRequest struct {
	Title string `json:"title,omitempty"`
	Color string `json:"color,omitempty"`
}

// ReorderColumnsRequest represents the request to reorder columns
type ReorderColumnsRequest struct {
	ColumnIDs []int `json:"columnIds"`
}

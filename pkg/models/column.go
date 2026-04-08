package models

import "time"

type Column struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Order     int       `json:"order"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CreateColumnRequest struct {
	Title string `json:"title"`
	Color string `json:"color,omitempty"`
}

type UpdateColumnRequest struct {
	Title string `json:"title,omitempty"`
	Color string `json:"color,omitempty"`
}

type ReorderColumnsRequest struct {
	ColumnIDs []int `json:"columnIds"`
}

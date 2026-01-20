package models

import "time"

// TimeEntry represents a time tracking entry for a task
type TimeEntry struct {
	ID          int        `json:"id"`
	TaskID      int        `json:"taskId"`
	UserID      int        `json:"userId"`
	StartTime   time.Time  `json:"startTime"`
	EndTime     *time.Time `json:"endTime,omitempty"`
	Duration    int        `json:"duration"` // in seconds
	Description string     `json:"description,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// CreateTimeEntryRequest represents the request to create a time entry
type CreateTimeEntryRequest struct {
	TaskID      int        `json:"taskId"`
	StartTime   time.Time  `json:"startTime"`
	EndTime     *time.Time `json:"endTime,omitempty"`
	Duration    int        `json:"duration"` // in seconds
	Description string     `json:"description,omitempty"`
}

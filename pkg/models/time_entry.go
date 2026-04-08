package models

import "time"

type TimeEntry struct {
	ID          int        `json:"id"`
	TaskID      int        `json:"taskId"`
	UserID      int        `json:"userId"`
	StartTime   time.Time  `json:"startTime"`
	EndTime     *time.Time `json:"endTime,omitempty"`
	Duration    int        `json:"duration"`
	Description string     `json:"description,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type CreateTimeEntryRequest struct {
	TaskID      int        `json:"taskId"`
	StartTime   time.Time  `json:"startTime"`
	EndTime     *time.Time `json:"endTime,omitempty"`
	Duration    int        `json:"duration"`
	Description string     `json:"description,omitempty"`
}

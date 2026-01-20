package models

import (
	"time"

	"github.com/lib/pq"
)

// UserBrief represents a brief user info for task assignee
type UserBrief struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatarUrl,omitempty"`
}

// Task represents a task in the board
type Task struct {
	ID            int        `json:"id"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	ColumnID      int        `json:"columnId"`
	Order         int        `json:"order"`
	Priority      string     `json:"priority"`
	AssigneeID    *int       `json:"assigneeId,omitempty"`
	Assignee      *UserBrief `json:"assignee,omitempty"`
	Deadline      *time.Time `json:"deadline,omitempty"`
	EstimatedTime int        `json:"estimatedTime"` // in minutes
	TrackedTime   int        `json:"trackedTime"`   // in minutes
	Tags          []string   `json:"tags"`
	CreatedBy     int        `json:"createdBy"`
	UserID        int        `json:"userId"` // owner of the task
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

// TaskDB represents the task as stored in database (with pq.StringArray for tags)
type TaskDB struct {
	ID            int
	Title         string
	Description   string
	ColumnID      int
	Order         int
	Priority      string
	AssigneeID    *int
	Deadline      *time.Time
	EstimatedTime int
	TrackedTime   int
	Tags          pq.StringArray
	CreatedBy     *int
	UserID        int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ToTask converts TaskDB to Task
func (t *TaskDB) ToTask() Task {
	task := Task{
		ID:            t.ID,
		Title:         t.Title,
		Description:   t.Description,
		ColumnID:      t.ColumnID,
		Order:         t.Order,
		Priority:      t.Priority,
		AssigneeID:    t.AssigneeID,
		Deadline:      t.Deadline,
		EstimatedTime: t.EstimatedTime,
		TrackedTime:   t.TrackedTime,
		Tags:          []string{},
		UserID:        t.UserID,
		CreatedAt:     t.CreatedAt,
		UpdatedAt:     t.UpdatedAt,
	}
	if t.CreatedBy != nil {
		task.CreatedBy = *t.CreatedBy
	}
	if t.Tags != nil {
		task.Tags = t.Tags
	}
	return task
}

// CreateTaskRequest represents the request to create a task
type CreateTaskRequest struct {
	Title         string     `json:"title"`
	Description   string     `json:"description,omitempty"`
	ColumnID      int        `json:"columnId"`
	Priority      string     `json:"priority,omitempty"`
	AssigneeID    *int       `json:"assigneeId,omitempty"`
	Deadline      *time.Time `json:"deadline,omitempty"`
	EstimatedTime int        `json:"estimatedTime,omitempty"`
	Tags          []string   `json:"tags,omitempty"`
}

// UpdateTaskRequest represents the request to update a task
type UpdateTaskRequest struct {
	Title         string     `json:"title,omitempty"`
	Description   string     `json:"description,omitempty"`
	ColumnID      int        `json:"columnId,omitempty"`
	Priority      string     `json:"priority,omitempty"`
	AssigneeID    *int       `json:"assigneeId,omitempty"`
	Deadline      *time.Time `json:"deadline,omitempty"`
	EstimatedTime int        `json:"estimatedTime,omitempty"`
	Tags          []string   `json:"tags,omitempty"`
}

// MoveTaskRequest represents the request to move a task
type MoveTaskRequest struct {
	ColumnID int `json:"columnId"`
	Order    int `json:"order"`
}

// ReorderTasksRequest represents the request to reorder tasks in a column
type ReorderTasksRequest struct {
	ColumnID int   `json:"columnId"`
	TaskIDs  []int `json:"taskIds"`
}

// BoardResponse represents the response for GET /tasks/board
type BoardResponse struct {
	Columns []Column `json:"columns"`
	Tasks   []Task   `json:"tasks"`
}

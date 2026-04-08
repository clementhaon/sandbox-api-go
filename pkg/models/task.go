package models

import (
	"time"

	"github.com/lib/pq"
)

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
	EstimatedTime int        `json:"estimatedTime"`
	TrackedTime   int        `json:"trackedTime"`
	Tags          []string   `json:"tags"`
	CreatedBy     int        `json:"createdBy"`
	UserID        int        `json:"userId"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

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

func (t *TaskDB) ToTask() Task {
	task := Task{
		ID: t.ID, Title: t.Title, Description: t.Description, ColumnID: t.ColumnID,
		Order: t.Order, Priority: t.Priority, AssigneeID: t.AssigneeID,
		Deadline: t.Deadline, EstimatedTime: t.EstimatedTime, TrackedTime: t.TrackedTime,
		Tags: []string{}, UserID: t.UserID, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
	if t.CreatedBy != nil {
		task.CreatedBy = *t.CreatedBy
	}
	if t.Tags != nil {
		task.Tags = t.Tags
	}
	return task
}

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

type MoveTaskRequest struct {
	ColumnID int `json:"columnId"`
	Order    int `json:"order"`
}

type ReorderTasksRequest struct {
	ColumnID int   `json:"columnId"`
	TaskIDs  []int `json:"taskIds"`
}

type BoardResponse struct {
	Columns []Column `json:"columns"`
	Tasks   []Task   `json:"tasks"`
}

package models

// UserRole constants
const (
	RoleAdmin   = "admin"
	RoleManager = "manager"
	RoleUser    = "user"
)

// UserStatus constants
const (
	StatusActive   = "active"
	StatusInactive = "inactive"
)

// TaskPriority constants
const (
	PriorityLow    = "low"
	PriorityMedium = "medium"
	PriorityHigh   = "high"
	PriorityUrgent = "urgent"
)

// NotificationType constants
const (
	NotifTaskAssigned  = "task_assigned"
	NotifTaskDeadline  = "task_deadline"
	NotifTaskOverdue   = "task_overdue"
	NotifTaskCompleted = "task_completed"
	NotifTaskComment   = "task_comment"
	NotifMention       = "mention"
	NotifSystem        = "system"
)

// ValidRoles returns all valid user roles
func ValidRoles() []string {
	return []string{RoleAdmin, RoleManager, RoleUser}
}

// ValidStatuses returns all valid user statuses
func ValidStatuses() []string {
	return []string{StatusActive, StatusInactive}
}

// ValidPriorities returns all valid task priorities
func ValidPriorities() []string {
	return []string{PriorityLow, PriorityMedium, PriorityHigh, PriorityUrgent}
}

// ValidNotificationTypes returns all valid notification types
func ValidNotificationTypes() []string {
	return []string{
		NotifTaskAssigned,
		NotifTaskDeadline,
		NotifTaskOverdue,
		NotifTaskCompleted,
		NotifTaskComment,
		NotifMention,
		NotifSystem,
	}
}

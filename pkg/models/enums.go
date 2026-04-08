package models

const (
	RoleAdmin   = "admin"
	RoleManager = "manager"
	RoleUser    = "user"
)

const (
	StatusActive   = "active"
	StatusInactive = "inactive"
)

const (
	PriorityLow    = "low"
	PriorityMedium = "medium"
	PriorityHigh   = "high"
	PriorityUrgent = "urgent"
)

const (
	NotifTaskAssigned  = "task_assigned"
	NotifTaskDeadline  = "task_deadline"
	NotifTaskOverdue   = "task_overdue"
	NotifTaskCompleted = "task_completed"
	NotifTaskComment   = "task_comment"
	NotifMention       = "mention"
	NotifSystem        = "system"
)

func ValidRoles() []string {
	return []string{RoleAdmin, RoleManager, RoleUser}
}

func ValidStatuses() []string {
	return []string{StatusActive, StatusInactive}
}

func ValidPriorities() []string {
	return []string{PriorityLow, PriorityMedium, PriorityHigh, PriorityUrgent}
}

func ValidNotificationTypes() []string {
	return []string{NotifTaskAssigned, NotifTaskDeadline, NotifTaskOverdue, NotifTaskCompleted, NotifTaskComment, NotifMention, NotifSystem}
}

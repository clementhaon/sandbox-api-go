package models

// Task represents a simple task/todo item
type Task struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
	UserID      int    `json:"user_id"` // Pour associer les tâches aux utilisateurs
} 
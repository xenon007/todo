package models

import "time"

// Project describes a scrum project that groups multiple tasks.
type Project struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Task represents a single card in the scrum board.
type Task struct {
	ID          int64     `json:"id"`
	ProjectID   int64     `json:"project_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Position    int64     `json:"position"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ValidTaskStatuses enumerates the statuses supported by the board columns.
var ValidTaskStatuses = map[string]struct{}{
	"todo":        {},
	"in_progress": {},
	"done":        {},
}

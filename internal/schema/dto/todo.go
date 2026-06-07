package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shah-dhwanil/tasker/internal/schema"
)

type Todo struct {
	ID          uuid.UUID      `db:"id"`
	UserID      string         `db:"user_id"`
	CategoryID  *uuid.UUID     `db:"category_id"`
	Title       string         `db:"title"`
	Description *string        `db:"description"`
	Status      string         `db:"status"`
	Priority    int            `db:"priority"`
	DueDate     *time.Time     `db:"due_date"`
	CompletedAt *time.Time     `db:"completed_at"`
	ParentID    *uuid.UUID     `db:"parent_id"`
	Metadata    map[string]any `db:"metadata"`
	IsDeleted   bool           `db:"is_deleted"`
	CreatedAt   time.Time      `db:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at"`
}

type CreateTodoRequest struct {
	UserID      string         `db:"user_id"`
	CategoryID  *uuid.UUID     `db:"category_id"`
	Title       string         `db:"title"`
	Description *string        `db:"description"`
	Status      string         `db:"status"`
	Priority    int            `db:"priority"`
	DueDate     *time.Time     `db:"due_date"`
	ParentID    *uuid.UUID     `db:"parent_id"`
	Metadata    map[string]any `db:"metadata"`
}

type UpdateTodoRequest struct {
	UserID      string         `db:"user_id"`
	CategoryID  *uuid.UUID     `db:"category_id,omitempty"`
	Title       *string        `db:"title,omitempty"`
	Description *string        `db:"description,omitempty"`
	Status      *string        `db:"status,omitempty"`
	Priority    *int           `db:"priority,omitempty"`
	DueDate     schema.Nullable[*time.Time]     `db:"due_date,omitempty"`
	CompletedAt schema.Nullable[*time.Time]     `db:"completed_at,omitempty"`
	ParentID    schema.Nullable[*uuid.UUID]     `db:"parent_id,omitempty"`
	Metadata    *map[string]any `db:"metadata,omitempty"`
}

type GetTodosQuery struct {
	Offset     int `db:"offset"`
	Limit      int `db:"limit"`
	Search     *string `db:"search,omitempty"`
	Status     *string `db:"status,omitempty"`
	CategoryID *uuid.UUID `db:"category_id,omitempty"`
	ParentID   *uuid.UUID `db:"parent_id,omitempty"`
	Priority   *int `db:"priority,omitempty"`
	OrderBy    []string 
}

type TodoListItems struct {
	ID         uuid.UUID  `db:"id"`
	Title      string     `db:"title"`
	Status     string     `db:"status"`
	Priority   int        `db:"priority"`
	DueDate    *time.Time `db:"due_date"`
	CategoryID *uuid.UUID `db:"category_id"`
}

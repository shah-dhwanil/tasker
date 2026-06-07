package schema

import (
	"time"

	"github.com/google/uuid"
	"github.com/shah-dhwanil/tasker/internal/validation"
)

type TodoStatus string

const (
	TodoStatusDraft      TodoStatus = "draft"
	TodoStatusPending    TodoStatus = "pending"
	TodoStatusInProgress TodoStatus = "in_progress"
	TodoStatusCompleted  TodoStatus = "completed"
	TodoStatusArchived   TodoStatus = "archived"
)

type Todo struct {
	ID          uuid.UUID
	UserID      string
	CategoryID  *uuid.UUID
	Title       string
	Description *string
	Status      TodoStatus
	Priority    int
	DueDate     *time.Time
	CompletedAt *time.Time
	ParentID    *uuid.UUID
	Metadata    map[string]any
	IsDeleted   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Children    []TodoListItems
}

type TodoListItems struct {
	ID         uuid.UUID
	Title      string
	Status     string
	Priority   int
	DueDate    *time.Time
	CategoryID *uuid.UUID
}

type CreateTodoRequest struct {
	CategoryID  *uuid.UUID  `json:"category_id" validate:"omitempty"`
	Title       string      `json:"title" validate:"required,max=255"`
	Description *string     `json:"description" validate:"omitempty"`
	Status      string      `json:"status" validate:"required,oneof=draft pending in_progress completed archived"`
	Priority    int         `json:"priority" validate:"required,min=1,max=5"`
	DueDate     *time.Time  `json:"due_date"`
	ParentID    *uuid.UUID  `json:"parent_id"`
	Metadata    map[string]any `json:"metadata"`
}

func (p *CreateTodoRequest) Validate(client validation.ValidatorClient) error {
	return client.Struct(p)
}

type UpdateTodoRequest struct {
	CategoryID  *uuid.UUID              `json:"category_id" validate:"omitempty"`
	Title       *string                 `json:"title" validate:"omitempty,max=255"`
	Description *string                 `json:"description" validate:"omitempty"`
	Status      *string                 `json:"status" validate:"omitempty,oneof=draft pending in_progress completed archived"`
	Priority    *int                    `json:"priority" validate:"omitempty,min=1,max=5"`
	DueDate     Nullable[*time.Time]    `json:"due_date"`
	CompletedAt Nullable[*time.Time]    `json:"completed_at"`
	ParentID    Nullable[*uuid.UUID]    `json:"parent_id"`
	Metadata    *map[string]any         `json:"metadata"`
}

func (p *UpdateTodoRequest) Validate(client validation.ValidatorClient) error {
	return client.Struct(p)
}

type GetTodosQuery struct {
	Page       *int       `query:"page" validate:"omitempty,gte=1" json:"page"`
	Limit      *int       `query:"limit" validate:"omitempty,gte=1,lte=100" json:"limit"`
	Search     *string    `query:"search" validate:"omitempty,max=32" json:"search"`
	Status     *string    `query:"status" validate:"omitempty,oneof=draft pending in_progress completed archived" json:"status"`
	Priority   *int       `query:"priority" validate:"omitempty,min=1,max=5" json:"priority"`
	CategoryID *uuid.UUID `query:"category_id" json:"category_id"`
	ParentID   *uuid.UUID `query:"parent_id" json:"parent_id"`
	OrderBy    []string   `query:"order_by" validate:"omitempty,dive,oneof=title -title +title status -status +status priority -priority +priority due_date -due_date +due_date created_at -created_at +created_at updated_at -updated_at +updated_at" json:"order_by"`
}

func (p *GetTodosQuery) Validate(client validation.ValidatorClient) error {
	return client.Struct(p)
}

func (p *GetTodosQuery) Normalize() (*GetTodosQuery, error) {
	if p.Page == nil {
		defaultPage := 1
		p.Page = &defaultPage
	}
	if p.Limit == nil {
		defaultLimit := 10
		p.Limit = &defaultLimit
	}
	if len(p.OrderBy) == 0 {
		p.OrderBy = []string{"-created_at"}
	}
	return p, nil
}

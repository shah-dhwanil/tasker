package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shah-dhwanil/tasker/internal/schema"
)

type Category struct {
	ID          uuid.UUID `db:"id"`
	Name        string    `db:"name"`
	UserID      string `db:"user_id"`
	Description *string    `db:"description"`
	Metadata    map[string]any    `db:"metadata"`
	CreatedAt   time.Time     `db:"created_at"`
	UpdatedAt   time.Time     `db:"updated_at"`
}


type CreateCategoryRequest struct {
	Name        string    `db:"name"`
	Description *string    `db:"description"`
	Metadata    map[string]any    `db:"metadata"`
}

type UpdateCategoryRequest struct {
	Name        *string    `db:"name,omitempty"`
	Description schema.Nullable[*string]    `db:"description,omitempty"`
	Metadata    *map[string]any    `db:"metadata,omitempty"`
}

type GetCategoriesQuery struct {
	Offset     int `db:"offset" `
	Limit    int `db:"limit"`
	Search  *string `db:"search"`
	OrderBy  []string `db:"order_by"`
}

type CategoriesListItems struct {
	ID		  uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
}
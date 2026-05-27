package schema

import (
	"time"

	"github.com/google/uuid"
	"github.com/shah-dhwanil/tasker/internal/validation"
)

type Category struct {
	ID          uuid.UUID `db:"id"`
	Name        string    `db:"name"`
	UserID      uuid.UUID `db:"user_id"`
	Description *string    `db:"description"`
	Metadata    map[string]any    `db:"metadata"`
	CreatedAt   time.Time     `db:"created_at"`
	UpdatedAt   time.Time     `db:"updated_at"`
}


type CreateCategoryRequest struct {
	Name        string    `json:"name" validate:"required,max=32" db:"name"`
	Description *string    `json:"description" validate:"omitempty,max=256" db:"description"`
	Metadata    map[string]any    `json:"metadata" db:"metadata"`
}

func (payload *CreateCategoryRequest) Validate(client validation.ValidatorClient) error {
	return client.Struct(payload)
}

type CreateCategoryResponse struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description *string    `json:"description" db:"description"`
	Metadata    map[string]any    `json:"metadata" db:"metadata"`
	CreatedAt   time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at" db:"updated_at"`
}

type UpdateCategoryRequest struct {
	Name        *string    `json:"name" validate:"omitempty,max=32" db:"name,omitempty"`
	Description *string    `json:"description" validate:"omitempty,max=256" db:"description,omitempty"`
	Metadata    *map[string]any    `json:"metadata" db:"metadata,omitempty"`
}

func (payload *UpdateCategoryRequest) Validate(client validation.ValidatorClient) error {
	return client.Struct(payload)
}

type UpdateCategoryResponse struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description *string    `json:"description" db:"description"`
	Metadata    map[string]any    `json:"metadata" db:"metadata"`
	CreatedAt   time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at" db:"updated_at"`
}

type GetCategoriesQuery struct {
	Page     *int `query:"page" validate:"gte=1" db:"page" `
	Limit    *int `query:"limit" validate:"gte=1,lte=100" db:"limit"`
	Search  *string `query:"search" validate:"omitempty,max=32" db:"search"`
	OrderBy  []string `query:"order_by" validate:"omitempty, dive,contains=name|contains=created_at|contains=updated_at" db:"order_by"`
}

func (payload *GetCategoriesQuery) Validate(client validation.ValidatorClient) error {
	return client.Struct(payload)
}

func (payload *GetCategoriesQuery) Normalize() *GetCategoriesQuery {
	if payload.Page == nil {
		defaultPage := 1
		payload.Page = &defaultPage
	}

	if payload.Limit == nil {
		defaultLimit := 10
		payload.Limit = &defaultLimit
	}

	if len(payload.OrderBy) == 0 {
		payload.OrderBy = []string{"-created_at"}
	}
	return payload
}	

type GetCategoriesResponse struct {
	ID		  uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
}